package sqlserver

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"mcp_sqlserver/internal/config"

	_ "github.com/microsoft/go-mssqldb"
)

type DB interface {
	*sql.DB
	Close() error
}

func Open(ctx context.Context, cfg config.DatabaseConfig) (*sql.DB, error) {
	connString := cfg.ConnectionString
	if connString == "" {
		q := url.Values{}
		q.Set("database", cfg.Database)
		q.Set("connection timeout", fmt.Sprintf("%.0f", cfg.ConnectionTimeout.Seconds()))
		q.Set("app name", cfg.ApplicationName)
		q.Set("encrypt", cfg.Encrypt)
		q.Set("TrustServerCertificate", fmt.Sprintf("%t", cfg.TrustServerCert))

		u := &url.URL{
			Scheme:   "sqlserver",
			User:     url.UserPassword(cfg.User, cfg.Password),
			Host:     cfg.Host + ":" + cfg.Port,
			RawQuery: q.Encode(),
		}
		connString = u.String()
	}

	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.MaxOpenConnections)
	db.SetMaxIdleConns(cfg.MaxIdleConnections)

	ctx, cancel := context.WithTimeout(ctx, cfg.ConnectionTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}