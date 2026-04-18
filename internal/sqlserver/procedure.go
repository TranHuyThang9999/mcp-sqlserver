package sqlserver

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type ProcedureParam struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

func (s *Service) ExecuteProcedure(ctx context.Context, schema, name string, params []ProcedureParam) (QueryResult, error) {
	if !s.cfg.AllowProcedureCalls {
		return QueryResult{}, fmt.Errorf("procedure calls are disabled")
	}

	procName, err := objectName(schema, name)
	if err != nil {
		return QueryResult{}, err
	}

	sort.Slice(params, func(i, j int) bool {
		return params[i].Name < params[j].Name
	})

	parts := make([]string, 0, len(params))
	args := make([]any, 0, len(params))
	for _, param := range params {
		cleanName := strings.TrimPrefix(strings.TrimSpace(param.Name), "@")
		if _, err := quoteName(cleanName); err != nil {
			return QueryResult{}, fmt.Errorf("invalid procedure parameter %q", param.Name)
		}
		parts = append(parts, "@"+cleanName+" = ?")
		args = append(args, param.Value)
	}

	sqlText := "SET NOCOUNT ON; EXEC " + procName
	if len(parts) > 0 {
		sqlText += " " + strings.Join(parts, ", ")
	}
	return s.query(ctx, sqlText, s.cfg.MaxRows, args...)
}
