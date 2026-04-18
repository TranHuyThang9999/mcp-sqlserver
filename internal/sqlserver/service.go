package sqlserver

import (
	"context"
	"database/sql"
	"time"

	"mcp_sqlserver/internal/config"
)

// Service provides methods for querying and executing SQL against SQL Server.
//
// Service wraps a sql.DB connection and provides high-level operations
// for the MCP server, including SQL validation and row result mapping.
//
// The type is safe for concurrent use by multiple goroutines.
type Service struct {
	db  *sql.DB
	cfg config.ServerConfig
}

// NewService creates a new Service instance.
//
//	db is the SQL Server database connection.
//	cfg is the server configuration with query limits and permissions.
//
// Returns a new Service configured with the given parameters.
func NewService(db *sql.DB, cfg config.ServerConfig) *Service {
	s := &Service{db: db, cfg: cfg}
	return s
}

// Health checks the connection to SQL Server and returns server information.
//
// ctx is the context for the query timeout.
//
// Returns a map containing connection status, server name, database name,
// and SQL Server version on success, or an error if the connection fails.
func (s *Service) Health(ctx context.Context) (map[string]any, error) {
	var serverName, version, databaseName string
	err := s.db.QueryRowContext(ctx, "SELECT @@SERVERNAME, @@VERSION, DB_NAME()").Scan(&serverName, &version, &databaseName)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"ok":       true,
		"server":   serverName,
		"database": databaseName,
		"version":  version,
	}, nil
}

// Select executes a read-only SELECT or WITH query.
//
// ctx is the context for the query.
// sqlText is the SELECT query to execute.
// maxRows limits the number of rows returned (0 uses default).
//
// Returns the query result with columns and rows on success,
// or an error if the query is invalid or execution fails.
func (s *Service) Select(ctx context.Context, sqlText string, maxRows int) (QueryResult, error) {
	if err := validateSelectSQL(sqlText); err != nil {
		return QueryResult{}, err
	}
	if maxRows <= 0 || maxRows > s.cfg.MaxRows {
		maxRows = s.cfg.MaxRows
	}
	return s.query(ctx, sqlText, maxRows)
}

// Execute executes a write statement (INSERT, UPDATE, DELETE, MERGE).
//
// ctx is the context for the query.
// sqlText is the write statement to execute.
//
// Returns the number of rows affected and a message on success,
// or an error if the statement is invalid or not allowed.
func (s *Service) Execute(ctx context.Context, sqlText string) (ExecuteResult, error) {
	if err := validateWriteSQL(sqlText, s.cfg.AllowDangerousSQL, s.cfg.AllowSchemaChanges); err != nil {
		return ExecuteResult{}, err
	}
	result, err := s.db.ExecContext(ctx, sqlText)
	if err != nil {
		return ExecuteResult{}, err
	}
	rowsAffected, _ := result.RowsAffected()
	return ExecuteResult{
		RowsAffected: rowsAffected,
		Message:    "statement executed",
	}, nil
}

// query executes a raw SQL query and maps results to QueryResult.
//
// ctx is the context for the query.
// sqlText is the SQL query to execute.
// maxRows limits the number of rows returned.
// args are optional query parameters.
//
// Returns the query result on success, or an error on failure.
func (s *Service) query(ctx context.Context, sqlText string, maxRows int, args ...any) (QueryResult, error) {
	rows, err := s.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return QueryResult{}, err
	}
	defer func() { _ = rows.Close() }()

	columns, err := rows.Columns()
	if err != nil {
		return QueryResult{}, err
	}

	colCount := len(columns)
	if maxRows > 0 && maxRows < 100 {
		maxRows = 100
	}
	out := make([]map[string]any, 0, maxRows)
	idx := 0
	for rows.Next() {
		values := make([]any, colCount)
		pointers := make([]any, colCount)
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return QueryResult{}, err
		}

		row := make(map[string]any, colCount+1)
		row["_index"] = idx
		for i, column := range columns {
			row[column] = normalizeValue(values[i])
		}
		out = append(out, row)
		idx++
	}
	if err := rows.Err(); err != nil {
		return QueryResult{}, err
	}

	return QueryResult{
		Columns:  columns,
		Rows:     out,
		RowCount: len(out),
	}, nil
}

func normalizeValue(value any) any {
	switch v := value.(type) {
	case nil:
		return nil
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.RFC3339Nano)
	default:
		return v
	}
}
