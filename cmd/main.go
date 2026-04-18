package main

import (
	"context"
	"log"

	"mcp_sqlserver/internal/config"
	mcpsql "mcp_sqlserver/internal/mcpserver"
	"mcp_sqlserver/internal/sqlserver"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()
	db, err := sqlserver.Open(ctx, cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-sqlserver",
		Version: "v1.0.0",
	}, nil)

	service := sqlserver.NewService(db, cfg.Server)
	mcpsql.RegisterTools(server, service)

	log.Println("SQL Server MCP server running on stdio")
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
