// Package sqlserver provides Microsoft SQL Server database connectivity for MCP.
//
// This package implements the database layer for connecting to Microsoft SQL Server
// using the go-mssqldb driver. It handles connection pooling, query execution,
// and result mapping for the MCP server.
//
// # Connection
//
// The Open function establishes a connection to SQL Server using the provided
// configuration. Connection pooling is managed automatically by sql.DB.
//
//	cfg := config.Load()
//	db, err := sqlserver.Open(context.Background(), cfg.Database)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer db.Close()
//
// # Query Execution
//
// The Service type provides high-level methods for querying the database,
// including SQL validation and row mapping.
package sqlserver

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"mcp_sqlserver/internal/config"

	_ "github.com/microsoft/go-mssqldb"
)

// DB is the database connection type.
type DB interface {
	*sql.DB
	Close() error
}

// Open establishes a new connection to Microsoft SQL Server.
//
// The function constructs a connection string from the provided config,
// or uses the configured ConnectionString if set. It sets up connection
// pooling and verifies the connection with a ping.
//
//	ctx is the context for connection timeout.
//	cfg is the database configuration.
//
// Returns a sql.DB connection handle on success, or an error on failure.
// The caller is responsible for closing the connection.
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

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectionTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close() //nolint:errcheck
		return nil, err
	}
	return db, nil
}
