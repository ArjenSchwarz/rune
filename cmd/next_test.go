package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
)

func TestNextCommand(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-next-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	tests := map[string]struct {
		setupTasks      func() *task.TaskList
		setupFile       string
		wantErr         bool
		expectNoTasks   bool
		expectTaskID    string
		expectTaskTitle string
	}{
		"find first incomplete task": {
			setupTasks: func() *task.TaskList {
				tl := task.NewTaskList("Test Project")
				tl.AddTask("", "First task", "")   // 1
				tl.AddTask("", "Second task", "")  // 2
				tl.AddTask("1", "Subtask 1.1", "") // 1.1
				tl.AddTask("1", "Subtask 1.2", "") // 1.2
				tl.UpdateStatus("1.1", task.Completed)
				return tl
			},
			setupFile:       "test-incomplete.md",
			wantErr:         false,
			expectTaskID:    "1",
			expectTaskTitle: "First task",
		},
		"find deeply nested incomplete task": {
			setupTasks: func() *task.TaskList {
				tl := task.NewTaskList("Deep Project")
				tl.AddTask("", "First task", "")       // 1
				tl.AddTask("1", "Subtask 1.1", "")     // 1.1
				tl.AddTask("1.1", "Subtask 1.1.1", "") // 1.1.1
				tl.AddTask("1.1", "Subtask 1.1.2", "") // 1.1.2
				tl.UpdateStatus("1", task.Completed)
				tl.UpdateStatus("1.1", task.Completed)
				tl.UpdateStatus("1.1.1", task.Completed)
				// 1.1.2 remains incomplete
				return tl
			},
			setupFile:       "test-deep.md",
			wantErr:         false,
			expectTaskID:    "1",
			expectTaskTitle: "First task",
		},
		"all tasks completed": {
			setupTasks: func() *task.TaskList {
				tl := task.NewTaskList("Complete Project")
				tl.AddTask("", "First task", "")
				tl.AddTask("", "Second task", "")
				tl.AddTask("1", "Subtask 1.1", "")
				tl.UpdateStatus("1", task.Completed)
				tl.UpdateStatus("1.1", task.Completed)
				tl.UpdateStatus("2", task.Completed)
				return tl
			},
			setupFile:     "test-complete.md",
			wantErr:       false,
			expectNoTasks: true,
		},
		"in-progress tasks are incomplete": {
			setupTasks: func() *task.TaskList {
				tl := task.NewTaskList("In Progress Project")
				tl.AddTask("", "First task", "")
				tl.UpdateStatus("1", task.InProgress)
				return tl
			},
			setupFile:       "test-inprogress.md",
			wantErr:         false,
			expectTaskID:    "1",
			expectTaskTitle: "First task",
		},
		"nonexistent file": {
			setupTasks:    nil, // Don't create any file
			setupFile:     "nonexistent.md",
			wantErr:       true,
			expectNoTasks: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Setup task file if needed
			if tc.setupTasks != nil {
				tl := tc.setupTasks()
				if err := tl.WriteFile(tc.setupFile); err != nil {
					t.Fatalf("failed to write test file: %v", err)
				}
			}

			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the command
			rootCmd.SetArgs([]string{"next", tc.setupFile})
			err := rootCmd.Execute()

			// Restore stdout and capture output
			w.Close()
			os.Stdout = oldStdout
			buf.ReadFrom(r)
			output := buf.String()

			// Reset command args for next test
			rootCmd.SetArgs([]string{})

			// Check error expectation
			if tc.wantErr && err == nil {
				t.Errorf("expected error but got none")
				return
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Skip output checks if we expected an error
			if tc.wantErr {
				return
			}

			// Check output expectations
			if tc.expectNoTasks {
				if !strings.Contains(output, "All tasks are complete!") {
					t.Errorf("expected 'All tasks are complete!' message, got: %s", output)
				}
			} else {
				if !strings.Contains(output, tc.expectTaskID) {
					t.Errorf("expected task ID %s in output, got: %s", tc.expectTaskID, output)
				}
				if !strings.Contains(output, tc.expectTaskTitle) {
					t.Errorf("expected task title '%s' in output, got: %s", tc.expectTaskTitle, output)
				}
			}
		})
	}
}

func TestNextCommandWithTaskDetailsAndReferences(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-next-details-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	tests := map[string]struct {
		fileContent    string
		fileName       string
		expectInOutput []string
		format         string
	}{
		"task with details and task-level references": {
			fileContent: `---
references:
  - ./docs/architecture.md
  - ./specs/api.yaml
---
# Project Tasks

- [ ] 1. Setup development environment
  - This involves setting up the complete development stack
  - including Docker containers and environment variables.
  - Make sure to follow the setup guide carefully.
  - References: ./setup-guide.md, ./docker-compose.yml
  - [x] 1.1. Install dependencies
  - [ ] 1.2. Configure database
    - Create database schema and initial migrations.
    - Make sure to use the latest PostgreSQL version.
    - References: ./db/migrations/, ./db/schema.sql
- [x] 2. Implement authentication
`,
			fileName: "test-with-details.md",
			expectInOutput: []string{
				"Setup development environment", // task title
				"This involves setting up",      // task details
				"./setup-guide.md",              // task-level reference
				"./docker-compose.yml",          // task-level reference
				"./docs/architecture.md",        // front matter reference
				"./specs/api.yaml",              // front matter reference
			},
			format: "table",
		},
		"nested task with details in markdown format": {
			fileContent: `# Complex Tasks

- [-] 1. Design Phase
  - [x] 1.1. User experience design
    - [x] 1.1.1. User persona development
      - Primary user research completed
      - Persona documentation created
      - References: personas.md
    - [-] 1.1.2. User journey mapping
      - Current state mapping in progress
      - Future state design pending
      - Pain point identification needed
      - References: journey-maps.png, research-notes.md
`,
			fileName: "test-nested-details.md",
			expectInOutput: []string{
				"# Next Task",                       // markdown header
				"- [-] 1. Design Phase",             // main task
				"- [-] 1.1.2. User journey mapping", // incomplete subtask
				"Current state mapping",             // task details
				"journey-maps.png",                  // task-level reference
			},
			format: "markdown",
		},
		"json format with details and references": {
			fileContent: `---
references:
  - ./global-api-spec.yaml
  - ./auth-guide.md
---
# JSON Test

- [ ] 1. API Implementation
  - Implement REST API endpoints following OpenAPI specification.
  - Ensure proper error handling and validation.
  - References: api-spec.yaml
  - [x] 1.1. Authentication endpoints
  - [ ] 1.2. User endpoints
    - CRUD operations for user management
    - Include role-based access control
    - References: user-api.md, rbac-spec.md
`,
			fileName: "test-json-details.md",
			expectInOutput: []string{
				`"id": "1"`,                     // task ID
				`"title": "API Implementation"`, // task title
				`"details"`,                     // details field
				`"Implement REST API endpoints following OpenAPI specification."`, // detail content
				`"references"`,              // references field
				`"api-spec.yaml"`,           // task-level reference
				`"task_references"`,         // task references field
				`"front_matter_references"`, // front matter references field
			},
			format: "json",
		},
		"task without details or references": {
			fileContent: `# Simple Tasks

- [ ] 1. Simple task
- [x] 2. Completed task
`,
			fileName: "test-simple.md",
			expectInOutput: []string{
				"Simple task", // task title should still appear
			},
			format: "table",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Write test file
			if err := os.WriteFile(tc.fileName, []byte(tc.fileContent), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the command with format flag
			args := []string{"next", tc.fileName}
			if tc.format != "" {
				args = append(args, "--format", tc.format)
			}
			rootCmd.SetArgs(args)
			err := rootCmd.Execute()

			// Restore stdout and capture output
			w.Close()
			os.Stdout = oldStdout
			buf.ReadFrom(r)
			output := buf.String()

			// Reset command args for next test
			rootCmd.SetArgs([]string{})

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check expected strings in output
			for _, expected := range tc.expectInOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("test %s: expected '%s' in output, got: %s", name, expected, output)
				}
			}

			// For JSON, validate it's valid JSON
			if tc.format == "json" {
				var jsonObj any
				if err := json.Unmarshal([]byte(output), &jsonObj); err != nil {
					t.Errorf("JSON format produced invalid JSON: %v\nOutput: %s", err, output)
				}
			}
		})
	}
}

func TestNextCommandOutputFormats(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-next-format-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Create a test file with front matter references
	taskContent := `---
references:
  - ./docs/architecture.md
  - ./specs/api.yaml
metadata:
  project: test-project
---
# Test Project

- [ ] 1. First task
  - [x] 1.1. Completed subtask
  - [ ] 1.2. Incomplete subtask
- [x] 2. Completed task
`

	testFile := "test-with-refs.md"
	if err := os.WriteFile(testFile, []byte(taskContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	tests := map[string]struct {
		format         string
		expectInOutput []string
	}{
		"table format": {
			format: "table",
			expectInOutput: []string{
				"Next Task",              // table title
				"First task",             // task title
				"./docs/architecture.md", // reference
				"./specs/api.yaml",       // reference
			},
		},
		"markdown format": {
			format: "markdown",
			expectInOutput: []string{
				"# Next Task",            // markdown header
				"- [ ] 1. First task",    // task in markdown format
				"## References",          // references section
				"./docs/architecture.md", // reference
			},
		},
		"json format": {
			format: "json",
			expectInOutput: []string{
				"\"id\": \"1\"",               // JSON field
				"\"title\": \"First task\"",   // JSON field
				"\"front_matter_references\"", // references field
				"./docs/architecture.md",      // reference in JSON
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the command with format flag
			rootCmd.SetArgs([]string{"next", testFile, "--format", tc.format})
			err := rootCmd.Execute()

			// Restore stdout and capture output
			w.Close()
			os.Stdout = oldStdout
			buf.ReadFrom(r)
			output := buf.String()

			// Reset command args for next test
			rootCmd.SetArgs([]string{})

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check expected strings in output
			for _, expected := range tc.expectInOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("format %s: expected '%s' in output, got: %s", tc.format, expected, output)
				}
			}

			// For JSON, also validate it's valid JSON
			if tc.format == "json" {
				var jsonObj any
				if err := json.Unmarshal([]byte(output), &jsonObj); err != nil {
					t.Errorf("JSON format produced invalid JSON: %v", err)
				}
			}
		})
	}
}

func TestResolveFilename(t *testing.T) {
	tests := map[string]struct {
		args        []string
		expectArg   bool
		expectError bool
	}{
		"explicit filename provided": {
			args:        []string{"test.md"},
			expectArg:   true,
			expectError: false,
		},
		"no filename provided": {
			args:        []string{},
			expectArg:   false,
			expectError: true, // Will error since git discovery is disabled by default
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := resolveFilename(tc.args)

			if tc.expectError && err == nil {
				t.Errorf("expected error but got none")
				return
			}

			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tc.expectArg && result != tc.args[0] {
				t.Errorf("expected result %s, got %s", tc.args[0], result)
			}
		})
	}
}

func TestNextCommandHelperFunctions(t *testing.T) {
	// Test formatStatusMarkdown
	tests := []struct {
		status   task.Status
		expected string
	}{
		{task.Pending, "[ ]"},
		{task.InProgress, "[-]"},
		{task.Completed, "[x]"},
	}

	for _, tc := range tests {
		result := formatStatusMarkdown(tc.status)
		if result != tc.expected {
			t.Errorf("formatStatusMarkdown(%v) = %s, want %s", tc.status, result, tc.expected)
		}
	}

	// Test getTaskLevel
	levelTests := []struct {
		id       string
		expected int
	}{
		{"1", 1},
		{"1.1", 2},
		{"1.2.3", 3},
		{"", 0},
	}

	for _, tc := range levelTests {
		result := getTaskLevel(tc.id)
		if result != tc.expected {
			t.Errorf("getTaskLevel(%s) = %d, want %d", tc.id, result, tc.expected)
		}
	}

	// Test formatStatus
	for _, tc := range []struct {
		status   task.Status
		expected string
	}{
		{task.Pending, "Pending"},
		{task.InProgress, "In Progress"},
		{task.Completed, "Completed"},
	} {
		result := formatStatus(tc.status)
		if result != tc.expected {
			t.Errorf("formatStatus(%v) = %s, want %s", tc.status, result, tc.expected)
		}
	}
}

func TestNextCommandWithPhases(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-next-phase-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	tests := map[string]struct {
		fileContent    string
		fileName       string
		usePhaseFlag   bool
		expectInOutput []string
		expectNoTasks  bool
		format         string
	}{
		"next phase with pending tasks": {
			fileContent: `# Project with Phases

## Planning
- [x] 1. Research requirements
- [x] 2. Define scope

## Implementation
- [ ] 3. Setup development environment
- [ ] 4. Implement core features
  - [ ] 4.1. Authentication
  - [ ] 4.2. Database layer

## Testing
- [ ] 5. Write unit tests
- [ ] 6. Write integration tests
`,
			fileName:       "phases-pending.md",
			usePhaseFlag:   true,
			expectInOutput: []string{"3", "Setup development environment", "4", "Implement core features"},
			format:         "table",
		},
		"all phases complete": {
			fileContent: `# Complete Project

## Planning
- [x] 1. Research requirements
- [x] 2. Define scope

## Implementation
- [x] 3. Setup development environment
- [x] 4. Implement core features
  - [x] 4.1. Authentication
  - [x] 4.2. Database layer

## Testing
- [x] 5. Write unit tests
- [x] 6. Write integration tests
`,
			fileName:      "phases-complete.md",
			usePhaseFlag:  true,
			expectNoTasks: true,
			format:        "table",
		},
		"skip first phase with all complete tasks": {
			fileContent: `# Mixed Completion Project

## Planning
- [x] 1. Research requirements
- [x] 2. Define scope

## Implementation
- [x] 3. Setup development environment
- [ ] 4. Implement core features
  - [x] 4.1. Authentication
  - [ ] 4.2. Database layer

## Testing
- [ ] 5. Write unit tests
`,
			fileName:       "phases-mixed.md",
			usePhaseFlag:   true,
			expectInOutput: []string{"4", "Implement core features", "4.2", "Database layer"},
			format:         "table",
		},
		"phase flag with json format": {
			fileContent: `# JSON Phase Test

## Development
- [ ] 1. Task one
  - Details for task one
  - References: task-one.md
- [ ] 2. Task two

## Testing
- [ ] 3. Task three
`,
			fileName:       "phases-json.md",
			usePhaseFlag:   true,
			expectInOutput: []string{`"id": "1"`, `"title": "Task one"`, `"id": "2"`, `"title": "Task two"`},
			format:         "json",
		},
		"phase flag with markdown format": {
			fileContent: `# Markdown Phase Test

## Phase One
- [ ] 1. First task
- [ ] 2. Second task
  - [ ] 2.1. Subtask

## Phase Two  
- [ ] 3. Third task
`,
			fileName:       "phases-markdown.md",
			usePhaseFlag:   true,
			expectInOutput: []string{"# Next Phase Tasks", "- [ ] 1. First task", "- [ ] 2. Second task", "- [ ] 2.1. Subtask"},
			format:         "markdown",
		},
		"existing behavior preserved without phase flag": {
			fileContent: `# Without Phase Flag

## Planning
- [x] 1. Complete task

## Implementation
- [ ] 2. Pending task
- [ ] 3. Another pending task
`,
			fileName:       "no-phase-flag.md",
			usePhaseFlag:   false,
			expectInOutput: []string{"2", "Pending task"},
			format:         "table",
		},
		"document without phases": {
			fileContent: `# No Phases Document

- [ ] 1. First task
- [x] 2. Complete task
- [ ] 3. Another task
`,
			fileName:       "no-phases.md",
			usePhaseFlag:   true,
			expectInOutput: []string{"1", "First task", "3", "Another task"},
			format:         "table",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Write test file
			if err := os.WriteFile(tc.fileName, []byte(tc.fileContent), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Build command args
			args := []string{"next", tc.fileName}
			if tc.usePhaseFlag {
				args = append(args, "--phase")
			}
			if tc.format != "" {
				args = append(args, "--format", tc.format)
			}

			rootCmd.SetArgs(args)
			err := rootCmd.Execute()

			// Restore stdout and capture output
			w.Close()
			os.Stdout = oldStdout
			buf.ReadFrom(r)
			output := buf.String()

			// Reset command args for next test
			rootCmd.SetArgs([]string{})

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check expected output
			if tc.expectNoTasks {
				if !strings.Contains(output, "No pending tasks found in any phase") && !strings.Contains(output, "All tasks are complete!") {
					t.Errorf("expected no tasks message, got: %s", output)
				}
			} else {
				for _, expected := range tc.expectInOutput {
					if !strings.Contains(output, expected) {
						t.Errorf("expected '%s' in output, got: %s", expected, output)
					}
				}
			}

			// For JSON, validate it's valid JSON
			if tc.format == "json" && !tc.expectNoTasks {
				var jsonObj any
				if err := json.Unmarshal([]byte(output), &jsonObj); err != nil {
					t.Errorf("JSON format produced invalid JSON: %v\nOutput: %s", err, output)
				}
			}
		})
	}
}
