package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ngtrvu/data-cli/internal/connector"
)

// makeResult builds a Result from plain column names and row values.
func makeResult(cols []string, rows [][]any, elapsed time.Duration) *connector.Result {
	columns := make([]connector.Column, len(cols))
	for i, name := range cols {
		columns[i] = connector.Column{Name: name, Type: "text"}
	}
	return &connector.Result{Columns: columns, Rows: rows, Elapsed: elapsed}
}

// ---- FormatValue -----------------------------------------------------------

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"nil", nil, ""},
		{"zero float64", float64(0), "0"},
		{"integer float64", float64(42), "42"},
		{"decimal float64", float64(1.5), "1.5"},
		// The user-reported case: large financial float shown in scientific notation before fix.
		{"large float64", float64(33599163017), "33599163017"},
		{"negative float64", float64(-99.99), "-99.99"},
		{"zero float32", float32(0), "0"},
		{"large float32", float32(1500000), "1500000"},
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatValue(tc.input)
			if got != tc.want {
				t.Errorf("FormatValue(%v) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFormatValueNoScientificNotation(t *testing.T) {
	floats := []float64{
		3.3599163017e+10,
		1e15,
		-2.5e8,
	}
	for _, f := range floats {
		got := FormatValue(f)
		if strings.ContainsAny(got, "eE") {
			t.Errorf("FormatValue(%v) = %q contains scientific notation", f, got)
		}
	}
}

// ---- writeTable ------------------------------------------------------------

func TestWriteTable(t *testing.T) {
	result := makeResult(
		[]string{"id", "name", "amount"},
		[][]any{
			{1, "alpha", float64(33599163017)},
			{2, "beta", nil},
		},
		42*time.Millisecond,
	)

	var buf bytes.Buffer
	if err := Write(&buf, result, FormatTable); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	// Headers must appear.
	for _, h := range []string{"id", "name", "amount"} {
		if !strings.Contains(out, h) {
			t.Errorf("table output missing header %q", h)
		}
	}
	// Separator row.
	if !strings.Contains(out, "──") {
		t.Error("table output missing separator")
	}
	// Row data: large float must be plain decimal.
	if !strings.Contains(out, "33599163017") {
		t.Error("table output: large float not formatted as plain decimal")
	}
	// nil should render as empty string, not "<nil>".
	if strings.Contains(out, "<nil>") {
		t.Error("table output: nil rendered as <nil>, want empty string")
	}
	// Footer.
	if !strings.Contains(out, "2 rows") {
		t.Error("table output missing row-count footer")
	}
	if !strings.Contains(out, "42ms") {
		t.Error("table output missing elapsed-time footer")
	}
}

func TestWriteTableEmpty(t *testing.T) {
	result := makeResult([]string{"col"}, nil, 0)
	var buf bytes.Buffer
	if err := Write(&buf, result, FormatTable); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "col") {
		t.Error("empty table: header missing")
	}
	if !strings.Contains(out, "0 rows") {
		t.Error("empty table: footer missing")
	}
}

// ---- writeMarkdown ---------------------------------------------------------

func TestWriteMarkdown(t *testing.T) {
	result := makeResult(
		[]string{"a", "b"},
		[][]any{{float64(1e10), "hello"}},
		10*time.Millisecond,
	)

	var buf bytes.Buffer
	if err := Write(&buf, result, FormatMarkdown); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	lines := strings.Split(strings.TrimSpace(out), "\n")

	// Header row contains pipes and column names.
	if !strings.Contains(lines[0], "| a") || !strings.Contains(lines[0], "b |") {
		t.Errorf("markdown header line malformed: %q", lines[0])
	}
	// Separator row.
	if !strings.Contains(lines[1], "---") {
		t.Errorf("markdown separator line malformed: %q", lines[1])
	}
	// Data row: large float must be plain decimal.
	if !strings.Contains(out, "10000000000") {
		t.Error("markdown: large float not plain decimal")
	}
	// Footer.
	if !strings.Contains(out, "1 rows") {
		t.Error("markdown: missing row-count footer")
	}
	if !strings.Contains(out, "10ms") {
		t.Error("markdown: missing elapsed footer")
	}
}

// ---- writeCSV --------------------------------------------------------------

func TestWriteCSV(t *testing.T) {
	result := makeResult(
		[]string{"x", "y"},
		[][]any{{"foo", float64(1.5)}, {nil, float64(0)}},
		0,
	)

	var buf bytes.Buffer
	if err := Write(&buf, result, FormatCSV); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")

	if len(lines) != 3 {
		t.Fatalf("CSV: want 3 lines (header + 2 rows), got %d", len(lines))
	}
	if lines[0] != "x,y" {
		t.Errorf("CSV header: got %q, want %q", lines[0], "x,y")
	}
	if lines[1] != "foo,1.5" {
		t.Errorf("CSV row 1: got %q, want %q", lines[1], "foo,1.5")
	}
	// nil renders as empty field.
	if lines[2] != ",0" {
		t.Errorf("CSV row 2: got %q, want %q", lines[2], ",0")
	}
}

// ---- writeJSON -------------------------------------------------------------

func TestWriteJSON(t *testing.T) {
	result := makeResult(
		[]string{"id", "val"},
		[][]any{{1, "hello"}, {2, nil}},
		0,
	)

	var buf bytes.Buffer
	if err := Write(&buf, result, FormatJSON); err != nil {
		t.Fatal(err)
	}

	var rows []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rows); err != nil {
		t.Fatalf("JSON output is not valid: %v\noutput: %s", err, buf.String())
	}
	if len(rows) != 2 {
		t.Fatalf("JSON: want 2 objects, got %d", len(rows))
	}
	if rows[0]["id"] == nil || rows[0]["val"] != "hello" {
		t.Errorf("JSON row 0 unexpected: %v", rows[0])
	}
	if rows[1]["val"] != nil {
		t.Errorf("JSON row 1: nil val should be JSON null, got %v", rows[1]["val"])
	}
}
