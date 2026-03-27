package cmd

import (
	"encoding/json"
	"maps"
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

// TestFilterTasksRecursiveExcludesNonMatchingParents verifies that
// filterTasksRecursive drops parents that don't match filters, even when
// their children do match. This aligns JSON output with table output (T-436).
func TestFilterTasksRecursiveExcludesNonMatchingParents(t *testing.T) {
	tests := map[string]struct {
		setupTasks      func(*task.TaskList)
		opts            listFilterOptions
		expectedTaskIDs []string // IDs that should appear in filtered output
		excludedTaskIDs []string // IDs that must NOT appear
		description     string
	}{
		"status filter excludes non-matching parent with matching child": {
			setupTasks: func(tl *task.TaskList) {
				tl.AddTask("", "Parent task", "")      // task 1, pending
				tl.AddTask("1", "Child task", "")      // task 1.1, pending
				tl.UpdateStatus("1.1", task.Completed) // child is completed
				tl.AddTask("", "Other task", "")       // task 2, pending
				tl.UpdateStatus("2", task.Completed)   // also completed
			},
			opts: listFilterOptions{
				statusFilter: "completed",
			},
			expectedTaskIDs: []string{"1.1", "2"},
			excludedTaskIDs: []string{"1"},
			description:     "Parent task 1 is pending, should not appear even though child 1.1 is completed",
		},
		"stream filter excludes non-matching parent with matching child": {
			setupTasks: func(tl *task.TaskList) {
				tl.AddTaskWithOptions("", "Stream 1 parent", task.AddOptions{Stream: 1})
				tl.AddTaskWithOptions("1", "Stream 2 child", task.AddOptions{Stream: 2})
				tl.AddTaskWithOptions("", "Stream 2 task", task.AddOptions{Stream: 2})
			},
			opts: listFilterOptions{
				streamFilter: 2,
			},
			expectedTaskIDs: []string{"1.1", "2"},
			excludedTaskIDs: []string{"1"},
			description:     "Parent task 1 is stream 1, should not appear even though child 1.1 is stream 2",
		},
		"owner filter excludes non-matching parent with matching child": {
			setupTasks: func(tl *task.TaskList) {
				tl.AddTaskWithOptions("", "Alice's parent", task.AddOptions{Owner: "alice"})
				tl.AddTaskWithOptions("1", "Bob's child", task.AddOptions{Owner: "bob"})
				tl.AddTaskWithOptions("", "Bob's task", task.AddOptions{Owner: "bob"})
			},
			opts: listFilterOptions{
				ownerFilter: "bob",
				ownerSet:    true,
			},
			expectedTaskIDs: []string{"1.1", "2"},
			excludedTaskIDs: []string{"1"},
			description:     "Parent task 1 is owned by alice, should not appear even though child 1.1 is owned by bob",
		},
		"deep nesting promotes grandchild through non-matching ancestors": {
			setupTasks: func(tl *task.TaskList) {
				tl.AddTask("", "Grandparent", "")        // task 1, pending
				tl.AddTask("1", "Parent", "")            // task 1.1, pending
				tl.AddTask("1.1", "Grandchild", "")      // task 1.1.1, pending
				tl.UpdateStatus("1.1.1", task.Completed) // only grandchild is completed
				tl.AddTask("", "Other completed", "")    // task 2, pending
				tl.UpdateStatus("2", task.Completed)     // also completed
			},
			opts: listFilterOptions{
				statusFilter: "completed",
			},
			expectedTaskIDs: []string{"1.1.1", "2"},
			excludedTaskIDs: []string{"1", "1.1"},
			description:     "Grandchild 1.1.1 should be promoted through two non-matching ancestors",
		},
		"matching parent with matching child includes both": {
			setupTasks: func(tl *task.TaskList) {
				tl.AddTask("", "Parent task", "")
				tl.AddTask("1", "Child task", "")
				tl.UpdateStatus("1", task.Completed)
				tl.UpdateStatus("1.1", task.Completed)
			},
			opts: listFilterOptions{
				statusFilter: "completed",
			},
			expectedTaskIDs: []string{"1", "1.1"},
			excludedTaskIDs: []string{},
			description:     "Both parent and child are completed, both should appear",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl := task.NewTaskList("Filter Test")
			tc.setupTasks(tl)

			filtered := filterTasksRecursive(tl.Tasks, tc.opts)

			// Collect all task IDs from the filtered result (recursively)
			gotIDs := collectTaskIDs(filtered)

			// Check expected tasks are present
			for _, expectedID := range tc.expectedTaskIDs {
				if !gotIDs[expectedID] {
					t.Errorf("%s: expected task %s to be in filtered output, but it was not. Got IDs: %v",
						tc.description, expectedID, gotIDs)
				}
			}

			// Check excluded tasks are absent
			for _, excludedID := range tc.excludedTaskIDs {
				if gotIDs[excludedID] {
					t.Errorf("%s: task %s should NOT be in filtered output, but it was. Got IDs: %v",
						tc.description, excludedID, gotIDs)
				}
			}
		})
	}
}

// collectTaskIDs recursively collects all task IDs from a task tree into a set.
func collectTaskIDs(tasks []task.Task) map[string]bool {
	ids := make(map[string]bool)
	for _, t := range tasks {
		ids[t.ID] = true
		maps.Copy(ids, collectTaskIDs(t.Children))
	}
	return ids
}

// TestFilteredJSONOutputPreservesPhaseBoundaries verifies that filtered JSON
// output retains correct phase labels even when the boundary task (the task
// referenced by PhaseMarker.AfterTaskID) is filtered out. Regression test
// for T-537.
func TestFilteredJSONOutputPreservesPhaseBoundaries(t *testing.T) {
	tests := map[string]struct {
		tasks       []task.Task
		phases      []task.PhaseMarker
		opts        listFilterOptions
		wantPhases  map[string]string // taskID -> expected phase
		description string
	}{
		"boundary task filtered out by status": {
			tasks: []task.Task{
				{ID: "1", Title: "Design docs", Status: task.Completed},
				{ID: "2", Title: "Prototype", Status: task.Completed},
				{ID: "3", Title: "Implement", Status: task.InProgress},
				{ID: "4", Title: "Deploy", Status: task.Pending},
			},
			phases: []task.PhaseMarker{
				{Name: "Design", AfterTaskID: ""},
				{Name: "Build", AfterTaskID: "2"},
			},
			opts: listFilterOptions{statusFilter: "pending"},
			wantPhases: map[string]string{
				"4": "Build",
			},
			description: "Task 2 is the Build boundary but is completed; filtered to pending only",
		},
		"boundary task filtered out by stream": {
			tasks: []task.Task{
				{ID: "1", Title: "Planning", Status: task.Pending, Stream: 1},
				{ID: "2", Title: "Boundary task", Status: task.Pending, Stream: 1},
				{ID: "3", Title: "Stream 2 work", Status: task.Pending, Stream: 2},
				{ID: "4", Title: "More stream 2", Status: task.Pending, Stream: 2},
			},
			phases: []task.PhaseMarker{
				{Name: "Phase A", AfterTaskID: ""},
				{Name: "Phase B", AfterTaskID: "2"},
			},
			opts: listFilterOptions{streamFilter: 2},
			wantPhases: map[string]string{
				"3": "Phase B",
				"4": "Phase B",
			},
			description: "Task 2 is the Phase B boundary but is stream 1; filtering to stream 2",
		},
		"boundary task filtered out by owner": {
			tasks: []task.Task{
				{ID: "1", Title: "Alice task", Status: task.Pending, Owner: "alice"},
				{ID: "2", Title: "Alice boundary", Status: task.Pending, Owner: "alice"},
				{ID: "3", Title: "Bob's work", Status: task.Pending, Owner: "bob"},
			},
			phases: []task.PhaseMarker{
				{Name: "Setup", AfterTaskID: ""},
				{Name: "Execution", AfterTaskID: "2"},
			},
			opts: listFilterOptions{ownerFilter: "bob", ownerSet: true},
			wantPhases: map[string]string{
				"3": "Execution",
			},
			description: "Task 2 is the Execution boundary but owned by alice; filtering to bob",
		},
		"all boundary tasks present (no filtering issue)": {
			tasks: []task.Task{
				{ID: "1", Title: "Task one", Status: task.Pending},
				{ID: "2", Title: "Task two", Status: task.Pending},
				{ID: "3", Title: "Task three", Status: task.Pending},
			},
			phases: []task.PhaseMarker{
				{Name: "Alpha", AfterTaskID: ""},
				{Name: "Beta", AfterTaskID: "1"},
			},
			opts: listFilterOptions{}, // no filters
			wantPhases: map[string]string{
				"1": "Alpha",
				"2": "Beta",
				"3": "Beta",
			},
			description: "No filters applied — baseline correctness check",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			originalTL := &task.TaskList{
				Title: "Phase Boundary Test",
				Tasks: tc.tasks,
			}

			// Filter tasks the same way outputJSONWithFilters does
			filteredTasks := filterTasksRecursive(originalTL.Tasks, tc.opts)
			filteredList := &task.TaskList{
				Title: originalTL.Title,
				Tasks: filteredTasks,
			}

			jsonData := task.RenderJSONWithPhases(filteredList, tc.phases, originalTL)

			var result struct {
				Tasks []struct {
					ID    string `json:"ID"`
					Phase string `json:"Phase"`
				} `json:"Tasks"`
			}
			if err := json.Unmarshal(jsonData, &result); err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			for _, got := range result.Tasks {
				wantPhase, ok := tc.wantPhases[got.ID]
				if !ok {
					continue // task not in our expectations
				}
				if got.Phase != wantPhase {
					t.Errorf("%s: task %s Phase = %q, want %q",
						tc.description, got.ID, got.Phase, wantPhase)
				}
			}
		})
	}
}

// TestFilterTasksRecursiveMatchesTableOutput verifies that the JSON filter path
// and the table filter path produce the same set of task IDs for various filter
// combinations. This is the core assertion for T-436.
func TestFilterTasksRecursiveMatchesTableOutput(t *testing.T) {
	tests := map[string]struct {
		setupTasks func(*task.TaskList)
		opts       listFilterOptions
	}{
		"status filter pending": {
			setupTasks: func(tl *task.TaskList) {
				tl.AddTask("", "Parent", "")
				tl.AddTask("1", "Child completed", "")
				tl.UpdateStatus("1.1", task.Completed)
				tl.AddTask("", "Other pending", "")
			},
			opts: listFilterOptions{statusFilter: "pending"},
		},
		"status filter completed with nested tasks": {
			setupTasks: func(tl *task.TaskList) {
				tl.AddTask("", "Pending parent", "")
				tl.AddTask("1", "Completed child", "")
				tl.UpdateStatus("1.1", task.Completed)
				tl.AddTask("", "Completed root", "")
				tl.UpdateStatus("2", task.Completed)
			},
			opts: listFilterOptions{statusFilter: "completed"},
		},
		"stream filter with mixed hierarchy": {
			setupTasks: func(tl *task.TaskList) {
				tl.AddTaskWithOptions("", "Stream 1 parent", task.AddOptions{Stream: 1})
				tl.AddTaskWithOptions("1", "Stream 2 child", task.AddOptions{Stream: 2})
				tl.AddTaskWithOptions("", "Stream 2 root", task.AddOptions{Stream: 2})
			},
			opts: listFilterOptions{streamFilter: 2},
		},
		"owner filter with mixed hierarchy": {
			setupTasks: func(tl *task.TaskList) {
				tl.AddTaskWithOptions("", "Alice parent", task.AddOptions{Owner: "alice"})
				tl.AddTaskWithOptions("1", "Bob child", task.AddOptions{Owner: "bob"})
				tl.AddTaskWithOptions("", "Bob root", task.AddOptions{Owner: "bob"})
			},
			opts: listFilterOptions{ownerFilter: "bob", ownerSet: true},
		},
		"deep nesting with grandchild promotion": {
			setupTasks: func(tl *task.TaskList) {
				tl.AddTask("", "Grandparent", "")
				tl.AddTask("1", "Parent", "")
				tl.AddTask("1.1", "Grandchild", "")
				tl.UpdateStatus("1.1.1", task.Completed)
				tl.AddTask("", "Root completed", "")
				tl.UpdateStatus("2", task.Completed)
			},
			opts: listFilterOptions{statusFilter: "completed"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl := task.NewTaskList("Consistency Test")
			tc.setupTasks(tl)

			depIndex := task.BuildDependencyIndex(tl.Tasks)
			hasStreams := detectNonDefaultStreams(tl.Tasks)

			// Table path: flattenTasksWithFilters
			tableData := flattenTasksWithFilters(tl, nil, depIndex, hasStreams, tc.opts)
			tableIDs := make(map[string]bool)
			for _, record := range tableData {
				tableIDs[record["ID"].(string)] = true
			}

			// JSON path: filterTasksRecursive
			jsonFiltered := filterTasksRecursive(tl.Tasks, tc.opts)
			jsonIDs := collectTaskIDs(jsonFiltered)

			// They must produce the same set of task IDs
			for id := range tableIDs {
				if !jsonIDs[id] {
					t.Errorf("task %s appears in table output but not JSON output", id)
				}
			}
			for id := range jsonIDs {
				if !tableIDs[id] {
					t.Errorf("task %s appears in JSON output but not table output", id)
				}
			}
		})
	}
}

// TestMarkdownOutputRespectsFilters verifies that markdown output applies the
// same filter semantics as table and JSON output (T-579). Before the fix,
// outputMarkdownWithPhases ignored filterOpts and rendered every task.
func TestMarkdownOutputRespectsFilters(t *testing.T) {
	tests := map[string]struct {
		tasks      []task.Task
		phases     []task.PhaseMarker
		opts       listFilterOptions
		wantTitles []string // titles that MUST appear
		dropTitles []string // titles that MUST NOT appear
	}{
		"status filter excludes completed tasks": {
			tasks: []task.Task{
				{ID: "1", Title: "Keep me", Status: task.Pending},
				{ID: "2", Title: "Remove me", Status: task.Completed},
			},
			opts:       listFilterOptions{statusFilter: "pending"},
			wantTitles: []string{"Keep me"},
			dropTitles: []string{"Remove me"},
		},
		"stream filter excludes other streams": {
			tasks: []task.Task{
				{ID: "1", Title: "Stream one", Status: task.Pending, Stream: 1},
				{ID: "2", Title: "Stream two", Status: task.Pending, Stream: 2},
			},
			opts:       listFilterOptions{streamFilter: 2},
			wantTitles: []string{"Stream two"},
			dropTitles: []string{"Stream one"},
		},
		"owner filter excludes other owners": {
			tasks: []task.Task{
				{ID: "1", Title: "Alice task", Status: task.Pending, Owner: "alice"},
				{ID: "2", Title: "Bob task", Status: task.Pending, Owner: "bob"},
			},
			opts:       listFilterOptions{ownerFilter: "bob", ownerSet: true},
			wantTitles: []string{"Bob task"},
			dropTitles: []string{"Alice task"},
		},
		"no filters shows all tasks": {
			tasks: []task.Task{
				{ID: "1", Title: "First", Status: task.Pending},
				{ID: "2", Title: "Second", Status: task.Completed},
			},
			opts:       listFilterOptions{},
			wantTitles: []string{"First", "Second"},
			dropTitles: nil,
		},
		"status filter with phases preserves phase headers": {
			tasks: []task.Task{
				{ID: "1", Title: "Design", Status: task.Completed},
				{ID: "2", Title: "Build", Status: task.Pending},
				{ID: "3", Title: "Deploy", Status: task.Pending},
			},
			phases: []task.PhaseMarker{
				{Name: "Planning", AfterTaskID: ""},
				{Name: "Execution", AfterTaskID: "1"},
			},
			opts:       listFilterOptions{statusFilter: "pending"},
			wantTitles: []string{"Build", "Deploy"},
			dropTitles: []string{"Design"},
		},
		"combined status and stream filters": {
			tasks: []task.Task{
				{ID: "1", Title: "Pending S1", Status: task.Pending, Stream: 1},
				{ID: "2", Title: "Pending S2", Status: task.Pending, Stream: 2},
				{ID: "3", Title: "Done S2", Status: task.Completed, Stream: 2},
			},
			opts:       listFilterOptions{statusFilter: "pending", streamFilter: 2},
			wantTitles: []string{"Pending S2"},
			dropTitles: []string{"Pending S1", "Done S2"},
		},
		"all tasks filtered out produces valid markdown": {
			tasks: []task.Task{
				{ID: "1", Title: "Done one", Status: task.Completed},
				{ID: "2", Title: "Done two", Status: task.Completed},
			},
			opts:       listFilterOptions{statusFilter: "pending"},
			wantTitles: nil,
			dropTitles: []string{"Done one", "Done two"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl := &task.TaskList{
				Title: "Markdown Filter Test",
				Tasks: tc.tasks,
			}

			md := outputMarkdownWithFilters(tl, tc.phases, tc.opts)

			for _, want := range tc.wantTitles {
				if !strings.Contains(md, want) {
					t.Errorf("expected markdown to contain %q, got:\n%s", want, md)
				}
			}
			for _, drop := range tc.dropTitles {
				if strings.Contains(md, drop) {
					t.Errorf("expected markdown NOT to contain %q, got:\n%s", drop, md)
				}
			}
		})
	}
}

// TestMarkdownFilterMatchesJSONFilter verifies that the set of task IDs in
// filtered markdown output matches the set in filtered JSON output (T-579).
func TestMarkdownFilterMatchesJSONFilter(t *testing.T) {
	tests := map[string]struct {
		tasks  []task.Task
		phases []task.PhaseMarker
		opts   listFilterOptions
	}{
		"status filter": {
			tasks: []task.Task{
				{ID: "1", Title: "Pending task", Status: task.Pending},
				{ID: "2", Title: "Completed task", Status: task.Completed},
				{ID: "3", Title: "In-progress task", Status: task.InProgress},
			},
			opts: listFilterOptions{statusFilter: "pending"},
		},
		"status filter with children": {
			tasks: []task.Task{
				{ID: "1", Title: "Parent done", Status: task.Completed, Children: []task.Task{
					{ID: "1.1", Title: "Child pending", Status: task.Pending},
				}},
				{ID: "2", Title: "Parent pending", Status: task.Pending},
			},
			opts: listFilterOptions{statusFilter: "pending"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl := &task.TaskList{Title: "Cross-Format Test", Tasks: tc.tasks}

			// Get filtered JSON task IDs
			filteredTasks := filterTasksRecursive(tl.Tasks, tc.opts)
			jsonIDs := collectIDs(filteredTasks)

			// Get markdown output and extract task IDs from rendered lines
			md := outputMarkdownWithFilters(tl, tc.phases, tc.opts)

			for _, id := range jsonIDs {
				// Each filtered task's title should appear in markdown
				title := findTitleByID(tc.tasks, id)
				if title != "" && !strings.Contains(md, title) {
					t.Errorf("task %s (%q) in JSON output but missing from markdown", id, title)
				}
			}
		})
	}
}

// collectIDs returns all task IDs from a (possibly nested) task slice.
func collectIDs(tasks []task.Task) []string {
	var ids []string
	for _, t := range tasks {
		ids = append(ids, t.ID)
		ids = append(ids, collectIDs(t.Children)...)
	}
	return ids
}

// findTitleByID searches tasks (recursively) for a task with the given ID.
func findTitleByID(tasks []task.Task, id string) string {
	for _, t := range tasks {
		if t.ID == id {
			return t.Title
		}
		if title := findTitleByID(t.Children, id); title != "" {
			return title
		}
	}
	return ""
}

// TestFilterTasksRecursiveClearsStaleParentID verifies that when
// filterTasksRecursive promotes a child (because its parent was excluded by
// filters), the child's ParentID is updated so it no longer references the
// now-absent parent. Regression test for T-549.
func TestFilterTasksRecursiveClearsStaleParentID(t *testing.T) {
	tests := map[string]struct {
		setupTasks       func(*task.TaskList)
		opts             listFilterOptions
		expectedParentID map[string]string // taskID → expected ParentID
		description      string
	}{
		"promoted child gets parent's ParentID cleared": {
			setupTasks: func(tl *task.TaskList) {
				tl.AddTask("", "Parent task", "")      // task 1, pending
				tl.AddTask("1", "Child task", "")      // task 1.1, pending
				tl.UpdateStatus("1.1", task.Completed) // child is completed
			},
			opts: listFilterOptions{statusFilter: "completed"},
			expectedParentID: map[string]string{
				"1.1": "", // promoted to root, ParentID must be empty
			},
			description: "Child promoted to root should have empty ParentID",
		},
		"grandchild promoted through two excluded ancestors": {
			setupTasks: func(tl *task.TaskList) {
				tl.AddTask("", "Grandparent", "")        // task 1, pending
				tl.AddTask("1", "Parent", "")            // task 1.1, pending
				tl.AddTask("1.1", "Grandchild", "")      // task 1.1.1, pending
				tl.UpdateStatus("1.1.1", task.Completed) // only grandchild completed
			},
			opts: listFilterOptions{statusFilter: "completed"},
			expectedParentID: map[string]string{
				"1.1.1": "", // promoted through two levels to root
			},
			description: "Grandchild promoted through two levels should have empty ParentID",
		},
		"child promoted to surviving grandparent": {
			setupTasks: func(tl *task.TaskList) {
				tl.AddTaskWithOptions("", "Grandparent", task.AddOptions{Stream: 2})
				tl.AddTaskWithOptions("1", "Parent", task.AddOptions{Stream: 1})       // excluded
				tl.AddTaskWithOptions("1.1", "Grandchild", task.AddOptions{Stream: 2}) // kept
			},
			opts: listFilterOptions{streamFilter: 2},
			expectedParentID: map[string]string{
				"1":     "",  // root task, ParentID stays empty
				"1.1.1": "1", // promoted past parent, should point to grandparent
			},
			description: "Child promoted past excluded parent should point to surviving grandparent",
		},
		"matching parent and child keep original ParentID": {
			setupTasks: func(tl *task.TaskList) {
				tl.AddTask("", "Parent", "")
				tl.AddTask("1", "Child", "")
				tl.UpdateStatus("1", task.Completed)
				tl.UpdateStatus("1.1", task.Completed)
			},
			opts: listFilterOptions{statusFilter: "completed"},
			expectedParentID: map[string]string{
				"1":   "",  // root task
				"1.1": "1", // parent is present, keep original
			},
			description: "When both parent and child match, ParentID should be unchanged",
		},
		"multiple children promoted from same excluded parent": {
			setupTasks: func(tl *task.TaskList) {
				tl.AddTask("", "Parent task", "") // task 1, pending
				tl.AddTask("1", "Child A", "")    // task 1.1
				tl.AddTask("1", "Child B", "")    // task 1.2
				tl.UpdateStatus("1.1", task.Completed)
				tl.UpdateStatus("1.2", task.Completed)
			},
			opts: listFilterOptions{statusFilter: "completed"},
			expectedParentID: map[string]string{
				"1.1": "", // both promoted to root
				"1.2": "",
			},
			description: "Multiple children promoted from same parent should both have empty ParentID",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl := task.NewTaskList("ParentID Test")
			tc.setupTasks(tl)

			filtered := filterTasksRecursive(tl.Tasks, tc.opts)

			// Build a map of taskID → ParentID from the filtered result
			gotParentIDs := collectParentIDs(filtered)

			for taskID, expectedPID := range tc.expectedParentID {
				gotPID, exists := gotParentIDs[taskID]
				if !exists {
					t.Errorf("%s: task %s not found in filtered output", tc.description, taskID)
					continue
				}
				if gotPID != expectedPID {
					t.Errorf("%s: task %s has ParentID=%q, want %q",
						tc.description, taskID, gotPID, expectedPID)
				}
			}
		})
	}
}

// TestFilteredJSONOutputNoStaleParentIDs verifies that the full JSON rendering
// pipeline (filterTasksRecursive → RenderJSON) never emits a ParentID that
// references a task absent from the output. Regression test for T-549.
func TestFilteredJSONOutputNoStaleParentIDs(t *testing.T) {
	tl := task.NewTaskList("Stale ParentID Test")
	tl.AddTask("", "Pending parent", "")   // task 1
	tl.AddTask("1", "Completed child", "") // task 1.1
	tl.UpdateStatus("1.1", task.Completed)
	tl.AddTask("", "Completed root", "") // task 2
	tl.UpdateStatus("2", task.Completed)

	opts := listFilterOptions{statusFilter: "completed"}
	filteredTasks := filterTasksRecursive(tl.Tasks, opts)

	filteredList := &task.TaskList{
		Title: tl.Title,
		Tasks: filteredTasks,
	}

	jsonData := task.RenderJSONWithPhases(filteredList, nil, tl)

	var result struct {
		Tasks []struct {
			ID       string `json:"ID"`
			ParentID string `json:"ParentID"`
		} `json:"Tasks"`
	}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Build a set of all task IDs in the output
	idSet := make(map[string]bool)
	for _, tsk := range result.Tasks {
		idSet[tsk.ID] = true
	}

	// Every non-empty ParentID must reference a task that exists in the output
	for _, tsk := range result.Tasks {
		if tsk.ParentID != "" && !idSet[tsk.ParentID] {
			t.Errorf("task %s has ParentID=%q which does not exist in filtered output (available IDs: %v)",
				tsk.ID, tsk.ParentID, idSet)
		}
	}
}

// collectParentIDs recursively collects taskID → ParentID from a task tree.
func collectParentIDs(tasks []task.Task) map[string]string {
	result := make(map[string]string)
	for _, t := range tasks {
		result[t.ID] = t.ParentID
		for k, v := range collectParentIDs(t.Children) {
			result[k] = v
		}
	}
	return result
}
