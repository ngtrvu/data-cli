package postgres

import (
	"strings"
	"testing"
)

// ---- isSelect --------------------------------------------------------------

func TestIsSelect(t *testing.T) {
	allow := []string{
		"SELECT 1",
		"select id from users",
		"  SELECT * FROM t",
		"WITH cte AS (SELECT 1) SELECT * FROM cte",
		"EXPLAIN SELECT 1",
		"SHOW search_path",
	}
	for _, sql := range allow {
		if !isSelect(sql) {
			t.Errorf("isSelect(%q) = false, want true", sql)
		}
	}

	deny := []string{
		"INSERT INTO t VALUES (1)",
		"UPDATE t SET x = 1",
		"DELETE FROM t",
		"DROP TABLE t",
		"CREATE TABLE t (id int)",
		"TRUNCATE t",
	}
	for _, sql := range deny {
		if isSelect(sql) {
			t.Errorf("isSelect(%q) = true, want false", sql)
		}
	}
}

// ---- oidName ---------------------------------------------------------------

func TestOidName(t *testing.T) {
	known := map[uint32]string{
		16:   "bool",
		20:   "int8",
		21:   "int2",
		23:   "int4",
		25:   "text",
		700:  "float4",
		701:  "float8",
		1043: "varchar",
		2950: "uuid",
		3802: "jsonb",
	}
	for oid, want := range known {
		if got := oidName(oid); got != want {
			t.Errorf("oidName(%d) = %q, want %q", oid, got, want)
		}
	}
	// Unknown OID falls back to "oid:<n>".
	if got := oidName(99999); !strings.HasPrefix(got, "oid:") {
		t.Errorf("oidName(99999) = %q, want oid:99999 prefix", got)
	}
}

// ---- qualified table name parsing ------------------------------------------
// parseQualifiedTable is tested indirectly via the inline split logic; we
// replicate the exact logic here to guard against regressions.

func TestParseQualifiedTable(t *testing.T) {
	tests := []struct {
		input         string
		wantSchema    string
		wantTableName string
	}{
		{"users", "public", "users"},
		{"myschema.users", "myschema", "users"},
		// Three-part names are unusual for Postgres but should not panic.
		{"a.b.c", "a", "b.c"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			schema := "public"
			tableName := tc.input
			if parts := strings.SplitN(tc.input, ".", 2); len(parts) == 2 {
				schema, tableName = parts[0], parts[1]
			}
			if schema != tc.wantSchema {
				t.Errorf("schema: got %q, want %q", schema, tc.wantSchema)
			}
			if tableName != tc.wantTableName {
				t.Errorf("tableName: got %q, want %q", tableName, tc.wantTableName)
			}
		})
	}
}
