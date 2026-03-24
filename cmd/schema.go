package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/ngtrvu/data-cli/internal/config"
	"github.com/ngtrvu/data-cli/internal/connector"
	_ "github.com/ngtrvu/data-cli/internal/connector/bigquery"
	_ "github.com/ngtrvu/data-cli/internal/connector/json"
	_ "github.com/ngtrvu/data-cli/internal/connector/postgres"
)

var schemaFormat string

var schemaCmd = &cobra.Command{
	Use:   "schema <connection> [table]",
	Short: "Inspect tables and columns",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runSchema,
}

func init() {
	schemaCmd.Flags().StringVar(&schemaFormat, "format", "table", "output format: table, json")
	rootCmd.AddCommand(schemaCmd)
}

func runSchema(cmd *cobra.Command, args []string) error {
	connName := args[0]

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

	if len(args) == 1 {
		return listTables(ctx, c, connName)
	}
	return describeTable(ctx, c, args[1])
}

func listTables(ctx context.Context, c connector.Connector, connName string) error {
	tables, err := c.ListTables(ctx)
	if err != nil {
		return err
	}
	if schemaFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(tables)
	}
	fmt.Printf("Tables in %s\n", connName)
	for _, t := range tables {
		fmt.Printf("  %s\n", t)
	}
	return nil
}

func describeTable(ctx context.Context, c connector.Connector, table string) error {
	cols, err := c.DescribeTable(ctx, table)
	if err != nil {
		return err
	}
	if schemaFormat == "json" {
		type colJSON struct {
			Name     string  `json:"name"`
			Type     string  `json:"type"`
			Nullable bool    `json:"nullable"`
			Default  *string `json:"default"`
		}
		out := struct {
			Table   string    `json:"table"`
			Columns []colJSON `json:"columns"`
		}{Table: table}
		for _, col := range cols {
			out.Columns = append(out.Columns, colJSON{
				Name:     col.Name,
				Type:     col.Type,
				Nullable: col.Nullable,
				Default:  col.Default,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	fmt.Printf("Table: %s\n", table)
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "COLUMN\tTYPE\tNULLABLE\tDEFAULT")
	fmt.Fprintln(tw, "──────\t────\t────────\t───────")
	for _, col := range cols {
		nullable := "NO"
		if col.Nullable {
			nullable = "YES"
		}
		def := ""
		if col.Default != nil {
			def = *col.Default
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", col.Name, col.Type, nullable, def)
	}
	return tw.Flush()
}
