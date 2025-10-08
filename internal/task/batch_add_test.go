package task

import (
	"fmt"
	"strings"
	"testing"
)

func TestExecuteBatch_AddWithRequirements(t *testing.T) {
	tl := NewTaskList("Test Tasks")
	tl.RequirementsFile = "requirements.md"

	ops := []Operation{
		{
			Type:         "add",
			Title:        "Implement feature",
			Requirements: []string{"1.1", "1.2", "2.3"},
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}
	if response.Applied != 1 {
		t.Errorf("Expected 1 applied operation, got %d", response.Applied)
	}

	// Verify requirements were added
	if len(tl.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(tl.Tasks))
	}
	task := tl.Tasks[0]
	if len(task.Requirements) != 3 {
		t.Errorf("Expected 3 requirements, got %d", len(task.Requirements))
	}
	expectedReqs := []string{"1.1", "1.2", "2.3"}
	for i, req := range expectedReqs {
		if i >= len(task.Requirements) || task.Requirements[i] != req {
			t.Errorf("Expected requirement %s at index %d, got %v", req, i, task.Requirements)
		}
	}
}

func TestExecuteBatch_PositionInsertionDryRun(t *testing.T) {
	tl := NewTaskList("Dry Run Position Test")

	// Set up initial tasks
	tl.AddTask("", "Task 1", "")
	tl.AddTask("", "Task 2", "")

	initialTaskCount := len(tl.Tasks)
	initialTitles := []string{tl.Tasks[0].Title, tl.Tasks[1].Title}

	ops := []Operation{
		{
			Type:     "add",
			Title:    "Inserted Task",
			Position: "2",
		},
		{
			Type:  "update",
			ID:    "1",
			Title: "Updated Task 1",
		},
	}

	response, err := tl.ExecuteBatch(ops, true) // dry-run = true
	if err != nil {
		t.Fatalf("ExecuteBatch dry-run failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success for dry-run, got errors: %v", response.Errors)
	}

	if response.Applied != 2 {
		t.Errorf("Expected 2 applied operations in dry-run, got %d", response.Applied)
	}

	if response.Preview == "" {
		t.Error("Expected preview content in dry-run")
	}

	// Original should be unchanged
	if len(tl.Tasks) != initialTaskCount {
		t.Errorf("Expected task count to remain %d after dry-run, got %d", initialTaskCount, len(tl.Tasks))
	}

	for i, task := range tl.Tasks {
		if task.Title != initialTitles[i] {
			t.Errorf("Task %d title should remain unchanged after dry-run: expected '%s', got '%s'", i, initialTitles[i], task.Title)
		}
	}

	// Preview should contain the expected changes
	if !strings.Contains(response.Preview, "Inserted Task") {
		t.Error("Preview should contain 'Inserted Task'")
	}
	if !strings.Contains(response.Preview, "Updated Task 1") {
		t.Error("Preview should contain 'Updated Task 1'")
	}
}

// TestExecuteBatch_UnifiedUpdateAutoCompleteTriggers tests auto-completion with unified update operations (Requirements 1.4)
func TestExecuteBatch_PositionInsertionAtomicFailure(t *testing.T) {
	tl := NewTaskList("Atomic Failure Test")

	// Set up initial tasks
	tl.AddTask("", "Task 1", "")
	tl.AddTask("", "Task 2", "")

	initialTaskCount := len(tl.Tasks)

	// Mix valid operations with invalid position insertion
	ops := []Operation{
		{
			Type:  "update",
			ID:    "1",
			Title: "Updated Task 1",
		},
		{
			Type:     "add",
			Title:    "Valid Task",
			Position: "invalid-position", // This should fail validation
		},
		{
			Type:   "update",
			ID:     "2",
			Status: StatusPtr(Completed),
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch returned error: %v", err)
	}

	// Should fail validation and not apply any operations
	if response.Success {
		t.Error("Expected batch to fail due to invalid position format")
	}

	if response.Applied != 0 {
		t.Errorf("Expected 0 applied operations due to atomic failure, got %d", response.Applied)
	}

	// Verify nothing was changed (atomic behavior)
	if len(tl.Tasks) != initialTaskCount {
		t.Errorf("Expected task count to remain %d, got %d", initialTaskCount, len(tl.Tasks))
	}

	task1 := tl.FindTask("1")
	if task1.Title != "Task 1" {
		t.Errorf("Task 1 title should remain unchanged, got '%s'", task1.Title)
	}

	task2 := tl.FindTask("2")
	if task2.Status != Pending {
		t.Errorf("Task 2 status should remain Pending, got %v", task2.Status)
	}
}

// TestExecuteBatch_PositionInsertionDryRun tests position insertion in dry-run mode
func TestExecuteBatch_PositionInsertionWithOtherOperations(t *testing.T) {
	tl := NewTaskList("Mixed Operations Test")

	// Set up initial structure
	tl.AddTask("", "Task 1", "")
	tl.AddTask("", "Task 2", "")
	tl.AddTask("", "Task 3", "")

	// Mix position insertion with updates and removes
	ops := []Operation{
		{
			Type:     "add",
			Title:    "New Task at 2",
			Position: "2",
		},
		{
			Type:   "update",
			ID:     "1",
			Title:  "Updated Task 1",
			Status: StatusPtr(InProgress),
		},
		{
			Type: "remove",
			ID:   "3", // This will be Task 2 after the insertion and renumbering
		},
		{
			Type:     "add",
			Parent:   "2", // Parent will be "New Task at 2"
			Title:    "Child of New Task",
			Position: "1",
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	if response.Applied != 4 {
		t.Errorf("Expected 4 applied operations, got %d", response.Applied)
	}

	// After all operations:
	// Initial: Task 1, Task 2, Task 3
	// After insert at 2: Task 1, New Task at 2, Task 2, Task 3 (IDs: 1, 2, 3, 4)
	// After update task 1: Updated Task 1, New Task at 2, Task 2, Task 3
	// After remove task 3 (which is Task 2): Updated Task 1, New Task at 2, Task 3 (IDs: 1, 2, 3)
	// After add child to task 2: Updated Task 1, New Task at 2 (with child), Task 3

	// Verify root task count
	if len(tl.Tasks) != 3 {
		t.Errorf("Expected 3 root tasks, got %d", len(tl.Tasks))
	}

	// Verify first task was updated
	task1 := tl.FindTask("1")
	if task1 == nil {
		t.Fatal("Task 1 not found")
	}
	if task1.Title != "Updated Task 1" {
		t.Errorf("Expected 'Updated Task 1', got '%s'", task1.Title)
	}
	if task1.Status != InProgress {
		t.Errorf("Expected InProgress status, got %v", task1.Status)
	}

	// Verify inserted task exists and has child
	task2 := tl.FindTask("2")
	if task2 == nil {
		t.Fatal("Task 2 (New Task at 2) not found")
	}
	if task2.Title != "New Task at 2" {
		t.Errorf("Expected 'New Task at 2', got '%s'", task2.Title)
	}
	if len(task2.Children) != 1 {
		t.Errorf("Expected 1 child for task 2, got %d", len(task2.Children))
	}
	if task2.Children[0].Title != "Child of New Task" {
		t.Errorf("Expected child title 'Child of New Task', got '%s'", task2.Children[0].Title)
	}

	// Verify third task remains
	task3 := tl.FindTask("3")
	if task3 == nil {
		t.Fatal("Task 3 not found")
	}
	if task3.Title != "Task 3" {
		t.Errorf("Expected 'Task 3', got '%s'", task3.Title)
	}
}

// TestExecuteBatch_PositionInsertionAtomicFailure tests atomic behavior when position insertion fails
func TestExecuteBatch_PositionInsertionMultiple(t *testing.T) {
	tl := NewTaskList("Multiple Position Test")

	// Add initial tasks
	tl.AddTask("", "Task 1", "")
	tl.AddTask("", "Task 2", "")
	tl.AddTask("", "Task 3", "")
	tl.AddTask("", "Task 4", "")

	// Multiple position insertions - processed in reverse order (per requirement 2.10)
	// to maintain position references to original pre-batch state
	ops := []Operation{
		{
			Type:     "add",
			Title:    "Insert at 2",
			Position: "2",
		},
		{
			Type:     "add",
			Title:    "Insert at 4",
			Position: "4",
		},
		{
			Type:     "add",
			Title:    "Insert at 1",
			Position: "1",
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	if response.Applied != 3 {
		t.Errorf("Expected 3 applied operations, got %d", response.Applied)
	}

	// Verify we have 7 tasks total
	if len(tl.Tasks) != 7 {
		t.Errorf("Expected 7 tasks after insertions, got %d", len(tl.Tasks))
	}

	// The expected final order after reverse-order insertions (4, 2, 1):
	// Original: [Task 1, Task 2, Task 3, Task 4]
	// 1. Insert at 4: [Task 1, Task 2, Task 3, Insert at 4, Task 4]
	// 2. Insert at 2: [Task 1, Insert at 2, Task 2, Task 3, Insert at 4, Task 4]
	// 3. Insert at 1: [Insert at 1, Task 1, Insert at 2, Task 2, Task 3, Insert at 4, Task 4]
	expectedTitles := []string{"Insert at 1", "Task 1", "Insert at 2", "Task 2", "Task 3", "Insert at 4", "Task 4"}

	for i, task := range tl.Tasks {
		if task.Title != expectedTitles[i] {
			t.Errorf("Task %d: expected title '%s', got '%s'", i, expectedTitles[i], task.Title)
		}
		expectedID := fmt.Sprintf("%d", i+1)
		if task.ID != expectedID {
			t.Errorf("Task %d: expected ID '%s', got '%s'", i, expectedID, task.ID)
		}
	}
}

// TestExecuteBatch_PositionInsertionWithOtherOperations tests position insertion mixed with other operation types
func TestExecuteBatch_PositionInsertionValidation(t *testing.T) {
	tl := NewTaskList("Validation Test")
	tl.AddTask("", "Task 1", "")

	tests := map[string]struct {
		position    string
		expectError bool
		errorMsg    string
	}{
		"valid position": {
			position:    "1",
			expectError: false,
		},
		"valid hierarchical position": {
			position:    "1.2",
			expectError: false,
		},
		"invalid position with letters": {
			position:    "1a",
			expectError: true,
			errorMsg:    "invalid position format: 1a",
		},
		"invalid position starting with 0": {
			position:    "0",
			expectError: true,
			errorMsg:    "invalid position format: 0",
		},
		"invalid position with dot at end": {
			position:    "1.",
			expectError: true,
			errorMsg:    "invalid position format: 1.",
		},
		"invalid position with double dots": {
			position:    "1..2",
			expectError: true,
			errorMsg:    "invalid position format: 1..2",
		},
		"invalid position with spaces": {
			position:    "1 2",
			expectError: true,
			errorMsg:    "invalid position format: 1 2",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ops := []Operation{
				{
					Type:     "add",
					Title:    "Test Task",
					Position: tc.position,
				},
			}

			response, err := tl.ExecuteBatch(ops, false)
			if err != nil {
				t.Fatalf("ExecuteBatch returned error: %v", err)
			}

			if tc.expectError {
				if response.Success {
					t.Errorf("Expected validation failure for position %q, but got success", tc.position)
				}
				if tc.errorMsg != "" && len(response.Errors) > 0 {
					if !strings.Contains(response.Errors[0], tc.errorMsg) {
						t.Errorf("Expected error message to contain %q, got %q", tc.errorMsg, response.Errors[0])
					}
				}
			} else {
				if !response.Success {
					t.Errorf("Expected success for position %q, but got errors: %v", tc.position, response.Errors)
				}
			}
		})
	}
}

// TestExecuteBatch_PositionInsertionMultiple tests multiple position insertions in single batch
func TestExecuteBatch_PositionInsertionHierarchical(t *testing.T) {
	tl := NewTaskList("Hierarchical Position Test")

	// Set up hierarchical structure
	tl.AddTask("", "Parent 1", "")
	tl.AddTask("1", "Child 1.1", "")
	tl.AddTask("1", "Child 1.2", "")
	tl.AddTask("1", "Child 1.3", "")

	ops := []Operation{
		{
			Type:     "add",
			Parent:   "1",
			Title:    "New Child",
			Position: "2", // Insert at position 2 within parent's children
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Verify parent has 4 children now
	parent := tl.FindTask("1")
	if parent == nil {
		t.Fatal("Parent task not found")
	}
	if len(parent.Children) != 4 {
		t.Errorf("Expected 4 children, got %d", len(parent.Children))
	}

	// Verify order: Child 1.1, New Child, Child 1.2, Child 1.3
	expectedTitles := []string{"Child 1.1", "New Child", "Child 1.2", "Child 1.3"}
	expectedIDs := []string{"1.1", "1.2", "1.3", "1.4"}

	for i, child := range parent.Children {
		if child.Title != expectedTitles[i] {
			t.Errorf("Child %d: expected title '%s', got '%s'", i, expectedTitles[i], child.Title)
		}
		if child.ID != expectedIDs[i] {
			t.Errorf("Child %d: expected ID '%s', got '%s'", i, expectedIDs[i], child.ID)
		}
		if child.ParentID != "1" {
			t.Errorf("Child %d: expected ParentID '1', got '%s'", i, child.ParentID)
		}
	}
}

// TestExecuteBatch_PositionInsertionValidation tests position format validation in batch operations
func TestExecuteBatch_PositionInsertionSingle(t *testing.T) {
	tl := NewTaskList("Position Test")

	// Add some initial tasks
	tl.AddTask("", "Task 1", "")
	tl.AddTask("", "Task 2", "")
	tl.AddTask("", "Task 3", "")

	tests := map[string]struct {
		op             Operation
		expectedTitles []string
		expectedIDs    []string
		desc           string
	}{
		"insert at beginning": {
			op: Operation{
				Type:     "add",
				Title:    "New First Task",
				Position: "1",
			},
			expectedTitles: []string{"New First Task", "Task 1", "Task 2", "Task 3"},
			expectedIDs:    []string{"1", "2", "3", "4"},
			desc:           "Task inserted at position 1 should become new task 1",
		},
		"insert in middle": {
			op: Operation{
				Type:     "add",
				Title:    "New Middle Task",
				Position: "2",
			},
			expectedTitles: []string{"Task 1", "New Middle Task", "Task 2", "Task 3"},
			expectedIDs:    []string{"1", "2", "3", "4"},
			desc:           "Task inserted at position 2 should become new task 2",
		},
		"insert beyond end should append": {
			op: Operation{
				Type:     "add",
				Title:    "Appended Task",
				Position: "10",
			},
			expectedTitles: []string{"Task 1", "Task 2", "Task 3", "Appended Task"},
			expectedIDs:    []string{"1", "2", "3", "4"},
			desc:           "Task inserted beyond list end should be appended",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create fresh test list for each test
			testTL := NewTaskList("Position Test")
			testTL.AddTask("", "Task 1", "")
			testTL.AddTask("", "Task 2", "")
			testTL.AddTask("", "Task 3", "")

			ops := []Operation{tc.op}

			response, err := testTL.ExecuteBatch(ops, false)
			if err != nil {
				t.Fatalf("ExecuteBatch failed: %v", err)
			}

			if !response.Success {
				t.Fatalf("%s: Expected success, got errors: %v", tc.desc, response.Errors)
			}

			if response.Applied != 1 {
				t.Errorf("%s: Expected 1 applied operation, got %d", tc.desc, response.Applied)
			}

			// Verify task count
			if len(testTL.Tasks) != 4 {
				t.Errorf("%s: Expected 4 tasks, got %d", tc.desc, len(testTL.Tasks))
			}

			// Verify titles and IDs
			for i, task := range testTL.Tasks {
				if i < len(tc.expectedTitles) && task.Title != tc.expectedTitles[i] {
					t.Errorf("%s: Task %d expected title '%s', got '%s'", tc.desc, i, tc.expectedTitles[i], task.Title)
				}
				if i < len(tc.expectedIDs) && task.ID != tc.expectedIDs[i] {
					t.Errorf("%s: Task %d expected ID '%s', got '%s'", tc.desc, i, tc.expectedIDs[i], task.ID)
				}
			}
		})
	}
}

// TestExecuteBatch_PositionInsertionHierarchical tests position insertion with hierarchical tasks
func TestExecuteBatch_ComplexOperations(t *testing.T) {
	tl := NewTaskList("Complex Test")

	// Set up initial structure
	tl.AddTask("", "Task 1", "")
	tl.AddTask("1", "Task 1.1", "")
	tl.AddTask("1", "Task 1.2", "")
	tl.AddTask("", "Task 2", "")

	ops := []Operation{
		{
			Type:       "update",
			ID:         "1",
			Title:      "Updated Task 1",
			Details:    []string{"New detail 1", "New detail 2"},
			References: []string{"ref1.md", "ref2.md"},
		},
		{
			Type: "remove",
			ID:   "1.2",
		},
		{
			Type:   "add",
			Parent: "2",
			Title:  "Task 2.1",
		},
		{
			Type:   "update",
			ID:     "2",
			Status: StatusPtr(InProgress),
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Verify results
	task1 := tl.FindTask("1")
	if task1.Title != "Updated Task 1" {
		t.Errorf("Expected 'Updated Task 1', got '%s'", task1.Title)
	}
	if len(task1.Details) != 2 {
		t.Errorf("Expected 2 details, got %d", len(task1.Details))
	}
	if len(task1.References) != 2 {
		t.Errorf("Expected 2 references, got %d", len(task1.References))
	}
	if len(task1.Children) != 1 {
		t.Errorf("Expected 1 child after removal, got %d", len(task1.Children))
	}

	task2 := tl.FindTask("2")
	if task2.Status != InProgress {
		t.Errorf("Expected InProgress status, got %v", task2.Status)
	}
	if len(task2.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(task2.Children))
	}
}

func TestExecuteBatch_MultipleOperations(t *testing.T) {
	tl := NewTaskList("Test Tasks")

	ops := []Operation{
		{
			Type:  "add",
			Title: "Parent task",
		},
		{
			Type:   "add",
			Parent: "1",
			Title:  "Child task",
		},
		{
			Type:   "update",
			ID:     "1",
			Status: StatusPtr(InProgress),
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}
	if response.Applied != 3 {
		t.Errorf("Expected 3 applied operations, got %d", response.Applied)
	}

	// Verify structure
	if len(tl.Tasks) != 1 {
		t.Errorf("Expected 1 root task, got %d", len(tl.Tasks))
	}
	if len(tl.Tasks[0].Children) != 1 {
		t.Errorf("Expected 1 child task, got %d", len(tl.Tasks[0].Children))
	}
	if tl.Tasks[0].Status != InProgress {
		t.Errorf("Expected parent status InProgress, got %v", tl.Tasks[0].Status)
	}
}

func TestExecuteBatch_SingleAdd(t *testing.T) {
	tl := NewTaskList("Test Tasks")

	ops := []Operation{
		{
			Type:  "add",
			Title: "First task",
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}
	if response.Applied != 1 {
		t.Errorf("Expected 1 applied operation, got %d", response.Applied)
	}
	if len(tl.Tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tl.Tasks))
	}
	if tl.Tasks[0].Title != "First task" {
		t.Errorf("Expected 'First task', got '%s'", tl.Tasks[0].Title)
	}
}
