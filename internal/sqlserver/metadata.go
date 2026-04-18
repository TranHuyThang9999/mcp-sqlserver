package sqlserver

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func (s *Service) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT name, state_desc, recovery_model_desc, compatibility_level
FROM sys.databases
ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]DatabaseInfo, 0, 64)
	for rows.Next() {
		var item DatabaseInfo
		if err := rows.Scan(&item.Name, &item.State, &item.RecoveryModel, &item.Compatibility); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) ListSchemas(ctx context.Context) ([]SchemaInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT s.name, USER_NAME(s.principal_id) AS owner_name
FROM sys.schemas AS s
ORDER BY s.name`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]SchemaInfo, 0, 64)
	for rows.Next() {
		var item SchemaInfo
		if err := rows.Scan(&item.Name, &item.Owner); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) ListTables(ctx context.Context, schema string) ([]TableInfo, error) {
	query := `
SELECT sch.name, obj.name, obj.type_desc, COALESCE(SUM(part.rows), 0) AS row_count
FROM sys.objects AS obj
JOIN sys.schemas AS sch ON sch.schema_id = obj.schema_id
LEFT JOIN sys.partitions AS part ON part.object_id = obj.object_id AND part.index_id IN (0, 1)
WHERE obj.type IN ('U', 'V') AND (@p1 = '' OR sch.name = @p1)
GROUP BY sch.name, obj.name, obj.type_desc
ORDER BY sch.name, obj.name`

	rows, err := s.db.QueryContext(ctx, query, strings.TrimSpace(schema))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]TableInfo, 0, 128)
	for rows.Next() {
		var item TableInfo
		if err := rows.Scan(&item.Schema, &item.Name, &item.Type, &item.RowCount); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) DescribeTable(ctx context.Context, schema, table string) (TableSchema, error) {
	objectID, err := s.objectID(ctx, schema, table)
	if err != nil {
		return TableSchema{}, err
	}

	results, err := s.describeTableDetails(ctx, objectID)
	if err != nil {
		return TableSchema{}, err
	}

	ts := TableSchema{
		Schema:      schema,
		Name:        table,
		Columns:     results.columns,
		PrimaryKeys: results.primaryKeys,
		ForeignKeys: results.foreignKeys,
		Indexes:     results.indexes,
		Triggers:    results.triggers,
	}

	if s.Knowledge != nil {
		data := map[string]any{
			"columns":     results.columns,
			"primaryKeys": results.primaryKeys,
			"foreignKeys": results.foreignKeys,
			"indexes":     results.indexes,
			"triggers":    results.triggers,
		}
		_ = s.Knowledge.LearnTable(ctx, schema, table, data)

		fkData := make([]map[string]any, len(results.foreignKeys))
		for i, fk := range results.foreignKeys {
			fkData[i] = map[string]any{
				"columns":    fk.Columns,
				"references": fk.References,
			}
		}
		_ = s.Knowledge.LearnRelations(ctx, schema, table, fkData)
	}

	return ts, nil
}

type describeResults struct {
	columns     []ColumnInfo
	primaryKeys []KeyInfo
	foreignKeys []KeyInfo
	indexes     []IndexInfo
	triggers    []ObjectInfo
}

func (s *Service) describeTableDetails(ctx context.Context, objectID int) (describeResults, error) {
	var r describeResults
	ec := make(chan error, 5)

	go func() { _, err := s.tableColumns(ctx, objectID); ec <- err }()
	go func() { _, err := s.primaryKeys(ctx, objectID); ec <- err }()
	go func() { _, err := s.foreignKeys(ctx, objectID); ec <- err }()
	go func() { _, err := s.indexes(ctx, objectID); ec <- err }()
	go func() { _, err := s.triggersForTable(ctx, objectID); ec <- err }()

	for i := 0; i < 5; i++ {
		if err := <-ec; err != nil {
			return r, err
		}
	}

	cols, _ := s.tableColumns(ctx, objectID)
	pks, _ := s.primaryKeys(ctx, objectID)
	fks, _ := s.foreignKeys(ctx, objectID)
	idx, _ := s.indexes(ctx, objectID)
	trg, _ := s.triggersForTable(ctx, objectID)

	return describeResults{
		columns:     cols,
		primaryKeys: pks,
		foreignKeys: fks,
		indexes:     idx,
		triggers:    trg,
	}, nil
}

func (s *Service) ListProcedures(ctx context.Context, schema string) ([]ObjectInfo, error) {
	return s.listObjects(ctx, "P", schema)
}

func (s *Service) ListViews(ctx context.Context, schema string) ([]ObjectInfo, error) {
	return s.listObjects(ctx, "V", schema)
}

func (s *Service) ListTriggers(ctx context.Context, schema string) ([]ObjectInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT COALESCE(s.name, OBJECT_SCHEMA_NAME(parent.object_id)) AS schema_name,
       tr.name,
       tr.type_desc,
       CONVERT(varchar(33), tr.create_date, 126),
       CONVERT(varchar(33), tr.modify_date, 126),
       COALESCE(OBJECT_SCHEMA_NAME(parent.object_id) + '.' + parent.name, ''),
       tr.is_disabled
FROM sys.triggers AS tr
LEFT JOIN sys.objects AS parent ON parent.object_id = tr.parent_id
LEFT JOIN sys.schemas AS s ON s.schema_id = tr.schema_id
WHERE (@p1 = '' OR COALESCE(s.name, OBJECT_SCHEMA_NAME(parent.object_id)) = @p1)
ORDER BY schema_name, tr.name`, strings.TrimSpace(schema))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]ObjectInfo, 0, 64)
	for rows.Next() {
		var item ObjectInfo
		if err := rows.Scan(&item.Schema, &item.Name, &item.Type, &item.CreateDate, &item.ModifyDate, &item.Parent, &item.Disabled); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) GetDefinition(ctx context.Context, schema, name string) (DefinitionResult, error) {
	var result DefinitionResult
	err := s.db.QueryRowContext(ctx, `
SELECT SCHEMA_NAME(o.schema_id), o.name, o.type_desc, OBJECT_DEFINITION(o.object_id)
FROM sys.objects AS o
WHERE o.object_id = OBJECT_ID(@p1)`, schema+"."+name).Scan(&result.Schema, &result.Name, &result.Type, &result.Definition)
	if err != nil {
		if err == sql.ErrNoRows {
			return DefinitionResult{}, fmt.Errorf("object %s.%s not found", schema, name)
		}
		return DefinitionResult{}, err
	}
	return result, nil
}

func (s *Service) objectID(ctx context.Context, schema, table string) (int, error) {
	var id sql.NullInt64
	err := s.db.QueryRowContext(ctx, "SELECT OBJECT_ID(@p1)", schema+"."+table).Scan(&id)
	if err != nil {
		return 0, err
	}
	if !id.Valid {
		return 0, fmt.Errorf("object %s.%s not found", schema, table)
	}
	return int(id.Int64), nil
}

func (s *Service) listObjects(ctx context.Context, objectType, schema string) ([]ObjectInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT s.name, o.name, o.type_desc, CONVERT(varchar(33), o.create_date, 126), CONVERT(varchar(33), o.modify_date, 126)
FROM sys.objects AS o
JOIN sys.schemas AS s ON s.schema_id = o.schema_id
WHERE o.type = @p1 AND (@p2 = '' OR s.name = @p2)
ORDER BY s.name, o.name`, objectType, strings.TrimSpace(schema))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]ObjectInfo, 0, 128)
	for rows.Next() {
		var item ObjectInfo
		if err := rows.Scan(&item.Schema, &item.Name, &item.Type, &item.CreateDate, &item.ModifyDate); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) tableColumns(ctx context.Context, objectID int) ([]ColumnInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT c.name, TYPE_NAME(c.user_type_id), c.max_length, c.precision, c.scale,
       c.is_nullable, c.is_identity, c.is_computed,
       COALESCE(OBJECT_DEFINITION(c.default_object_id), ''),
       COALESCE(c.collation_name, '')
FROM sys.columns AS c
WHERE c.object_id = @p1
ORDER BY c.column_id`, objectID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]ColumnInfo, 0, 64)
	for rows.Next() {
		var item ColumnInfo
		if err := rows.Scan(&item.Name, &item.Type, &item.MaxLength, &item.Precision, &item.Scale, &item.Nullable, &item.Identity, &item.Computed, &item.DefaultValue, &item.CollationName); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) primaryKeys(ctx context.Context, objectID int) ([]KeyInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT kc.name, STRING_AGG(c.name, ',') WITHIN GROUP (ORDER BY ic.key_ordinal)
FROM sys.key_constraints AS kc
JOIN sys.index_columns AS ic ON ic.object_id = kc.parent_object_id AND ic.index_id = kc.unique_index_id
JOIN sys.columns AS c ON c.object_id = ic.object_id AND c.column_id = ic.column_id
WHERE kc.type = 'PK' AND kc.parent_object_id = @p1
GROUP BY kc.name`, objectID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]KeyInfo, 0, 4)
	for rows.Next() {
		var item KeyInfo
		var columns string
		if err := rows.Scan(&item.Name, &columns); err != nil {
			return nil, err
		}
		item.Type = "PRIMARY_KEY"
		item.Columns = splitCSV(columns)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) foreignKeys(ctx context.Context, objectID int) ([]KeyInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT fk.name,
       STRING_AGG(pc.name, ',') WITHIN GROUP (ORDER BY fkc.constraint_column_id),
       OBJECT_SCHEMA_NAME(fk.referenced_object_id) + '.' + OBJECT_NAME(fk.referenced_object_id) AS referenced_table
FROM sys.foreign_keys AS fk
JOIN sys.foreign_key_columns AS fkc ON fkc.constraint_object_id = fk.object_id
JOIN sys.columns AS pc ON pc.object_id = fkc.parent_object_id AND pc.column_id = fkc.parent_column_id
WHERE fk.parent_object_id = @p1
GROUP BY fk.name, fk.referenced_object_id`, objectID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]KeyInfo, 0, 8)
	for rows.Next() {
		var item KeyInfo
		var columns string
		if err := rows.Scan(&item.Name, &columns, &item.References); err != nil {
			return nil, err
		}
		item.Type = "FOREIGN_KEY"
		item.Columns = splitCSV(columns)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) indexes(ctx context.Context, objectID int) ([]IndexInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT i.name, i.type_desc, i.is_unique, i.is_primary_key,
       STRING_AGG(c.name, ',') WITHIN GROUP (ORDER BY ic.key_ordinal),
       COALESCE(i.filter_definition, ''),
       i.is_disabled
FROM sys.indexes AS i
JOIN sys.index_columns AS ic ON ic.object_id = i.object_id AND ic.index_id = i.index_id
JOIN sys.columns AS c ON c.object_id = ic.object_id AND c.column_id = ic.column_id
WHERE i.object_id = @p1 AND i.name IS NOT NULL AND ic.is_included_column = 0
GROUP BY i.name, i.type_desc, i.is_unique, i.is_primary_key, i.filter_definition, i.is_disabled
ORDER BY i.name`, objectID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]IndexInfo, 0, 8)
	for rows.Next() {
		var item IndexInfo
		var columns string
		if err := rows.Scan(&item.Name, &item.Type, &item.Unique, &item.Primary, &columns, &item.Filter, &item.Disabled); err != nil {
			return nil, err
		}
		item.Columns = splitCSV(columns)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) triggersForTable(ctx context.Context, objectID int) ([]ObjectInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT OBJECT_SCHEMA_NAME(tr.object_id), tr.name, tr.type_desc,
       CONVERT(varchar(33), tr.create_date, 126),
       CONVERT(varchar(33), tr.modify_date, 126),
       tr.is_disabled
FROM sys.triggers AS tr
WHERE tr.parent_id = @p1
ORDER BY tr.name`, objectID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]ObjectInfo, 0, 4)
	for rows.Next() {
		var item ObjectInfo
		if err := rows.Scan(&item.Schema, &item.Name, &item.Type, &item.CreateDate, &item.ModifyDate, &item.Disabled); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
