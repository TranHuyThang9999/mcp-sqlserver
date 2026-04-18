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
