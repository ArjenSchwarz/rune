package cmd

import (
	"os"
	"os/exec"
	"testing"
)

// TestIntegrationAddPhase tests add-phase command workflows that require a
// real subprocess (e.g. simulating an OS-level write failure via ulimit).
func TestIntegrationAddPhase(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION=1 to run.")
	}

	tests := map[string]struct {
		name        string
		workflow    func(t *testing.T, tempDir string)
		description string
	}{
		"add_phase_write_failure_preserves_file": {
			name:        "Add Phase Write Failure Preserves File",
			description: "Test that a write failure partway through add-phase leaves the original file byte-for-byte intact",
			workflow:    testAddPhaseWriteFailurePreservesFile,
		},
	}

	for testName, tc := range tests {
		t.Run(testName, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "rune-integration-add-phase-"+testName)
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			oldDir, _ := os.Getwd()
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("failed to change directory: %v", err)
			}
			defer func() {
				_ = os.Chdir(oldDir)
			}()

			t.Logf("Running integration test: %s", tc.description)
			tc.workflow(t, tempDir)
		})
	}
}

// testAddPhaseWriteFailurePreservesFile is a regression test for T-1854:
// runAddPhase used to write the appended content back to the target file with
// a plain os.WriteFile, which opens the file with O_TRUNC before writing. If
// the write failed partway (disk full, quota exceeded, a file-size limit),
// the file was left truncated and the original content was lost.
//
// This reproduces the exact scenario confirmed during triage: copy
// examples/complex.md (4,169 bytes) to a working file, then run
// `rune add-phase` under `ulimit -f 1` so the write fails partway through
// with "file too large". Before the fix, the target file ends up truncated
// to whatever fit under the limit (e.g. 512 bytes) and no longer matches the
// original. After the fix (routing the write through
// task.WriteFileAtomic, which writes a temp file and renames it into place),
// the failed write never touches the original file at all.
func testAddPhaseWriteFailurePreservesFile(t *testing.T, tempDir string) {
	examplePath, err := getExamplePath("examples/complex.md")
	if err != nil {
		t.Fatalf("failed to resolve example file: %v", err)
	}

	originalContent, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("failed to read example file: %v", err)
	}

	const filename = "tasks.md"
	if err := os.WriteFile(filename, originalContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Run add-phase in a subprocess with a 1-block (512 byte) file-size
	// ulimit so the write fails partway through, mirroring the ulimit -f 1
	// reproduction confirmed during triage. This must be a real subprocess:
	// applying the limit to the test binary's own process would risk
	// crashing or corrupting unrelated test state.
	cmd := exec.Command("sh", "-c", `ulimit -f 1 && exec "$0" add-phase "$1" "$2"`,
		runeBinaryPath, filename, "Test Phase")
	output, runErr := cmd.CombinedOutput()

	if runErr == nil {
		t.Fatalf("expected add-phase to fail under ulimit -f 1, but it succeeded. Output: %s", output)
	}
	t.Logf("add-phase output (expected failure): %s", output)

	// The core regression assertion: the file on disk must be byte-for-byte
	// identical to the original, not truncated.
	currentContent, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read file after failed add-phase: %v", err)
	}
	if len(currentContent) != len(originalContent) || string(currentContent) != string(originalContent) {
		t.Errorf("original file was truncated/modified by a failed write: got %d bytes, want %d bytes (original preserved)",
			len(currentContent), len(originalContent))
	}

	// No leftover temp file from the atomic write attempt.
	if _, err := os.Stat(filename + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("temporary file %s.tmp should have been cleaned up", filename)
	}
}
