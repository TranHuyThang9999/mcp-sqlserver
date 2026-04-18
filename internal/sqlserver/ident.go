package sqlserver

import (
	"fmt"
	"regexp"
	"strings"
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_@$#]*$`)

func quoteName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("identifier cannot be empty")
	}
	parts := strings.Split(name, ".")
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(part, " []")
		if !identifierPattern.MatchString(part) {
			return "", fmt.Errorf("invalid identifier %q", part)
		}
		quoted = append(quoted, "["+strings.ReplaceAll(part, "]", "]]")+"]")
	}
	return strings.Join(quoted, "."), nil
}

func objectName(schema, name string) (string, error) {
	quotedSchema, err := quoteName(schema)
	if err != nil {
		return "", err
	}
	quotedName, err := quoteName(name)
	if err != nil {
		return "", err
	}
	return quotedSchema + "." + quotedName, nil
}
