package sqlserver

import "testing"

func TestValidateSelectSQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{name: "select", sql: "SELECT * FROM dbo.Users"},
		{name: "cte", sql: "WITH cte AS (SELECT 1 AS id) SELECT * FROM cte"},
		{name: "comment then select", sql: "-- explain\nSELECT 1"},
		{name: "update blocked", sql: "UPDATE dbo.Users SET Name = 'A'", wantErr: true},
		{name: "dangerous blocked", sql: "SELECT * FROM OPENROWSET('x', 'y', 'z')", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateSelectSQL(tt.sql)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateWriteSQL(t *testing.T) {
	t.Parallel()

	if err := validateWriteSQL("UPDATE dbo.Users SET Name = 'A'", false, false); err != nil {
		t.Fatalf("update should be allowed: %v", err)
	}
	if err := validateWriteSQL("CREATE TABLE dbo.T (ID int)", false, false); err == nil {
		t.Fatalf("schema change should be blocked")
	}
	if err := validateWriteSQL("CREATE TABLE dbo.T (ID int)", false, true); err != nil {
		t.Fatalf("schema change should be allowed when enabled: %v", err)
	}
	if err := validateWriteSQL("EXEC xp_cmdshell 'dir'", false, true); err == nil {
		t.Fatalf("dangerous SQL should be blocked")
	}
}
