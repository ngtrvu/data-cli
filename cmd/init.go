package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/ngtrvu/data-cli/internal/config"
)

const initTemplate = `# Data CLI — local config
# Run "data connect <name> <dsn>" to add connections, or edit this file directly.
#
# Example:
#   [connections.prod]
#   driver    = "postgres"
#   dsn       = "postgres://user:password@localhost:5432/mydb"
#   readonly  = true
#   row_limit = 1000

[defaults]
row_limit = 500
timeout   = 30
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a local config file in the current directory",
	Args:  cobra.NoArgs,
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	path := config.LocalPath()
	if config.FileExists(path) {
		fmt.Printf("config already exists: %s\n", path)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("init config: %w", err)
	}
	if err := os.WriteFile(path, []byte(initTemplate), 0600); err != nil {
		return fmt.Errorf("init config: %w", err)
	}
	fmt.Printf("created %s\n", path)
	return nil
}
