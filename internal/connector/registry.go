package connector

import (
	"fmt"

	"github.com/ngtrvu/data-cli/internal/config"
)

// Factory creates a Connector from a ConnectionConfig.
type Factory func(cfg config.ConnectionConfig) (Connector, error)

var registry = map[string]Factory{}

// Register adds a driver factory. Called from each driver's init().
func Register(driver string, fn Factory) {
	registry[driver] = fn
}

// Open returns a Connector for the given config.
func Open(cfg config.ConnectionConfig) (Connector, error) {
	fn, ok := registry[cfg.Driver]
	if !ok {
		return nil, fmt.Errorf("unknown driver %q — supported: postgres, json, bigquery", cfg.Driver)
	}
	return fn(cfg)
}
