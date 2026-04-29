package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const repo = "ngtrvu/data-cli"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the current version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade data-cli to the latest release",
	RunE:  runUpgrade,
}

func init() {
	rootCmd.AddCommand(versionCmd, upgradeCmd)
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	fmt.Println("Checking for updates...")

	latest, err := latestRelease()
	if err != nil {
		return fmt.Errorf("could not fetch latest release: %w", err)
	}

	current := strings.TrimPrefix(version, "v")
	latestClean := strings.TrimPrefix(latest, "v")

	if current == latestClean {
		fmt.Printf("Already on latest version (%s).\n", version)
		return nil
	}

	fmt.Printf("Upgrading %s → %s\n", version, latest)

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("could not resolve executable path: %w", err)
	}

	if err := downloadAndReplace(latest, execPath); err != nil {
		return err
	}

	fmt.Printf("Upgraded to %s.\n", latest)
	return nil
}

func latestRelease() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.TagName == "" {
		return "", fmt.Errorf("empty tag_name in response")
	}
	return payload.TagName, nil
}

func downloadAndReplace(tag, dest string) error {
	goos := runtime.GOOS
	arch := runtime.GOARCH
	ver := strings.TrimPrefix(tag, "v")

	archive := fmt.Sprintf("data-cli_%s_%s_%s.tar.gz", ver, goos, arch)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, archive)

	fmt.Printf("Downloading %s...\n", archive)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	binary, err := extractBinary(resp.Body, archive)
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Write to a temp file next to the binary, then rename atomically.
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, binary, 0755); err != nil {
		return fmt.Errorf("could not write new binary: %w", err)
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("could not replace binary: %w", err)
	}
	return nil
}

// extractBinary pulls the "data" binary out of the tarball.
func extractBinary(r io.Reader, archiveName string) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) == "data" && hdr.Typeflag == tar.TypeReg {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("binary 'data' not found in %s", archiveName)
}
