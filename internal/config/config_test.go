package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalPath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	got := LocalPath()
	want := filepath.Join(cwd, ".data", "config.toml")
	if got != want {
		t.Errorf("LocalPath() = %q, want %q", got, want)
	}
}

func TestGlobalPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := GlobalPath()
	if !strings.HasPrefix(got, home) {
		t.Errorf("GlobalPath() = %q, expected path under home dir %q", got, home)
	}
	if !strings.HasSuffix(got, "config.toml") {
		t.Errorf("GlobalPath() = %q, expected to end in config.toml", got)
	}
}

func TestFileExists(t *testing.T) {
	f, err := os.CreateTemp("", "config_test_*.toml")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	if !FileExists(f.Name()) {
		t.Errorf("FileExists(%q) = false, want true for existing file", f.Name())
	}
	if FileExists(f.Name() + ".nonexistent") {
		t.Errorf("FileExists returned true for non-existent file")
	}
}

func TestDefaultPath_prefersLocalOverGlobal(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		orig, _ := os.Getwd()
		_ = orig
	}()

	// No local config yet — should fall through to global path.
	got := DefaultPath()
	local := filepath.Join(dir, ".data", "config.toml")
	if got == local {
		t.Errorf("DefaultPath() returned local path before local config was created")
	}

	// Create local config.
	if err := os.MkdirAll(filepath.Dir(local), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(local, []byte("[defaults]\n"), 0600); err != nil {
		t.Fatal(err)
	}

	got = DefaultPath()
	// Resolve symlinks on both sides (macOS /var → /private/var).
	gotReal, _ := filepath.EvalSymlinks(got)
	localReal, _ := filepath.EvalSymlinks(local)
	if gotReal != localReal {
		t.Errorf("DefaultPath() = %q, want %q (local config should win)", got, local)
	}
}

func TestLoad_missingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.toml")
	if err != nil {
		t.Fatalf("Load on missing file returned error: %v", err)
	}
	if cfg.Defaults.RowLimit != 500 {
		t.Errorf("default RowLimit = %d, want 500", cfg.Defaults.RowLimit)
	}
	if cfg.Defaults.Timeout != 30 {
		t.Errorf("default Timeout = %d, want 30", cfg.Defaults.Timeout)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	orig := &Config{
		Connections: map[string]ConnectionConfig{
			"test": {Driver: "postgres", DSN: "postgres://localhost/db"},
		},
		Defaults: DefaultsConfig{RowLimit: 100, Timeout: 10},
	}

	if err := Save(path, orig); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	conn, ok := loaded.Connections["test"]
	if !ok {
		t.Fatal("connection 'test' not found after load")
	}
	if conn.Driver != "postgres" {
		t.Errorf("Driver = %q, want postgres", conn.Driver)
	}
	if loaded.Defaults.RowLimit != 100 {
		t.Errorf("RowLimit = %d, want 100", loaded.Defaults.RowLimit)
	}
}
