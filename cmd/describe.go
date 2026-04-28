package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/ngtrvu/data-cli/internal/config"
	"github.com/ngtrvu/data-cli/internal/connector"
	_ "github.com/ngtrvu/data-cli/internal/connector/bigquery"
	_ "github.com/ngtrvu/data-cli/internal/connector/json"
	_ "github.com/ngtrvu/data-cli/internal/connector/postgres"
	"github.com/ngtrvu/data-cli/internal/output"
)

var describeCmd = &cobra.Command{
	Use:   "describe <connection> <table>",
	Short: "Show row count and per-column stats (min, max, non-null count)",
	Args:  cobra.ExactArgs(2),
	RunE:  runDescribe,
}

func init() {
	rootCmd.AddCommand(describeCmd)
}

func runDescribe(cmd *cobra.Command, args []string) error {
	connName, table := args[0], args[1]

	cfg, err := config.Load(cfgPath())
	if err != nil {
		return err
	}
	connCfg, ok := cfg.Connections[connName]
	if !ok {
		return fmt.Errorf("connection %q not found", connName)
	}

	c, err := connector.Open(connCfg)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	if err := c.Connect(ctx); err != nil {
		return err
	}
	defer c.Close()

	cols, err := c.DescribeTable(ctx, table)
	if err != nil {
		return err
	}

	// Build a single aggregation query: COUNT(*) + per-column COUNT/MIN/MAX.
	selects := make([]string, 0, 1+len(cols)*3)
	selects = append(selects, "COUNT(*) AS _total")
	for _, col := range cols {
		n := col.Name
		selects = append(selects,
			fmt.Sprintf(`COUNT("%s") AS "%s__nn"`, n, n),
			fmt.Sprintf(`MIN("%s") AS "%s__min"`, n, n),
			fmt.Sprintf(`MAX("%s") AS "%s__max"`, n, n),
		)
	}
	sql := fmt.Sprintf("SELECT %s FROM %s", strings.Join(selects, ", "), table)

	result, err := c.Query(ctx, sql, connector.QueryOptions{
		Timeout: queryTimeout(connCfg, cfg.Defaults),
	})
	if err != nil {
		return fmt.Errorf("stats query: %w", err)
	}

	// Map result columns → values.
	stats := make(map[string]string, len(result.Columns))
	if len(result.Rows) > 0 {
		for i, col := range result.Columns {
			stats[col.Name] = output.FormatValue(result.Rows[0][i])
		}
	}

	fmt.Printf("Table: %s\n%s rows\n\n", table, stats["_total"])

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "COLUMN\tTYPE\tNON_NULL\tMIN\tMAX")
	fmt.Fprintln(tw, "──────\t────\t────────\t───\t───")
	for _, col := range cols {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			col.Name, col.Type,
			stats[col.Name+"__nn"],
			stats[col.Name+"__min"],
			stats[col.Name+"__max"],
		)
	}
	return tw.Flush()
}
