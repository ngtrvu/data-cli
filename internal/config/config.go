package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Connections map[string]ConnectionConfig `toml:"connections"`
	Defaults    DefaultsConfig              `toml:"defaults"`
}

type ConnectionConfig struct {
	Driver      string `toml:"driver"`
	DSN         string `toml:"dsn,omitempty"`
	Path        string `toml:"path,omitempty"`
	Project     string `toml:"project,omitempty"`
	Dataset     string `toml:"dataset,omitempty"`
	Credentials string `toml:"credentials,omitempty"`
	ReadOnly    bool   `toml:"readonly,omitempty"`
	RowLimit    int    `toml:"row_limit,omitempty"`
	Timeout     int    `toml:"timeout,omitempty"`
}

type DefaultsConfig struct {
	RowLimit int `toml:"row_limit"`
	Timeout  int `toml:"timeout"`
}

// DefaultPath resolves the config file location using this priority:
//  1. conf/config.toml next to the running binary  (portable / collocated install)
//  2. ~/.data/conf/config.toml                     (home-based install)
func DefaultPath() string {
	if exe, err := os.Executable(); err == nil {
		local := filepath.Join(filepath.Dir(exe), "config", "config.toml")
		if _, err := os.Stat(local); err == nil {
			return local
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".data", "config", "config.toml")
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Connections: make(map[string]ConnectionConfig),
		Defaults:    DefaultsConfig{RowLimit: 500, Timeout: 30},
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}

func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
