package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

// TestCreateBackup tests the createBackup function
func TestCreateBackup(t *testing.T) {
	// Create temporary directory for test files within the current working directory
	// This is required due to path traversal protection in validateFilePath
	tempDir := filepath.Join(".", "test-tmp-renumber")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := map[string]struct {
		setupFile      func(string) error
		fileMode       os.FileMode
		validateBackup func(*testing.T, string, string)
		expectError    bool
		errorContains  string
	}{
		"create backup with correct content": {
			setupFile: func(filename string) error {
				content := []byte("# Test Tasks\n\n- [ ] Task 1\n- [ ] Task 2\n")
				return os.WriteFile(filename, content, 0644)
			},
			fileMode: 0644,
			validateBackup: func(t *testing.T, original, backup string) {
				// Read original file
				originalContent, err := os.ReadFile(original)
				if err != nil {
					t.Fatalf("Failed to read original file: %v", err)
				}

				// Read backup file
				backupContent, err := os.ReadFile(backup)
				if err != nil {
					t.Fatalf("Failed to read backup file: %v", err)
				}

				// Verify content matches
				if string(originalContent) != string(backupContent) {
					t.Errorf("Backup content does not match original.\nOriginal: %s\nBackup: %s",
						string(originalContent), string(backupContent))
				}
			},
			expectError: false,
		},
		"backup preserves file permissions": {
			setupFile: func(filename string) error {
				content := []byte("# Test Tasks\n\n- [ ] Task 1\n")
				return os.WriteFile(filename, content, 0600)
			},
			fileMode: 0600,
			validateBackup: func(t *testing.T, original, backup string) {
				// Get original file permissions
				originalInfo, err := os.Stat(original)
				if err != nil {
					t.Fatalf("Failed to stat original file: %v", err)
				}

				// Get backup file permissions
				backupInfo, err := os.Stat(backup)
				if err != nil {
					t.Fatalf("Failed to stat backup file: %v", err)
				}

				// Verify permissions match
				if originalInfo.Mode().Perm() != backupInfo.Mode().Perm() {
					t.Errorf("Backup permissions (%v) do not match original (%v)",
						backupInfo.Mode().Perm(), originalInfo.Mode().Perm())
				}
			},
			expectError: false,
		},
		"backup overwrites existing .bak file": {
			setupFile: func(filename string) error {
				// Create original file
				content := []byte("# New Content\n\n- [ ] New Task\n")
				if err := os.WriteFile(filename, content, 0644); err != nil {
					return err
				}

				// Create existing backup file with different content
				backupPath := filename + ".bak"
				oldContent := []byte("# Old Content\n\n- [ ] Old Task\n")
				return os.WriteFile(backupPath, oldContent, 0644)
			},
			fileMode: 0644,
			validateBackup: func(t *testing.T, original, backup string) {
				// Read original file
				originalContent, err := os.ReadFile(original)
				if err != nil {
					t.Fatalf("Failed to read original file: %v", err)
				}

				// Read backup file
				backupContent, err := os.ReadFile(backup)
				if err != nil {
					t.Fatalf("Failed to read backup file: %v", err)
				}

				// Verify backup was overwritten with new content
				if string(backupContent) != string(originalContent) {
					t.Errorf("Backup was not overwritten with new content.\nOriginal: %s\nBackup: %s",
						string(originalContent), string(backupContent))
				}

				// Verify backup does not contain old content
				if string(backupContent) == "# Old Content\n\n- [ ] Old Task\n" {
					t.Error("Backup still contains old content - was not overwritten")
				}
			},
			expectError: false,
		},
		"error handling for read failures": {
			setupFile: func(filename string) error {
				// Create a file but make it unreadable
				content := []byte("# Test Tasks\n")
				if err := os.WriteFile(filename, content, 0644); err != nil {
					return err
				}
				// Make file unreadable (chmod 000)
				return os.Chmod(filename, 0000)
			},
			fileMode:      0000,
			expectError:   true,
			errorContains: "reading original file",
		},
	}

	testNum := 0
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			testNum++
			// Create unique test file for each test case
			testFile := filepath.Join(tempDir, fmt.Sprintf("test-tasks-%d.md", testNum))

			// Ensure cleanup happens even on panic
			defer func() {
				// Restore permissions before cleanup
				if tc.fileMode == 0000 {
					os.Chmod(testFile, 0644)
				}
				os.Remove(testFile)
				os.Remove(testFile + ".bak")
			}()

			if tc.setupFile != nil {
				if err := tc.setupFile(testFile); err != nil {
					t.Fatalf("Failed to setup test file: %v", err)
				}
			}

			// Get file info
			fileInfo, err := os.Stat(testFile)
			if err != nil && !tc.expectError {
				t.Fatalf("Failed to stat test file: %v", err)
			}

			// Call createBackup
			backupPath, err := createBackup(testFile, fileInfo)

			// Check error expectations
			if tc.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tc.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify backup path
			expectedBackupPath := testFile + ".bak"
			if backupPath != expectedBackupPath {
				t.Errorf("Expected backup path '%s', got '%s'", expectedBackupPath, backupPath)
			}

			// Verify backup file exists
			if _, err := os.Stat(backupPath); os.IsNotExist(err) {
				t.Fatal("Backup file was not created")
			}

			// Run validation if provided
			if tc.validateBackup != nil {
				tc.validateBackup(t, testFile, backupPath)
			}
		})
	}
}

// TestRenumberValidation tests the validation phase of runRenumber
func TestRenumberValidation(t *testing.T) {
	// Create temporary directory for test files within the current working directory
	tempDir := filepath.Join(".", "test-tmp-renumber-validation")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := map[string]struct {
		setupFile     func(string) error
		filePath      string
		expectError   bool
		errorContains string
	}{
		"invalid path - path traversal": {
			filePath:      "../../../etc/passwd",
			expectError:   true,
			errorContains: "invalid file path",
		},
		"invalid path - absolute path outside working dir": {
			filePath:      "/tmp/tasks.md",
			expectError:   true,
			errorContains: "invalid file path",
		},
		"file does not exist": {
			filePath:      filepath.Join(tempDir, "nonexistent.md"),
			expectError:   true,
			errorContains: "file not found",
		},
		"file exceeds 10MB limit": {
			setupFile: func(filename string) error {
				// Create a file slightly larger than 10MB
				// MaxFileSize is 10 * 1024 * 1024 = 10485760 bytes
				largeContent := make([]byte, 10485761)
				for i := range largeContent {
					largeContent[i] = 'x'
				}
				return os.WriteFile(filename, largeContent, 0644)
			},
			expectError:   true,
			errorContains: "file exceeds 10MB limit",
		},
		"task count exceeds limit": {
			setupFile: func(filename string) error {
				// Create a task list with just over 10000 tasks
				// MaxTaskCount is 10000
				tl := task.NewTaskList("Test Tasks")
				// Add 10001 tasks to exceed the limit
				for range 10001 {
					tl.AddTask("", "Test task", "")
				}
				return tl.WriteFile(filename)
			},
			expectError:   true,
			errorContains: "task count",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var testFile string
			if tc.setupFile != nil {
				testFile = filepath.Join(tempDir, "test-tasks.md")
				if err := tc.setupFile(testFile); err != nil {
					t.Fatalf("Failed to setup test file: %v", err)
				}
				defer os.Remove(testFile)
			} else {
				testFile = tc.filePath
			}

			// Create a fake cobra command for testing
			cmd := &cobra.Command{}
			args := []string{testFile}

			// Call runRenumber
			err := runRenumber(cmd, args)

			// Check error expectations
			if tc.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tc.errorContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestRenumberValidationOrder tests that file size validation occurs before parsing
func TestRenumberValidationOrder(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-renumber-order")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file that is both too large AND contains invalid content
	// If file size validation happens first (as it should), we'll get a size error
	// If parsing happens first, we'd get a parse error
	testFile := filepath.Join(tempDir, "large-invalid.md")
	largeInvalidContent := make([]byte, 10485761) // Just over 10MB
	for i := range largeInvalidContent {
		largeInvalidContent[i] = 'x'
	}
	if err := os.WriteFile(testFile, largeInvalidContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	cmd := &cobra.Command{}
	args := []string{testFile}

	err := runRenumber(cmd, args)

	// We should get a file size error, NOT a parse error
	if err == nil {
		t.Fatal("Expected error but got none")
	}

	if !strings.Contains(err.Error(), "file exceeds 10MB limit") {
		t.Errorf("Expected file size error, got: %v", err)
	}

	// Verify we did NOT get a parse error (which would indicate wrong order)
	if strings.Contains(err.Error(), "parse") || strings.Contains(err.Error(), "invalid") {
		t.Errorf("Got parse error instead of size error, indicating wrong validation order: %v", err)
	}
}

// TestConvertPhaseMarkersToPositions tests the phase marker to position conversion
func TestConvertPhaseMarkersToPositions(t *testing.T) {
	tests := map[string]struct {
		inputMarkers      []task.PhaseMarker
		fileTaskIDOrder   []string
		expectedPositions []phasePosition
	}{
		"empty markers": {
			inputMarkers:      []task.PhaseMarker{},
			fileTaskIDOrder:   []string{"1", "2", "3"},
			expectedPositions: []phasePosition{},
		},
		"phase at beginning": {
			inputMarkers: []task.PhaseMarker{
				{Name: "Phase 1", AfterTaskID: ""},
			},
			fileTaskIDOrder: []string{"1", "2", "3"},
			expectedPositions: []phasePosition{
				{Name: "Phase 1", AfterPosition: -1},
			},
		},
		"phase after task in order": {
			inputMarkers: []task.PhaseMarker{
				{Name: "Phase 1", AfterTaskID: "2"},
			},
			fileTaskIDOrder: []string{"1", "2", "3"},
			expectedPositions: []phasePosition{
				{Name: "Phase 1", AfterPosition: 1}, // Task 2 is at position 1
			},
		},
		"phase after out of order task": {
			inputMarkers: []task.PhaseMarker{
				{Name: "Phase 1", AfterTaskID: "7"},
			},
			fileTaskIDOrder: []string{"1", "2", "6", "7", "3", "4", "5"},
			expectedPositions: []phasePosition{
				{Name: "Phase 1", AfterPosition: 3}, // Task 7 is at position 3 in file
			},
		},
		"multiple phases": {
			inputMarkers: []task.PhaseMarker{
				{Name: "Phase 1", AfterTaskID: "2"},
				{Name: "Phase 2", AfterTaskID: "7"},
			},
			fileTaskIDOrder: []string{"1", "2", "6", "7", "3", "4", "5"},
			expectedPositions: []phasePosition{
				{Name: "Phase 1", AfterPosition: 1}, // Task 2 at position 1
				{Name: "Phase 2", AfterPosition: 3}, // Task 7 at position 3
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := convertPhaseMarkersToPositions(tc.inputMarkers, tc.fileTaskIDOrder)

			if len(result) != len(tc.expectedPositions) {
				t.Fatalf("Expected %d positions, got %d", len(tc.expectedPositions), len(result))
			}

			for i, expected := range tc.expectedPositions {
				if result[i].Name != expected.Name {
					t.Errorf("Position %d: expected name '%s', got '%s'", i, expected.Name, result[i].Name)
				}
				if result[i].AfterPosition != expected.AfterPosition {
					t.Errorf("Position %d: expected AfterPosition %d, got %d", i, expected.AfterPosition, result[i].AfterPosition)
				}
			}
		})
	}
}

// Helper function to get all task IDs from a TaskList
func getTaskIDs(tl *task.TaskList) []string {
	var ids []string
	for _, t := range tl.Tasks {
		ids = append(ids, t.ID)
		ids = append(ids, getChildTaskIDs(t.Children)...)
	}
	return ids
}

func getChildTaskIDs(tasks []task.Task) []string {
	var ids []string
	for _, t := range tasks {
		ids = append(ids, t.ID)
		ids = append(ids, getChildTaskIDs(t.Children)...)
	}
	return ids
}

// TestGetRootTaskID tests the getRootTaskID function
func TestGetRootTaskID(t *testing.T) {
	tests := map[string]struct {
		taskID   string
		expected string
	}{
		"root task": {
			taskID:   "3",
			expected: "3",
		},
		"first level nested": {
			taskID:   "5.2",
			expected: "5",
		},
		"deeply nested": {
			taskID:   "7.2.1",
			expected: "7",
		},
		"very deeply nested": {
			taskID:   "15.4.3.2.1",
			expected: "15",
		},
		"single digit": {
			taskID:   "1",
			expected: "1",
		},
		"double digit root": {
			taskID:   "42",
			expected: "42",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := getRootTaskID(tc.taskID)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

// TestExtractTaskIDOrder tests the extractTaskIDOrder function
func TestExtractTaskIDOrder(t *testing.T) {
	tests := map[string]struct {
		content  string
		expected []string
	}{
		"sequential tasks": {
			content: `# Test
- [ ] 1. First
- [ ] 2. Second
- [ ] 3. Third`,
			expected: []string{"1", "2", "3"},
		},
		"out of order tasks": {
			content: `# Test
- [ ] 1. First
- [ ] 2. Second
- [ ] 6. Sixth
- [ ] 7. Seventh
- [ ] 3. Third`,
			expected: []string{"1", "2", "6", "7", "3"},
		},
		"tasks with subtasks": {
			content: `# Test
- [ ] 1. First
  - [ ] 1.1. Subtask
- [ ] 2. Second`,
			expected: []string{"1", "2"}, // Only root tasks
		},
		"mixed with phases": {
			content: `# Test
- [ ] 1. First
## Phase 1
- [ ] 2. Second`,
			expected: []string{"1", "2"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := extractTaskIDOrder(tc.content)

			if len(result) != len(tc.expected) {
				t.Fatalf("Expected %d IDs, got %d", len(tc.expected), len(result))
			}

			for i, expected := range tc.expected {
				if result[i] != expected {
					t.Errorf("Position %d: expected ID '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}

// TestRenumberSimpleFile tests renumbering a file with simple flat tasks
func TestRenumberSimpleFile(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-renumber-simple")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "simple.md")

	// Create a task list with gaps in numbering
	tl := task.NewTaskList("Simple Tasks")
	tl.AddTask("", "First task", "")
	tl.AddTask("", "Second task", "")
	tl.AddTask("", "Third task", "")

	// Manually set IDs with gaps to simulate manual editing
	tl.Tasks[0].ID = "1"
	tl.Tasks[1].ID = "5"
	tl.Tasks[2].ID = "10"

	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run renumber command
	cmd := &cobra.Command{}
	args := []string{testFile}
	if err := runRenumber(cmd, args); err != nil {
		t.Fatalf("Renumber failed: %v", err)
	}

	// Parse the file and check results
	result, _, err := task.ParseFileWithPhases(testFile)
	if err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Verify IDs are sequential
	expectedIDs := []string{"1", "2", "3"}
	if len(result.Tasks) != len(expectedIDs) {
		t.Fatalf("Expected %d tasks, got %d", len(expectedIDs), len(result.Tasks))
	}

	for i, expectedID := range expectedIDs {
		if result.Tasks[i].ID != expectedID {
			t.Errorf("Task %d: expected ID '%s', got '%s'", i, expectedID, result.Tasks[i].ID)
		}
	}

	// Verify backup was created
	backupPath := testFile + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Backup file was not created")
	}
}

// TestRenumberWithHierarchy tests renumbering a file with nested tasks
func TestRenumberWithHierarchy(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-renumber-hierarchy")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "hierarchy.md")

	// Create a task list with hierarchy
	tl := task.NewTaskList("Hierarchical Tasks")
	tl.AddTask("", "Parent 1", "")
	tl.AddTask("1", "Child 1.1", "")
	tl.AddTask("1", "Child 1.2", "")
	tl.AddTask("", "Parent 2", "")
	tl.AddTask("2", "Child 2.1", "")
	tl.AddTask("2.1", "Grandchild 2.1.1", "")

	// Manually set IDs with gaps
	tl.Tasks[0].ID = "5"
	tl.Tasks[0].Children[0].ID = "5.1"
	tl.Tasks[0].Children[1].ID = "5.3"
	tl.Tasks[1].ID = "10"
	tl.Tasks[1].Children[0].ID = "10.2"
	tl.Tasks[1].Children[0].Children[0].ID = "10.2.5"

	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run renumber command
	cmd := &cobra.Command{}
	args := []string{testFile}
	if err := runRenumber(cmd, args); err != nil {
		t.Fatalf("Renumber failed: %v", err)
	}

	// Parse the file and check results
	result, _, err := task.ParseFileWithPhases(testFile)
	if err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Verify hierarchical IDs are correct
	if result.Tasks[0].ID != "1" {
		t.Errorf("Expected root task ID '1', got '%s'", result.Tasks[0].ID)
	}
	if result.Tasks[0].Children[0].ID != "1.1" {
		t.Errorf("Expected child task ID '1.1', got '%s'", result.Tasks[0].Children[0].ID)
	}
	if result.Tasks[0].Children[1].ID != "1.2" {
		t.Errorf("Expected child task ID '1.2', got '%s'", result.Tasks[0].Children[1].ID)
	}
	if result.Tasks[1].ID != "2" {
		t.Errorf("Expected root task ID '2', got '%s'", result.Tasks[1].ID)
	}
	if result.Tasks[1].Children[0].ID != "2.1" {
		t.Errorf("Expected child task ID '2.1', got '%s'", result.Tasks[1].Children[0].ID)
	}
	if result.Tasks[1].Children[0].Children[0].ID != "2.1.1" {
		t.Errorf("Expected grandchild task ID '2.1.1', got '%s'", result.Tasks[1].Children[0].Children[0].ID)
	}
}

// TestRenumberWithPhases tests renumbering a file with phase markers
func TestRenumberWithPhases(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-renumber-phases")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "phases.md")

	// Create task content with phases (using sequential IDs)
	content := `# Project with Phases

## Phase 1

- [ ] 1. First task in phase 1
- [ ] 2. Second task in phase 1

## Phase 2

- [ ] 3. First task in phase 2
  - [ ] 3.1. Subtask in phase 2
- [ ] 4. Second task in phase 2
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Parse before renumber to see what we start with
	beforeResult, beforePhaseMarkers, err := task.ParseFileWithPhases(testFile)
	if err != nil {
		t.Fatalf("Failed to parse before renumber: %v", err)
	}
	t.Logf("Before renumber - phase markers: %+v", beforePhaseMarkers)
	t.Logf("Before renumber - task IDs: %v", getTaskIDs(beforeResult))

	// Run renumber command
	cmd := &cobra.Command{}
	args := []string{testFile}
	if err := runRenumber(cmd, args); err != nil {
		t.Fatalf("Renumber failed: %v", err)
	}

	// Parse the file and check results
	result, phaseMarkers, err := task.ParseFileWithPhases(testFile)
	if err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Verify task IDs are sequential
	if result.Tasks[0].ID != "1" {
		t.Errorf("Expected task ID '1', got '%s'", result.Tasks[0].ID)
	}
	if result.Tasks[1].ID != "2" {
		t.Errorf("Expected task ID '2', got '%s'", result.Tasks[1].ID)
	}
	if result.Tasks[2].ID != "3" {
		t.Errorf("Expected task ID '3', got '%s'", result.Tasks[2].ID)
	}
	if result.Tasks[2].Children[0].ID != "3.1" {
		t.Errorf("Expected task ID '3.1', got '%s'", result.Tasks[2].Children[0].ID)
	}
	if result.Tasks[3].ID != "4" {
		t.Errorf("Expected task ID '4', got '%s'", result.Tasks[3].ID)
	}

	// Verify phase markers are preserved
	if len(phaseMarkers) != 2 {
		t.Logf("Phase markers: %+v", phaseMarkers)
		t.Fatalf("Expected 2 phase markers, got %d", len(phaseMarkers))
	}

	// Phase markers should be adjusted to point to root tasks
	if phaseMarkers[0].Name != "Phase 1" {
		t.Errorf("Expected phase name 'Phase 1', got '%s'", phaseMarkers[0].Name)
	}
	if phaseMarkers[0].AfterTaskID != "" {
		t.Errorf("Expected empty AfterTaskID for first phase, got '%s'", phaseMarkers[0].AfterTaskID)
	}

	if phaseMarkers[1].Name != "Phase 2" {
		t.Errorf("Expected phase name 'Phase 2', got '%s'", phaseMarkers[1].Name)
	}
	if phaseMarkers[1].AfterTaskID != "2" {
		t.Errorf("Expected AfterTaskID '2', got '%s'", phaseMarkers[1].AfterTaskID)
	}
}

// TestRenumberWithFrontMatter tests renumbering a file with YAML front matter
func TestRenumberWithFrontMatter(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-renumber-frontmatter")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "frontmatter.md")

	// Create task content with YAML front matter
	content := `---
references:
  - requirements.md
  - design.md
metadata:
  title: Project Tasks
  created: 2024-01-01
---

# Project Tasks

- [ ] 1. First task
- [ ] 2. Second task
- [ ] 3. Third task
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run renumber command
	cmd := &cobra.Command{}
	args := []string{testFile}
	if err := runRenumber(cmd, args); err != nil {
		t.Fatalf("Renumber failed: %v", err)
	}

	// Read the result file and verify front matter is preserved
	resultContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	resultStr := string(resultContent)
	t.Logf("Result file content:\n%s", resultStr)

	// Check front matter is preserved
	if !strings.Contains(resultStr, "---") {
		t.Error("YAML front matter delimiters not found")
	}
	if !strings.Contains(resultStr, "references:") {
		t.Error("Front matter references not preserved")
	}
	if !strings.Contains(resultStr, "requirements.md") {
		t.Error("Front matter reference requirements.md not preserved")
	}
	if !strings.Contains(resultStr, "design.md") {
		t.Error("Front matter reference design.md not preserved")
	}
	if !strings.Contains(resultStr, "metadata:") {
		t.Error("Front matter metadata section not preserved")
	}
	if !strings.Contains(resultStr, "title: Project Tasks") {
		t.Error("Front matter metadata title not preserved")
	}
	if !strings.Contains(resultStr, "created:") || !strings.Contains(resultStr, "2024-01-01") {
		t.Error("Front matter metadata created date not preserved")
	}

	// Parse and verify tasks are renumbered
	result, _, err := task.ParseFileWithPhases(testFile)
	if err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if result.Tasks[0].ID != "1" {
		t.Errorf("Expected task ID '1', got '%s'", result.Tasks[0].ID)
	}
	if result.Tasks[1].ID != "2" {
		t.Errorf("Expected task ID '2', got '%s'", result.Tasks[1].ID)
	}
	if result.Tasks[2].ID != "3" {
		t.Errorf("Expected task ID '3', got '%s'", result.Tasks[2].ID)
	}
}

// TestRenumberPreservesTaskOrder tests that task order is maintained
func TestRenumberPreservesTaskOrder(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-renumber-order")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "order.md")

	// Create a task list with specific titles in order
	tl := task.NewTaskList("Task Order Test")
	tl.AddTask("", "Alpha", "")
	tl.AddTask("", "Beta", "")
	tl.AddTask("", "Gamma", "")
	tl.AddTask("", "Delta", "")

	// Set IDs out of order
	tl.Tasks[0].ID = "10"
	tl.Tasks[1].ID = "5"
	tl.Tasks[2].ID = "20"
	tl.Tasks[3].ID = "15"

	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run renumber command
	cmd := &cobra.Command{}
	args := []string{testFile}
	if err := runRenumber(cmd, args); err != nil {
		t.Fatalf("Renumber failed: %v", err)
	}

	// Parse the file and check task order is preserved
	result, _, err := task.ParseFileWithPhases(testFile)
	if err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	expectedTitles := []string{"Alpha", "Beta", "Gamma", "Delta"}
	for i, expectedTitle := range expectedTitles {
		if result.Tasks[i].Title != expectedTitle {
			t.Errorf("Task %d: expected title '%s', got '%s'", i, expectedTitle, result.Tasks[i].Title)
		}
	}
}

// TestRenumberPreservesMetadata tests that task metadata is preserved
func TestRenumberPreservesMetadata(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-renumber-metadata")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "metadata.md")

	// Create task content with various metadata
	content := `# Task Metadata Test

- [x] 1. Completed task with details
  - This task is completed
  - It has multiple details
  - References: doc1.md, doc2.md
- [-] 2. In-progress task
  - Currently working on this
  - References: spec.md
- [ ] 3. Pending task with requirements
  - Requirements: [1](requirements.md#1), [2](requirements.md#2)
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run renumber command
	cmd := &cobra.Command{}
	args := []string{testFile}
	if err := runRenumber(cmd, args); err != nil {
		t.Fatalf("Renumber failed: %v", err)
	}

	// Parse the file and check metadata is preserved
	result, _, err := task.ParseFileWithPhases(testFile)
	if err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Check task 1 (previously 5)
	if result.Tasks[0].Status != task.Completed {
		t.Errorf("Task 1: expected status Completed, got %v", result.Tasks[0].Status)
	}
	if len(result.Tasks[0].Details) != 2 {
		t.Errorf("Task 1: expected 2 details, got %d", len(result.Tasks[0].Details))
	}
	if len(result.Tasks[0].References) != 2 {
		t.Errorf("Task 1: expected 2 references, got %d", len(result.Tasks[0].References))
	}

	// Check task 2 (previously 10)
	if result.Tasks[1].Status != task.InProgress {
		t.Errorf("Task 2: expected status InProgress, got %v", result.Tasks[1].Status)
	}
	if len(result.Tasks[1].Details) != 1 {
		t.Errorf("Task 2: expected 1 detail, got %d", len(result.Tasks[1].Details))
	}
	if len(result.Tasks[1].References) != 1 {
		t.Errorf("Task 2: expected 1 reference, got %d", len(result.Tasks[1].References))
	}

	// Check task 3 (previously 15)
	if result.Tasks[2].Status != task.Pending {
		t.Errorf("Task 3: expected status Pending, got %v", result.Tasks[2].Status)
	}
	t.Logf("Task 3 requirements: %+v", result.Tasks[2].Requirements)
	if len(result.Tasks[2].Requirements) == 0 {
		t.Error("Task 3: expected requirements to be preserved")
	}
}

// TestRenumberPreservesStableIDs tests that stable IDs, blocked-by dependencies,
// streams, and owners survive renumbering
func TestRenumberPreservesStableIDs(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-renumber-stableids")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "stableids.md")

	// Create task content with stable IDs, dependencies, streams, and owners.
	// Use out-of-order IDs so renumbering actually changes them.
	content := `# Stable ID Test

- [ ] 5. Setup database <!-- id:abc1234 -->
  - Stream: 1
  - Owner: agent-a
- [ ] 10. Build API <!-- id:def5678 -->
  - Blocked-by: abc1234
  - Stream: 2
  - Owner: agent-b
- [ ] 15. Write tests <!-- id:ghi9012 -->
  - Blocked-by: abc1234, def5678
  - Stream: 1
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run renumber command
	cmd := &cobra.Command{}
	args := []string{testFile}
	if err := runRenumber(cmd, args); err != nil {
		t.Fatalf("Renumber failed: %v", err)
	}

	// Parse the renumbered file
	result, _, err := task.ParseFileWithPhases(testFile)
	if err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if len(result.Tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(result.Tasks))
	}

	// Verify hierarchical IDs were renumbered (5, 10, 15 -> 1, 2, 3)
	expectedIDs := []string{"1", "2", "3"}
	for i, want := range expectedIDs {
		if result.Tasks[i].ID != want {
			t.Errorf("Task %d: expected ID %q, got %q", i, want, result.Tasks[i].ID)
		}
	}

	// Verify stable IDs are preserved unchanged
	expectedStableIDs := []string{"abc1234", "def5678", "ghi9012"}
	for i, want := range expectedStableIDs {
		if result.Tasks[i].StableID != want {
			t.Errorf("Task %d: expected stable ID %q, got %q", i, want, result.Tasks[i].StableID)
		}
	}

	// Verify blocked-by dependencies are preserved
	if len(result.Tasks[0].BlockedBy) != 0 {
		t.Errorf("Task 1: expected 0 blocked-by refs, got %d", len(result.Tasks[0].BlockedBy))
	}
	if len(result.Tasks[1].BlockedBy) != 1 || result.Tasks[1].BlockedBy[0] != "abc1234" {
		t.Errorf("Task 2: expected blocked-by [abc1234], got %v", result.Tasks[1].BlockedBy)
	}
	if len(result.Tasks[2].BlockedBy) != 2 {
		t.Errorf("Task 3: expected 2 blocked-by refs, got %d", len(result.Tasks[2].BlockedBy))
	}

	// Verify streams are preserved
	if result.Tasks[0].Stream != 1 {
		t.Errorf("Task 1: expected stream 1, got %d", result.Tasks[0].Stream)
	}
	if result.Tasks[1].Stream != 2 {
		t.Errorf("Task 2: expected stream 2, got %d", result.Tasks[1].Stream)
	}
	if result.Tasks[2].Stream != 1 {
		t.Errorf("Task 3: expected stream 1, got %d", result.Tasks[2].Stream)
	}

	// Verify owners are preserved
	if result.Tasks[0].Owner != "agent-a" {
		t.Errorf("Task 1: expected owner %q, got %q", "agent-a", result.Tasks[0].Owner)
	}
	if result.Tasks[1].Owner != "agent-b" {
		t.Errorf("Task 2: expected owner %q, got %q", "agent-b", result.Tasks[1].Owner)
	}

	// Also verify the raw file contains the stable ID comments
	rawContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read result file: %v", err)
	}
	for _, stableID := range expectedStableIDs {
		marker := fmt.Sprintf("<!-- id:%s -->", stableID)
		if !strings.Contains(string(rawContent), marker) {
			t.Errorf("Raw file missing stable ID marker %q", marker)
		}
	}
}

// TestRenumberParseError tests that parse errors are reported correctly
func TestRenumberParseError(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-renumber-parse-error")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := map[string]struct {
		content       string
		errorContains string
	}{
		"invalid status marker": {
			content: `# Test Tasks

- [?] 1. Invalid status marker
- [ ] 2. Valid task
`,
			errorContains: "failed to parse task file",
		},
		"task with tabs instead of spaces": {
			content:       "# Test Tasks\n\n\t- [ ] 1. Task with tab\n- [ ] 2. Normal task\n",
			errorContains: "failed to parse task file",
		},
		"missing space in checkbox": {
			content: `# Test Tasks

-[] 1. Missing space
- [ ] 2. Valid task
`,
			errorContains: "failed to parse task file",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, "test-tasks.md")
			if err := os.WriteFile(testFile, []byte(tc.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}
			defer os.Remove(testFile)

			// Create a fake cobra command for testing
			cmd := &cobra.Command{}
			args := []string{testFile}

			// Call runRenumber
			err := runRenumber(cmd, args)

			// Expect an error
			if err == nil {
				t.Fatal("Expected parse error but got none")
			}

			if !strings.Contains(err.Error(), tc.errorContains) {
				t.Errorf("Expected error containing '%s', got: %v", tc.errorContains, err)
			}
		})
	}
}

// TestRenumberBackupFailure tests backup creation failure scenarios
func TestRenumberBackupFailure(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-renumber-backup-fail")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test-tasks.md")

	// Create a valid task file
	tl := task.NewTaskList("Test Tasks")
	tl.AddTask("", "Task 1", "")
	tl.AddTask("", "Task 2", "")
	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	defer os.Remove(testFile)

	// Create a directory with the .bak extension to prevent backup file creation
	backupPath := testFile + ".bak"
	if err := os.Mkdir(backupPath, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	defer os.RemoveAll(backupPath)

	// Create a fake cobra command for testing
	cmd := &cobra.Command{}
	args := []string{testFile}

	// Call runRenumber
	err := runRenumber(cmd, args)

	// Expect a backup error
	if err == nil {
		t.Fatal("Expected backup creation error but got none")
	}

	if !strings.Contains(err.Error(), "failed to create backup") {
		t.Errorf("Expected error containing 'failed to create backup', got: %v", err)
	}
}

// TestRenumberWriteFailure tests write failure and cleanup
func TestRenumberWriteFailure(t *testing.T) {
	t.Skip("Skipping write failure test - difficult to simulate in a cross-platform way without mocking")
	// Note: Write failures are implicitly tested by the atomic write mechanism in WriteFile
	// which is already tested in internal/task/operations_test.go
	// The backup-before-write behavior ensures original file safety even on write failures
}

// TestRenumberEdgeCases tests edge cases for the renumber command
func TestRenumberEdgeCases(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-renumber-edge-cases")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := map[string]struct {
		content         string
		expectError     bool
		errorContains   string
		expectedCount   int
		validateSuccess func(*testing.T, string)
	}{
		"empty file with no tasks": {
			content: `# Empty Task List
`,
			expectError:   false,
			expectedCount: 0,
			validateSuccess: func(t *testing.T, testFile string) {
				// Verify backup was created
				backupPath := testFile + ".bak"
				if _, err := os.Stat(backupPath); os.IsNotExist(err) {
					t.Error("Backup file was not created for empty file")
				}

				// Verify file still exists and is unchanged (no tasks to renumber)
				if _, err := os.Stat(testFile); os.IsNotExist(err) {
					t.Error("Original file was removed")
				}
			},
		},
		"file with only phase markers": {
			content: `# Project with Phases

## Phase 1

## Phase 2

## Phase 3
`,
			expectError:   false,
			expectedCount: 0,
			validateSuccess: func(t *testing.T, testFile string) {
				// Read result and verify phase markers are preserved
				result, phaseMarkers, err := task.ParseFileWithPhases(testFile)
				if err != nil {
					t.Fatalf("Failed to parse result: %v", err)
				}

				if len(result.Tasks) != 0 {
					t.Errorf("Expected 0 tasks, got %d", len(result.Tasks))
				}

				if len(phaseMarkers) != 3 {
					t.Errorf("Expected 3 phase markers, got %d", len(phaseMarkers))
				}
			},
		},
		"truly empty file": {
			content:       "",
			expectError:   false,
			expectedCount: 0,
			validateSuccess: func(t *testing.T, testFile string) {
				// Verify backup was created
				backupPath := testFile + ".bak"
				if _, err := os.Stat(backupPath); os.IsNotExist(err) {
					t.Error("Backup file was not created for empty file")
				}

				// Don't try to parse truly empty file - it's not valid markdown
				// The important thing is that renumbering succeeded without error
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, "test-tasks.md")
			if err := os.WriteFile(testFile, []byte(tc.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}
			defer os.Remove(testFile)
			defer os.Remove(testFile + ".bak")

			// Create a fake cobra command for testing
			cmd := &cobra.Command{}
			args := []string{testFile}

			// Call runRenumber
			err := runRenumber(cmd, args)

			if tc.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tc.errorContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				// Only verify task count if we have validation function or content is parseable
				if tc.validateSuccess != nil {
					tc.validateSuccess(t, testFile)
				} else if tc.content != "" {
					// Verify task count in result (skip for empty files)
					result, _, err := task.ParseFileWithPhases(testFile)
					if err != nil {
						t.Fatalf("Failed to parse result: %v", err)
					}

					actualCount := result.CountTotalTasks()
					if actualCount != tc.expectedCount {
						t.Errorf("Expected task count %d, got %d", tc.expectedCount, actualCount)
					}
				}
			}
		})
	}
}

// TestDisplaySummaryTable tests the displaySummary function with table format
func TestDisplaySummaryTable(t *testing.T) {
	tests := map[string]struct {
		taskCount       int
		backupPath      string
		expectedStrings []string
	}{
		"table format with 3 tasks": {
			taskCount:  3,
			backupPath: "/test/path/tasks.md.bak",
			expectedStrings: []string{
				"Renumbering Summary",
				"Total Tasks",
				"3",
				"Backup File",
				"/test/path/tasks.md.bak",
				"Status",
				"✓ Success",
			},
		},
		"table format with 0 tasks": {
			taskCount:  0,
			backupPath: "/test/path/empty.md.bak",
			expectedStrings: []string{
				"Renumbering Summary",
				"Total Tasks",
				"0",
				"Backup File",
				"/test/path/empty.md.bak",
				"Status",
				"✓ Success",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a task list
			tl := task.NewTaskList("Test Tasks")
			for range tc.taskCount {
				tl.AddTask("", "Test task", "")
			}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call displaySummary with table format
			err := displaySummary(tl, tc.backupPath, "table")

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Errorf("displaySummary with table format failed: %v", err)
			}

			// Read captured output
			var buf []byte
			buf = make([]byte, 4096)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			// Verify expected strings are present
			for _, expected := range tc.expectedStrings {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but it was not found. Output:\n%s", expected, output)
				}
			}
		})
	}
}

// TestDisplaySummaryMarkdown tests the displaySummary function with markdown format
func TestDisplaySummaryMarkdown(t *testing.T) {
	tests := map[string]struct {
		taskCount       int
		backupPath      string
		expectedStrings []string
	}{
		"markdown format with 3 tasks": {
			taskCount:  3,
			backupPath: "/test/path/tasks.md.bak",
			expectedStrings: []string{
				"# Renumbering Summary",
				"- **Total Tasks**: 3",
				"- **Backup File**: /test/path/tasks.md.bak",
				"- **Status**: ✓ Success",
			},
		},
		"markdown format with 0 tasks": {
			taskCount:  0,
			backupPath: "/test/path/empty.md.bak",
			expectedStrings: []string{
				"# Renumbering Summary",
				"- **Total Tasks**: 0",
				"- **Backup File**: /test/path/empty.md.bak",
				"- **Status**: ✓ Success",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a task list
			tl := task.NewTaskList("Test Tasks")
			for range tc.taskCount {
				tl.AddTask("", "Test task", "")
			}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call displaySummary with markdown format
			err := displaySummary(tl, tc.backupPath, "markdown")

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Errorf("displaySummary with markdown format failed: %v", err)
			}

			// Read captured output
			var buf []byte
			buf = make([]byte, 4096)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			// Verify expected strings are present
			for _, expected := range tc.expectedStrings {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but it was not found. Output:\n%s", expected, output)
				}
			}
		})
	}
}

// TestDisplaySummaryJSON tests the displaySummary function with JSON format
func TestDisplaySummaryJSON(t *testing.T) {
	tests := map[string]struct {
		taskCount            int
		backupPath           string
		expectedTaskCount    int
		expectedBackupFile   string
		expectedSuccessValue bool
	}{
		"json format with 3 tasks": {
			taskCount:            3,
			backupPath:           "/test/path/tasks.md.bak",
			expectedTaskCount:    3,
			expectedBackupFile:   "/test/path/tasks.md.bak",
			expectedSuccessValue: true,
		},
		"json format with 0 tasks": {
			taskCount:            0,
			backupPath:           "/test/path/empty.md.bak",
			expectedTaskCount:    0,
			expectedBackupFile:   "/test/path/empty.md.bak",
			expectedSuccessValue: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a task list
			tl := task.NewTaskList("Test Tasks")
			for range tc.taskCount {
				tl.AddTask("", "Test task", "")
			}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call displaySummary with JSON format
			err := displaySummary(tl, tc.backupPath, "json")

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Errorf("displaySummary with JSON format failed: %v", err)
			}

			// Read captured output
			var buf []byte
			buf = make([]byte, 4096)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			// Parse JSON output
			var result map[string]any
			if err := json.Unmarshal(buf[:n], &result); err != nil {
				t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
			}

			// Verify required fields are present
			if _, ok := result["task_count"]; !ok {
				t.Error("JSON output missing 'task_count' field")
			}
			if _, ok := result["backup_file"]; !ok {
				t.Error("JSON output missing 'backup_file' field")
			}
			if _, ok := result["success"]; !ok {
				t.Error("JSON output missing 'success' field")
			}

			// Verify field values
			if taskCount, ok := result["task_count"].(float64); !ok || int(taskCount) != tc.expectedTaskCount {
				t.Errorf("Expected task_count=%d, got %v", tc.expectedTaskCount, result["task_count"])
			}
			if backupFile, ok := result["backup_file"].(string); !ok || backupFile != tc.expectedBackupFile {
				t.Errorf("Expected backup_file=%s, got %v", tc.expectedBackupFile, result["backup_file"])
			}
			if success, ok := result["success"].(bool); !ok || success != tc.expectedSuccessValue {
				t.Errorf("Expected success=%v, got %v", tc.expectedSuccessValue, result["success"])
			}
		})
	}
}
