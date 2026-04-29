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

// LocalPath returns the local project config path: .data/config.toml in the CWD.
func LocalPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return filepath.Join(cwd, ".data", "config.toml")
}

// GlobalPath returns the user-level (home) config path.
func GlobalPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".data", "config", "config.toml")
}

// DefaultPath resolves the config file location using this priority:
//  1. .data/config.toml in the current working directory  (local project config)
//  2. config/config.toml next to the running binary       (portable / collocated install)
//  3. ~/.data/config/config.toml                          (home-based / global install)
func DefaultPath() string {
	if local := LocalPath(); FileExists(local) {
		return local
	}
	if exe, err := os.Executable(); err == nil {
		portable := filepath.Join(filepath.Dir(exe), "config", "config.toml")
		if FileExists(portable) {
			return portable
		}
	}
	return GlobalPath()
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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
