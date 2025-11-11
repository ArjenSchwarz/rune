package cmd

import (
	"fmt"
	"os"
	"path/filepath"
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
				if tc.errorContains != "" && !contains(err.Error(), tc.errorContains) {
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

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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
				if tc.errorContains != "" && !contains(err.Error(), tc.errorContains) {
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

	if !contains(err.Error(), "file exceeds 10MB limit") {
		t.Errorf("Expected file size error, got: %v", err)
	}

	// Verify we did NOT get a parse error (which would indicate wrong order)
	if contains(err.Error(), "parse") || contains(err.Error(), "invalid") {
		t.Errorf("Got parse error instead of size error, indicating wrong validation order: %v", err)
	}
}

// TestAdjustPhaseMarkersAfterRenumber tests the adjustPhaseMarkersAfterRenumber function
func TestAdjustPhaseMarkersAfterRenumber(t *testing.T) {
	tests := map[string]struct {
		inputMarkers    []task.PhaseMarker
		expectedMarkers []task.PhaseMarker
	}{
		"no phase markers (empty array)": {
			inputMarkers:    []task.PhaseMarker{},
			expectedMarkers: []task.PhaseMarker{},
		},
		"phase at beginning (empty AfterTaskID)": {
			inputMarkers: []task.PhaseMarker{
				{Name: "Phase 1", AfterTaskID: ""},
			},
			expectedMarkers: []task.PhaseMarker{
				{Name: "Phase 1", AfterTaskID: ""},
			},
		},
		"phase after root task": {
			inputMarkers: []task.PhaseMarker{
				{Name: "Phase 1", AfterTaskID: "3"},
			},
			expectedMarkers: []task.PhaseMarker{
				{Name: "Phase 1", AfterTaskID: "3"},
			},
		},
		"phase after nested task": {
			inputMarkers: []task.PhaseMarker{
				{Name: "Phase 1", AfterTaskID: "2.3"},
			},
			expectedMarkers: []task.PhaseMarker{
				{Name: "Phase 1", AfterTaskID: "2"},
			},
		},
		"multiple phases with various depths": {
			inputMarkers: []task.PhaseMarker{
				{Name: "Phase 1", AfterTaskID: ""},
				{Name: "Phase 2", AfterTaskID: "5"},
				{Name: "Phase 3", AfterTaskID: "7.2.1"},
				{Name: "Phase 4", AfterTaskID: "10.3"},
			},
			expectedMarkers: []task.PhaseMarker{
				{Name: "Phase 1", AfterTaskID: ""},
				{Name: "Phase 2", AfterTaskID: "5"},
				{Name: "Phase 3", AfterTaskID: "7"},
				{Name: "Phase 4", AfterTaskID: "10"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := adjustPhaseMarkersAfterRenumber(tc.inputMarkers)

			if len(result) != len(tc.expectedMarkers) {
				t.Fatalf("Expected %d markers, got %d", len(tc.expectedMarkers), len(result))
			}

			for i, expected := range tc.expectedMarkers {
				if result[i].Name != expected.Name {
					t.Errorf("Marker %d: expected name '%s', got '%s'", i, expected.Name, result[i].Name)
				}
				if result[i].AfterTaskID != expected.AfterTaskID {
					t.Errorf("Marker %d: expected AfterTaskID '%s', got '%s'", i, expected.AfterTaskID, result[i].AfterTaskID)
				}
			}
		})
	}
}

// TestGetRootTaskNumber tests the getRootTaskNumber function
func TestGetRootTaskNumber(t *testing.T) {
	tests := map[string]struct {
		taskID   string
		expected int
	}{
		"root task": {
			taskID:   "3",
			expected: 3,
		},
		"first level nested": {
			taskID:   "5.2",
			expected: 5,
		},
		"deeply nested": {
			taskID:   "7.2.1",
			expected: 7,
		},
		"very deeply nested": {
			taskID:   "15.4.3.2.1",
			expected: 15,
		},
		"single digit": {
			taskID:   "1",
			expected: 1,
		},
		"double digit root": {
			taskID:   "42",
			expected: 42,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := getRootTaskNumber(tc.taskID)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}
