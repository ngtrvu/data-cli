package config

import (
	"fmt"
	"os"
	"strings"
)

// Resolve resolves a DSN string. Supports:
//   - literal:    "postgres://user:pass@host/db"
//   - env ref:    "env:DATABASE_URL"
//   - gcp secret: "gcp-secret:projects/p/secrets/s/versions/latest"
func Resolve(dsn string) (string, error) {
	switch {
	case strings.HasPrefix(dsn, "env:"):
		key := strings.TrimPrefix(dsn, "env:")
		val := os.Getenv(key)
		if val == "" {
			return "", fmt.Errorf("environment variable %q is not set", key)
		}
		return val, nil
	case strings.HasPrefix(dsn, "gcp-secret:"):
		return "", fmt.Errorf("gcp-secret DSN resolution not yet implemented")
	default:
		return dsn, nil
	}
}
