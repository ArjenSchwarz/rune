package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runRuneIn runs the built rune binary with dir as its working directory and
// returns combined output and the error.
func runRuneIn(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	if runeBinaryPath == "" {
		t.Fatal("rune binary path not set - TestMain should have built the binary")
	}
	c := exec.Command(runeBinaryPath, args...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	return string(out), err
}

// TestIntegrationDiscoveryFromSubdirectory is a CLI round trip for T-959: a
// file found by git discovery from a repo subdirectory must be opened and
// written correctly, and the path rune reports must be usable again from the
// same caller directory.
func TestIntegrationDiscoveryFromSubdirectory(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION=1 to run.")
	}

	repo, subDir := setupSubdirDiscoveryRepo(t)
	taskFile := filepath.Join(repo, "specs", "my-feature", "tasks.md")

	// Write via discovery and capture the reported path.
	out, err := runRuneIn(t, subDir, "add-phase", "Planning", "--format", "json")
	if err != nil {
		t.Fatalf("add-phase from subdirectory failed: %v\n%s", err, out)
	}
	var resp AddPhaseResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("invalid JSON from add-phase: %v\n%s", err, out)
	}
	content, err := os.ReadFile(taskFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "## Planning") {
		t.Fatalf("phase not written to repo-root task file:\n%s", content)
	}

	// The reported path must resolve to the task file from the caller's
	// directory, and be accepted by another command run there.
	reported := resp.File
	if !filepath.IsAbs(reported) {
		reported = filepath.Join(subDir, reported)
	}
	if !sameFile(t, reported, taskFile) {
		t.Fatalf("reported file %q does not resolve to %q from %s", resp.File, taskFile, subDir)
	}
	if out, err := runRuneIn(t, subDir, "list", resp.File, "--format", "json"); err != nil {
		t.Fatalf("list with reported path %q failed from subdirectory: %v\n%s", resp.File, err, out)
	}

	// Text and dry-run output report the same reusable path.
	out, err = runRuneIn(t, subDir, "add-phase", "Review", "--dry-run")
	if err != nil {
		t.Fatalf("add-phase --dry-run failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Would add phase to file: "+resp.File+"\n") {
		t.Errorf("dry-run output does not report %q:\n%s", resp.File, out)
	}
	out, err = runRuneIn(t, subDir, "complete", "1", "--verbose")
	if err != nil {
		t.Fatalf("complete via discovery failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, resp.File) {
		t.Errorf("complete output does not report %q:\n%s", resp.File, out)
	}
	if content, _ := os.ReadFile(taskFile); !strings.Contains(string(content), "- [x] 1. First task") {
		t.Errorf("task 1 not completed in repo-root task file:\n%s", content)
	}

	// Explicit paths keep resolving against the caller's directory: reading
	// works, while writing outside it is still rejected.
	rel := filepath.Join("..", "..", "specs", "my-feature", "tasks.md")
	if out, err := runRuneIn(t, subDir, "list", rel); err != nil {
		t.Errorf("explicit relative read failed: %v\n%s", err, out)
	}
	if out, err := runRuneIn(t, subDir, "uncomplete", rel, "1"); err == nil || !strings.Contains(out, "path traversal") {
		t.Errorf("explicit write outside the caller directory should still be rejected, got err=%v\n%s", err, out)
	}

	// From the repo root the reported path stays repo-relative.
	out, err = runRuneIn(t, repo, "add-phase", "Done", "--format", "json")
	if err != nil {
		t.Fatalf("add-phase from repo root failed: %v\n%s", err, out)
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if resp.File != "specs/my-feature/tasks.md" {
		t.Errorf("file reported from repo root = %q, want specs/my-feature/tasks.md", resp.File)
	}
}

func sameFile(t *testing.T, a, b string) bool {
	t.Helper()
	ai, err := os.Stat(a)
	if err != nil {
		return false
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(ai, bi)
}
