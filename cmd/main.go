package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	"mcp_sqlserver/internal/config"
	mcpsql "mcp_sqlserver/internal/mcpserver"
	"mcp_sqlserver/internal/rag"
	"mcp_sqlserver/internal/sqlserver"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/fx"
)

const version = "v1.3.0"

func main() {
	var server *mcp.Server

	app := fx.New(
		fx.Provide(
			fx.Annotate(provideDatabase, fx.ParamTags(`name:"database"`)),
			fx.Annotate(provideKnowledge, fx.ParamTags(`optional:"true"`)),
			provideService,
		),
		fx.Invoke(func(svc *sqlserver.Service, ks *rag.Store) {
			server = mcp.NewServer(&mcp.Implementation{
				Name:    "mcp-sqlserver",
				Version: version,
			}, nil)
			mcpsql.RegisterTools(server, svc, ks)
		}),
		fx.NopLogger,
	)

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatal(err)
	}
	if err := app.Stop(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	log.Println("SQL Server MCP server running on stdio")
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		os.Exit(1)
	}
}

func provideDatabase(lc fx.Lifecycle, cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sqlserver.Open(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return db.Close() }})
	return db, nil
}

func provideKnowledge(lc fx.Lifecycle, cfg config.RAGConfig) (*rag.Store, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	store, err := rag.NewStore("")
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return store.Close() }})
	return store, nil
}

func provideService(db *sql.DB, cfg config.ServerConfig, store *rag.Store) *sqlserver.Service {
	return sqlserver.NewService(db, cfg, store)
}
