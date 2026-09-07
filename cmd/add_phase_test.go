package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

func TestAddPhaseCommand(t *testing.T) {
	tests := map[string]struct {
		existingContent string
		phaseName       string
		wantErr         bool
		wantContent     []string
	}{
		"add phase to empty task file": {
			existingContent: "# My Tasks\n\n",
			phaseName:       "Planning",
			wantErr:         false,
			wantContent: []string{
				"# My Tasks",
				"",
				"## Planning",
			},
		},
		"add phase to file with existing tasks": {
			existingContent: "# My Tasks\n\n- [ ] 1. First task\n- [ ] 2. Second task\n",
			phaseName:       "Implementation",
			wantErr:         false,
			wantContent: []string{
				"# My Tasks",
				"",
				"- [ ] 1. First task",
				"- [ ] 2. Second task",
				"## Implementation",
			},
		},
		"add phase to file with existing phases": {
			existingContent: "# My Tasks\n\n## Phase 1\n\n- [ ] 1. Task one\n\n## Phase 2\n\n- [ ] 2. Task two\n",
			phaseName:       "Phase 3",
			wantErr:         false,
			wantContent: []string{
				"# My Tasks",
				"",
				"## Phase 1",
				"",
				"- [ ] 1. Task one",
				"",
				"## Phase 2",
				"",
				"- [ ] 2. Task two",
				"## Phase 3",
			},
		},
		"add phase with special characters": {
			existingContent: "# My Tasks\n\n",
			phaseName:       "Q&A / Testing",
			wantErr:         false,
			wantContent: []string{
				"# My Tasks",
				"",
				"## Q&A / Testing",
			},
		},
		"add phase preserves empty phases": {
			existingContent: "# My Tasks\n\n## Empty Phase\n\n## Another Phase\n\n- [ ] 1. Task\n",
			phaseName:       "New Phase",
			wantErr:         false,
			wantContent: []string{
				"# My Tasks",
				"",
				"## Empty Phase",
				"",
				"## Another Phase",
				"",
				"- [ ] 1. Task",
				"## New Phase",
			},
		},
		"add phase with spaces in name": {
			existingContent: "# My Tasks\n\n",
			phaseName:       "  Phase with Spaces  ",
			wantErr:         false,
			wantContent: []string{
				"# My Tasks",
				"",
				"## Phase with Spaces",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp directory for test
			tempDir, err := os.MkdirTemp("", "rune-add-phase-test")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Change to temp directory
			oldDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(oldDir)

			// Create test file with existing content
			testFile := "tasks.md"
			if err := os.WriteFile(testFile, []byte(tc.existingContent), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			// Add the phase
			phaseHeader := "## " + strings.TrimSpace(tc.phaseName)

			// Read existing content
			content, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			// Ensure content ends with newline, then append phase
			contentStr := string(content)
			if !strings.HasSuffix(contentStr, "\n") {
				contentStr += "\n"
			}
			contentStr += phaseHeader + "\n"

			// Write back to file
			err = os.WriteFile(testFile, []byte(contentStr), 0644)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Read and verify content
			content, err = os.ReadFile(testFile)
			if err != nil {
				t.Errorf("failed to read result file: %v", err)
				return
			}

			lines := strings.Split(strings.TrimSpace(string(content)), "\n")
			for i, wantLine := range tc.wantContent {
				if i >= len(lines) {
					t.Errorf("expected line %d to be %q, but file has only %d lines", i, wantLine, len(lines))
					continue
				}
				if lines[i] != wantLine {
					t.Errorf("line %d: expected %q, got %q", i, wantLine, lines[i])
				}
			}

			// Ensure task list is still valid after adding phase
			_, err = task.ParseFile(testFile)
			if err != nil {
				t.Errorf("file is invalid after adding phase: %v", err)
			}
		})
	}
}

// TestRunAddPhaseRejectsNewline is a regression test for T-1603: the
// add-phase command must reject a phase name containing a newline instead of
// writing it verbatim into the "## {name}" header, which lets the phase name
// inject arbitrary markdown/task lines into the file.
func TestRunAddPhaseRejectsNewline(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rune-add-phase-newline-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	testFile := "tasks.md"
	original := "# My Tasks\n\n- [ ] 1. Existing task\n"
	if err := os.WriteFile(testFile, []byte(original), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	maliciousPhase := "Bad\n- [ ] 999. Injected"

	err = runAddPhase(&cobra.Command{}, []string{testFile, maliciousPhase})
	if err == nil {
		t.Fatal("expected error for phase name containing newline, got nil")
	}

	content, readErr := os.ReadFile(testFile)
	if readErr != nil {
		t.Fatalf("failed to read file: %v", readErr)
	}
	if string(content) != original {
		t.Errorf("file was modified despite validation error; got:\n%s", string(content))
	}
	if strings.Contains(string(content), "999. Injected") {
		t.Error("injected line was written to the task file")
	}
}

// TestRunAddPhaseTrimsTrailingNewline pins the behaviour that a phase name
// whose only problem is surrounding whitespace is trimmed and accepted, on
// every path. T-1603 briefly broke this on the batch path while the CLI path
// kept working, so the same logical operation had two different outcomes.
func TestRunAddPhaseTrimsTrailingNewline(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rune-add-phase-trim-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	testFile := "tasks.md"
	if err := os.WriteFile(testFile, []byte("# My Tasks\n\n- [ ] 1. Existing task\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if err := runAddPhase(&cobra.Command{}, []string{testFile, "Planning\n"}); err != nil {
		t.Fatalf("expected trailing newline to be trimmed and accepted, got: %v", err)
	}

	content, readErr := os.ReadFile(testFile)
	if readErr != nil {
		t.Fatalf("failed to read file: %v", readErr)
	}
	if !strings.Contains(string(content), "## Planning\n") {
		t.Errorf("expected trimmed phase header, got:\n%s", string(content))
	}
}

func TestAddPhaseCommandEmptyFile(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "rune-add-phase-empty-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Create empty test file
	testFile := "empty.md"
	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Add phase to empty file
	phaseHeader := "## First Phase\n"
	err = os.WriteFile(testFile, []byte(phaseHeader), 0644)
	if err != nil {
		t.Errorf("failed to add phase to empty file: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("failed to read result file: %v", err)
		return
	}

	if string(content) != phaseHeader {
		t.Errorf("expected content %q, got %q", phaseHeader, string(content))
	}
}

func TestAddPhaseCommandPreservesTaskStructure(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "rune-add-phase-structure-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Create test file with hierarchical tasks
	testFile := "tasks.md"
	existingContent := `# My Tasks

- [ ] 1. First task
  - [ ] 1.1. Subtask one
  - [ ] 1.2. Subtask two
- [ ] 2. Second task
  - [ ] 2.1. Another subtask
`
	if err := os.WriteFile(testFile, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse original file to get task count
	originalTl, err := task.ParseFile(testFile)
	if err != nil {
		t.Fatalf("failed to parse original file: %v", err)
	}
	originalTaskCount := len(originalTl.Tasks)
	if len(originalTl.Tasks) > 0 {
		originalTaskCount += len(originalTl.Tasks[0].Children)
	}
	if len(originalTl.Tasks) > 1 {
		originalTaskCount += len(originalTl.Tasks[1].Children)
	}

	// Add phase
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	contentStr := string(content)
	if !strings.HasSuffix(contentStr, "\n") {
		contentStr += "\n"
	}
	contentStr += "## New Phase\n"

	err = os.WriteFile(testFile, []byte(contentStr), 0644)
	if err != nil {
		t.Errorf("failed to add phase: %v", err)
	}

	// Parse modified file
	modifiedTl, err := task.ParseFile(testFile)
	if err != nil {
		t.Errorf("file is invalid after adding phase: %v", err)
		return
	}

	// Verify task count is preserved
	modifiedTaskCount := len(modifiedTl.Tasks)
	if len(modifiedTl.Tasks) > 0 {
		modifiedTaskCount += len(modifiedTl.Tasks[0].Children)
	}
	if len(modifiedTl.Tasks) > 1 {
		modifiedTaskCount += len(modifiedTl.Tasks[1].Children)
	}
	if originalTaskCount != modifiedTaskCount {
		t.Errorf("task count changed after adding phase: was %d, now %d", originalTaskCount, modifiedTaskCount)
	}

	// Verify task structure is preserved
	if len(originalTl.Tasks) != len(modifiedTl.Tasks) {
		t.Errorf("top-level task count changed: was %d, now %d", len(originalTl.Tasks), len(modifiedTl.Tasks))
	}

	// Check that phase header was added
	content, _ = os.ReadFile(testFile)
	if !strings.Contains(string(content), "## New Phase") {
		t.Error("phase header was not added to file")
	}
}

func TestAddPhaseCommandWithVariousFormats(t *testing.T) {
	tests := map[string]struct {
		phaseName      string
		expectedHeader string
	}{
		"simple name": {
			phaseName:      "Planning",
			expectedHeader: "## Planning",
		},
		"name with spaces": {
			phaseName:      "Implementation Phase",
			expectedHeader: "## Implementation Phase",
		},
		"name with numbers": {
			phaseName:      "Phase 1",
			expectedHeader: "## Phase 1",
		},
		"name with special chars": {
			phaseName:      "Q&A / Testing",
			expectedHeader: "## Q&A / Testing",
		},
		"name with leading/trailing spaces": {
			phaseName:      "  Trimmed  ",
			expectedHeader: "## Trimmed",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp directory for test
			tempDir, err := os.MkdirTemp("", "rune-add-phase-format-test")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Change to temp directory
			oldDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(oldDir)

			// Create test file
			testFile := "tasks.md"
			if err := os.WriteFile(testFile, []byte("# My Tasks\n\n"), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			// Add phase
			phaseHeader := "## " + strings.TrimSpace(tc.phaseName)

			content, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			contentStr := string(content)
			if !strings.HasSuffix(contentStr, "\n") {
				contentStr += "\n"
			}
			contentStr += phaseHeader + "\n"

			err = os.WriteFile(testFile, []byte(contentStr), 0644)
			if err != nil {
				t.Errorf("failed to add phase: %v", err)
			}

			// Read and verify content
			content, err = os.ReadFile(testFile)
			if err != nil {
				t.Errorf("failed to read result file: %v", err)
				return
			}

			if !strings.Contains(string(content), tc.expectedHeader) {
				t.Errorf("expected header %q not found in content:\n%s", tc.expectedHeader, string(content))
			}
		})
	}
}

// TestAddPhaseCommandRejectsPathOutsideWorkingDirectory is a regression test for T-1473/T-1752:
// runAddPhase used to read and write the target file directly via os.ReadFile/os.WriteFile
// without ever calling task.ValidateFilePath, so it could mutate a markdown file outside the
// current working directory. Every other mutating command enforces this containment check, so
// add-phase must too. Before the fix this test fails because the outside file gets modified;
// after the fix runAddPhase returns a path containment error and leaves the file untouched.
func TestAddPhaseCommandRejectsPathOutsideWorkingDirectory(t *testing.T) {
	// Run rune from an isolated temporary working directory. t.Chdir restores the
	// original working directory automatically when the test finishes.
	t.Chdir(t.TempDir())

	// The target file lives in a completely separate directory, outside the working directory.
	outsideDir := t.TempDir()

	outsideFile := filepath.Join(outsideDir, "tasks.md")
	originalContent := "# Outside\n\n- [ ] 1. Outside task\n"
	if err := os.WriteFile(outsideFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to create outside test file: %v", err)
	}

	cmd := &cobra.Command{}
	args := []string{outsideFile, "Escaped"}

	err := runAddPhase(cmd, args)
	if err == nil {
		t.Fatal("expected add-phase to reject a file outside the working directory, got nil error")
	}
	// Assert on both halves: the "invalid file path" prefix runAddPhase wraps around the
	// validator's error, and the containment reason task.ValidateFilePath itself returns.
	// Matching only one half would also accept an unrelated failure that happens to
	// mention it -- runAddPhase returns several other errors (not-exist, read, write)
	// that carry neither string.
	if !strings.Contains(err.Error(), "invalid file path") || !strings.Contains(err.Error(), "path traversal") {
		t.Errorf("expected a path containment error, got: %v", err)
	}

	// The outside file must be left completely untouched.
	content, readErr := os.ReadFile(outsideFile)
	if readErr != nil {
		t.Fatalf("failed to read outside file after rejected add-phase: %v", readErr)
	}
	if string(content) != originalContent {
		t.Errorf("outside file was modified despite path containment violation:\ngot:  %q\nwant: %q", string(content), originalContent)
	}
}

// TestAddPhaseCommandAcceptsPathInsideWorkingDirectory is the positive counterpart to
// TestAddPhaseCommandRejectsPathOutsideWorkingDirectory. The T-1473 fix added a check that
// can reject input, so this test pins down that it does not reject a legitimate path inside
// the working directory. Without it, `make test` alone cannot catch a regression where the
// containment check wrongly rejects a valid file -- only the INTEGRATION=1 suite would.
func TestAddPhaseCommandAcceptsPathInsideWorkingDirectory(t *testing.T) {
	t.Chdir(t.TempDir())

	// runAddPhase reads these package-level flag globals. Pin them to their defaults so the
	// test cannot be affected by a sibling test that left them set.
	verbose, format, dryRun = false, "table", false

	const filename = "tasks.md"
	originalContent := "# Inside\n\n- [ ] 1. Inside task\n"
	if err := os.WriteFile(filename, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	captureStdout(t, func() {
		if err := runAddPhase(&cobra.Command{}, []string{filename, "Planning"}); err != nil {
			t.Errorf("add-phase rejected a file inside the working directory: %v", err)
		}
	})

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read task file after add-phase: %v", err)
	}
	want := originalContent + "## Planning\n"
	if string(content) != want {
		t.Errorf("unexpected file content after add-phase:\ngot:  %q\nwant: %q", string(content), want)
	}
}

// TestAddPhaseCommandWriteFailurePreservesFile is a regression test for T-1854:
// runAddPhase used to write the appended content back to the target file with
// a plain os.WriteFile, which truncates the target (O_TRUNC) before writing
// any bytes. If the write failed partway through -- disk full, quota
// exceeded, a file-size limit -- the original file was left truncated and
// the pre-existing content was lost. See TestIntegrationAddPhase in
// cmd/integration_add_phase_test.go (INTEGRATION=1) for a faithful
// reproduction of the exact ulimit -f scenario confirmed during triage.
//
// This test forces the write step to fail deterministically, without
// needing ulimit/disk-full, by pre-creating "tasks.md.tmp" as a directory:
// task.WriteFileAtomic's os.WriteFile to that path then fails with "is a
// directory" before ever touching "tasks.md". Before the fix, runAddPhase
// wrote directly to "tasks.md" via os.WriteFile and never created a ".tmp"
// file at all, so this setup had no effect on it -- the write succeeded,
// silently truncating and overwriting the original content instead of
// failing. After the fix, the write fails and "tasks.md" is left
// byte-for-byte unchanged.
func TestAddPhaseCommandWriteFailurePreservesFile(t *testing.T) {
	t.Chdir(t.TempDir())

	// runAddPhase reads these package-level flag globals. Save and restore
	// them so this test cannot leak state into sibling tests.
	origVerbose, origFormat, origDryRun := verbose, format, dryRun
	t.Cleanup(func() { verbose, format, dryRun = origVerbose, origFormat, origDryRun })
	verbose, format, dryRun = false, "table", false

	const filename = "tasks.md"
	originalContent := "# My Tasks\n\n- [ ] 1. Existing task\n"
	if err := os.WriteFile(filename, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Block the atomic writer's temp file with a directory of the same name,
	// forcing the write step to fail.
	if err := os.Mkdir(filename+".tmp", 0755); err != nil {
		t.Fatalf("failed to create blocking directory: %v", err)
	}

	err := runAddPhase(&cobra.Command{}, []string{filename, "Planning"})
	if err == nil {
		t.Fatal("expected add-phase to fail when the write step cannot complete, got nil")
	}

	content, readErr := os.ReadFile(filename)
	if readErr != nil {
		t.Fatalf("failed to read file after failed add-phase: %v", readErr)
	}
	if string(content) != originalContent {
		t.Errorf("original file was modified despite write failure:\ngot:  %q\nwant: %q", string(content), originalContent)
	}
}
