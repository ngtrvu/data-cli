package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/ngtrvu/data-cli/internal/config"
)

var listFormat string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured data sources",
	RunE:  runList,
}

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a saved connection",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemove,
}

func init() {
	listCmd.Flags().StringVar(&listFormat, "format", "table", "output format: table, json")
	rootCmd.AddCommand(listCmd, removeCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgPath())
	if err != nil {
		return err
	}

	// Sort names for stable output
	names := make([]string, 0, len(cfg.Connections))
	for name := range cfg.Connections {
		names = append(names, name)
	}
	sort.Strings(names)

	if listFormat == "json" {
		type entry struct {
			Name    string `json:"name"`
			Driver  string `json:"driver"`
			Details string `json:"details,omitempty"`
		}
		var list []entry
		for _, name := range names {
			conn := cfg.Connections[name]
			list = append(list, entry{
				Name:    name,
				Driver:  conn.Driver,
				Details: connectionDetails(conn),
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	}

	if len(names) == 0 {
		fmt.Println("No connections configured.")
		fmt.Printf("Add one with: data connect <name> <dsn>\n")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tDRIVER\tDETAILS")
	fmt.Fprintln(tw, "────\t──────\t───────")
	for _, name := range names {
		conn := cfg.Connections[name]
		fmt.Fprintf(tw, "%s\t%s\t%s\n", name, conn.Driver, connectionDetails(conn))
	}
	return tw.Flush()
}

func runRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	cfg, err := config.Load(cfgPath())
	if err != nil {
		return err
	}
	if _, ok := cfg.Connections[name]; !ok {
		return fmt.Errorf("connection %q not found", name)
	}
	delete(cfg.Connections, name)
	if err := config.Save(cfgPath(), cfg); err != nil {
		return err
	}
	fmt.Printf("Connection %q removed.\n", name)
	return nil
}

// connectionDetails returns a safe, non-sensitive summary of a connection.
func connectionDetails(conn config.ConnectionConfig) string {
	switch conn.Driver {
	case "postgres":
		if conn.DSN == "" {
			return ""
		}
		// Show DSN form but mask credentials
		if len(conn.DSN) > 4 && conn.DSN[:4] == "env:" {
			return conn.DSN
		}
		if len(conn.DSN) > 11 && conn.DSN[:11] == "gcp-secret:" {
			return "gcp-secret:***"
		}
		return maskDSN(conn.DSN)
	case "json":
		return conn.Path
	case "bigquery":
		return fmt.Sprintf("%s.%s", conn.Project, conn.Dataset)
	default:
		return ""
	}
}

// maskDSN hides the password in a postgres:// URL.
func maskDSN(dsn string) string {
	// postgres://user:pass@host:port/db → postgres://user:***@host:port/db
	start := len("postgres://")
	if len(dsn) <= start {
		return dsn
	}
	rest := dsn[start:]
	atIdx := -1
	for i, c := range rest {
		if c == '@' {
			atIdx = i
			break
		}
	}
	if atIdx < 0 {
		return dsn
	}
	userInfo := rest[:atIdx]
	hostPart := rest[atIdx:]
	colonIdx := -1
	for i, c := range userInfo {
		if c == ':' {
			colonIdx = i
			break
		}
	}
	if colonIdx < 0 {
		return dsn
	}
	return "postgres://" + userInfo[:colonIdx+1] + "***" + hostPart
}
