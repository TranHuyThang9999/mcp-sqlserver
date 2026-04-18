package sqlserver

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"mcp_sqlserver/internal/config"

	_ "github.com/microsoft/go-mssqldb"
)

func Open(ctx context.Context, cfg config.DatabaseConfig) (*sql.DB, error) {
	connString := cfg.ConnectionString
	if connString == "" {
		query := url.Values{}
		query.Set("database", cfg.Database)
		query.Set("connection timeout", fmt.Sprintf("%.0f", cfg.ConnectionTimeout.Seconds()))
		query.Set("app name", cfg.ApplicationName)
		query.Set("encrypt", cfg.Encrypt)
		query.Set("TrustServerCertificate", fmt.Sprintf("%t", cfg.TrustServerCert))

		u := &url.URL{
			Scheme:   "sqlserver",
			User:     url.UserPassword(cfg.User, cfg.Password),
			Host:     cfg.Host + ":" + cfg.Port,
			RawQuery: query.Encode(),
		}
		connString = u.String()
	}

	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.MaxOpenConnections)
	db.SetMaxIdleConns(cfg.MaxIdleConnections)

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectionTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
