package sqlserver

import (
	"fmt"
	"regexp"
	"strings"
)

var firstTokenPattern = regexp.MustCompile(`(?is)^\s*(?:--[^\n]*\n|/\*.*?\*/\s*)*([a-z]+)`)

var dangerousTokens = []string{
	"xp_cmdshell",
	"sp_configure",
	"openrowset",
	"opendatasource",
	"bulk insert",
	"backup database",
	"restore database",
	"shutdown",
}

func validateSelectSQL(sqlText string) error {
	verb := firstVerb(sqlText)
	if verb != "select" && verb != "with" {
		return fmt.Errorf("only SELECT or WITH statements are allowed")
	}
	if hasDangerousToken(sqlText) {
		return fmt.Errorf("query contains a blocked SQL Server capability")
	}
	return nil
}

func validateWriteSQL(sqlText string, allowDangerous, allowSchemaChanges bool) error {
	verb := firstVerb(sqlText)
	switch verb {
	case "insert", "update", "delete", "merge":
	case "create", "alter", "drop", "truncate":
		if !allowSchemaChanges {
			return fmt.Errorf("schema changes are disabled; set MCP_SQLSERVER_ALLOW_SCHEMA_CHANGES=true to allow %s", strings.ToUpper(verb))
		}
	default:
		return fmt.Errorf("unsupported write statement %q", verb)
	}

	if !allowDangerous && hasDangerousToken(sqlText) {
		return fmt.Errorf("statement contains a blocked SQL Server capability")
	}
	return nil
}

func firstVerb(sqlText string) string {
	match := firstTokenPattern.FindStringSubmatch(sqlText)
	if len(match) < 2 {
		return ""
	}
	return strings.ToLower(match[1])
}

func hasDangerousToken(sqlText string) bool {
	lower := strings.ToLower(sqlText)
	for _, token := range dangerousTokens {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}
