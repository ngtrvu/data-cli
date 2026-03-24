package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ngtrvu/data-cli/internal/config"
	"github.com/ngtrvu/data-cli/internal/connector"
	_ "github.com/ngtrvu/data-cli/internal/connector/bigquery"
	_ "github.com/ngtrvu/data-cli/internal/connector/json"
	_ "github.com/ngtrvu/data-cli/internal/connector/postgres"
	"github.com/ngtrvu/data-cli/internal/output"
)

var (
	queryFormat string
	queryLimit  int
)

var queryCmd = &cobra.Command{
	Use:   "query <connection> <sql>",
	Short: "Run a SQL query",
	Args:  cobra.ExactArgs(2),
	RunE:  runQuery,
}

func init() {
	queryCmd.Flags().StringVar(&queryFormat, "format", "table", "output format: table, csv, json, md")
	queryCmd.Flags().IntVar(&queryLimit, "limit", 0, "max rows (overrides config)")
	rootCmd.AddCommand(queryCmd)
}

func runQuery(cmd *cobra.Command, args []string) error {
	connName, sql := args[0], args[1]

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

	limit := queryLimit
	if limit == 0 {
		limit = rowLimit(connCfg, cfg.Defaults)
	}

	result, err := c.Query(ctx, sql, connector.QueryOptions{
		RowLimit: limit,
		Timeout:  queryTimeout(connCfg, cfg.Defaults),
		ReadOnly: connCfg.ReadOnly,
	})
	if err != nil {
		return err
	}

	return output.Write(os.Stdout, result, output.Format(queryFormat))
}
