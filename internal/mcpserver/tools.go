package mcpserver

import (
	"context"
	"fmt"
	"log"

	"mcp_sqlserver/internal/rag"
	"mcp_sqlserver/internal/sqlserver"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type handlers struct {
	service   *sqlserver.Service
	knowledge *rag.Store
}

func RegisterTools(server *mcp.Server, service *sqlserver.Service, knowledge *rag.Store) {
	h := &handlers{service: service, knowledge: knowledge}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "health_check",
		Description: "Check SQL Server connectivity and return server/database information.",
	}, h.health)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sql_select",
		Description: "Run a read-only SELECT/WITH query and return rows as JSON.",
	}, h.selectSQL)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sql_execute",
		Description: "Run INSERT, UPDATE, DELETE, MERGE, and optionally schema-changing statements.",
	}, h.executeSQL)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_databases",
		Description: "List SQL Server databases visible to the current login.",
	}, h.listDatabases)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_schemas",
		Description: "List schemas in the current database.",
	}, h.listSchemas)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_tables",
		Description: "List tables and views in the current database, optionally filtered by schema.",
	}, h.listTables)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "describe_table",
		Description: "Return columns, primary keys, foreign keys, indexes, and triggers for a table.",
	}, h.describeTable)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_views",
		Description: "List views in the current database.",
	}, h.listViews)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_object_definition",
		Description: "Return SQL definition for a procedure, view, trigger, or function.",
	}, h.getDefinition)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_procedures",
		Description: "List stored procedures in the current database.",
	}, h.listProcedures)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "execute_procedure",
		Description: "Execute a stored procedure with named parameters and return any result set.",
	}, h.executeProcedure)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_triggers",
		Description: "List database and table triggers.",
	}, h.listTriggers)

	if h.knowledge != nil {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "rag_query",
			Description: "Query the persistent knowledge base for schema/table information learned from previous queries.",
		}, h.ragQuery)

		mcp.AddTool(server, &mcp.Tool{
			Name:        "rag_learn_table",
			Description: "Learn and store table schema in the knowledge base.",
		}, h.ragLearnTable)

		mcp.AddTool(server, &mcp.Tool{
			Name:        "rag_stats",
			Description: "Get knowledge base statistics (tables, relations, queries learned).",
		}, h.ragStats)

		mcp.AddTool(server, &mcp.Tool{
			Name:        "rag_list_tables",
			Description: "List all tables stored in knowledge base.",
		}, h.ragListTables)

		mcp.AddTool(server, &mcp.Tool{
			Name:        "rag_list_relations",
			Description: "List all relationships stored in knowledge base.",
		}, h.ragListRelations)
	}
}

func toolError(err error) *mcp.CallToolResult {
	log.Printf("tool error: %v", err)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		IsError: true,
	}
}

func (h *handlers) health(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, HealthOutput, error) {
	info, err := h.service.Health(ctx)
	if err != nil {
		return toolError(err), HealthOutput{}, nil
	}
	return nil, HealthOutput{Info: info}, nil
}

func (h *handlers) selectSQL(ctx context.Context, req *mcp.CallToolRequest, input SelectInput) (*mcp.CallToolResult, QueryOutput, error) {
	result, err := h.service.Select(ctx, input.SQL, input.MaxRows)
	if err != nil {
		return toolError(err), QueryOutput{}, nil
	}
	return nil, QueryOutput{Result: result}, nil
}

func (h *handlers) executeSQL(ctx context.Context, req *mcp.CallToolRequest, input SQLInput) (*mcp.CallToolResult, ExecuteOutput, error) {
	result, err := h.service.Execute(ctx, input.SQL)
	if err != nil {
		return toolError(err), ExecuteOutput{}, nil
	}
	return nil, ExecuteOutput{Result: result}, nil
}

func (h *handlers) listDatabases(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, DatabasesOutput, error) {
	items, err := h.service.ListDatabases(ctx)
	if err != nil {
		return toolError(err), DatabasesOutput{}, nil
	}
	return nil, DatabasesOutput{Databases: items}, nil
}

func (h *handlers) listSchemas(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, SchemasOutput, error) {
	items, err := h.service.ListSchemas(ctx)
	if err != nil {
		return toolError(err), SchemasOutput{}, nil
	}
	return nil, SchemasOutput{Schemas: items}, nil
}

func (h *handlers) listTables(ctx context.Context, req *mcp.CallToolRequest, input SchemaFilterInput) (*mcp.CallToolResult, TablesOutput, error) {
	items, err := h.service.ListTables(ctx, input.Schema)
	if err != nil {
		return toolError(err), TablesOutput{}, nil
	}
	return nil, TablesOutput{Tables: items}, nil
}

func (h *handlers) describeTable(ctx context.Context, req *mcp.CallToolRequest, input ObjectInput) (*mcp.CallToolResult, TableSchemaOutput, error) {
	if input.Schema == "" || input.Name == "" {
		return toolError(fmt.Errorf("schema and name are required")), TableSchemaOutput{}, nil
	}
	item, err := h.service.DescribeTable(ctx, input.Schema, input.Name)
	if err != nil {
		return toolError(err), TableSchemaOutput{}, nil
	}
	return nil, TableSchemaOutput{Table: item}, nil
}

func (h *handlers) listViews(ctx context.Context, req *mcp.CallToolRequest, input SchemaFilterInput) (*mcp.CallToolResult, ObjectsOutput, error) {
	items, err := h.service.ListViews(ctx, input.Schema)
	if err != nil {
		return toolError(err), ObjectsOutput{}, nil
	}
	return nil, ObjectsOutput{Objects: items}, nil
}

func (h *handlers) listProcedures(ctx context.Context, req *mcp.CallToolRequest, input SchemaFilterInput) (*mcp.CallToolResult, ObjectsOutput, error) {
	items, err := h.service.ListProcedures(ctx, input.Schema)
	if err != nil {
		return toolError(err), ObjectsOutput{}, nil
	}
	return nil, ObjectsOutput{Objects: items}, nil
}

func (h *handlers) listTriggers(ctx context.Context, req *mcp.CallToolRequest, input SchemaFilterInput) (*mcp.CallToolResult, ObjectsOutput, error) {
	items, err := h.service.ListTriggers(ctx, input.Schema)
	if err != nil {
		return toolError(err), ObjectsOutput{}, nil
	}
	return nil, ObjectsOutput{Objects: items}, nil
}

func (h *handlers) getDefinition(ctx context.Context, req *mcp.CallToolRequest, input ObjectInput) (*mcp.CallToolResult, DefinitionOutput, error) {
	if input.Schema == "" || input.Name == "" {
		return toolError(fmt.Errorf("schema and name are required")), DefinitionOutput{}, nil
	}
	item, err := h.service.GetDefinition(ctx, input.Schema, input.Name)
	if err != nil {
		return toolError(err), DefinitionOutput{}, nil
	}
	return nil, DefinitionOutput{Object: item}, nil
}

func (h *handlers) executeProcedure(ctx context.Context, req *mcp.CallToolRequest, input ProcedureInput) (*mcp.CallToolResult, QueryOutput, error) {
	if input.Schema == "" || input.Name == "" {
		return toolError(fmt.Errorf("schema and name are required")), QueryOutput{}, nil
	}
	result, err := h.service.ExecuteProcedure(ctx, input.Schema, input.Name, input.Parameters)
	if err != nil {
		return toolError(err), QueryOutput{}, nil
	}
	return nil, QueryOutput{Result: result}, nil
}

func (h *handlers) ragQuery(ctx context.Context, req *mcp.CallToolRequest, input RAGQueryInput) (*mcp.CallToolResult, RAGSearchOutput, error) {
	if h.knowledge == nil {
		return toolError(fmt.Errorf("vector store not initialized")), RAGSearchOutput{}, nil
	}
	results, err := h.knowledge.Search(ctx, input.Query, 10)
	if err != nil {
		return toolError(err), RAGSearchOutput{}, nil
	}
	out := make([]RAGSearchResult, len(results))
	for i, r := range results {
		out[i] = RAGSearchResult{Schema: r.Schema, Name: r.Name, Type: r.Type, Text: r.Text, Score: r.Score}
	}
	return nil, RAGSearchOutput{Results: out}, nil
}

func (h *handlers) ragLearnTable(ctx context.Context, req *mcp.CallToolRequest, input ObjectInput) (*mcp.CallToolResult, RAGStatsOutput, error) {
	if h.knowledge == nil {
		return toolError(fmt.Errorf("vector store not initialized")), RAGStatsOutput{}, nil
	}
	if input.Schema == "" || input.Name == "" {
		return toolError(fmt.Errorf("schema and name are required")), RAGStatsOutput{}, nil
	}
	ts, err := h.service.DescribeTable(ctx, input.Schema, input.Name)
	if err != nil {
		return toolError(err), RAGStatsOutput{}, nil
	}
	data := map[string]any{"columns": ts.Columns, "primaryKeys": ts.PrimaryKeys, "foreignKeys": ts.ForeignKeys, "indexes": ts.Indexes}
	_ = h.knowledge.LearnTable(ctx, input.Schema, input.Name, data)
	fks := make([]map[string]any, len(ts.ForeignKeys))
	for i, fk := range ts.ForeignKeys {
		fks[i] = map[string]any{"columns": fk.Columns, "references": fk.References}
	}
	_ = h.knowledge.LearnRelations(ctx, input.Schema, input.Name, fks)
	stats, _ := h.knowledge.Stats(ctx)
	return nil, RAGStatsOutput{Stats: stats, Status: "learned"}, nil
}

func (h *handlers) ragStats(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, RAGStatsOutput, error) {
	if h.knowledge == nil {
		return toolError(fmt.Errorf("vector store not initialized")), RAGStatsOutput{}, nil
	}
	stats, err := h.knowledge.Stats(ctx)
	if err != nil {
		return toolError(err), RAGStatsOutput{}, nil
	}
	return nil, RAGStatsOutput{Stats: stats, Status: "ok"}, nil
}

func (h *handlers) ragListTables(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, RAGTablesOutput, error) {
	if h.knowledge == nil {
		return toolError(fmt.Errorf("vector store not initialized")), RAGTablesOutput{}, nil
	}
	results, err := h.knowledge.GetAllTables(ctx)
	if err != nil {
		return toolError(err), RAGTablesOutput{}, nil
	}
	tables := make([]RAGTableInfo, len(results))
	for i, r := range results {
		tables[i] = RAGTableInfo{Schema: r.Schema, Name: r.Name, LastLearned: r.LastLearned}
	}
	return nil, RAGTablesOutput{Tables: tables}, nil
}

func (h *handlers) ragListRelations(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, RAGRelationsOutput, error) {
	if h.knowledge == nil {
		return toolError(fmt.Errorf("vector store not initialized")), RAGRelationsOutput{}, nil
	}
	results, err := h.knowledge.GetAllRelations(ctx)
	if err != nil {
		return toolError(err), RAGRelationsOutput{}, nil
	}
	relations := make([]RAGRelationInfo, len(results))
	for i, r := range results {
		relations[i] = RAGRelationInfo{FromSchema: r.Schema, FromTable: r.Name, FromColumn: "", ToSchema: "", ToTable: "", ToColumn: "", RelationType: r.Type}
	}
	return nil, RAGRelationsOutput{Relations: relations}, nil
}
