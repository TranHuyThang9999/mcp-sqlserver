package mcpserver

import "mcp_sqlserver/internal/sqlserver"

type EmptyInput struct{}

type SQLInput struct {
	SQL string `json:"sql" jsonschema:"SQL statement to execute"`
}

type SelectInput struct {
	SQL     string `json:"sql" jsonschema:"SELECT or WITH query"`
	MaxRows int    `json:"maxRows,omitempty" jsonschema:"maximum rows to return"`
}

type SchemaFilterInput struct {
	Schema string `json:"schema,omitempty" jsonschema:"optional SQL Server schema name"`
}

type ObjectInput struct {
	Schema string `json:"schema" jsonschema:"SQL Server schema name"`
	Name   string `json:"name" jsonschema:"object name"`
}

type ProcedureInput struct {
	Schema     string                     `json:"schema" jsonschema:"procedure schema name"`
	Name       string                     `json:"name" jsonschema:"procedure name"`
	Parameters []sqlserver.ProcedureParam `json:"parameters,omitempty" jsonschema:"procedure parameters"`
}

type HealthOutput struct {
	Info map[string]any `json:"info"`
}

type DatabasesOutput struct {
	Databases []sqlserver.DatabaseInfo `json:"databases"`
}

type SchemasOutput struct {
	Schemas []sqlserver.SchemaInfo `json:"schemas"`
}

type TablesOutput struct {
	Tables []sqlserver.TableInfo `json:"tables"`
}

type ObjectsOutput struct {
	Objects []sqlserver.ObjectInfo `json:"objects"`
}

type TableSchemaOutput struct {
	Table sqlserver.TableSchema `json:"table"`
}

type DefinitionOutput struct {
	Object sqlserver.DefinitionResult `json:"object"`
}

type QueryOutput struct {
	Result sqlserver.QueryResult `json:"result"`
}

type ExecuteOutput struct {
	Result sqlserver.ExecuteResult `json:"result"`
}

type RAGQueryInput struct {
	Query string `json:"query" jsonschema:"query to search in knowledge base"`
	Type  string `json:"type,omitempty" jsonschema:"type of knowledge: tables, relations, all"`
}

type RAGLearnInput struct {
	Schema string `json:"schema" jsonschema:"schema name"`
	Table  string `json:"table" jsonschema:"table name"`
}

type RAGStatsOutput struct {
	Stats  map[string]int `json:"stats"`
	Status string         `json:"status"`
}

type RAGTablesOutput struct {
	Tables []RAGTableInfo `json:"tables"`
}

type RAGTableInfo struct {
	Schema      string `json:"schema"`
	Name        string `json:"name"`
	Columns     string `json:"columns,omitempty"`
	LastLearned string `json:"lastLearned,omitempty"`
}

type RAGRelationsOutput struct {
	Relations []RAGRelationInfo `json:"relations"`
}

type RAGRelationInfo struct {
	FromSchema   string `json:"fromSchema"`
	FromTable    string `json:"fromTable"`
	FromColumn   string `json:"fromColumn"`
	ToSchema     string `json:"toSchema"`
	ToTable      string `json:"toTable"`
	ToColumn     string `json:"toColumn"`
	RelationType string `json:"relationType"`
}

type RAGSearchOutput struct {
	Results []RAGSearchResult `json:"results"`
}

type RAGSearchResult struct {
	Schema string  `json:"schema"`
	Name   string  `json:"name"`
	Type   string  `json:"type"`
	Text   string  `json:"text"`
	Score  float64 `json:"score"`
}
