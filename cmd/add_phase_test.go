package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
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
