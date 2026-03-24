package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ngtrvu/data-cli/internal/config"
	"github.com/ngtrvu/data-cli/internal/connector"
	_ "github.com/ngtrvu/data-cli/internal/connector/bigquery"
	_ "github.com/ngtrvu/data-cli/internal/connector/json"
	_ "github.com/ngtrvu/data-cli/internal/connector/postgres"
)

var (
	connectDriver      string
	connectProject     string
	connectDataset     string
	connectCredentials string
	connectTest        bool
)

var connectCmd = &cobra.Command{
	Use:   "connect <name> <dsn-or-path>",
	Short: "Save a named connection",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runConnect,
}

func init() {
	connectCmd.Flags().StringVar(&connectDriver, "driver", "", "driver: postgres, json, bigquery")
	connectCmd.Flags().StringVar(&connectProject, "project", "", "GCP project ID (bigquery)")
	connectCmd.Flags().StringVar(&connectDataset, "dataset", "", "BigQuery dataset")
	connectCmd.Flags().StringVar(&connectCredentials, "credentials", "", "service account JSON path (bigquery)")
	connectCmd.Flags().BoolVar(&connectTest, "test", false, "test connection before saving")
	rootCmd.AddCommand(connectCmd)
}

func runConnect(cmd *cobra.Command, args []string) error {
	name := args[0]
	cfg, err := config.Load(cfgPath())
	if err != nil {
		return err
	}

	var conn config.ConnectionConfig

	if connectDriver == "bigquery" {
		conn = config.ConnectionConfig{
			Driver:      "bigquery",
			Project:     connectProject,
			Dataset:     connectDataset,
			Credentials: connectCredentials,
		}
	} else {
		if len(args) < 2 {
			return fmt.Errorf("usage: data connect <name> <dsn-or-path>")
		}
		target := args[1]
		driver := inferDriver(target, connectDriver)
		conn = config.ConnectionConfig{Driver: driver}
		switch driver {
		case "json":
			conn.Path = target
		default:
			conn.DSN = target
		}
	}

	if connectTest {
		c, err := connector.Open(conn)
		if err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
		ctx := context.Background()
		if err := c.Connect(ctx); err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
		c.Close()
		fmt.Fprintln(os.Stderr, "Connection OK")
	}

	cfg.Connections[name] = conn
	if err := config.Save(cfgPath(), cfg); err != nil {
		return err
	}
	fmt.Printf("Connection %q saved.\n", name)
	return nil
}

func inferDriver(target, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if strings.HasPrefix(target, "postgres://") || strings.HasPrefix(target, "postgresql://") {
		return "postgres"
	}
	if strings.HasSuffix(target, ".json") {
		return "json"
	}
	return "postgres"
}
