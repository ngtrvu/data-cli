package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveSQL(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		args    []string
		want    string
		wantErr string
	}{
		{
			name: "inline sql arg",
			file: "",
			args: []string{"conn", "SELECT 1"},
			want: "SELECT 1",
		},
		{
			name:    "no sql and no file",
			file:    "",
			args:    []string{"conn"},
			wantErr: "provide SQL as an argument or use --file",
		},
	}

	dir := t.TempDir()

	// file exists
	sqlPath := filepath.Join(dir, "query.sql")
	if err := os.WriteFile(sqlPath, []byte("SELECT * FROM orders"), 0600); err != nil {
		t.Fatal(err)
	}
	tests = append(tests, struct {
		name    string
		file    string
		args    []string
		want    string
		wantErr string
	}{
		name: "reads sql from file",
		file: sqlPath,
		args: []string{"conn"},
		want: "SELECT * FROM orders",
	})

	// file not found
	tests = append(tests, struct {
		name    string
		file    string
		args    []string
		want    string
		wantErr string
	}{
		name:    "file not found",
		file:    filepath.Join(dir, "missing.sql"),
		args:    []string{"conn"},
		wantErr: "read file:",
	})

	// --file takes priority over inline arg
	tests = append(tests, struct {
		name    string
		file    string
		args    []string
		want    string
		wantErr string
	}{
		name: "file flag takes priority over inline arg",
		file: sqlPath,
		args: []string{"conn", "SELECT 1"},
		want: "SELECT * FROM orders",
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveSQL(tc.file, tc.args)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveSQL_WithVars(t *testing.T) {
	dir := t.TempDir()
	sqlPath := filepath.Join(dir, "query.sql")
	if err := os.WriteFile(sqlPath, []byte("SELECT * FROM orders WHERE date > '{{cutoff}}'"), 0600); err != nil {
		t.Fatal(err)
	}

	sql, err := resolveSQL(sqlPath, []string{"conn"})
	if err != nil {
		t.Fatalf("resolveSQL: %v", err)
	}

	sql, err = applyVars(sql, []string{"cutoff=2026-01-01"})
	if err != nil {
		t.Fatalf("applyVars: %v", err)
	}

	want := "SELECT * FROM orders WHERE date > '2026-01-01'"
	if sql != want {
		t.Errorf("got %q, want %q", sql, want)
	}
}
