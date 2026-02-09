package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	tests := map[string]struct {
		status   task.Status
		expected string
	}{
		"pending":     {task.Pending, "[ ]"},
		"in progress": {task.InProgress, "[-]"},
		"completed":   {task.Completed, "[x]"},
	}

	for name, tc := range tests {
		t.Run("formatStatusMarkdown/"+name, func(t *testing.T) {
			result := formatStatusMarkdown(tc.status)
			if result != tc.expected {
				t.Errorf("got %s, want %s", result, tc.expected)
			}
		})
	}

	// Test getTaskLevel
	levelTests := map[string]struct {
		id       string
		expected int
	}{
		"root task":  {"1", 1},
		"child task": {"1.1", 2},
		"grandchild": {"1.2.3", 3},
		"empty id":   {"", 0},
	}

	for name, tc := range levelTests {
		t.Run("getTaskLevel/"+name, func(t *testing.T) {
			result := getTaskLevel(tc.id)
			if result != tc.expected {
				t.Errorf("got %d, want %d", result, tc.expected)
			}
		})
	}

	// Test formatStatus
	statusTests := map[string]struct {
		status   task.Status
		expected string
	}{
		"pending":     {task.Pending, "Pending"},
		"in progress": {task.InProgress, "In Progress"},
		"completed":   {task.Completed, "Completed"},
	}

	for name, tc := range statusTests {
		t.Run("formatStatus/"+name, func(t *testing.T) {
			result := formatStatus(tc.status)
			if result != tc.expected {
				t.Errorf("got %s, want %s", result, tc.expected)
			}
		})
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

func TestNextCommandStreamFilter(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-next-stream-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	tests := map[string]struct {
		fileContent       string
		fileName          string
		streamFlag        int
		format            string
		expectInOutput    []string
		expectNotInOutput []string
	}{
		"stream filter returns only stream tasks": {
			fileContent: `# Project Tasks

- [ ] 1. Task in stream 1 <!-- id:abc1234 -->
  - Stream: 1

- [ ] 2. Task in stream 2 <!-- id:def5678 -->
  - Stream: 2

- [ ] 3. Another task in stream 1 <!-- id:ghi9012 -->
  - Stream: 1
`,
			fileName:          "stream-filter.md",
			streamFlag:        2,
			format:            "json",
			expectInOutput:    []string{`"id": "2"`, `"title": "Task in stream 2"`},
			expectNotInOutput: []string{`"id": "1"`, `"id": "3"`},
		},
		"stream filter with default stream": {
			fileContent: `# Project Tasks

- [ ] 1. Task without explicit stream <!-- id:abc1234 -->

- [ ] 2. Task in stream 2 <!-- id:def5678 -->
  - Stream: 2
`,
			fileName:          "default-stream.md",
			streamFlag:        1,
			format:            "json",
			expectInOutput:    []string{`"id": "1"`, `"title": "Task without explicit stream"`},
			expectNotInOutput: []string{`"id": "2"`},
		},
		"stream filter with no matching tasks": {
			fileContent: `# Project Tasks

- [ ] 1. Task in stream 1 <!-- id:abc1234 -->
  - Stream: 1
`,
			fileName:       "no-matching-stream.md",
			streamFlag:     5,
			format:         "json",
			expectInOutput: []string{`"success": true`},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset flags before each test to ensure isolation
			streamFlag = 0
			claimFlag = ""
			phaseFlag = false

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
			args := []string{"next", tc.fileName, "--format", tc.format}
			if tc.streamFlag > 0 {
				args = append(args, "--stream", fmt.Sprintf("%d", tc.streamFlag))
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
					t.Errorf("expected '%s' in output, got: %s", expected, output)
				}
			}

			// Check strings NOT expected in output
			for _, notExpected := range tc.expectNotInOutput {
				if strings.Contains(output, notExpected) {
					t.Errorf("did NOT expect '%s' in output, got: %s", notExpected, output)
				}
			}
		})
	}
}

func TestNextCommandClaim(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-next-claim-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	tests := map[string]struct {
		fileContent       string
		fileName          string
		streamFlag        int
		claimFlag         string
		format            string
		expectInOutput    []string
		expectInFile      []string
		expectNotInOutput []string
	}{
		"claim without stream claims single task": {
			fileContent: `# Project Tasks

- [ ] 1. First ready task <!-- id:abc1234 -->
  - Stream: 1

- [ ] 2. Second ready task <!-- id:def5678 -->
  - Stream: 1
`,
			fileName:          "claim-single.md",
			claimFlag:         "agent-1",
			format:            "json",
			expectInOutput:    []string{`"id": "1"`, `"owner": "agent-1"`},
			expectInFile:      []string{"Owner: agent-1", "[-] 1."},
			expectNotInOutput: []string{`"id": "2"`},
		},
		"stream claim claims all ready tasks in stream": {
			fileContent: `# Project Tasks

- [ ] 1. Task in stream 1 <!-- id:abc1234 -->
  - Stream: 1

- [ ] 2. Task in stream 2 <!-- id:def5678 -->
  - Stream: 2

- [ ] 3. Another task in stream 2 <!-- id:ghi9012 -->
  - Stream: 2
`,
			fileName:          "claim-stream.md",
			streamFlag:        2,
			claimFlag:         "agent-2",
			format:            "json",
			expectInOutput:    []string{`"id": "2"`, `"id": "3"`, `"owner": "agent-2"`},
			expectInFile:      []string{"Owner: agent-2"},
			expectNotInOutput: []string{`"id": "1"`},
		},
		"claim sets status to in-progress and owner": {
			fileContent: `# Project Tasks

- [ ] 1. Ready task <!-- id:abc1234 -->
`,
			fileName:       "claim-status.md",
			claimFlag:      "test-agent",
			format:         "json",
			expectInOutput: []string{`"status": "In Progress"`, `"owner": "test-agent"`},
			expectInFile:   []string{"[-] 1.", "Owner: test-agent"},
		},
		"already-claimed tasks are skipped": {
			fileContent: `# Project Tasks

- [-] 1. Already claimed task <!-- id:abc1234 -->
  - Owner: other-agent

- [ ] 2. Available task <!-- id:def5678 -->
`,
			fileName:          "skip-claimed.md",
			claimFlag:         "new-agent",
			format:            "json",
			expectInOutput:    []string{`"id": "2"`, `"owner": "new-agent"`},
			expectNotInOutput: []string{`"id": "1"`},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset flags before each test to ensure isolation
			streamFlag = 0
			claimFlag = ""
			phaseFlag = false

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
			args := []string{"next", tc.fileName, "--format", tc.format}
			if tc.streamFlag > 0 {
				args = append(args, "--stream", fmt.Sprintf("%d", tc.streamFlag))
			}
			if tc.claimFlag != "" {
				args = append(args, "--claim", tc.claimFlag)
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
					t.Errorf("expected '%s' in output, got: %s", expected, output)
				}
			}

			// Check strings NOT expected in output
			for _, notExpected := range tc.expectNotInOutput {
				if strings.Contains(output, notExpected) {
					t.Errorf("did NOT expect '%s' in output, got: %s", notExpected, output)
				}
			}

			// Check file contents if expectInFile is specified
			if len(tc.expectInFile) > 0 {
				fileContent, err := os.ReadFile(tc.fileName)
				if err != nil {
					t.Fatalf("failed to read file after claim: %v", err)
				}
				for _, expected := range tc.expectInFile {
					if !strings.Contains(string(fileContent), expected) {
						t.Errorf("expected '%s' in file, got: %s", expected, string(fileContent))
					}
				}
			}
		})
	}
}

func TestNextCommandPhaseWithStreamInfo(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-next-phase-stream-test")
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
		streamFlag     int
		format         string
		expectInOutput []string
	}{
		"phase output includes stream and dependency info": {
			fileContent: `# Project Tasks

## Phase 1

- [ ] 1. First task <!-- id:abc1234 -->
  - Stream: 1

- [ ] 2. Second task <!-- id:def5678 -->
  - Stream: 2
  - Blocked-by: abc1234 (First task)
`,
			fileName:       "phase-stream-info.md",
			format:         "json",
			expectInOutput: []string{`"stream": 1`, `"stream": 2`, `"blockedBy"`},
		},
		"phase json includes streams summary": {
			fileContent: `# Project Tasks

## Phase 1

- [ ] 1. Ready task stream 1 <!-- id:abc1234 -->
  - Stream: 1

- [ ] 2. Ready task stream 2 <!-- id:def5678 -->
  - Stream: 2

- [ ] 3. Blocked task <!-- id:ghi9012 -->
  - Stream: 1
  - Blocked-by: def5678 (Ready task stream 2)
`,
			fileName:       "phase-streams-summary.md",
			format:         "json",
			expectInOutput: []string{`"streams_summary"`, `"ready"`, `"blocked"`, `"active"`, `"available"`},
		},
		"phase with stream filter": {
			fileContent: `# Project Tasks

## Phase 1

- [ ] 1. Task stream 1 <!-- id:abc1234 -->
  - Stream: 1

- [ ] 2. Task stream 2 <!-- id:def5678 -->
  - Stream: 2
`,
			fileName:       "phase-stream-filter.md",
			streamFlag:     1,
			format:         "json",
			expectInOutput: []string{`"id": "1"`, `"stream": 1`},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset flags before each test to ensure isolation
			streamFlag = 0
			claimFlag = ""
			phaseFlag = false

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
			args := []string{"next", tc.fileName, "--phase", "--format", tc.format}
			if tc.streamFlag > 0 {
				args = append(args, "--stream", fmt.Sprintf("%d", tc.streamFlag))
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
					t.Errorf("expected '%s' in output, got: %s", expected, output)
				}
			}

			// Validate JSON
			if tc.format == "json" {
				var jsonObj any
				if err := json.Unmarshal([]byte(output), &jsonObj); err != nil {
					t.Errorf("JSON format produced invalid JSON: %v\nOutput: %s", err, output)
				}
			}
		})
	}
}

func TestNextCommandPhaseTableShowsReadyAndBlockedStatus(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rune-next-phase-table-blocked-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	streamFlag = 0
	claimFlag = ""
	phaseFlag = false

	const taskFile = "phase-table-blocked.md"
	content := `# Project Tasks

## Phase 1
- [ ] 1. Ready task <!-- id:abc1234 -->

- [ ] 2. Blocked task <!-- id:def5678 -->
  - Blocked-by: abc1234 (Ready task)
`

	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"next", taskFile, "--phase", "--format", "table"})
	err = rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	rootCmd.SetArgs([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Pending (ready)") {
		t.Fatalf("expected ready indicator in output, got: %s", output)
	}
	if !strings.Contains(output, "Pending (blocked)") {
		t.Fatalf("expected blocked indicator in output, got: %s", output)
	}
}

func TestNextCommandPhaseMarkdownShowsBlockedByNotation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rune-next-phase-markdown-blocked-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	streamFlag = 0
	claimFlag = ""
	phaseFlag = false

	const taskFile = "phase-markdown-blocked.md"
	content := `# Project Tasks

## Phase 1
- [ ] 1. Ready task <!-- id:abc1234 -->

- [ ] 2. Blocked task <!-- id:def5678 -->
  - Blocked-by: abc1234 (Ready task)
`

	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"next", taskFile, "--phase", "--format", "markdown"})
	err = rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	rootCmd.SetArgs([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "- [ ] 1. Ready task") {
		t.Fatalf("expected ready task in output, got: %s", output)
	}
	if !strings.Contains(output, "- [ ] 2. Blocked task (blocked by: 1)") {
		t.Fatalf("expected blocked-by notation in output, got: %s", output)
	}
}

func TestNextCommandPhaseStreamSelectsFirstPhaseWithReadyStreamTasks(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rune-next-phase-stream-select-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	streamFlag = 0
	claimFlag = ""
	phaseFlag = false

	const taskFile = "phase-stream-selection.md"
	content := `# Project Tasks

## Phase A
- [ ] 1. External blocker <!-- id:aaa0001 -->
  - Stream: 1

## Phase B
- [ ] 2. Stream 2 blocked <!-- id:bbb0002 -->
  - Stream: 2
  - Blocked-by: aaa0001 (External blocker)

## Phase C
- [ ] 3. Stream 2 ready <!-- id:ccc0003 -->
  - Stream: 2
- [ ] 4. Stream 2 blocked in selected phase <!-- id:ddd0004 -->
  - Stream: 2
  - Blocked-by: ccc0003 (Stream 2 ready)
`

	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"next", taskFile, "--phase", "--stream", "2", "--format", "json"})
	err = rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	rootCmd.SetArgs([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, `"phase_name": "Phase C"`) {
		t.Fatalf("expected phase C in output, got: %s", output)
	}
	if !strings.Contains(output, `"id": "3"`) || !strings.Contains(output, `"id": "4"`) {
		t.Fatalf("expected stream 2 tasks from selected phase, got: %s", output)
	}
	if strings.Contains(output, `"id": "2"`) {
		t.Fatalf("did not expect blocked-only prior phase task in output, got: %s", output)
	}
}

func TestNextCommandPhaseStreamClaimClaimsOnlyReadyTasksFromSelectedPhase(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rune-next-phase-stream-claim-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	streamFlag = 0
	claimFlag = ""
	phaseFlag = false

	const taskFile = "phase-stream-claim.md"
	content := `# Project Tasks

## Phase A
- [ ] 1. External blocker <!-- id:aaa0001 -->
  - Stream: 1

## Phase B
- [ ] 2. Stream 2 blocked <!-- id:bbb0002 -->
  - Stream: 2
  - Blocked-by: aaa0001 (External blocker)

## Phase C
- [ ] 3. Stream 2 ready <!-- id:ccc0003 -->
  - Stream: 2
- [ ] 4. Stream 2 blocked in selected phase <!-- id:ddd0004 -->
  - Stream: 2
  - Blocked-by: ccc0003 (Stream 2 ready)

## Phase D
- [ ] 5. Later phase stream 2 ready <!-- id:eee0005 -->
  - Stream: 2
`

	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"next", taskFile, "--phase", "--stream", "2", "--claim", "agent-1", "--format", "json"})
	err = rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	rootCmd.SetArgs([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, `"count": 1`) || !strings.Contains(output, `"id": "3"`) {
		t.Fatalf("expected only task 3 claimed, got: %s", output)
	}
	if strings.Contains(output, `"id": "4"`) || strings.Contains(output, `"id": "5"`) {
		t.Fatalf("did not expect blocked or later-phase tasks to be claimed, got: %s", output)
	}

	fileContent, err := os.ReadFile(taskFile)
	if err != nil {
		t.Fatalf("failed to read task file after claim: %v", err)
	}

	updated := string(fileContent)
	if !strings.Contains(updated, "- [-] 3. Stream 2 ready") || !strings.Contains(updated, "Owner: agent-1") {
		t.Fatalf("expected task 3 to be in-progress and owned after claim, got: %s", updated)
	}
	if strings.Contains(updated, "- [-] 5. Later phase stream 2 ready") {
		t.Fatalf("did not expect later phase task to be claimed, got: %s", updated)
	}
}

func TestNextCommandNoReadyTasks(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-next-no-ready-test")
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
		claimFlag      string
		streamFlag     int
		expectInOutput []string
		expectSuccess  bool
	}{
		"claim with no ready tasks returns success": {
			fileContent: `# Project Tasks

- [x] 1. Completed task <!-- id:abc1234 -->
`,
			fileName:       "no-ready-for-claim.md",
			claimFlag:      "agent-1",
			expectInOutput: []string{`"success": true`},
			expectSuccess:  true,
		},
		"stream claim with no ready tasks in stream": {
			fileContent: `# Project Tasks

- [ ] 1. Task in stream 1 <!-- id:abc1234 -->
  - Stream: 1
`,
			fileName:       "no-ready-in-stream.md",
			claimFlag:      "agent-1",
			streamFlag:     2,
			expectInOutput: []string{`"success": true`, `"claimed": []`},
			expectSuccess:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset flags before each test to ensure isolation
			streamFlag = 0
			claimFlag = ""
			phaseFlag = false

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
			args := []string{"next", tc.fileName, "--format", "json"}
			if tc.claimFlag != "" {
				args = append(args, "--claim", tc.claimFlag)
			}
			if tc.streamFlag > 0 {
				args = append(args, "--stream", fmt.Sprintf("%d", tc.streamFlag))
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

			// Check error/success expectation
			if tc.expectSuccess && err != nil {
				t.Errorf("expected success but got error: %v", err)
				return
			}

			// Check expected strings in output
			for _, expected := range tc.expectInOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("expected '%s' in output, got: %s", expected, output)
				}
			}
		})
	}
}

func TestNextCommandOneFlag(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rune-next-one-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	content := `# Project Tasks

- [ ] 1. Parent task
  - [ ] 1.1. First child
    - [ ] 1.1.1. First grandchild
    - [ ] 1.1.2. Second grandchild
  - [ ] 1.2. Second child
    - [ ] 1.2.1. Third grandchild
`

	tests := map[string]struct {
		format            string
		expectInOutput    []string
		expectNotInOutput []string
	}{
		"table format with --one shows single path": {
			format:            "table",
			expectInOutput:    []string{"1", "Parent task", "1.1", "First child", "1.1.1", "First grandchild"},
			expectNotInOutput: []string{"1.1.2", "Second grandchild", "1.2", "Second child", "1.2.1", "Third grandchild"},
		},
		"markdown format with --one shows single path": {
			format:            "markdown",
			expectInOutput:    []string{"- [ ] 1. Parent task", "- [ ] 1.1. First child", "- [ ] 1.1.1. First grandchild"},
			expectNotInOutput: []string{"1.1.2. Second grandchild", "1.2. Second child", "1.2.1. Third grandchild"},
		},
		"json format with --one shows single path": {
			format:            "json",
			expectInOutput:    []string{`"id": "1"`, `"title": "Parent task"`, `"id": "1.1"`, `"title": "First child"`, `"id": "1.1.1"`, `"title": "First grandchild"`},
			expectNotInOutput: []string{`"id": "1.1.2"`, `"id": "1.2"`, `"id": "1.2.1"`},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset all flags before each test
			oneFlag = false
			streamFlag = 0
			claimFlag = ""
			phaseFlag = false

			testFile := fmt.Sprintf("test-one-%s.md", tc.format)
			if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			rootCmd.SetArgs([]string{"next", testFile, "--one", "--format", tc.format})
			err := rootCmd.Execute()

			w.Close()
			os.Stdout = oldStdout
			buf.ReadFrom(r)
			output := buf.String()

			rootCmd.SetArgs([]string{})

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			for _, expected := range tc.expectInOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("expected '%s' in output, got: %s", expected, output)
				}
			}

			for _, notExpected := range tc.expectNotInOutput {
				if strings.Contains(output, notExpected) {
					t.Errorf("did NOT expect '%s' in output, got: %s", notExpected, output)
				}
			}

			if tc.format == "json" {
				var jsonObj any
				if err := json.Unmarshal([]byte(output), &jsonObj); err != nil {
					t.Errorf("JSON format produced invalid JSON: %v\nOutput: %s", err, output)
				}
			}
		})
	}
}

func TestNextCommandOneFlagValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rune-next-one-validation-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	content := `# Project Tasks

- [ ] 1. Parent task
  - [ ] 1.1. First child
`

	testFile := "test-validation.md"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	tests := map[string]struct {
		args        []string
		expectError bool
	}{
		"--one with --phase should error": {
			args:        []string{"next", testFile, "--one", "--phase"},
			expectError: true,
		},
		"--one with --stream should error": {
			args:        []string{"next", testFile, "--one", "--stream", "2"},
			expectError: true,
		},
		"--one with --claim should work": {
			args:        []string{"next", testFile, "--one", "--claim", "agent-1"},
			expectError: false,
		},
		"--one alone should work": {
			args:        []string{"next", testFile, "--one"},
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset flags
			oneFlag = false
			streamFlag = 0
			claimFlag = ""
			phaseFlag = false

			rootCmd.SetArgs(tc.args)
			err := rootCmd.Execute()
			rootCmd.SetArgs([]string{})

			if tc.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNextCommandOneWithClaim(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rune-next-one-claim-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	content := `# Project Tasks

- [ ] 1. Parent task <!-- id:abc1234 -->
  - [ ] 1.1. First child <!-- id:def5678 -->
    - [ ] 1.1.1. First grandchild <!-- id:ghi9012 -->
    - [ ] 1.1.2. Second grandchild <!-- id:jkl3456 -->
  - [ ] 1.2. Second child <!-- id:mno7890 -->
    - [ ] 1.2.1. Third grandchild <!-- id:pqr1234 -->
`

	testFile := "test-one-claim.md"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Reset flags
	oneFlag = false
	streamFlag = 0
	claimFlag = ""
	phaseFlag = false

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"next", testFile, "--one", "--claim", "test-agent", "--format", "json"})
	err = rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	rootCmd.SetArgs([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify JSON output contains only the first incomplete task
	if !strings.Contains(output, `"id": "1.1.1"`) {
		t.Errorf("expected first grandchild (1.1.1) in output, got: %s", output)
	}

	// Verify siblings are NOT in output
	if strings.Contains(output, `"id": "1.1.2"`) {
		t.Errorf("did not expect second grandchild (1.1.2) in output, got: %s", output)
	}
	if strings.Contains(output, `"id": "1.2"`) {
		t.Errorf("did not expect second child (1.2) in output, got: %s", output)
	}

	// Verify the task was claimed
	if !strings.Contains(output, `"owner": "test-agent"`) {
		t.Errorf("expected owner to be test-agent, got: %s", output)
	}
	if !strings.Contains(output, `"status": "In Progress"`) {
		t.Errorf("expected status to be In Progress, got: %s", output)
	}

	// Verify file was updated with claim
	fileContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file after claim: %v", err)
	}

	fileStr := string(fileContent)
	if !strings.Contains(fileStr, "Owner: test-agent") {
		t.Errorf("expected Owner: test-agent in file, got: %s", fileStr)
	}
	if !strings.Contains(fileStr, "[-] 1.1.1. First grandchild") {
		t.Errorf("expected task 1.1.1 to be in-progress, got: %s", fileStr)
	}

	// Verify only one task was claimed (not siblings)
	if strings.Contains(fileStr, "[-] 1.1.2") {
		t.Errorf("did not expect task 1.1.2 to be claimed, got: %s", fileStr)
	}
	if strings.Contains(fileStr, "[-] 1.2") {
		t.Errorf("did not expect task 1.2 to be claimed, got: %s", fileStr)
	}
}
