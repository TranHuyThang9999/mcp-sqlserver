package sqlserver

import (
	"context"
	"database/sql"
	"time"

	"mcp_sqlserver/internal/config"
)

type Service struct {
	db  *sql.DB
	cfg config.ServerConfig
}

func NewService(db *sql.DB, cfg config.ServerConfig) *Service {
	return &Service{db: db, cfg: cfg}
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
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return QueryResult{}, err
	}

	var out []map[string]any
	for rows.Next() {
		if maxRows > 0 && len(out) >= maxRows {
			break
		}
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return QueryResult{}, err
		}

		row := make(map[string]any, len(columns))
		for i, column := range columns {
			row[column] = normalizeValue(values[i])
		}
		out = append(out, row)
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
