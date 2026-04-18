// Package config provides configuration loading from environment variables.
//
// This package loads configuration for the MCP SQL Server from environment
// variables, with sensible defaults for local development.
//
// Configuration precedence:
//
//  1. Environment variables (highest priority)
//  2. Default values (lowest priority)
//
// Example:
//
//	cfg := config.Load()
//	// cfg.Database.Host contains the configured SQL Server host
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the complete configuration for the MCP SQL Server.
type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
	RAG      RAGConfig
}

type RAGConfig struct {
	Enabled   bool
	AutoLearn bool
}

type DatabaseConfig struct {
	Host               string
	Port               string
	User               string
	Password           string
	Database           string
	Encrypt            string
	TrustServerCert    bool
	ApplicationName    string
	ConnectionString   string
	ConnectionTimeout  time.Duration
	QueryTimeout       time.Duration
	MaxOpenConnections int
	MaxIdleConnections int
}

type ServerConfig struct {
	MaxRows             int
	AllowDangerousSQL   bool
	AllowSchemaChanges  bool
	AllowProcedureCalls bool
}

func Load() Config {
	return Config{
		Database: DatabaseConfig{
			Host:               env("SQL_SERVER_HOST", "localhost"),
			Port:               env("SQL_SERVER_PORT", "1433"),
			User:               env("SQL_SERVER_USER", "sa"),
			Password:           env("SQL_SERVER_PASSWORD", ""),
			Database:           env("SQL_SERVER_DATABASE", "master"),
			Encrypt:            env("SQL_SERVER_ENCRYPT", "disable"),
			TrustServerCert:    envBool("SQL_SERVER_TRUST_CERT", true),
			ApplicationName:    env("SQL_SERVER_APP_NAME", "mcp-sqlserver"),
			ConnectionString:   os.Getenv("SQL_SERVER_CONNECTION_STRING"),
			ConnectionTimeout:  envDuration("SQL_SERVER_CONNECT_TIMEOUT", 10*time.Second),
			QueryTimeout:       envDuration("SQL_SERVER_QUERY_TIMEOUT", 60*time.Second),
			MaxOpenConnections: envInt("SQL_SERVER_MAX_OPEN_CONNS", 10),
			MaxIdleConnections: envInt("SQL_SERVER_MAX_IDLE_CONNS", 5),
		},
		Server: ServerConfig{
			MaxRows:             envInt("MCP_SQLSERVER_MAX_ROWS", 500),
			AllowDangerousSQL:   envBool("MCP_SQLSERVER_ALLOW_DANGEROUS_SQL", false),
			AllowSchemaChanges:  envBool("MCP_SQLSERVER_ALLOW_SCHEMA_CHANGES", false),
			AllowProcedureCalls: envBool("MCP_SQLSERVER_ALLOW_PROCEDURE_CALLS", true),
		},
		RAG: RAGConfig{
			Enabled:   envBool("MCP_RAG_ENABLED", true),
			AutoLearn: envBool("MCP_RAG_AUTO_LEARN", true),
		},
	}
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	if parsed, err := time.ParseDuration(value); err == nil {
		return parsed
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}
