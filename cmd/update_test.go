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

func TestRunUpdate(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-update")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := map[string]struct {
		setupFile       func(string) error
		taskID          string
		title           string
		details         string
		references      string
		clearDetails    bool
		clearReferences bool
		expectError     bool
		errorContains   string
		validateFile    func(*testing.T, string)
	}{
		"update title only": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Original title", "")
				return tl.WriteFile(filename)
			},
			taskID: "1",
			title:  "Updated title",
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if tl.Tasks[0].Title != "Updated title" {
					t.Fatalf("Expected title 'Updated title', got '%s'", tl.Tasks[0].Title)
				}
				// Other fields should remain unchanged
				if len(tl.Tasks[0].Details) != 0 {
					t.Fatalf("Expected no details, got %v", tl.Tasks[0].Details)
				}
				if len(tl.Tasks[0].References) != 0 {
					t.Fatalf("Expected no references, got %v", tl.Tasks[0].References)
				}
			},
		},
		"update details only": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task with details", "")
				tl.UpdateTask("1", "", []string{"Old detail"}, []string{})
				return tl.WriteFile(filename)
			},
			taskID:  "1",
			details: "New detail,Another detail",
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				expectedDetails := []string{"New detail", "Another detail"}
				if len(tl.Tasks[0].Details) != len(expectedDetails) {
					t.Fatalf("Expected %d details, got %d", len(expectedDetails), len(tl.Tasks[0].Details))
				}
				for i, expected := range expectedDetails {
					if tl.Tasks[0].Details[i] != expected {
						t.Fatalf("Expected detail %d to be '%s', got '%s'", i, expected, tl.Tasks[0].Details[i])
					}
				}
			},
		},
		"update references only": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task with references", "")
				return tl.WriteFile(filename)
			},
			taskID:     "1",
			references: "doc.md,spec.md,readme.md",
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				expectedRefs := []string{"doc.md", "spec.md", "readme.md"}
				if len(tl.Tasks[0].References) != len(expectedRefs) {
					t.Fatalf("Expected %d references, got %d", len(expectedRefs), len(tl.Tasks[0].References))
				}
				for i, expected := range expectedRefs {
					if tl.Tasks[0].References[i] != expected {
						t.Fatalf("Expected reference %d to be '%s', got '%s'", i, expected, tl.Tasks[0].References[i])
					}
				}
			},
		},
		"update all fields": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Original task", "")
				tl.UpdateTask("1", "", []string{"Old detail"}, []string{"old.md"})
				return tl.WriteFile(filename)
			},
			taskID:     "1",
			title:      "New title",
			details:    "First detail,Second detail",
			references: "new.md,other.md",
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if tl.Tasks[0].Title != "New title" {
					t.Fatalf("Expected title 'New title', got '%s'", tl.Tasks[0].Title)
				}
				expectedDetails := []string{"First detail", "Second detail"}
				if len(tl.Tasks[0].Details) != len(expectedDetails) {
					t.Fatalf("Expected %d details, got %d", len(expectedDetails), len(tl.Tasks[0].Details))
				}
				expectedRefs := []string{"new.md", "other.md"}
				if len(tl.Tasks[0].References) != len(expectedRefs) {
					t.Fatalf("Expected %d references, got %d", len(expectedRefs), len(tl.Tasks[0].References))
				}
			},
		},
		"clear details": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task with details", "")
				tl.UpdateTask("1", "", []string{"Detail to clear"}, []string{})
				return tl.WriteFile(filename)
			},
			taskID:       "1",
			clearDetails: true,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if len(tl.Tasks[0].Details) != 0 {
					t.Fatalf("Expected no details after clearing, got %v", tl.Tasks[0].Details)
				}
			},
		},
		"clear references": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task with references", "")
				tl.UpdateTask("1", "", []string{}, []string{"ref.md"})
				return tl.WriteFile(filename)
			},
			taskID:          "1",
			clearReferences: true,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if len(tl.Tasks[0].References) != 0 {
					t.Fatalf("Expected no references after clearing, got %v", tl.Tasks[0].References)
				}
			},
		},
		"update subtask": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Parent task", "")
				tl.AddTask("1", "Child task", "")
				return tl.WriteFile(filename)
			},
			taskID: "1.1",
			title:  "Updated child task",
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if len(tl.Tasks[0].Children) == 0 {
					t.Fatalf("Expected child task to exist")
				}
				if tl.Tasks[0].Children[0].Title != "Updated child task" {
					t.Fatalf("Expected child title 'Updated child task', got '%s'", tl.Tasks[0].Children[0].Title)
				}
			},
		},
		"non-existent task ID": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				return tl.WriteFile(filename)
			},
			taskID:        "999",
			title:         "New title",
			expectError:   true,
			errorContains: "task 999 not found",
		},
		"no update flags provided": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task", "")
				return tl.WriteFile(filename)
			},
			taskID:        "1",
			expectError:   true,
			errorContains: "at least one update flag must be provided",
		},
		"file does not exist": {
			setupFile: func(filename string) error {
				// Don't create the file
				return nil
			},
			taskID:        "1",
			title:         "New title",
			expectError:   true,
			errorContains: "does not exist",
		},
		"details with whitespace handling": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task", "")
				return tl.WriteFile(filename)
			},
			taskID:  "1",
			details: " Detail 1 , Detail 2 ,Detail 3 ",
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				expectedDetails := []string{"Detail 1", "Detail 2", "Detail 3"}
				if len(tl.Tasks[0].Details) != len(expectedDetails) {
					t.Fatalf("Expected %d details, got %d: %v", len(expectedDetails), len(tl.Tasks[0].Details), tl.Tasks[0].Details)
				}
				for i, expected := range expectedDetails {
					if tl.Tasks[0].Details[i] != expected {
						t.Fatalf("Expected detail %d to be '%s', got '%s'", i, expected, tl.Tasks[0].Details[i])
					}
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

			// Set command flags
			updateTitle = tt.title
			updateDetails = tt.details
			updateReferences = tt.references
			clearDetails = tt.clearDetails
			clearReferences = tt.clearReferences
			dryRun = false

			cmd := &cobra.Command{}
			args := []string{filename, tt.taskID}

			err := runUpdate(cmd, args)

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

			// Reset flags for next test
			updateTitle = ""
			updateDetails = ""
			updateReferences = ""
			clearDetails = false
			clearReferences = false
		})
	}
}

func TestRunUpdateDryRun(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-update-dry")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filename := filepath.Join(tempDir, "test.md")

	// Create test file
	tl := task.NewTaskList("Test Tasks")
	tl.AddTask("", "Original task", "")
	tl.UpdateTask("1", "", []string{"Original detail"}, []string{"original.md"})
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Read initial content
	initialContent, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read initial file: %v", err)
	}

	// Set up dry run
	updateTitle = "New title"
	updateDetails = "New detail"
	updateReferences = "new.md"
	dryRun = true

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run command
	cmd := &cobra.Command{}
	args := []string{filename, "1"}
	err = runUpdate(cmd, args)

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
		"Would update task in file",
		"Task ID: 1",
		"Current title: Original task",
		"Current details: Original detail",
		"Current references: original.md",
		"New title: New title",
		"New details: New detail",
		"New references: new.md",
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

	// Reset flags
	updateTitle = ""
	updateDetails = ""
	updateReferences = ""
	dryRun = false
}

func TestUpdateCmdFlags(t *testing.T) {
	// Test that flags are properly configured
	titleFlag := updateCmd.Flag("title")
	if titleFlag == nil {
		t.Fatal("Title flag not found")
	}

	detailsFlag := updateCmd.Flag("details")
	if detailsFlag == nil {
		t.Fatal("Details flag not found")
	}

	referencesFlag := updateCmd.Flag("references")
	if referencesFlag == nil {
		t.Fatal("References flag not found")
	}

	clearDetailsFlag := updateCmd.Flag("clear-details")
	if clearDetailsFlag == nil {
		t.Fatal("Clear-details flag not found")
	}

	clearReferencesFlag := updateCmd.Flag("clear-references")
	if clearReferencesFlag == nil {
		t.Fatal("Clear-references flag not found")
	}
}

func TestFormatFunctions(t *testing.T) {
	// Test formatDetailsForDisplay
	tests := []struct {
		input    []string
		expected string
	}{
		{[]string{}, "(none)"},
		{[]string{"detail1"}, "detail1"},
		{[]string{"detail1", "detail2"}, "detail1, detail2"},
		{[]string{"detail1", "detail2", "detail3"}, "detail1, detail2, detail3"},
	}

	for _, tt := range tests {
		result := formatDetailsForDisplay(tt.input)
		if result != tt.expected {
			t.Fatalf("formatDetailsForDisplay(%v) = %s, expected %s", tt.input, result, tt.expected)
		}
	}

	// Test formatReferencesForDisplay (should behave the same)
	for _, tt := range tests {
		result := formatReferencesForDisplay(tt.input)
		if result != tt.expected {
			t.Fatalf("formatReferencesForDisplay(%v) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}
