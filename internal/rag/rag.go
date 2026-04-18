package rag

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db     *sql.DB
	mu     sync.RWMutex
	dbPath string
}

func NewStore(dbPath string) (*Store, error) {
	if dbPath == "" {
		dir, _ := os.UserHomeDir()
		dbPath = filepath.Join(dir, ".mcp-sqlserver", "knowledge.db")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath+"?cache=shared&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	ks := &Store{db: db, dbPath: dbPath}
	if err := ks.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return ks, nil
}

func (ks *Store) init() error {
	_, err := ks.db.Exec(`
	CREATE TABLE IF NOT EXISTS knowledge (
		id INTEGER PRIMARY KEY,
		schema TEXT,
		name TEXT,
		type TEXT,
		content TEXT,
		hash TEXT UNIQUE,
		created_at TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_hash ON knowledge(hash);
	CREATE INDEX IF NOT EXISTS idx_name ON knowledge(schema, name);
	`)
	return err
}

func (ks *Store) LearnTable(ctx context.Context, schema, name string, data map[string]any) error {
	content := formatContent(schema, name, data)
	hash := hashContent(content)

	ks.mu.Lock()
	defer ks.mu.Unlock()

	_, err := ks.db.ExecContext(ctx, `
	INSERT INTO knowledge (schema, name, type, content, hash, created_at)
	VALUES (?, ?, 'table', ?, ?, ?)
	ON CONFLICT(hash) DO UPDATE SET content = excluded.content, created_at = excluded.created_at
	`, schema, name, content, hash, time.Now().Format(time.RFC3339))
	return err
}

func (ks *Store) LearnRelations(ctx context.Context, schema, name string, rels []map[string]any) error {
	if len(rels) == 0 {
		return nil
	}

	var content string
	for _, r := range rels {
		cols, _ := r["columns"].([]string)
		refs, _ := r["references"].(string)
		if len(cols) > 0 && refs != "" {
			content += fmt.Sprintf("%s.%s -> %s; ", name, cols[0], refs)
		}
	}
	if content == "" {
		return nil
	}
	hash := hashContent(content)

	ks.mu.Lock()
	defer ks.mu.Unlock()

	_, err := ks.db.ExecContext(ctx, `
	INSERT INTO knowledge (schema, name, type, content, hash, created_at)
	VALUES (?, ?, 'relation', ?, ?, ?)
	ON CONFLICT(hash) DO UPDATE SET content = excluded.content
	`, schema, name, content, hash, time.Now().Format(time.RFC3339))
	return err
}

func (ks *Store) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 5
	}

	ks.mu.RLock()
	defer ks.mu.RUnlock()

	rows, err := ks.db.QueryContext(ctx, `
	SELECT schema, name, type, content FROM knowledge ORDER BY created_at DESC LIMIT ?
	`, limit*2)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.Schema, &r.Name, &r.Type, &r.Text); err != nil {
			continue
		}
		score := matchScore(query, r.Text)
		if score > 0 {
			r.Score = score
			results = append(results, r)
		}
		if len(results) >= limit {
			break
		}
	}
	return results, rows.Err()
}

func (ks *Store) GetAllTables(ctx context.Context) ([]SchemaDoc, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	rows, err := ks.db.QueryContext(ctx, `
	SELECT schema, name, created_at FROM knowledge WHERE type = 'table' ORDER BY schema, name
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []SchemaDoc
	for rows.Next() {
		var r SchemaDoc
		if err := rows.Scan(&r.Schema, &r.Name, &r.LastLearned); err != nil {
			continue
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (ks *Store) GetAllRelations(ctx context.Context) ([]SearchResult, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	rows, err := ks.db.QueryContext(ctx, `
	SELECT schema, name, type, content FROM knowledge WHERE type = 'relation'
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.Schema, &r.Name, &r.Type, &r.Text); err != nil {
			continue
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (ks *Store) Stats(ctx context.Context) (map[string]int, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	var tables, relations int
	_ = ks.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM knowledge WHERE type = 'table'`).Scan(&tables)
	_ = ks.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM knowledge WHERE type = 'relation'`).Scan(&relations)
	return map[string]int{
		"tables":    tables,
		"relations": relations,
	}, nil
}

func (ks *Store) Close() error {
	return ks.db.Close()
}

type SearchResult struct {
	Schema string
	Name   string
	Type   string
	Text   string
	Score  float64
}

type SchemaDoc struct {
	Schema      string
	Name        string
	LastLearned string
}

func formatContent(schema, name string, data map[string]any) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Table %s.%s", schema, name))

	if cols, ok := data["columns"].([]interface{}); ok {
		var colStrs []string
		for _, c := range cols {
			colStrs = append(colStrs, fmt.Sprintf("%v", c))
		}
		parts = append(parts, "columns: "+strings.Join(colStrs, ", "))
	}
	if pks, ok := data["primaryKeys"].([]interface{}); ok && len(pks) > 0 {
		parts = append(parts, fmt.Sprintf("PK: %v", pks))
	}
	if fks, ok := data["foreignKeys"].([]interface{}); ok && len(fks) > 0 {
		parts = append(parts, fmt.Sprintf("FK: %v", fks))
	}
	return strings.Join(parts, ". ")
}

func hashContent(s string) string {
	h := uint32(2166136261)
	for _, c := range s {
		h ^= uint32(c)
		h *= 16777619
	}
	return fmt.Sprintf("%x", h)
}

func matchScore(query, text string) float64 {
	query = strings.ToLower(query)
	text = strings.ToLower(text)

	if strings.Contains(text, query) {
		return 1.0
	}
	terms := strings.Fields(query)
	count := 0
	for _, t := range terms {
		if strings.Contains(text, t) {
			count++
		}
	}
	return float64(count) / float64(len(terms))
}
