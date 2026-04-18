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

func (s *Service) Select(ctx context.Context, sqlText string, maxRows int) (QueryResult, error) {
	if err := validateSelectSQL(sqlText); err != nil {
		return QueryResult{}, err
	}
	if maxRows <= 0 || maxRows > s.cfg.MaxRows {
		maxRows = s.cfg.MaxRows
	}
	return s.query(ctx, sqlText, maxRows)
}

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
		Message:      "statement executed",
	}, nil
}

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
