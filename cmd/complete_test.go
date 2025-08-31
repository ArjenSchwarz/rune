package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/go-tasks/internal/task"
	"github.com/spf13/cobra"
)

func TestRunComplete(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-complete")
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
		"mark pending task as completed": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task to complete")
				return tl.WriteFile(filename)
			},
			taskID:      "1",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if len(tl.Tasks) != 1 {
					t.Fatalf("Expected 1 task, got %d", len(tl.Tasks))
				}
				if tl.Tasks[0].Status != task.Completed {
					t.Fatalf("Expected task to be completed, got status %v", tl.Tasks[0].Status)
				}
			},
		},
		"mark in-progress task as completed": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task in progress")
				tl.UpdateStatus("1", task.InProgress)
				return tl.WriteFile(filename)
			},
			taskID:      "1",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if tl.Tasks[0].Status != task.Completed {
					t.Fatalf("Expected task to be completed, got status %v", tl.Tasks[0].Status)
				}
			},
		},
		"mark already completed task as completed": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Already completed task")
				tl.UpdateStatus("1", task.Completed)
				return tl.WriteFile(filename)
			},
			taskID:      "1",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if tl.Tasks[0].Status != task.Completed {
					t.Fatalf("Expected task to remain completed, got status %v", tl.Tasks[0].Status)
				}
			},
		},
		"complete subtask": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Parent task")
				tl.AddTask("1", "Child task")
				return tl.WriteFile(filename)
			},
			taskID:      "1.1",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				child := &tl.Tasks[0].Children[0]
				if child.Status != task.Completed {
					t.Fatalf("Expected child task to be completed, got status %v", child.Status)
				}
				// Parent should be auto-completed since all children are complete
				if tl.Tasks[0].Status != task.Completed {
					t.Fatalf("Expected parent task to be auto-completed, got status %v", tl.Tasks[0].Status)
				}
			},
		},
		"complete last subtask auto-completes parent": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Parent task")
				tl.AddTask("1", "Child task 1")
				tl.AddTask("1", "Child task 2")
				// Complete first child
				tl.UpdateStatus("1.1", task.Completed)
				return tl.WriteFile(filename)
			},
			taskID:      "1.2",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				// Both children should be completed
				if tl.Tasks[0].Children[0].Status != task.Completed {
					t.Fatalf("Expected first child to be completed, got status %v", tl.Tasks[0].Children[0].Status)
				}
				if tl.Tasks[0].Children[1].Status != task.Completed {
					t.Fatalf("Expected second child to be completed, got status %v", tl.Tasks[0].Children[1].Status)
				}
				// Parent should be auto-completed
				if tl.Tasks[0].Status != task.Completed {
					t.Fatalf("Expected parent task to be auto-completed, got status %v", tl.Tasks[0].Status)
				}
			},
		},
		"multi-level auto-completion": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Grandparent task")
				tl.AddTask("1", "Parent task")
				tl.AddTask("1.1", "Child task 1")
				tl.AddTask("1.1", "Child task 2")
				// Complete first child
				tl.UpdateStatus("1.1.1", task.Completed)
				return tl.WriteFile(filename)
			},
			taskID:      "1.1.2",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				// All tasks in the chain should be completed
				grandchild := &tl.Tasks[0].Children[0].Children[1]
				if grandchild.Status != task.Completed {
					t.Fatalf("Expected grandchild to be completed, got status %v", grandchild.Status)
				}
				parent := &tl.Tasks[0].Children[0]
				if parent.Status != task.Completed {
					t.Fatalf("Expected parent to be auto-completed, got status %v", parent.Status)
				}
				grandparent := &tl.Tasks[0]
				if grandparent.Status != task.Completed {
					t.Fatalf("Expected grandparent to be auto-completed, got status %v", grandparent.Status)
				}
			},
		},
		"non-existent task ID": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
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
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			filename := filepath.Join(tempDir, "test-"+strings.ReplaceAll(name, " ", "-")+".md")

			if err := tt.setupFile(filename); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Set dry run to false for actual execution
			dryRun = false

			cmd := &cobra.Command{}
			args := []string{filename, tt.taskID}

			err := runComplete(cmd, args)

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

func TestRunCompleteDryRun(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-complete-dry")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filename := filepath.Join(tempDir, "test.md")

	// Create test file
	tl := task.NewTaskList("Test Tasks")
	tl.AddTask("", "Task to complete")
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
	err = runComplete(cmd, args)

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
	if !strings.Contains(output, "Would mark task as completed") {
		t.Fatalf("Expected dry run output to contain completion info, got: %s", output)
	}
	if !strings.Contains(output, "Task ID: 1") {
		t.Fatalf("Expected dry run output to show task ID, got: %s", output)
	}
	if !strings.Contains(output, "Current status: pending") {
		t.Fatalf("Expected dry run output to show current status, got: %s", output)
	}
	if !strings.Contains(output, "New status: completed") {
		t.Fatalf("Expected dry run output to show new status, got: %s", output)
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

func TestRunUncomplete(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-uncomplete")
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
		"mark completed task as pending": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Completed task")
				tl.UpdateStatus("1", task.Completed)
				return tl.WriteFile(filename)
			},
			taskID:      "1",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if tl.Tasks[0].Status != task.Pending {
					t.Fatalf("Expected task to be pending, got status %v", tl.Tasks[0].Status)
				}
			},
		},
		"mark in-progress task as pending": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "In progress task")
				tl.UpdateStatus("1", task.InProgress)
				return tl.WriteFile(filename)
			},
			taskID:      "1",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if tl.Tasks[0].Status != task.Pending {
					t.Fatalf("Expected task to be pending, got status %v", tl.Tasks[0].Status)
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

			err := runUncomplete(cmd, args)

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

func TestRunProgress(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-progress")
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
		"mark pending task as in-progress": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Pending task")
				return tl.WriteFile(filename)
			},
			taskID:      "1",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if tl.Tasks[0].Status != task.InProgress {
					t.Fatalf("Expected task to be in-progress, got status %v", tl.Tasks[0].Status)
				}
			},
		},
		"mark completed task as in-progress": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Completed task")
				tl.UpdateStatus("1", task.Completed)
				return tl.WriteFile(filename)
			},
			taskID:      "1",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if tl.Tasks[0].Status != task.InProgress {
					t.Fatalf("Expected task to be in-progress, got status %v", tl.Tasks[0].Status)
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

			err := runProgress(cmd, args)

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
