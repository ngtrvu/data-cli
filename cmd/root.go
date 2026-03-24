package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/ngtrvu/data-cli/internal/config"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "data",
	Short: "One data CLI for your AI agents",
}

func execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.data/config.toml)")
}

func cfgPath() string {
	if cfgFile != "" {
		return cfgFile
	}
	return config.DefaultPath()
}

func rowLimit(conn config.ConnectionConfig, defaults config.DefaultsConfig) int {
	if conn.RowLimit > 0 {
		return conn.RowLimit
	}
	if defaults.RowLimit > 0 {
		return defaults.RowLimit
	}
	return 500
}

func queryTimeout(conn config.ConnectionConfig, defaults config.DefaultsConfig) time.Duration {
	t := conn.Timeout
	if t == 0 {
		t = defaults.Timeout
	}
	if t == 0 {
		t = 30
	}
	return time.Duration(t) * time.Second
}
