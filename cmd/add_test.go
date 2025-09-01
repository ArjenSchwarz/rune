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
		position      string
		expectError   bool
		errorContains string
		validateFile  func(*testing.T, string)
	}{
		"add top-level task to existing file": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Existing task", "")
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
				tl.AddTask("", "Parent task", "")
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
				tl.AddTask("", "Parent task", "")
				tl.AddTask("1", "First child", "")
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
				tl.AddTask("", "Parent task", "")
				tl.AddTask("1", "Child task", "")
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
		"add task at specific position": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task 1", "")
				tl.AddTask("", "Task 2", "")
				tl.AddTask("", "Task 3", "")
				return tl.WriteFile(filename)
			},
			title:       "Inserted task",
			position:    "2",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if len(tl.Tasks) != 4 {
					t.Fatalf("Expected 4 tasks, got %d", len(tl.Tasks))
				}
				// Task 1 should remain as task 1
				if tl.Tasks[0].Title != "Task 1" || tl.Tasks[0].ID != "1" {
					t.Fatalf("Task 1 unexpected: title='%s' id='%s'", tl.Tasks[0].Title, tl.Tasks[0].ID)
				}
				// Inserted task should be at position 2
				if tl.Tasks[1].Title != "Inserted task" || tl.Tasks[1].ID != "2" {
					t.Fatalf("Inserted task unexpected: title='%s' id='%s'", tl.Tasks[1].Title, tl.Tasks[1].ID)
				}
				// Original task 2 should now be task 3
				if tl.Tasks[2].Title != "Task 2" || tl.Tasks[2].ID != "3" {
					t.Fatalf("Task 2 unexpected: title='%s' id='%s'", tl.Tasks[2].Title, tl.Tasks[2].ID)
				}
				// Original task 3 should now be task 4
				if tl.Tasks[3].Title != "Task 3" || tl.Tasks[3].ID != "4" {
					t.Fatalf("Task 3 unexpected: title='%s' id='%s'", tl.Tasks[3].Title, tl.Tasks[3].ID)
				}
			},
		},
		"add task at position beyond list size": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task 1", "")
				tl.AddTask("", "Task 2", "")
				return tl.WriteFile(filename)
			},
			title:       "New task",
			position:    "10",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if len(tl.Tasks) != 3 {
					t.Fatalf("Expected 3 tasks, got %d", len(tl.Tasks))
				}
				// New task should be appended at the end
				if tl.Tasks[2].Title != "New task" || tl.Tasks[2].ID != "3" {
					t.Fatalf("New task unexpected: title='%s' id='%s'", tl.Tasks[2].Title, tl.Tasks[2].ID)
				}
			},
		},
		"add subtask with position and parent": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Parent task", "")
				tl.AddTask("1", "Child 1", "")
				tl.AddTask("1", "Child 2", "")
				tl.AddTask("1", "Child 3", "")
				return tl.WriteFile(filename)
			},
			title:       "Inserted child",
			parent:      "1",
			position:    "1.2",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if len(tl.Tasks[0].Children) != 4 {
					t.Fatalf("Expected 4 children, got %d", len(tl.Tasks[0].Children))
				}
				children := tl.Tasks[0].Children
				// Child 1 should remain as 1.1
				if children[0].Title != "Child 1" || children[0].ID != "1.1" {
					t.Fatalf("Child 1 unexpected: title='%s' id='%s'", children[0].Title, children[0].ID)
				}
				// Inserted child should be at position 1.2
				if children[1].Title != "Inserted child" || children[1].ID != "1.2" {
					t.Fatalf("Inserted child unexpected: title='%s' id='%s'", children[1].Title, children[1].ID)
				}
				// Original Child 2 should now be 1.3
				if children[2].Title != "Child 2" || children[2].ID != "1.3" {
					t.Fatalf("Child 2 unexpected: title='%s' id='%s'", children[2].Title, children[2].ID)
				}
				// Original Child 3 should now be 1.4
				if children[3].Title != "Child 3" || children[3].ID != "1.4" {
					t.Fatalf("Child 3 unexpected: title='%s' id='%s'", children[3].Title, children[3].ID)
				}
			},
		},
		"add task with invalid position format": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task 1", "")
				return tl.WriteFile(filename)
			},
			title:         "New task",
			position:      "invalid-position",
			expectError:   true,
			errorContains: "invalid position format",
		},
		"add task at position 1": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task 1", "")
				tl.AddTask("", "Task 2", "")
				return tl.WriteFile(filename)
			},
			title:       "First task",
			position:    "1",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if len(tl.Tasks) != 3 {
					t.Fatalf("Expected 3 tasks, got %d", len(tl.Tasks))
				}
				// New task should be at position 1
				if tl.Tasks[0].Title != "First task" || tl.Tasks[0].ID != "1" {
					t.Fatalf("First task unexpected: title='%s' id='%s'", tl.Tasks[0].Title, tl.Tasks[0].ID)
				}
				// Original task 1 should now be task 2
				if tl.Tasks[1].Title != "Task 1" || tl.Tasks[1].ID != "2" {
					t.Fatalf("Task 1 unexpected: title='%s' id='%s'", tl.Tasks[1].Title, tl.Tasks[1].ID)
				}
				// Original task 2 should now be task 3
				if tl.Tasks[2].Title != "Task 2" || tl.Tasks[2].ID != "3" {
					t.Fatalf("Task 2 unexpected: title='%s' id='%s'", tl.Tasks[2].Title, tl.Tasks[2].ID)
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
			addPosition = tt.position
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
			addPosition = ""
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
	tl.AddTask("", "Existing task", "")
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
	tl.AddTask("", "Parent task", "")
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

func TestRunAddDryRunWithPosition(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-position-dry")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	filename := filepath.Join(tempDir, "test.md")

	// Create initial file with multiple tasks
	tl := task.NewTaskList("Test Tasks")
	tl.AddTask("", "Task 1", "")
	tl.AddTask("", "Task 2", "")
	tl.AddTask("", "Task 3", "")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Read initial content
	initialContent, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read initial file: %v", err)
	}

	// Set up dry run with position
	addTitle = "Inserted task"
	addPosition = "2"
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

	// Check that output contains position information
	if !strings.Contains(output, "Position: 2") {
		t.Fatalf("Expected dry run output to show position, got: %s", output)
	}
	if !strings.Contains(output, "Title: Inserted task") {
		t.Fatalf("Expected dry run output to contain title, got: %s", output)
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
	addPosition = ""
	dryRun = false
}

func TestRunAddDryRunWithPositionAndParent(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-pos-parent-dry")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	filename := filepath.Join(tempDir, "test.md")

	// Create initial file with parent and children
	tl := task.NewTaskList("Test Tasks")
	tl.AddTask("", "Parent task", "")
	tl.AddTask("1", "Child 1", "")
	tl.AddTask("1", "Child 2", "")
	tl.AddTask("1", "Child 3", "")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Read initial content
	initialContent, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read initial file: %v", err)
	}

	// Set up dry run with position and parent
	addTitle = "Inserted child"
	addParent = "1"
	addPosition = "1.2"
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

	// Check that output contains both position and parent information
	if !strings.Contains(output, "Parent: 1 (Parent task)") {
		t.Fatalf("Expected dry run output to show parent info, got: %s", output)
	}
	if !strings.Contains(output, "Position: 1.2") {
		t.Fatalf("Expected dry run output to show position, got: %s", output)
	}
	if !strings.Contains(output, "Title: Inserted child") {
		t.Fatalf("Expected dry run output to contain title, got: %s", output)
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
	addParent = ""
	addPosition = ""
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

	positionFlag := addCmd.Flag("position")
	if positionFlag == nil {
		t.Fatal("Position flag not found")
	}
	if positionFlag.Usage == "" {
		t.Fatal("Position flag should have usage description")
	}
}
