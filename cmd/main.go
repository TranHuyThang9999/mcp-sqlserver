// mcp-sqlserver is an MCP server for connecting LLM tools to Microsoft SQL Server.
//
// This program provides a Model Context Protocol (MCP) server that enables AI tools
// to query and interact with Microsoft SQL Server databases over stdio.
//
// Usage:
//
//	SQL_SERVER_HOST=localhost \
//	SQL_SERVER_USER=sa \
//	SQL_SERVER_PASSWORD=your_password \
//	SQL_SERVER_DATABASE=your_database \
//	  go run ./cmd
//
// Or use a pre-built binary:
//
//	./mcp-sqlserver.exe
//
// The server communicates with MCP clients via stdio, following the MCP specification.
package main

import (
	"context"
	"log"
	"os"

	"mcp_sqlserver/internal/config"
	mcpsql "mcp_sqlserver/internal/mcpserver"
	"mcp_sqlserver/internal/sqlserver"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const version = "v1.1.0"

func main() {
	ctx := context.Background()

	cfg := config.Load()
	db, err := sqlserver.Open(ctx, cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-sqlserver",
		Version: version,
	}, nil)

	service := sqlserver.NewService(db, cfg.Server)
	mcpsql.RegisterTools(server, service)

	log.Println("SQL Server MCP server running on stdio")
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		os.Exit(1)
	}
}
