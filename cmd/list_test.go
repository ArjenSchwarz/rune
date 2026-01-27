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
	tl.UpdateTask("2", "", []string{"Detail 1", "Detail 2"}, []string{"ref1.md", "ref2.md"}, nil)

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
			filename: "../../../nonexistent-path-for-security-test.md",
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

// Tests for list command enhancements (task dependencies and streams)

func TestListStreamDisplayWhenNonDefaultStreamsExist(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-stream-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	tests := map[string]struct {
		setupTasks    func(*task.TaskList) // Setup tasks with different streams
		expectStreams bool                 // Whether streams column should be in table data
	}{
		"only default stream": {
			setupTasks: func(tl *task.TaskList) {
				// Add tasks without explicit stream (default to 1)
				tl.AddTaskWithOptions("", "Task 1", task.AddOptions{})
				tl.AddTaskWithOptions("", "Task 2", task.AddOptions{})
			},
			expectStreams: false, // Stream column should be omitted when all tasks are stream 1
		},
		"non-default streams exist": {
			setupTasks: func(tl *task.TaskList) {
				// Add tasks with different streams
				tl.AddTaskWithOptions("", "Task 1", task.AddOptions{Stream: 1})
				tl.AddTaskWithOptions("", "Task 2", task.AddOptions{Stream: 2})
				tl.AddTaskWithOptions("", "Task 3", task.AddOptions{Stream: 3})
			},
			expectStreams: true, // Stream column should appear
		},
		"mix of default and explicit stream 1": {
			setupTasks: func(tl *task.TaskList) {
				// All tasks are effectively stream 1
				tl.AddTaskWithOptions("", "Task 1", task.AddOptions{})          // Default (becomes 1)
				tl.AddTaskWithOptions("", "Task 2", task.AddOptions{Stream: 1}) // Explicit 1
			},
			expectStreams: false, // All are stream 1, so omit column
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl := task.NewTaskList("Stream Test")
			tc.setupTasks(tl)

			testFile := "stream-test.md"
			if err := tl.WriteFile(testFile); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Parse and check if streams other than 1 exist
			parsedList, err := task.ParseFile(testFile)
			if err != nil {
				t.Fatalf("failed to parse file: %v", err)
			}

			hasNonDefaultStream := hasNonDefaultStreams(parsedList.Tasks)
			if hasNonDefaultStream != tc.expectStreams {
				t.Errorf("expected hasNonDefaultStream=%v, got %v", tc.expectStreams, hasNonDefaultStream)
			}
		})
	}
}

// hasNonDefaultStreams checks if any task has a stream other than 1
func hasNonDefaultStreams(tasks []task.Task) bool {
	for _, t := range tasks {
		if task.GetEffectiveStream(&t) != 1 {
			return true
		}
		if hasNonDefaultStreams(t.Children) {
			return true
		}
	}
	return false
}

func TestListBlockedByDisplayAsHierarchicalIDs(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-blockedby-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Create task file with dependencies
	tl := task.NewTaskList("Dependency Test")

	// Add first task (will be a blocker)
	id1, err := tl.AddTaskWithOptions("", "First task", task.AddOptions{})
	if err != nil {
		t.Fatalf("failed to add task 1: %v", err)
	}

	// Add second task (will be a blocker)
	id2, err := tl.AddTaskWithOptions("", "Second task", task.AddOptions{})
	if err != nil {
		t.Fatalf("failed to add task 2: %v", err)
	}

	// Add third task that depends on first two
	_, err = tl.AddTaskWithOptions("", "Third task blocked by 1 and 2", task.AddOptions{
		BlockedBy: []string{id1, id2},
	})
	if err != nil {
		t.Fatalf("failed to add task 3: %v", err)
	}

	testFile := "blockedby-test.md"
	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Parse and verify blocked-by can be translated to hierarchical IDs
	parsedList, err := task.ParseFile(testFile)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	// Build dependency index
	index := task.BuildDependencyIndex(parsedList.Tasks)

	// Find task 3 and verify its blocked-by references
	task3 := parsedList.FindTask("3")
	if task3 == nil {
		t.Fatal("task 3 not found")
	}

	if len(task3.BlockedBy) != 2 {
		t.Errorf("expected 2 blocked-by references, got %d", len(task3.BlockedBy))
	}

	// Translate stable IDs to hierarchical IDs
	hierarchicalIDs := index.TranslateToHierarchical(task3.BlockedBy)
	if len(hierarchicalIDs) != 2 {
		t.Errorf("expected 2 hierarchical IDs, got %d", len(hierarchicalIDs))
	}

	// Verify the translated IDs are "1" and "2"
	expectedIDs := map[string]bool{"1": true, "2": true}
	for _, hid := range hierarchicalIDs {
		if !expectedIDs[hid] {
			t.Errorf("unexpected hierarchical ID %q in blocked-by", hid)
		}
		delete(expectedIDs, hid)
	}
	if len(expectedIDs) > 0 {
		t.Errorf("missing expected hierarchical IDs: %v", expectedIDs)
	}
}

func TestListFilterByStream(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-stream-filter-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Create task file with different streams
	tl := task.NewTaskList("Stream Filter Test")
	tl.AddTaskWithOptions("", "Stream 1 Task A", task.AddOptions{Stream: 1})
	tl.AddTaskWithOptions("", "Stream 1 Task B", task.AddOptions{Stream: 1})
	tl.AddTaskWithOptions("", "Stream 2 Task A", task.AddOptions{Stream: 2})
	tl.AddTaskWithOptions("", "Stream 2 Task B", task.AddOptions{Stream: 2})
	tl.AddTaskWithOptions("", "Stream 3 Task", task.AddOptions{Stream: 3})
	tl.AddTaskWithOptions("", "Default Stream Task", task.AddOptions{}) // No stream = default 1

	testFile := "stream-filter-test.md"
	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Parse and filter by stream
	parsedList, err := task.ParseFile(testFile)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	tests := map[string]struct {
		streamFilter  int
		expectedCount int
	}{
		"filter stream 1": {
			streamFilter:  1,
			expectedCount: 3, // Stream 1 Task A, B, and Default Stream Task
		},
		"filter stream 2": {
			streamFilter:  2,
			expectedCount: 2, // Stream 2 Task A, B
		},
		"filter stream 3": {
			streamFilter:  3,
			expectedCount: 1, // Stream 3 Task
		},
		"filter nonexistent stream": {
			streamFilter:  99,
			expectedCount: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			filtered := testFilterTasksByStream(parsedList.Tasks, tc.streamFilter)
			if len(filtered) != tc.expectedCount {
				t.Errorf("expected %d tasks for stream %d, got %d",
					tc.expectedCount, tc.streamFilter, len(filtered))
			}
		})
	}
}

// testFilterTasksByStream filters tasks by stream (recursive) - for testing
func testFilterTasksByStream(tasks []task.Task, stream int) []task.Task {
	var result []task.Task
	for _, t := range tasks {
		if task.GetEffectiveStream(&t) == stream {
			result = append(result, t)
		}
		// Also check children
		childFiltered := testFilterTasksByStream(t.Children, stream)
		result = append(result, childFiltered...)
	}
	return result
}

func TestListFilterByOwner(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-owner-filter-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Create task file with different owners
	tl := task.NewTaskList("Owner Filter Test")
	tl.AddTaskWithOptions("", "Alice's Task 1", task.AddOptions{Owner: "alice"})
	tl.AddTaskWithOptions("", "Alice's Task 2", task.AddOptions{Owner: "alice"})
	tl.AddTaskWithOptions("", "Bob's Task", task.AddOptions{Owner: "bob"})
	tl.AddTaskWithOptions("", "Unowned Task", task.AddOptions{}) // No owner

	testFile := "owner-filter-test.md"
	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Parse and filter by owner
	parsedList, err := task.ParseFile(testFile)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	tests := map[string]struct {
		ownerFilter   string
		expectedCount int
	}{
		"filter alice": {
			ownerFilter:   "alice",
			expectedCount: 2,
		},
		"filter bob": {
			ownerFilter:   "bob",
			expectedCount: 1,
		},
		"filter nonexistent owner": {
			ownerFilter:   "charlie",
			expectedCount: 0,
		},
		"filter empty owner": {
			ownerFilter:   "",
			expectedCount: 1, // Only unowned task
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			filtered := testFilterTasksByOwner(parsedList.Tasks, tc.ownerFilter)
			if len(filtered) != tc.expectedCount {
				t.Errorf("expected %d tasks for owner %q, got %d",
					tc.expectedCount, tc.ownerFilter, len(filtered))
			}
		})
	}
}

// testFilterTasksByOwner filters tasks by owner (recursive) - for testing
func testFilterTasksByOwner(tasks []task.Task, owner string) []task.Task {
	var result []task.Task
	for _, t := range tasks {
		if t.Owner == owner {
			result = append(result, t)
		}
		// Also check children
		childFiltered := testFilterTasksByOwner(t.Children, owner)
		result = append(result, childFiltered...)
	}
	return result
}

func TestListJSONOutputIncludesNewFields(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-json-fields-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Create task file with all new fields populated
	tl := task.NewTaskList("JSON Fields Test")

	// Add first task (blocker)
	id1, err := tl.AddTaskWithOptions("", "Blocker task", task.AddOptions{
		Stream: 1,
		Owner:  "agent-1",
	})
	if err != nil {
		t.Fatalf("failed to add blocker task: %v", err)
	}

	// Add second task with dependency
	_, err = tl.AddTaskWithOptions("", "Dependent task", task.AddOptions{
		Stream:    2,
		Owner:     "agent-2",
		BlockedBy: []string{id1},
	})
	if err != nil {
		t.Fatalf("failed to add dependent task: %v", err)
	}

	testFile := "json-fields-test.md"
	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Parse and render as JSON
	parsedList, err := task.ParseFile(testFile)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	jsonOutput, err := task.RenderJSON(parsedList)
	if err != nil {
		t.Fatalf("failed to render JSON: %v", err)
	}

	jsonStr := string(jsonOutput)

	// Verify JSON includes blockedBy field
	if !strings.Contains(jsonStr, "blockedBy") {
		t.Errorf("JSON output missing 'blockedBy' field")
	}

	// Verify JSON includes stream field
	if !strings.Contains(jsonStr, "stream") {
		t.Errorf("JSON output missing 'stream' field")
	}

	// Verify JSON includes owner field
	if !strings.Contains(jsonStr, "owner") {
		t.Errorf("JSON output missing 'owner' field")
	}

	// Verify specific values are present
	if !strings.Contains(jsonStr, "agent-1") {
		t.Errorf("JSON output missing owner value 'agent-1'")
	}

	if !strings.Contains(jsonStr, "agent-2") {
		t.Errorf("JSON output missing owner value 'agent-2'")
	}

	// Verify stableID is NOT in JSON output (per requirement 1.5)
	if strings.Contains(jsonStr, "stableID") || strings.Contains(jsonStr, "StableID") {
		t.Errorf("JSON output should NOT include stableID field")
	}
}
