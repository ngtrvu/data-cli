package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInit_createsConfig(t *testing.T) {
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	var out bytes.Buffer
	initCmd.SetOut(&out)

	if err := runInit(initCmd, nil); err != nil {
		t.Fatalf("runInit returned error: %v", err)
	}

	path := filepath.Join(dir, ".data", "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	if !strings.Contains(string(data), "[defaults]") {
		t.Errorf("config file missing [defaults] section, got:\n%s", data)
	}
}

func TestRunInit_alreadyExists(t *testing.T) {
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	// Pre-create the config.
	cfgDir := filepath.Join(dir, ".data")
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		t.Fatal(err)
	}
	existing := []byte("# existing\n")
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), existing, 0600); err != nil {
		t.Fatal(err)
	}

	if err := runInit(initCmd, nil); err != nil {
		t.Fatalf("runInit returned error: %v", err)
	}

	// File should remain unchanged.
	data, _ := os.ReadFile(filepath.Join(cfgDir, "config.toml"))
	if string(data) != string(existing) {
		t.Errorf("existing config was overwritten")
	}
}
