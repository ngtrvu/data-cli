package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/ngtrvu/data-cli/internal/connector"
)

type Format string

const (
	FormatTable    Format = "table"
	FormatCSV      Format = "csv"
	FormatJSON     Format = "json"
	FormatMarkdown Format = "md"
)

func Write(w io.Writer, result *connector.Result, format Format) error {
	switch format {
	case FormatCSV:
		return writeCSV(w, result)
	case FormatJSON:
		return writeJSON(w, result)
	case FormatMarkdown:
		return writeMarkdown(w, result)
	default:
		return writeTable(w, result)
	}
}

func writeTable(w io.Writer, result *connector.Result) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	headers := colNames(result)
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	seps := make([]string, len(headers))
	for i, h := range headers {
		seps[i] = strings.Repeat("─", len(h))
	}
	fmt.Fprintln(tw, strings.Join(seps, "\t"))
	for _, row := range result.Rows {
		fmt.Fprintln(tw, strings.Join(rowStrings(row), "\t"))
	}
	tw.Flush()
	fmt.Fprintf(w, "\n%d rows  (%dms)\n", len(result.Rows), result.Elapsed.Milliseconds())
	return nil
}

func writeCSV(w io.Writer, result *connector.Result) error {
	cw := csv.NewWriter(w)
	cw.Write(colNames(result))
	for _, row := range result.Rows {
		cw.Write(rowStrings(row))
	}
	cw.Flush()
	return cw.Error()
}

func writeJSON(w io.Writer, result *connector.Result) error {
	out := make([]map[string]any, 0, len(result.Rows))
	for _, row := range result.Rows {
		m := make(map[string]any, len(row))
		for i, v := range row {
			m[result.Columns[i].Name] = v
		}
		out = append(out, m)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeMarkdown(w io.Writer, result *connector.Result) error {
	headers := colNames(result)
	fmt.Fprintf(w, "| %s |\n", strings.Join(headers, " | "))
	seps := make([]string, len(headers))
	for i := range seps {
		seps[i] = "---"
	}
	fmt.Fprintf(w, "| %s |\n", strings.Join(seps, " | "))
	for _, row := range result.Rows {
		fmt.Fprintf(w, "| %s |\n", strings.Join(rowStrings(row), " | "))
	}
	return nil
}

func colNames(result *connector.Result) []string {
	names := make([]string, len(result.Columns))
	for i, c := range result.Columns {
		names[i] = c.Name
	}
	return names
}

func rowStrings(row []any) []string {
	s := make([]string, len(row))
	for i, v := range row {
		s[i] = fmt.Sprintf("%v", v)
	}
	return s
}
