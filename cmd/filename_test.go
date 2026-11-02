package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/config"
	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
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

// setupSubdirDiscoveryRepo creates a git repo on branch feature/my-feature with
// specs/my-feature/tasks.md and a .rune.yml at the root, and returns the repo
// root and a nested subdirectory (src/pkg).
func setupSubdirDiscoveryRepo(t *testing.T) (string, string) {
	t.Helper()
	repo := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	run("init", "--quiet")
	run("checkout", "-b", "feature/my-feature", "--quiet")

	for _, dir := range []string{"specs/my-feature", "src/pkg"} {
		if err := os.MkdirAll(filepath.Join(repo, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		"specs/my-feature/tasks.md": "# Tasks\n\n- [ ] 1. First task\n",
		".rune.yml":                 "discovery:\n  enabled: true\n  template: specs/{branch}/tasks.md\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	run("add", ".")
	run("-c", "user.email=a@example.com", "-c", "user.name=A", "commit", "--quiet", "-m", "init")

	return repo, filepath.Join(repo, "src", "pkg")
}

// TestResolveFilenameFromSubdirectory is a regression test for T-959: git
// discovery returned a repo-relative path, which commands then opened relative
// to the current subdirectory and failed with "no such file or directory".
func TestResolveFilenameFromSubdirectory(t *testing.T) {
	repo, subDir := setupSubdirDiscoveryRepo(t)

	config.ResetConfigCache()
	t.Cleanup(config.ResetConfigCache)
	t.Chdir(subDir)

	filename, err := resolveFilename([]string{})
	if err != nil {
		t.Fatalf("resolveFilename failed: %v", err)
	}

	// The returned path must be openable as-is by command code.
	tl, err := task.ParseFile(filename)
	if err != nil {
		t.Fatalf("discovered file %q not readable from subdirectory: %v", filename, err)
	}
	if len(tl.Tasks) != 1 || tl.Tasks[0].Title != "First task" {
		t.Fatalf("unexpected task list parsed from %q: %+v", filename, tl.Tasks)
	}

	// Writes go through the working-directory containment check, so they
	// must succeed for the discovered file too.
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("discovered file %q not writable from subdirectory: %v", filename, err)
	}
	if _, err := os.Stat(filepath.Join(repo, "specs", "my-feature", "tasks.md")); err != nil {
		t.Fatalf("task file missing at repo root after write: %v", err)
	}
}

// TestCompleteFromSubdirectoryWithDiscovery runs a mutating command end to end
// from a repo subdirectory using git discovery (T-959).
func TestCompleteFromSubdirectoryWithDiscovery(t *testing.T) {
	repo, subDir := setupSubdirDiscoveryRepo(t)

	config.ResetConfigCache()
	t.Cleanup(config.ResetConfigCache)
	t.Chdir(subDir)

	oldFormat, oldDryRun := format, dryRun
	format, dryRun = formatTable, false
	t.Cleanup(func() { format, dryRun = oldFormat, oldDryRun })

	if err := runComplete(&cobra.Command{}, []string{"1"}); err != nil {
		t.Fatalf("complete from subdirectory failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(repo, "specs", "my-feature", "tasks.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "- [x] 1. First task") {
		t.Errorf("task 1 not completed in repo-root task file, got:\n%s", content)
	}
}
