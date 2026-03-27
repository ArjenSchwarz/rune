package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/config"
)

func TestResolveFilenameConfigErrors(t *testing.T) {
	t.Run("returns error when config is invalid", func(t *testing.T) {
		config.ResetConfigCache()

		tempDir := t.TempDir()
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}
		t.Cleanup(func() {
			os.Chdir(originalDir)
			config.ResetConfigCache()
		})

		// Initialize git repo so getRepoRoot finds .rune.yml
		cmd := exec.Command("git", "-C", tempDir, "init")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to init git repo: %v", err)
		}

		// Write invalid config
		invalidYAML := "discovery:\n  enabled: not-a-boolean\n  template: [invalid\n"
		if err := os.WriteFile(filepath.Join(tempDir, ".rune.yml"), []byte(invalidYAML), 0644); err != nil {
			t.Fatal(err)
		}

		os.Chdir(tempDir)

		_, err = resolveFilename([]string{})
		if err == nil {
			t.Fatal("expected error for invalid config, got none")
		}
		if !strings.Contains(err.Error(), "configuration error") {
			t.Errorf("error should mention configuration error, got: %v", err)
		}
	})

	t.Run("returns error when config has unknown fields", func(t *testing.T) {
		config.ResetConfigCache()

		tempDir := t.TempDir()
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}
		t.Cleanup(func() {
			os.Chdir(originalDir)
			config.ResetConfigCache()
		})

		cmd := exec.Command("git", "-C", tempDir, "init")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to init git repo: %v", err)
		}

		unknownFieldYAML := "unknown_key: true\ndiscovery:\n  enabled: true\n"
		if err := os.WriteFile(filepath.Join(tempDir, ".rune.yml"), []byte(unknownFieldYAML), 0644); err != nil {
			t.Fatal(err)
		}

		os.Chdir(tempDir)

		_, err = resolveFilename([]string{})
		if err == nil {
			t.Fatal("expected error for unknown fields in config, got none")
		}
		if !strings.Contains(err.Error(), "configuration error") {
			t.Errorf("error should mention configuration error, got: %v", err)
		}
	})
}
