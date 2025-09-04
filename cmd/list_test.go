package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
)

func TestListCommand(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-list-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Create a test file with some tasks
	tl := task.NewTaskList("Test Project")
	tl.AddTask("", "First task", "")
	tl.AddTask("", "Second task", "")
	tl.AddTask("1", "Subtask 1.1", "")
	tl.AddTask("1", "Subtask 1.2", "")

	// Update some task statuses
	tl.UpdateStatus("1", task.InProgress)
	tl.UpdateStatus("1.1", task.Completed)

	// Add details and references to a task
	tl.UpdateTask("2", "", []string{"Detail 1", "Detail 2"}, []string{"ref1.md", "ref2.md"})

	testFile := "test-tasks.md"
	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	tests := map[string]struct {
		filename    string
		wantErr     bool
		expectTasks int
	}{
		"list existing file": {
			filename:    testFile,
			wantErr:     false,
			expectTasks: 4, // 2 root tasks + 2 subtasks
		},
		"list non-existent file": {
			filename: "non-existent.md",
			wantErr:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Test the core functionality (not CLI output)
			parsedList, err := task.ParseFile(tc.filename)

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

			// Count all tasks (recursive)
			taskCount := countTasks(parsedList.Tasks)
			if taskCount != tc.expectTasks {
				t.Errorf("expected %d tasks, got %d", tc.expectTasks, taskCount)
			}

			// Check title
			if parsedList.Title != "Test Project" {
				t.Errorf("expected title 'Test Project', got %q", parsedList.Title)
			}

			// Verify task statuses
			task1 := parsedList.FindTask("1")
			if task1 == nil || task1.Status != task.InProgress {
				t.Errorf("expected task 1 to be in progress")
			}

			task11 := parsedList.FindTask("1.1")
			if task11 == nil || task11.Status != task.Completed {
				t.Errorf("expected task 1.1 to be completed")
			}
		})
	}
}

// countTasks recursively counts all tasks
func countTasks(tasks []task.Task) int {
	count := len(tasks)
	for _, t := range tasks {
		count += countTasks(t.Children)
	}
	return count
}

func TestListCommandFormats(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-formats-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Create a simple test file
	tl := task.NewTaskList("Format Test")
	tl.AddTask("", "Task 1", "")
	tl.AddTask("", "Task 2", "")

	testFile := "format-test.md"
	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Test JSON format
	t.Run("JSON format", func(t *testing.T) {
		parsedList, err := task.ParseFile(testFile)
		if err != nil {
			t.Fatalf("failed to parse file: %v", err)
		}

		jsonOutput, err := task.RenderJSON(parsedList)
		if err != nil {
			t.Errorf("failed to render JSON: %v", err)
		}

		if !strings.Contains(string(jsonOutput), "Format Test") {
			t.Errorf("JSON output doesn't contain expected title")
		}

		if !strings.Contains(string(jsonOutput), "Task 1") {
			t.Errorf("JSON output doesn't contain expected task")
		}
	})

	// Test Markdown format
	t.Run("Markdown format", func(t *testing.T) {
		parsedList, err := task.ParseFile(testFile)
		if err != nil {
			t.Fatalf("failed to parse file: %v", err)
		}

		markdownOutput := task.RenderMarkdown(parsedList)

		if !strings.Contains(string(markdownOutput), "# Format Test") {
			t.Errorf("Markdown output doesn't contain expected title")
		}

		if !strings.Contains(string(markdownOutput), "- [ ] 1. Task 1") {
			t.Errorf("Markdown output doesn't contain expected task format")
		}
	})
}

func TestListCommandPathValidation(t *testing.T) {
	tests := map[string]struct {
		filename string
		wantErr  bool
		errMsg   string
	}{
		"valid file": {
			filename: "tasks.md",
			wantErr:  false,
		},
		"path traversal attempt": {
			filename: "../../../etc/passwd",
			wantErr:  true,
			errMsg:   "no such file",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp directory for test
			tempDir, err := os.MkdirTemp("", "rune-path-test")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Change to temp directory
			oldDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(oldDir)

			// Create valid file for non-error cases
			if !tc.wantErr {
				tl := task.NewTaskList("Test")
				tl.WriteFile(tc.filename)
			}

			_, err = task.ParseFile(tc.filename)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q but got none", tc.errMsg)
				} else if !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("expected error containing %q, got %q", tc.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
