package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test file size limits (10MB maximum)
func TestFileSizeLimits(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "normal file size",
			size:    1024,
			wantErr: false,
		},
		{
			name:    "large but acceptable file",
			size:    5 * 1024 * 1024, // 5MB
			wantErr: false,
		},
		{
			name:    "max file size",
			size:    MaxFileSize,
			wantErr: false,
		},
		{
			name:    "exceeds max file size",
			size:    MaxFileSize + 1,
			wantErr: true,
			errMsg:  "file exceeds maximum size",
		},
		{
			name:    "very large file",
			size:    50 * 1024 * 1024, // 50MB
			wantErr: true,
			errMsg:  "file exceeds maximum size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create valid markdown content of specified size
			var content []byte
			if tt.size <= 100 {
				// For small sizes, create minimal valid content
				content = []byte("# Test\n\n- [ ] 1. Test task\n")
				// Pad with comments to reach desired size
				for len(content) < tt.size {
					padding := "<!-- padding -->\n"
					if len(content)+len(padding) <= tt.size {
						content = append(content, padding...)
					} else {
						break
					}
				}
			} else {
				// For larger sizes, create repeated valid tasks
				content = []byte("# Test Tasks\n\n")
				taskTemplate := "- [ ] %d. Test task with some content\n  - Detail line\n  - References: test.md\n\n"

				taskNum := 1
				for len(content) < tt.size {
					taskContent := fmt.Sprintf(taskTemplate, taskNum)
					if len(content)+len(taskContent) <= tt.size {
						content = append(content, taskContent...)
						taskNum++
					} else {
						// Pad with spaces to reach exact size for overflow test
						remaining := tt.size - len(content)
						if remaining > 0 {
							content = append(content, make([]byte, remaining)...)
							// Fill padding with spaces to make it valid
							for i := len(content) - remaining; i < len(content); i++ {
								content[i] = ' '
							}
						}
						break
					}
				}
			}

			_, err := ParseMarkdown(content)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for size %d, got nil", tt.size)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for size %d: %v", tt.size, err)
				}
			}
		})
	}
}

// Test path traversal protection
func TestPathTraversalProtection(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir, err := os.MkdirTemp("", "go-tasks-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory for testing
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Create a sensitive file outside the working directory
	parentDir := filepath.Dir(tmpDir)
	sensitiveFile := filepath.Join(parentDir, "sensitive.txt")
	err = os.WriteFile(sensitiveFile, []byte("sensitive data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create sensitive file: %v", err)
	}
	defer os.Remove(sensitiveFile)

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "normal relative path",
			path:    "tasks.md",
			wantErr: false,
		},
		{
			name:    "normal subdirectory path",
			path:    "subdir/tasks.md",
			wantErr: false,
		},
		{
			name:    "path with dot",
			path:    "./tasks.md",
			wantErr: false,
		},
		{
			name:    "simple parent directory attempt",
			path:    "../tasks.md",
			wantErr: true,
			errMsg:  "path traversal attempt detected",
		},
		{
			name:    "complex parent directory attempt",
			path:    "../../etc/passwd",
			wantErr: true,
			errMsg:  "path traversal attempt detected",
		},
		{
			name:    "absolute path outside working dir",
			path:    "/etc/passwd",
			wantErr: true,
			errMsg:  "path traversal attempt detected",
		},
		{
			name:    "nested parent directory attempt",
			path:    "subdir/../../../etc/passwd",
			wantErr: true,
			errMsg:  "path traversal attempt detected",
		},
		{
			name:    "attempt to access parent temp directory",
			path:    "../sensitive.txt",
			wantErr: true,
			errMsg:  "path traversal attempt detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilePath(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for path %s, got nil", tt.path)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for path %s: %v", tt.path, err)
				}
			}
		})
	}
}

// Test atomic write operations
func TestAtomicWriteOperations(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "go-tasks-atomic-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	t.Run("successful atomic write", func(t *testing.T) {
		tl := NewTaskList("Test Tasks")
		_, err := tl.AddTask("", "Test task", "")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		testFile := "test.md"

		// Write should succeed
		err = tl.WriteFile(testFile)
		if err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		// File should exist
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Error("file was not created")
		}

		// Temp file should not exist
		tempFile := testFile + ".tmp"
		if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
			t.Error("temporary file was not cleaned up")
		}

		// Content should be correct
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if !strings.Contains(string(content), "Test Tasks") {
			t.Error("file content is incorrect")
		}
		if !strings.Contains(string(content), "Test task") {
			t.Error("file content is missing task")
		}
	})

	t.Run("atomic write with existing file", func(t *testing.T) {
		testFile := "existing.md"

		// Create existing file with content
		originalContent := "# Original Content\n\n- [ ] 1. Original task"
		err := os.WriteFile(testFile, []byte(originalContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create existing file: %v", err)
		}

		// Create new task list
		tl := NewTaskList("New Tasks")
		_, err = tl.AddTask("", "New task", "")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		// Write should replace existing file atomically
		err = tl.WriteFile(testFile)
		if err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		// Read the file content
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		// Should contain new content, not original
		if strings.Contains(string(content), "Original Content") {
			t.Error("file still contains original content")
		}
		if !strings.Contains(string(content), "New Tasks") {
			t.Error("file doesn't contain new content")
		}
		if !strings.Contains(string(content), "New task") {
			t.Error("file doesn't contain new task")
		}
	})

	t.Run("atomic write failure cleanup", func(t *testing.T) {
		// Create a directory with the same name as our target file
		// This will cause the rename to fail
		testFile := "will-fail.md"
		err := os.Mkdir(testFile, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		defer os.RemoveAll(testFile)

		tl := NewTaskList("Test")
		_, err = tl.AddTask("", "Test task", "")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		// Write should fail
		err = tl.WriteFile(testFile)
		if err == nil {
			t.Fatal("WriteFile should have failed")
		}

		// Temp file should be cleaned up
		tempFile := testFile + ".tmp"
		if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
			t.Error("temporary file was not cleaned up after failure")
		}
	})
}

// Test input sanitization
func TestInputSanitization(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "normal title",
			title:   "Normal task title",
			wantErr: false,
		},
		{
			name:    "empty title",
			title:   "",
			wantErr: true,
			errMsg:  "task title cannot be empty",
		},
		{
			name:    "title with special characters",
			title:   "Task with @#$%^&*() characters",
			wantErr: false,
		},
		{
			name:    "title with unicode characters",
			title:   "Task with émojis 🚀 and ñ characters",
			wantErr: false,
		},
		{
			name:    "very long title (500 chars)",
			title:   strings.Repeat("a", 500),
			wantErr: false,
		},
		{
			name:    "too long title (501 chars)",
			title:   strings.Repeat("a", 501),
			wantErr: true,
			errMsg:  "task title exceeds 500 characters",
		},
		{
			name:    "extremely long title",
			title:   strings.Repeat("a", 10000),
			wantErr: true,
			errMsg:  "task title exceeds 500 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{
				ID:     "1",
				Title:  tt.title,
				Status: Pending,
			}

			err := task.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for title '%s', got nil", tt.title)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for title '%s': %v", tt.title, err)
				}
			}
		})
	}
}

// Test task ID validation
func TestTaskIDValidation(t *testing.T) {
	tests := []struct {
		name    string
		taskID  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid root ID",
			taskID:  "1",
			wantErr: false,
		},
		{
			name:    "valid child ID",
			taskID:  "1.1",
			wantErr: false,
		},
		{
			name:    "valid deep nested ID",
			taskID:  "1.2.3.4.5",
			wantErr: false,
		},
		{
			name:    "empty ID",
			taskID:  "",
			wantErr: true,
			errMsg:  "invalid task ID format",
		},
		{
			name:    "invalid starting with dot",
			taskID:  ".1",
			wantErr: true,
			errMsg:  "invalid task ID format",
		},
		{
			name:    "invalid ending with dot",
			taskID:  "1.",
			wantErr: true,
			errMsg:  "invalid task ID format",
		},
		{
			name:    "invalid double dot",
			taskID:  "1..2",
			wantErr: true,
			errMsg:  "invalid task ID format",
		},
		{
			name:    "invalid with letters",
			taskID:  "1.a",
			wantErr: true,
			errMsg:  "invalid task ID format",
		},
		{
			name:    "invalid with zero",
			taskID:  "0",
			wantErr: true,
			errMsg:  "invalid task ID format",
		},
		{
			name:    "invalid with leading zero",
			taskID:  "01",
			wantErr: true,
			errMsg:  "invalid task ID format",
		},
		{
			name:    "invalid with special characters",
			taskID:  "1.2@3",
			wantErr: true,
			errMsg:  "invalid task ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{
				ID:     tt.taskID,
				Title:  "Test task",
				Status: Pending,
			}

			err := task.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for ID '%s', got nil", tt.taskID)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for ID '%s': %v", tt.taskID, err)
				}
			}
		})
	}
}

// Test concurrent access safety (within single process)
func TestConcurrentAccessSafety(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "go-tasks-concurrent-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	t.Run("concurrent writes to different files", func(t *testing.T) {
		// This tests that our atomic write operations don't interfere
		// with each other when writing to different files
		numGoroutines := 10
		done := make(chan error, numGoroutines)

		for i := range numGoroutines {
			go func(id int) {
				tl := NewTaskList("Test Tasks")
				_, err := tl.AddTask("", "Task from goroutine", "")
				if err != nil {
					done <- err
					return
				}

				fileName := filepath.Join("concurrent", "tasks-"+string(rune(id+'0'))+".md")

				// Ensure directory exists
				os.MkdirAll(filepath.Dir(fileName), 0755)

				err = tl.WriteFile(fileName)
				done <- err
			}(i)
		}

		// Wait for all goroutines to complete
		for i := range numGoroutines {
			if err := <-done; err != nil {
				t.Errorf("Goroutine %d failed: %v", i, err)
			}
		}

		// Verify all files were created
		for i := range numGoroutines {
			fileName := filepath.Join("concurrent", "tasks-"+string(rune(i+'0'))+".md")
			if _, err := os.Stat(fileName); os.IsNotExist(err) {
				t.Errorf("File %s was not created", fileName)
			}
		}
	})

	t.Run("atomic write prevents partial reads", func(t *testing.T) {
		// This test ensures that atomic writes prevent other processes
		// from reading partially written files
		testFile := "atomic-test.md"

		// Create a large task list to increase write time
		tl := NewTaskList("Large Task List")
		for i := 1; i <= 100; i++ {
			tl.AddTask("", "Task with some long content that will take time to write", "")
			if i%10 == 0 {
				// Add subtasks to some tasks
				parentID := string(rune(i/10 + '0'))
				tl.AddTask(parentID, "Subtask with additional content", "")
			}
		}

		// Write the file
		err := tl.WriteFile(testFile)
		if err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		// Verify the file is complete and valid
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		// Parse the content to ensure it's valid
		parsedTL, err := ParseMarkdown(content)
		if err != nil {
			t.Fatalf("Failed to parse written file: %v", err)
		}

		if len(parsedTL.Tasks) != 100 {
			t.Errorf("Expected 100 tasks, got %d", len(parsedTL.Tasks))
		}
	})
}
