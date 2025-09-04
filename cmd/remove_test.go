package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

func TestRunRemove(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-remove")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := map[string]struct {
		setupFile     func(string) error
		taskID        string
		expectError   bool
		errorContains string
		validateFile  func(*testing.T, string)
	}{
		"remove single top-level task": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task 1", "")
				tl.AddTask("", "Task 2", "")
				tl.AddTask("", "Task 3", "")
				return tl.WriteFile(filename)
			},
			taskID: "2",
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				// Should have 2 tasks remaining
				if len(tl.Tasks) != 2 {
					t.Fatalf("Expected 2 tasks remaining, got %d", len(tl.Tasks))
				}
				// Check renumbering
				if tl.Tasks[0].ID != "1" || tl.Tasks[0].Title != "Task 1" {
					t.Fatalf("Expected first task to be '1. Task 1', got '%s. %s'", tl.Tasks[0].ID, tl.Tasks[0].Title)
				}
				if tl.Tasks[1].ID != "2" || tl.Tasks[1].Title != "Task 3" {
					t.Fatalf("Expected second task to be '2. Task 3', got '%s. %s'", tl.Tasks[1].ID, tl.Tasks[1].Title)
				}
			},
		},
		"remove task with children": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Parent task", "")
				tl.AddTask("1", "Child 1", "")
				tl.AddTask("1", "Child 2", "")
				tl.AddTask("1.1", "Grandchild", "")
				tl.AddTask("", "Task 2", "")
				return tl.WriteFile(filename)
			},
			taskID: "1",
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				// Should have 1 task remaining (Task 2 becomes task 1)
				if len(tl.Tasks) != 1 {
					t.Fatalf("Expected 1 task remaining, got %d", len(tl.Tasks))
				}
				if tl.Tasks[0].ID != "1" || tl.Tasks[0].Title != "Task 2" {
					t.Fatalf("Expected remaining task to be '1. Task 2', got '%s. %s'", tl.Tasks[0].ID, tl.Tasks[0].Title)
				}
			},
		},
		"remove subtask": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Parent task", "")
				tl.AddTask("1", "Child 1", "")
				tl.AddTask("1", "Child 2", "")
				tl.AddTask("1", "Child 3", "")
				return tl.WriteFile(filename)
			},
			taskID: "1.2",
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				// Parent should still exist with 2 children
				if len(tl.Tasks) != 1 {
					t.Fatalf("Expected 1 parent task, got %d", len(tl.Tasks))
				}
				if len(tl.Tasks[0].Children) != 2 {
					t.Fatalf("Expected 2 children, got %d", len(tl.Tasks[0].Children))
				}
				// Check renumbering of children
				if tl.Tasks[0].Children[0].ID != "1.1" || tl.Tasks[0].Children[0].Title != "Child 1" {
					t.Fatalf("Expected first child to be '1.1. Child 1', got '%s. %s'", tl.Tasks[0].Children[0].ID, tl.Tasks[0].Children[0].Title)
				}
				if tl.Tasks[0].Children[1].ID != "1.2" || tl.Tasks[0].Children[1].Title != "Child 3" {
					t.Fatalf("Expected second child to be '1.2. Child 3', got '%s. %s'", tl.Tasks[0].Children[1].ID, tl.Tasks[0].Children[1].Title)
				}
			},
		},
		"remove nested subtask with children": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Parent", "")
				tl.AddTask("1", "Child 1", "")
				tl.AddTask("1", "Child 2", "")
				tl.AddTask("1.1", "Grandchild 1", "")
				tl.AddTask("1.1", "Grandchild 2", "")
				return tl.WriteFile(filename)
			},
			taskID: "1.1",
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				// Parent should exist with 1 child (Child 2 becomes 1.1)
				if len(tl.Tasks[0].Children) != 1 {
					t.Fatalf("Expected 1 child, got %d", len(tl.Tasks[0].Children))
				}
				if tl.Tasks[0].Children[0].ID != "1.1" || tl.Tasks[0].Children[0].Title != "Child 2" {
					t.Fatalf("Expected remaining child to be '1.1. Child 2', got '%s. %s'", tl.Tasks[0].Children[0].ID, tl.Tasks[0].Children[0].Title)
				}
				// Should have no grandchildren
				if len(tl.Tasks[0].Children[0].Children) != 0 {
					t.Fatalf("Expected no grandchildren, got %d", len(tl.Tasks[0].Children[0].Children))
				}
			},
		},
		"remove last task": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Only task", "")
				return tl.WriteFile(filename)
			},
			taskID: "1",
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				// Should have no tasks
				if len(tl.Tasks) != 0 {
					t.Fatalf("Expected no tasks remaining, got %d", len(tl.Tasks))
				}
			},
		},
		"non-existent task ID": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task 1", "")
				return tl.WriteFile(filename)
			},
			taskID:        "999",
			expectError:   true,
			errorContains: "task 999 not found",
		},
		"file does not exist": {
			setupFile: func(filename string) error {
				// Don't create the file
				return nil
			},
			taskID:        "1",
			expectError:   true,
			errorContains: "does not exist",
		},
		"remove from multiple level hierarchy": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task 1", "")
				tl.AddTask("", "Task 2", "")
				tl.AddTask("", "Task 3", "")
				tl.AddTask("2", "Child of 2", "")
				tl.AddTask("3", "Child of 3", "")
				return tl.WriteFile(filename)
			},
			taskID: "2",
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				// Should have 2 tasks (1 and 3 becomes 2)
				if len(tl.Tasks) != 2 {
					t.Fatalf("Expected 2 tasks, got %d", len(tl.Tasks))
				}
				if tl.Tasks[0].Title != "Task 1" {
					t.Fatalf("Expected first task to be 'Task 1', got '%s'", tl.Tasks[0].Title)
				}
				if tl.Tasks[1].Title != "Task 3" {
					t.Fatalf("Expected second task to be 'Task 3', got '%s'", tl.Tasks[1].Title)
				}
				// Task 3 should have its child with updated ID
				if len(tl.Tasks[1].Children) != 1 {
					t.Fatalf("Expected task 3 to have 1 child, got %d", len(tl.Tasks[1].Children))
				}
				if tl.Tasks[1].Children[0].ID != "2.1" {
					t.Fatalf("Expected child ID to be '2.1', got '%s'", tl.Tasks[1].Children[0].ID)
				}
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			filename := filepath.Join(tempDir, "test-"+strings.ReplaceAll(name, " ", "-")+".md")

			if err := tt.setupFile(filename); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			dryRun = false

			cmd := &cobra.Command{}
			args := []string{filename, tt.taskID}

			err := runRemove(cmd, args)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validateFile != nil {
				tt.validateFile(t, filename)
			}
		})
	}
}

func TestRunRemoveDryRun(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-remove-dry")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filename := filepath.Join(tempDir, "test.md")

	// Create test file with task that has children
	tl := task.NewTaskList("Test Tasks")
	tl.AddTask("", "Parent task", "")
	tl.AddTask("1", "Child 1", "")
	tl.AddTask("1", "Child 2", "")
	tl.AddTask("1.1", "Grandchild", "")
	tl.UpdateStatus("1.1.1", task.InProgress)
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Read initial content
	initialContent, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read initial file: %v", err)
	}

	// Set up dry run
	dryRun = true

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run command
	cmd := &cobra.Command{}
	args := []string{filename, "1"}
	err = runRemove(cmd, args)

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Unexpected error in dry run: %v", err)
	}

	// Check output contains expected information
	expectedPhrases := []string{
		"Would remove task from file",
		"Task ID: 1",
		"Title: Parent task",
		"This task has 3 subtask(s) that will also be removed",
		"Subtasks to be removed:",
		"- [ ] 1.1. Child 1",
		"- [-] 1.1.1. Grandchild",
		"- [ ] 1.2. Child 2",
		"All remaining tasks will be renumbered",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(output, phrase) {
			t.Fatalf("Expected dry run output to contain '%s', got: %s", phrase, output)
		}
	}

	// Verify file wasn't modified
	finalContent, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read final file: %v", err)
	}

	if !bytes.Equal(initialContent, finalContent) {
		t.Fatal("File was modified during dry run")
	}

	// Reset dry run
	dryRun = false
}

func TestRunRemoveDryRunSingleTask(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-remove-dry-single")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filename := filepath.Join(tempDir, "test.md")

	// Create test file with single task
	tl := task.NewTaskList("Test Tasks")
	tl.AddTask("", "Single task", "")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	dryRun = true

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run command
	cmd := &cobra.Command{}
	args := []string{filename, "1"}
	err := runRemove(cmd, args)

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Unexpected error in dry run: %v", err)
	}

	// Should not mention subtasks for a task with no children
	if strings.Contains(output, "subtask") {
		t.Fatalf("Expected dry run output to not mention subtasks for single task, got: %s", output)
	}
	if !strings.Contains(output, "Title: Single task") {
		t.Fatalf("Expected dry run output to show task title, got: %s", output)
	}

	dryRun = false
}

func TestCountTaskChildren(t *testing.T) {
	// Create a task with nested children
	parent := task.Task{
		ID:    "1",
		Title: "Parent",
		Children: []task.Task{
			{
				ID:    "1.1",
				Title: "Child 1",
				Children: []task.Task{
					{ID: "1.1.1", Title: "Grandchild 1"},
					{ID: "1.1.2", Title: "Grandchild 2"},
				},
			},
			{
				ID:    "1.2",
				Title: "Child 2",
			},
		},
	}

	count := countTaskChildren(&parent)
	// Should be 4: Child 1, Child 2, Grandchild 1, Grandchild 2
	if count != 4 {
		t.Fatalf("Expected 4 children, got %d", count)
	}
}

func TestStatusToCheckbox(t *testing.T) {
	tests := []struct {
		status   task.Status
		expected string
	}{
		{task.Pending, "[ ]"},
		{task.InProgress, "[-]"},
		{task.Completed, "[x]"},
	}

	for _, tt := range tests {
		result := statusToCheckbox(tt.status)
		if result != tt.expected {
			t.Fatalf("statusToCheckbox(%v) = %s, expected %s", tt.status, result, tt.expected)
		}
	}
}
