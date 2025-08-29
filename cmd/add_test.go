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

func TestRunAdd(t *testing.T) {
	// Create temporary directory for test files within the current working directory
	// This is required due to path traversal protection in validateFilePath
	tempDir := filepath.Join(".", "test-tmp-add")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := map[string]struct {
		setupFile     func(string) error
		title         string
		parent        string
		expectError   bool
		errorContains string
		validateFile  func(*testing.T, string)
	}{
		"add top-level task to existing file": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Existing task")
				return tl.WriteFile(filename)
			},
			title:       "New top-level task",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if len(tl.Tasks) != 2 {
					t.Fatalf("Expected 2 tasks, got %d", len(tl.Tasks))
				}
				if tl.Tasks[1].Title != "New top-level task" {
					t.Fatalf("Expected new task title 'New top-level task', got '%s'", tl.Tasks[1].Title)
				}
				if tl.Tasks[1].ID != "2" {
					t.Fatalf("Expected new task ID '2', got '%s'", tl.Tasks[1].ID)
				}
			},
		},
		"add subtask to existing parent": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Parent task")
				return tl.WriteFile(filename)
			},
			title:       "Child task",
			parent:      "1",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if len(tl.Tasks) != 1 {
					t.Fatalf("Expected 1 top-level task, got %d", len(tl.Tasks))
				}
				if len(tl.Tasks[0].Children) != 1 {
					t.Fatalf("Expected 1 child task, got %d", len(tl.Tasks[0].Children))
				}
				child := tl.Tasks[0].Children[0]
				if child.Title != "Child task" {
					t.Fatalf("Expected child task title 'Child task', got '%s'", child.Title)
				}
				if child.ID != "1.1" {
					t.Fatalf("Expected child task ID '1.1', got '%s'", child.ID)
				}
				if child.ParentID != "1" {
					t.Fatalf("Expected child parent ID '1', got '%s'", child.ParentID)
				}
			},
		},
		"add task with non-existent parent": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				return tl.WriteFile(filename)
			},
			title:         "Child task",
			parent:        "999",
			expectError:   true,
			errorContains: "parent task 999 not found",
		},
		"file does not exist": {
			setupFile: func(filename string) error {
				// Don't create the file
				return nil
			},
			title:         "New task",
			expectError:   true,
			errorContains: "does not exist",
		},
		"add multiple subtasks": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Parent task")
				tl.AddTask("1", "First child")
				return tl.WriteFile(filename)
			},
			title:       "Second child",
			parent:      "1",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if len(tl.Tasks[0].Children) != 2 {
					t.Fatalf("Expected 2 child tasks, got %d", len(tl.Tasks[0].Children))
				}
				secondChild := tl.Tasks[0].Children[1]
				if secondChild.Title != "Second child" {
					t.Fatalf("Expected second child title 'Second child', got '%s'", secondChild.Title)
				}
				if secondChild.ID != "1.2" {
					t.Fatalf("Expected second child ID '1.2', got '%s'", secondChild.ID)
				}
			},
		},
		"add nested subtask": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Parent task")
				tl.AddTask("1", "Child task")
				return tl.WriteFile(filename)
			},
			title:       "Grandchild task",
			parent:      "1.1",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				child := &tl.Tasks[0].Children[0]
				if len(child.Children) != 1 {
					t.Fatalf("Expected 1 grandchild task, got %d", len(child.Children))
				}
				grandchild := child.Children[0]
				if grandchild.Title != "Grandchild task" {
					t.Fatalf("Expected grandchild title 'Grandchild task', got '%s'", grandchild.Title)
				}
				if grandchild.ID != "1.1.1" {
					t.Fatalf("Expected grandchild ID '1.1.1', got '%s'", grandchild.ID)
				}
				if grandchild.ParentID != "1.1" {
					t.Fatalf("Expected grandchild parent ID '1.1', got '%s'", grandchild.ParentID)
				}
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create test file
			filename := filepath.Join(tempDir, "test-"+strings.ReplaceAll(name, " ", "-")+".md")

			if err := tt.setupFile(filename); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Set command flags
			addTitle = tt.title
			addParent = tt.parent
			dryRun = false

			// Create command and capture output
			cmd := &cobra.Command{}
			args := []string{filename}

			err := runAdd(cmd, args)

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

			// Validate the file if validator is provided
			if tt.validateFile != nil {
				tt.validateFile(t, filename)
			}

			// Reset flags for next test
			addTitle = ""
			addParent = ""
		})
	}
}

func TestRunAddDryRun(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-dry")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	filename := filepath.Join(tempDir, "test.md")

	// Create initial file
	tl := task.NewTaskList("Test Tasks")
	tl.AddTask("", "Existing task")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Read initial content
	initialContent, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read initial file: %v", err)
	}

	// Set up dry run
	addTitle = "New task"
	addParent = ""
	dryRun = true

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run command
	cmd := &cobra.Command{}
	args := []string{filename}
	err = runAdd(cmd, args)

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Unexpected error in dry run: %v", err)
	}

	// Check that output contains expected information
	if !strings.Contains(output, "Would add task to file") {
		t.Fatalf("Expected dry run output to contain 'Would add task to file', got: %s", output)
	}
	if !strings.Contains(output, "Title: New task") {
		t.Fatalf("Expected dry run output to contain title, got: %s", output)
	}
	if !strings.Contains(output, "New task ID would be: 2") {
		t.Fatalf("Expected dry run output to show new task ID, got: %s", output)
	}

	// Verify file wasn't modified
	finalContent, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read final file: %v", err)
	}

	if !bytes.Equal(initialContent, finalContent) {
		t.Fatal("File was modified during dry run")
	}

	// Reset flags
	addTitle = ""
	dryRun = false
}

func TestRunAddDryRunWithParent(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-parent")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	filename := filepath.Join(tempDir, "test.md")

	// Create initial file with a parent task
	tl := task.NewTaskList("Test Tasks")
	tl.AddTask("", "Parent task")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Set up dry run with parent
	addTitle = "Child task"
	addParent = "1"
	dryRun = true

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run command
	cmd := &cobra.Command{}
	args := []string{filename}
	err := runAdd(cmd, args)

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Unexpected error in dry run: %v", err)
	}

	// Check that output contains parent information
	if !strings.Contains(output, "Parent: 1 (Parent task)") {
		t.Fatalf("Expected dry run output to show parent info, got: %s", output)
	}
	if !strings.Contains(output, "New task ID would be: 1.1") {
		t.Fatalf("Expected dry run output to show child task ID, got: %s", output)
	}

	// Reset flags
	addTitle = ""
	addParent = ""
	dryRun = false
}

func TestAddCmdFlags(t *testing.T) {
	// Test that required flags are properly configured
	if !addCmd.Flag("title").Changed && addCmd.Flag("title").Value.String() == "" {
		// This should be required
		addCmd.MarkFlagRequired("title")
	}

	// Test flag descriptions
	titleFlag := addCmd.Flag("title")
	if titleFlag == nil {
		t.Fatal("Title flag not found")
	}
	if titleFlag.Usage == "" {
		t.Fatal("Title flag should have usage description")
	}

	parentFlag := addCmd.Flag("parent")
	if parentFlag == nil {
		t.Fatal("Parent flag not found")
	}
	if parentFlag.Usage == "" {
		t.Fatal("Parent flag should have usage description")
	}
}
