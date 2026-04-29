package main

import (
	"strings"
	"testing"
)

func TestApplyVars(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		vars    []string
		want    string
		wantErr string
	}{
		{
			name: "no vars",
			sql:  "SELECT 1",
			vars: nil,
			want: "SELECT 1",
		},
		{
			name: "single substitution",
			sql:  "WHERE date = '{{as_of}}'",
			vars: []string{"as_of=2026-04-23"},
			want: "WHERE date = '2026-04-23'",
		},
		{
			name: "multiple substitutions",
			sql:  "WHERE a = '{{x}}' AND b = '{{y}}'",
			vars: []string{"x=hello", "y=world"},
			want: "WHERE a = 'hello' AND b = 'world'",
		},
		{
			name: "repeated placeholder",
			sql:  "{{v}} OR {{v}}",
			vars: []string{"v=1"},
			want: "1 OR 1",
		},
		{
			name: "value contains equals sign",
			sql:  "WHERE expr = '{{val}}'",
			vars: []string{"val=a=b"},
			want: "WHERE expr = 'a=b'",
		},
		{
			name:    "missing equals sign",
			sql:     "SELECT 1",
			vars:    []string{"badvar"},
			wantErr: "expected key=value",
		},
		{
			name: "unused placeholder left intact",
			sql:  "WHERE x = '{{missing}}'",
			vars: []string{"other=val"},
			want: "WHERE x = '{{missing}}'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := applyVars(tc.sql, tc.vars)
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
