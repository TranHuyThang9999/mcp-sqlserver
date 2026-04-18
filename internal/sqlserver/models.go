package sqlserver

type QueryResult struct {
	Columns  []string         `json:"columns"`
	Rows     []map[string]any `json:"rows"`
	RowCount int              `json:"rowCount"`
}

type ExecuteResult struct {
	RowsAffected int64  `json:"rowsAffected"`
	Message      string `json:"message"`
}

type DatabaseInfo struct {
	Name          string `json:"name"`
	State         string `json:"state"`
	RecoveryModel string `json:"recoveryModel"`
	Compatibility int    `json:"compatibilityLevel"`
}

type SchemaInfo struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`
}

type TableInfo struct {
	Schema   string `json:"schema"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	RowCount int64  `json:"rowCount"`
}

type ColumnInfo struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	MaxLength     int16  `json:"maxLength"`
	Precision     uint8  `json:"precision"`
	Scale         uint8  `json:"scale"`
	Nullable      bool   `json:"nullable"`
	Identity      bool   `json:"identity"`
	Computed      bool   `json:"computed"`
	DefaultValue  string `json:"defaultValue,omitempty"`
	CollationName string `json:"collationName,omitempty"`
}

type KeyInfo struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Columns    []string `json:"columns"`
	References string   `json:"references,omitempty"`
}

type IndexInfo struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Unique   bool     `json:"unique"`
	Primary  bool     `json:"primary"`
	Columns  []string `json:"columns"`
	Filter   string   `json:"filter,omitempty"`
	Disabled bool     `json:"disabled"`
}

type TableSchema struct {
	Schema      string       `json:"schema"`
	Name        string       `json:"name"`
	Columns     []ColumnInfo `json:"columns"`
	PrimaryKeys []KeyInfo    `json:"primaryKeys"`
	ForeignKeys []KeyInfo    `json:"foreignKeys"`
	Indexes     []IndexInfo  `json:"indexes"`
	Triggers    []ObjectInfo `json:"triggers"`
}

type ObjectInfo struct {
	Schema     string `json:"schema"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	CreateDate string `json:"createDate,omitempty"`
	ModifyDate string `json:"modifyDate,omitempty"`
	Parent     string `json:"parent,omitempty"`
	Disabled   bool   `json:"disabled,omitempty"`
}

type DefinitionResult struct {
	Schema     string `json:"schema"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Definition string `json:"definition"`
}
