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
				tl.UpdateTask("1", "", []string{"Old detail"}, []string{}, nil)
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
				tl.UpdateTask("1", "", []string{"Old detail"}, []string{"old.md"}, nil)
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
				tl.UpdateTask("1", "", []string{"Detail to clear"}, []string{}, nil)
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
				tl.UpdateTask("1", "", []string{}, []string{"ref.md"}, nil)
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
	tl.UpdateTask("1", "", []string{"Original detail"}, []string{"original.md"}, nil)
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

func TestRunUpdateRequirements(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-update-requirements")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := map[string]struct {
		setupFile         func(string) error
		taskID            string
		requirements      string
		clearRequirements bool
		expectError       bool
		errorContains     string
		validateFile      func(*testing.T, string)
	}{
		"update requirements": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task", "")
				return tl.WriteFile(filename)
			},
			taskID:       "1",
			requirements: "1.1,1.2,2.3",
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				expectedReqs := []string{"1.1", "1.2", "2.3"}
				if len(tl.Tasks[0].Requirements) != len(expectedReqs) {
					t.Fatalf("Expected %d requirements, got %d", len(expectedReqs), len(tl.Tasks[0].Requirements))
				}
				for i, expected := range expectedReqs {
					if tl.Tasks[0].Requirements[i] != expected {
						t.Fatalf("Expected requirement %d to be '%s', got '%s'", i, expected, tl.Tasks[0].Requirements[i])
					}
				}
			},
		},
		"clear requirements": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task with requirements", "")
				// Add requirements
				task := tl.FindTask("1")
				task.Requirements = []string{"1.1", "1.2"}
				return tl.WriteFile(filename)
			},
			taskID:            "1",
			clearRequirements: true,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if len(tl.Tasks[0].Requirements) != 0 {
					t.Fatalf("Expected no requirements after clearing, got %v", tl.Tasks[0].Requirements)
				}
			},
		},
		"invalid requirement ID format": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task", "")
				return tl.WriteFile(filename)
			},
			taskID:        "1",
			requirements:  "invalid,1.2",
			expectError:   true,
			errorContains: "invalid requirement ID format",
		},
		"requirements with whitespace handling": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				tl.AddTask("", "Task", "")
				return tl.WriteFile(filename)
			},
			taskID:       "1",
			requirements: " 1.1 , 1.2 ,2.3 ",
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				expectedReqs := []string{"1.1", "1.2", "2.3"}
				if len(tl.Tasks[0].Requirements) != len(expectedReqs) {
					t.Fatalf("Expected %d requirements, got %d: %v", len(expectedReqs), len(tl.Tasks[0].Requirements), tl.Tasks[0].Requirements)
				}
				for i, expected := range expectedReqs {
					if tl.Tasks[0].Requirements[i] != expected {
						t.Fatalf("Expected requirement %d to be '%s', got '%s'", i, expected, tl.Tasks[0].Requirements[i])
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
			updateTitle = "" // Don't update title
			updateRequirements = tt.requirements
			clearRequirements = tt.clearRequirements
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
			updateRequirements = ""
			clearRequirements = false
		})
	}
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

	requirementsFlag := updateCmd.Flag("requirements")
	if requirementsFlag == nil {
		t.Fatal("Requirements flag not found")
	}

	clearDetailsFlag := updateCmd.Flag("clear-details")
	if clearDetailsFlag == nil {
		t.Fatal("Clear-details flag not found")
	}

	clearReferencesFlag := updateCmd.Flag("clear-references")
	if clearReferencesFlag == nil {
		t.Fatal("Clear-references flag not found")
	}

	clearRequirementsFlag := updateCmd.Flag("clear-requirements")
	if clearRequirementsFlag == nil {
		t.Fatal("Clear-requirements flag not found")
	}
}

func TestFormatFunctions(t *testing.T) {
	// Test formatDetailsForDisplay
	tests := map[string]struct {
		input    []string
		expected string
	}{
		"empty slice":   {[]string{}, "(none)"},
		"single detail": {[]string{"detail1"}, "detail1"},
		"two details":   {[]string{"detail1", "detail2"}, "detail1, detail2"},
		"three details": {[]string{"detail1", "detail2", "detail3"}, "detail1, detail2, detail3"},
	}

	for name, tc := range tests {
		t.Run("formatDetailsForDisplay/"+name, func(t *testing.T) {
			result := formatDetailsForDisplay(tc.input)
			if result != tc.expected {
				t.Errorf("got %s, want %s", result, tc.expected)
			}
		})
	}

	// Test formatReferencesForDisplay (should behave the same)
	for name, tc := range tests {
		t.Run("formatReferencesForDisplay/"+name, func(t *testing.T) {
			result := formatReferencesForDisplay(tc.input)
			if result != tc.expected {
				t.Errorf("got %s, want %s", result, tc.expected)
			}
		})
	}

	// Test formatRequirementsForDisplay (should behave the same)
	for name, tc := range tests {
		t.Run("formatRequirementsForDisplay/"+name, func(t *testing.T) {
			result := formatRequirementsForDisplay(tc.input)
			if result != tc.expected {
				t.Errorf("got %s, want %s", result, tc.expected)
			}
		})
	}
}

func TestRunUpdateWithStream(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-update-stream")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := map[string]struct {
		setupFile     func(string) error
		taskID        string
		stream        int
		expectError   bool
		errorContains string
		validateFile  func(*testing.T, string)
	}{
		"update stream to 2": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				_, err := tl.AddTaskWithOptions("", "Task", task.AddOptions{Stream: 1})
				if err != nil {
					return err
				}
				return tl.WriteFile(filename)
			},
			taskID:      "1",
			stream:      2,
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if tl.Tasks[0].Stream != 2 {
					t.Fatalf("Expected stream 2, got %d", tl.Tasks[0].Stream)
				}
			},
		},
		"update stream with invalid value (negative)": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				_, err := tl.AddTaskWithOptions("", "Task", task.AddOptions{})
				if err != nil {
					return err
				}
				return tl.WriteFile(filename)
			},
			taskID:        "1",
			stream:        -1,
			expectError:   true,
			errorContains: "stream must be a positive integer",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			filename := filepath.Join(tempDir, "test-"+strings.ReplaceAll(name, " ", "-")+".md")

			if err := tt.setupFile(filename); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Reset all flags
			updateTitle = ""
			updateDetails = ""
			updateReferences = ""
			updateRequirements = ""
			clearDetails = false
			clearReferences = false
			clearRequirements = false
			updateStream = tt.stream
			updateStreamSet = true
			updateBlockedBy = ""
			updateOwner = ""
			updateRelease = false
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

			// Reset flags
			updateStream = 0
			updateStreamSet = false
		})
	}
}

func TestRunUpdateWithBlockedBy(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-update-blocked-by")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := map[string]struct {
		setupFile     func(string) error
		taskID        string
		blockedBy     string
		expectError   bool
		errorContains string
		validateFile  func(*testing.T, string)
	}{
		"update blocked-by to reference another task": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				_, err := tl.AddTaskWithOptions("", "Blocking task", task.AddOptions{})
				if err != nil {
					return err
				}
				_, err = tl.AddTaskWithOptions("", "Task to update", task.AddOptions{})
				if err != nil {
					return err
				}
				return tl.WriteFile(filename)
			},
			taskID:      "2",
			blockedBy:   "1",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if len(tl.Tasks[1].BlockedBy) != 1 {
					t.Fatalf("Expected 1 blocked-by reference, got %d", len(tl.Tasks[1].BlockedBy))
				}
				// BlockedBy should contain the stable ID of task 1
				if tl.Tasks[1].BlockedBy[0] != tl.Tasks[0].StableID {
					t.Fatalf("Expected blocked-by to reference %s, got %s", tl.Tasks[0].StableID, tl.Tasks[1].BlockedBy[0])
				}
			},
		},
		"update blocked-by with multiple tasks": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				for i := 0; i < 3; i++ {
					_, err := tl.AddTaskWithOptions("", "Task", task.AddOptions{})
					if err != nil {
						return err
					}
				}
				return tl.WriteFile(filename)
			},
			taskID:      "3",
			blockedBy:   "1,2",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if len(tl.Tasks[2].BlockedBy) != 2 {
					t.Fatalf("Expected 2 blocked-by references, got %d", len(tl.Tasks[2].BlockedBy))
				}
			},
		},
		"update blocked-by to non-existent task": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				_, err := tl.AddTaskWithOptions("", "Task", task.AddOptions{})
				if err != nil {
					return err
				}
				return tl.WriteFile(filename)
			},
			taskID:        "1",
			blockedBy:     "999",
			expectError:   true,
			errorContains: "task 999 not found",
		},
		"update blocked-by to legacy task without stable ID": {
			setupFile: func(filename string) error {
				content := `# Test Tasks

- [ ] 1. Legacy task
- [ ] 2. Task to update <!-- id:abc1234 -->
`
				return os.WriteFile(filename, []byte(content), 0644)
			},
			taskID:        "2",
			blockedBy:     "1",
			expectError:   true,
			errorContains: "task does not have a stable ID",
		},
		"cycle detection error": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				// Create task 1
				_, err := tl.AddTaskWithOptions("", "Task 1", task.AddOptions{})
				if err != nil {
					return err
				}
				// Create task 2 that depends on task 1
				_, err = tl.AddTaskWithOptions("", "Task 2", task.AddOptions{BlockedBy: []string{"1"}})
				if err != nil {
					return err
				}
				return tl.WriteFile(filename)
			},
			taskID:        "1",
			blockedBy:     "2", // Task 1 blocked by Task 2, but Task 2 is already blocked by Task 1 -> cycle
			expectError:   true,
			errorContains: "circular dependency detected",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			filename := filepath.Join(tempDir, "test-"+strings.ReplaceAll(name, " ", "-")+".md")

			if err := tt.setupFile(filename); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Reset all flags
			updateTitle = ""
			updateDetails = ""
			updateReferences = ""
			updateRequirements = ""
			clearDetails = false
			clearReferences = false
			clearRequirements = false
			updateStream = 0
			updateStreamSet = false
			updateBlockedBy = tt.blockedBy
			updateOwner = ""
			updateRelease = false
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

			// Reset flags
			updateBlockedBy = ""
		})
	}
}

func TestRunUpdateWithOwner(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-update-owner")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := map[string]struct {
		setupFile     func(string) error
		taskID        string
		owner         string
		expectError   bool
		errorContains string
		validateFile  func(*testing.T, string)
	}{
		"update owner": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				_, err := tl.AddTaskWithOptions("", "Task", task.AddOptions{})
				if err != nil {
					return err
				}
				return tl.WriteFile(filename)
			},
			taskID:      "1",
			owner:       "agent-1",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if tl.Tasks[0].Owner != "agent-1" {
					t.Fatalf("Expected owner 'agent-1', got '%s'", tl.Tasks[0].Owner)
				}
			},
		},
		"update owner with spaces": {
			setupFile: func(filename string) error {
				tl := task.NewTaskList("Test Tasks")
				_, err := tl.AddTaskWithOptions("", "Task", task.AddOptions{})
				if err != nil {
					return err
				}
				return tl.WriteFile(filename)
			},
			taskID:      "1",
			owner:       "Agent Number One",
			expectError: false,
			validateFile: func(t *testing.T, filename string) {
				tl, err := task.ParseFile(filename)
				if err != nil {
					t.Fatalf("Failed to parse file: %v", err)
				}
				if tl.Tasks[0].Owner != "Agent Number One" {
					t.Fatalf("Expected owner 'Agent Number One', got '%s'", tl.Tasks[0].Owner)
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

			// Reset all flags
			updateTitle = ""
			updateDetails = ""
			updateReferences = ""
			updateRequirements = ""
			clearDetails = false
			clearReferences = false
			clearRequirements = false
			updateStream = 0
			updateStreamSet = false
			updateBlockedBy = ""
			updateOwner = tt.owner
			updateOwnerSet = true
			updateRelease = false
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

			// Reset flags
			updateOwner = ""
			updateOwnerSet = false
		})
	}
}

func TestRunUpdateWithRelease(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-update-release")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filename := filepath.Join(tempDir, "test.md")

	// Create a file with a task that has an owner
	tl := task.NewTaskList("Test Tasks")
	_, err := tl.AddTaskWithOptions("", "Task with owner", task.AddOptions{Owner: "agent-1"})
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Verify owner is set
	tl, err = task.ParseFile(filename)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}
	if tl.Tasks[0].Owner != "agent-1" {
		t.Fatalf("Expected initial owner 'agent-1', got '%s'", tl.Tasks[0].Owner)
	}

	// Reset all flags and set release
	updateTitle = ""
	updateDetails = ""
	updateReferences = ""
	updateRequirements = ""
	clearDetails = false
	clearReferences = false
	clearRequirements = false
	updateStream = 0
	updateStreamSet = false
	updateBlockedBy = ""
	updateOwner = ""
	updateOwnerSet = false
	updateRelease = true
	dryRun = false

	cmd := &cobra.Command{}
	args := []string{filename, "1"}

	err = runUpdate(cmd, args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify owner is cleared
	tl, err = task.ParseFile(filename)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}
	if tl.Tasks[0].Owner != "" {
		t.Fatalf("Expected owner to be cleared, got '%s'", tl.Tasks[0].Owner)
	}

	// Reset flags
	updateRelease = false
}

func TestUpdateCmdDependencyFlags(t *testing.T) {
	// Test that new flags are properly configured
	streamFlag := updateCmd.Flag("stream")
	if streamFlag == nil {
		t.Fatal("Stream flag not found")
	}
	if streamFlag.Usage == "" {
		t.Fatal("Stream flag should have usage description")
	}

	blockedByFlag := updateCmd.Flag("blocked-by")
	if blockedByFlag == nil {
		t.Fatal("Blocked-by flag not found")
	}
	if blockedByFlag.Usage == "" {
		t.Fatal("Blocked-by flag should have usage description")
	}

	ownerFlag := updateCmd.Flag("owner")
	if ownerFlag == nil {
		t.Fatal("Owner flag not found")
	}
	if ownerFlag.Usage == "" {
		t.Fatal("Owner flag should have usage description")
	}

	releaseFlag := updateCmd.Flag("release")
	if releaseFlag == nil {
		t.Fatal("Release flag not found")
	}
	if releaseFlag.Usage == "" {
		t.Fatal("Release flag should have usage description")
	}
}
