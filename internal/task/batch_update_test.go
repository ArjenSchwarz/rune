package task

import (
	"strings"
	"testing"
)

func TestExecuteBatch_UpdateWithRequirements(t *testing.T) {
	tl := NewTaskList("Test Tasks")
	tl.RequirementsFile = "requirements.md"
	tl.AddTask("", "Existing task", "")

	ops := []Operation{
		{
			Type:         "update",
			ID:           "1",
			Requirements: []string{"3.1", "3.2"},
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

	// Verify requirements were updated
	task := tl.Tasks[0]
	if len(task.Requirements) != 2 {
		t.Errorf("Expected 2 requirements, got %d", len(task.Requirements))
	}
	expectedReqs := []string{"3.1", "3.2"}
	for i, req := range expectedReqs {
		if i >= len(task.Requirements) || task.Requirements[i] != req {
			t.Errorf("Expected requirement %s at index %d, got %v", req, i, task.Requirements)
		}
	}
}

func TestExecuteBatch_UpdateStatusOperationInvalid(t *testing.T) {
	tl := NewTaskList("Test Tasks")
	tl.AddTask("", "Task 1", "")

	// update_status operations should now fail validation
	ops := []Operation{
		{
			Type:   "update_status", // This should fail - operation type no longer exists
			ID:     "1",
			Status: StatusPtr(Completed),
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch returned error: %v", err)
	}

	// Batch should fail due to invalid operation type
	if response.Success {
		t.Error("Expected batch to fail due to invalid update_status operation type")
	}

	// Should have error for invalid operation type
	if len(response.Errors) == 0 {
		t.Error("Expected error for invalid update_status operation type")
	} else if !strings.Contains(response.Errors[0], "unknown operation type") && !strings.Contains(response.Errors[0], "invalid operation") {
		t.Errorf("Expected error about unknown operation type, got: %s", response.Errors[0])
	}

	// No operations should be applied (atomic failure)
	if response.Applied != 0 {
		t.Errorf("Expected 0 applied operations due to failure, got %d", response.Applied)
	}

	// Task should remain unchanged
	task1 := tl.FindTask("1")
	if task1.Status != Pending {
		t.Errorf("Task 1 should remain Pending, got %v", task1.Status)
	}
}

// TestExecuteBatch_PhaseAddOperation tests adding tasks with phase field
func TestExecuteBatch_UnifiedUpdateAutoCompleteTriggers(t *testing.T) {
	tests := map[string]struct {
		setup       func(*TaskList)
		ops         []Operation
		expectAuto  bool
		autoTaskIDs []string
		desc        string
	}{
		"status=completed only should trigger auto-completion": {
			setup: func(tl *TaskList) {
				tl.AddTask("", "Parent task", "")
				tl.AddTask("1", "Child task", "")
			},
			ops: []Operation{
				{
					Type:   "update",
					ID:     "1.1",
					Status: StatusPtr(Completed),
				},
			},
			expectAuto:  true,
			autoTaskIDs: []string{"1"},
			desc:        "Auto-completion should trigger when status field is set to completed",
		},
		"title + status=completed should trigger auto-completion": {
			setup: func(tl *TaskList) {
				tl.AddTask("", "Parent task", "")
				tl.AddTask("1", "Child task", "")
			},
			ops: []Operation{
				{
					Type:   "update",
					ID:     "1.1",
					Title:  "Updated child task",
					Status: StatusPtr(Completed),
				},
			},
			expectAuto:  true,
			autoTaskIDs: []string{"1"},
			desc:        "Auto-completion should trigger when both title and status=completed are updated",
		},
		"details + references + status=completed should trigger auto-completion": {
			setup: func(tl *TaskList) {
				tl.AddTask("", "Parent task", "")
				tl.AddTask("1", "Child task", "")
			},
			ops: []Operation{
				{
					Type:       "update",
					ID:         "1.1",
					Details:    []string{"Updated detail"},
					References: []string{"updated-ref.md"},
					Status:     StatusPtr(Completed),
				},
			},
			expectAuto:  true,
			autoTaskIDs: []string{"1"},
			desc:        "Auto-completion should trigger when status=completed is updated with other fields",
		},
		"title only update should NOT trigger auto-completion": {
			setup: func(tl *TaskList) {
				tl.AddTask("", "Parent task", "")
				tl.AddTask("1", "Child task", "")
			},
			ops: []Operation{
				{
					Type:  "update",
					ID:    "1.1",
					Title: "Updated title only",
				},
			},
			expectAuto:  false,
			autoTaskIDs: []string{},
			desc:        "Auto-completion should NOT trigger when only non-status fields are updated",
		},
		"status=pending should NOT trigger auto-completion": {
			setup: func(tl *TaskList) {
				tl.AddTask("", "Parent task", "")
				tl.AddTask("1", "Child task", "")
			},
			ops: []Operation{
				{
					Type:   "update",
					ID:     "1.1",
					Status: StatusPtr(Pending),
				},
			},
			expectAuto:  false,
			autoTaskIDs: []string{},
			desc:        "Auto-completion should NOT trigger when status is not completed",
		},
		"status=in_progress should NOT trigger auto-completion": {
			setup: func(tl *TaskList) {
				tl.AddTask("", "Parent task", "")
				tl.AddTask("1", "Child task", "")
			},
			ops: []Operation{
				{
					Type:   "update",
					ID:     "1.1",
					Status: StatusPtr(InProgress),
				},
			},
			expectAuto:  false,
			autoTaskIDs: []string{},
			desc:        "Auto-completion should NOT trigger when status is in progress",
		},
		"title + status=pending should NOT trigger auto-completion": {
			setup: func(tl *TaskList) {
				tl.AddTask("", "Parent task", "")
				tl.AddTask("1", "Child task", "")
			},
			ops: []Operation{
				{
					Type:   "update",
					ID:     "1.1",
					Title:  "Updated title",
					Status: StatusPtr(Pending),
				},
			},
			expectAuto:  false,
			autoTaskIDs: []string{},
			desc:        "Auto-completion should NOT trigger when status is pending even with title update",
		},
		"details + references without status should NOT trigger auto-completion": {
			setup: func(tl *TaskList) {
				tl.AddTask("", "Parent task", "")
				tl.AddTask("1", "Child task", "")
			},
			ops: []Operation{
				{
					Type:       "update",
					ID:         "1.1",
					Details:    []string{"New detail"},
					References: []string{"new-ref.md"},
				},
			},
			expectAuto:  false,
			autoTaskIDs: []string{},
			desc:        "Auto-completion should NOT trigger when only details and references are updated",
		},
		"empty update should NOT trigger auto-completion": {
			setup: func(tl *TaskList) {
				tl.AddTask("", "Parent task", "")
				tl.AddTask("1", "Child task", "")
			},
			ops: []Operation{
				{
					Type: "update",
					ID:   "1.1",
					// No fields - empty update
				},
			},
			expectAuto:  false,
			autoTaskIDs: []string{},
			desc:        "Auto-completion should NOT trigger for empty update operations",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl := NewTaskList("Auto-completion Test")
			tc.setup(tl)

			response, err := tl.ExecuteBatch(tc.ops, false)
			if err != nil {
				t.Fatalf("ExecuteBatch failed: %v", err)
			}

			if !response.Success {
				t.Fatalf("Expected success, got errors: %v", response.Errors)
			}

			// Verify auto-completion expectation
			if tc.expectAuto {
				if len(response.AutoCompleted) == 0 {
					t.Errorf("%s: Expected auto-completion to occur, but no tasks were auto-completed", tc.desc)
				} else {
					// Check that expected task IDs were auto-completed
					completedSet := make(map[string]bool)
					for _, id := range response.AutoCompleted {
						completedSet[id] = true
					}
					for _, expectedID := range tc.autoTaskIDs {
						if !completedSet[expectedID] {
							t.Errorf("%s: Expected task %s to be auto-completed, but it wasn't", tc.desc, expectedID)
						}
					}
				}
			} else {
				if len(response.AutoCompleted) > 0 {
					t.Errorf("%s: Expected no auto-completion, but got auto-completed tasks: %v", tc.desc, response.AutoCompleted)
				}
			}
		})
	}
}

// TestExecuteBatch_UpdateStatusOperationInvalid tests that update_status operations are no longer supported
func TestExecuteBatch_UnifiedUpdateTitleLengthValidation(t *testing.T) {
	tl := NewTaskList("Test Tasks")
	tl.AddTask("", "Task 1", "")
	tl.AddTask("", "Task 2", "")
	tl.AddTask("", "Task 3", "")

	longTitle := string(make([]byte, 501)) // Exceeds 500 character limit

	tests := map[string]struct {
		op      Operation
		wantErr bool
	}{
		"valid title update": {
			op: Operation{
				Type:  "update",
				ID:    "1",
				Title: "Valid title",
			},
			wantErr: false,
		},
		"title too long should fail": {
			op: Operation{
				Type:  "update",
				ID:    "2",
				Title: longTitle,
			},
			wantErr: true,
		},
		"no title field should pass (even with other fields)": {
			op: Operation{
				Type:   "update",
				ID:     "3",
				Status: StatusPtr(InProgress),
				// No title field - validation should be skipped
			},
			wantErr: false,
		},
		"empty title should pass (means no update to title)": {
			op: Operation{
				Type:  "update",
				ID:    "1",
				Title: "", // Empty title means don't update title
			},
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ops := []Operation{tc.op}

			response, err := tl.ExecuteBatch(ops, false)
			if err != nil {
				t.Fatalf("ExecuteBatch returned error: %v", err)
			}

			if tc.wantErr && response.Success {
				t.Error("Expected validation failure for long title, but got success")
			}
			if !tc.wantErr && !response.Success {
				t.Errorf("Expected success, but got errors: %v", response.Errors)
			}
		})
	}
}

// TestExecuteBatch_PositionInsertionSingle tests single position insertion in batch operations
func TestExecuteBatch_UnifiedUpdateEmptyOperation(t *testing.T) {
	tl := NewTaskList("Test Tasks")
	tl.AddTask("", "Unchanged Task", "")

	// Store original task state
	originalTask := *tl.FindTask("1")

	ops := []Operation{
		{
			Type: "update",
			ID:   "1",
			// No fields provided - should be no-op
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success for empty update, got errors: %v", response.Errors)
	}

	// Verify task remains unchanged
	task := tl.FindTask("1")
	if task.Title != originalTask.Title {
		t.Errorf("Title should be unchanged: expected '%s', got '%s'", originalTask.Title, task.Title)
	}
	if task.Status != originalTask.Status {
		t.Errorf("Status should be unchanged: expected %v, got %v", originalTask.Status, task.Status)
	}
	if len(task.Details) != len(originalTask.Details) {
		t.Errorf("Details length should be unchanged: expected %d, got %d", len(originalTask.Details), len(task.Details))
	}
	if len(task.References) != len(originalTask.References) {
		t.Errorf("References length should be unchanged: expected %d, got %d", len(originalTask.References), len(task.References))
	}

	// Operation should still count as applied (even though it's a no-op)
	if response.Applied != 1 {
		t.Errorf("Expected 1 applied operation, got %d", response.Applied)
	}

	// No auto-completion should occur
	if len(response.AutoCompleted) != 0 {
		t.Errorf("Expected no auto-completion, got %v", response.AutoCompleted)
	}
}

// TestExecuteBatch_UnifiedUpdateTitleLengthValidation tests that title length validation only occurs when title field is provided
func TestExecuteBatch_UnifiedUpdateDetailsAndReferencesWithoutStatus(t *testing.T) {
	tl := NewTaskList("Test Tasks")
	tl.AddTask("", "Task with details", "")
	tl.AddTask("", "Task with references", "")
	tl.AddTask("", "Task with both", "")

	ops := []Operation{
		{
			Type:    "update",
			ID:      "1",
			Details: []string{"Detail 1", "Detail 2", "Detail 3"},
		},
		{
			Type:       "update",
			ID:         "2",
			References: []string{"ref1.md", "ref2.md"},
		},
		{
			Type:       "update",
			ID:         "3",
			Details:    []string{"Combined detail"},
			References: []string{"combined-ref.md"},
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Verify task 1 details were updated
	task1 := tl.FindTask("1")
	if len(task1.Details) != 3 {
		t.Errorf("Expected 3 details, got %d", len(task1.Details))
	}
	if task1.Details[0] != "Detail 1" || task1.Details[1] != "Detail 2" || task1.Details[2] != "Detail 3" {
		t.Errorf("Details not set correctly: %v", task1.Details)
	}
	// Status should remain unchanged
	if task1.Status != Pending {
		t.Errorf("Expected status Pending (unchanged), got %v", task1.Status)
	}

	// Verify task 2 references were updated
	task2 := tl.FindTask("2")
	if len(task2.References) != 2 {
		t.Errorf("Expected 2 references, got %d", len(task2.References))
	}
	if task2.References[0] != "ref1.md" || task2.References[1] != "ref2.md" {
		t.Errorf("References not set correctly: %v", task2.References)
	}
	// Status should remain unchanged
	if task2.Status != Pending {
		t.Errorf("Expected status Pending (unchanged), got %v", task2.Status)
	}

	// Verify task 3 has both details and references
	task3 := tl.FindTask("3")
	if len(task3.Details) != 1 || task3.Details[0] != "Combined detail" {
		t.Errorf("Expected 1 detail 'Combined detail', got %v", task3.Details)
	}
	if len(task3.References) != 1 || task3.References[0] != "combined-ref.md" {
		t.Errorf("Expected 1 reference 'combined-ref.md', got %v", task3.References)
	}
	// Status should remain unchanged
	if task3.Status != Pending {
		t.Errorf("Expected status Pending (unchanged), got %v", task3.Status)
	}

	// No auto-completion should occur since no status changes
	if len(response.AutoCompleted) != 0 {
		t.Errorf("Expected no auto-completion, got %v", response.AutoCompleted)
	}
}

// TestExecuteBatch_UnifiedUpdateEmptyOperation tests empty update operations (no-op behavior)
func TestExecuteBatch_UnifiedUpdateTitleAndStatus(t *testing.T) {
	tl := NewTaskList("Test Tasks")
	tl.AddTask("", "Original Title", "")
	tl.AddTask("1", "Child Task", "")

	ops := []Operation{
		{
			Type:   "update",
			ID:     "1",
			Title:  "Updated Parent Title",
			Status: StatusPtr(InProgress),
		},
		{
			Type:   "update",
			ID:     "1.1",
			Title:  "Updated Child Title",
			Status: StatusPtr(Completed),
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Verify parent was updated with both title and status
	parent := tl.FindTask("1")
	if parent.Title != "Updated Parent Title" {
		t.Errorf("Expected title 'Updated Parent Title', got '%s'", parent.Title)
	}
	// Parent should be auto-completed since only child is completed
	if parent.Status != Completed {
		t.Errorf("Expected parent status Completed (auto-completed), got %v", parent.Status)
	}

	// Verify child was updated with both title and status
	child := tl.FindTask("1.1")
	if child.Title != "Updated Child Title" {
		t.Errorf("Expected title 'Updated Child Title', got '%s'", child.Title)
	}
	if child.Status != Completed {
		t.Errorf("Expected child status Completed, got %v", child.Status)
	}

	// Verify auto-completion occurred
	if len(response.AutoCompleted) != 1 || response.AutoCompleted[0] != "1" {
		t.Errorf("Expected auto-completed task '1', got %v", response.AutoCompleted)
	}
}

// TestExecuteBatch_UnifiedUpdateDetailsAndReferencesWithoutStatus tests update operations with details and references but no status
func TestExecuteBatch_UnifiedUpdateStatusOnly(t *testing.T) {
	tl := NewTaskList("Test Tasks")
	tl.AddTask("", "Task 1", "")
	tl.AddTask("", "Task 2", "")

	tests := map[string]struct {
		taskID         string
		status         Status
		expectComplete bool
	}{
		"mark task as in progress": {
			taskID:         "1",
			status:         InProgress,
			expectComplete: false,
		},
		"mark task as completed": {
			taskID:         "2",
			status:         Completed,
			expectComplete: false, // No children to trigger auto-completion
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ops := []Operation{
				{
					Type:   "update",
					ID:     tc.taskID,
					Status: StatusPtr(tc.status),
				},
			}

			response, err := tl.ExecuteBatch(ops, false)
			if err != nil {
				t.Fatalf("ExecuteBatch failed: %v", err)
			}

			if !response.Success {
				t.Fatalf("Expected success, got errors: %v", response.Errors)
			}

			// Verify status was updated
			task := tl.FindTask(tc.taskID)
			if task.Status != tc.status {
				t.Errorf("Expected status %v, got %v", tc.status, task.Status)
			}

			// Verify auto-completion behavior
			if tc.expectComplete && len(response.AutoCompleted) == 0 {
				t.Error("Expected auto-completion to occur")
			}
			if !tc.expectComplete && len(response.AutoCompleted) > 0 {
				t.Errorf("Unexpected auto-completion: %v", response.AutoCompleted)
			}
		})
	}
}

// TestExecuteBatch_UnifiedUpdateTitleAndStatus tests update operations with both title and status fields
func TestExecuteBatch_UnifiedUpdateOperations(t *testing.T) {
	tl := NewTaskList("Test Tasks")
	tl.AddTask("", "Task 1", "")
	tl.AddTask("", "Task 2", "")
	tl.AddTask("", "Task 3", "")

	// All unified update operations should succeed
	ops := []Operation{
		{
			Type:   "update",
			ID:     "1",
			Status: StatusPtr(InProgress),
		},
		{
			Type:   "update", // Now uses unified update
			ID:     "2",
			Status: StatusPtr(Completed),
		},
		{
			Type:   "update",
			ID:     "3",
			Status: StatusPtr(Completed),
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch returned error: %v", err)
	}

	// All operations should succeed with unified update
	if !response.Success {
		t.Fatalf("Expected batch to succeed with unified update operations, got errors: %v", response.Errors)
	}

	// All operations should be applied
	if response.Applied != 3 {
		t.Errorf("Expected 3 applied operations, got %d", response.Applied)
	}

	// All tasks should be updated as expected
	task1 := tl.FindTask("1")
	if task1.Status != InProgress {
		t.Errorf("Expected Task 1 to be InProgress, got %v", task1.Status)
	}
	task2 := tl.FindTask("2")
	if task2.Status != Completed {
		t.Errorf("Expected Task 2 to be Completed, got %v", task2.Status)
	}
	task3 := tl.FindTask("3")
	if task3.Status != Completed {
		t.Errorf("Expected Task 3 to be Completed, got %v", task3.Status)
	}
}

// TestExecuteBatch_UnifiedUpdateStatusOnly tests update operations with only status field
func TestExecuteBatch_UnifiedUpdateWithStatus(t *testing.T) {
	tl := NewTaskList("Test Tasks")
	tl.AddTask("", "Parent task", "")
	tl.AddTask("1", "Child 1", "")
	tl.AddTask("1", "Child 2", "")

	// Test unified update operation with status field
	ops := []Operation{
		{
			Type:   "update",
			ID:     "1",
			Status: StatusPtr(InProgress),
			Title:  "Updated Parent",
		},
		{
			Type:       "update",
			ID:         "1.1",
			Status:     StatusPtr(Completed),
			Details:    []string{"Detail 1", "Detail 2"},
			References: []string{"ref1.md"},
		},
		{
			Type:   "update",
			ID:     "1.2",
			Status: StatusPtr(Completed),
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Verify parent update worked (title and status)
	parent := tl.FindTask("1")
	if parent.Title != "Updated Parent" {
		t.Errorf("Expected title 'Updated Parent', got '%s'", parent.Title)
	}
	// Parent should be auto-completed since both children are completed
	if parent.Status != Completed {
		t.Errorf("Expected parent status Completed (auto-completed), got %v", parent.Status)
	}

	// Verify child 1 update worked (status, details, references)
	child1 := tl.FindTask("1.1")
	if child1.Status != Completed {
		t.Errorf("Expected child1 status Completed, got %v", child1.Status)
	}
	if len(child1.Details) != 2 {
		t.Errorf("Expected 2 details, got %d", len(child1.Details))
	}
	if len(child1.References) != 1 {
		t.Errorf("Expected 1 reference, got %d", len(child1.References))
	}

	// Parent should be in auto-completed list
	if len(response.AutoCompleted) != 1 {
		t.Errorf("Expected 1 auto-completed task, got %d", len(response.AutoCompleted))
	}
	if len(response.AutoCompleted) > 0 && response.AutoCompleted[0] != "1" {
		t.Errorf("Expected auto-completed task '1', got '%s'", response.AutoCompleted[0])
	}
}

func TestExecuteBatch_AutoCompletePartialCompletion(t *testing.T) {
	tl := NewTaskList("Test Tasks")

	// Set up hierarchy with multiple children
	tl.AddTask("", "Parent task", "")
	tl.AddTask("1", "Child 1", "")
	tl.AddTask("1", "Child 2", "")
	tl.AddTask("1", "Child 3", "")

	// Mark first child as already complete
	tl.UpdateStatus("1.1", Completed)

	// Complete only one more child (not all)
	ops := []Operation{
		{
			Type:   "update",
			ID:     "1.2",
			Status: StatusPtr(Completed),
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Parent should NOT be auto-completed (child 3 is still pending)
	parent := tl.FindTask("1")
	if parent.Status == Completed {
		t.Error("Parent task should not be auto-completed when children remain incomplete")
	}

	// No tasks should be auto-completed
	if len(response.AutoCompleted) != 0 {
		t.Errorf("Expected 0 auto-completed tasks, got %d", len(response.AutoCompleted))
	}
}

func TestExecuteBatch_AutoCompleteMultiLevel(t *testing.T) {
	tl := NewTaskList("Test Tasks")

	// Set up multi-level hierarchy
	tl.AddTask("", "Root task", "")
	tl.AddTask("1", "Level 1 task", "")
	tl.AddTask("1.1", "Level 2 task 1", "")
	tl.AddTask("1.1", "Level 2 task 2", "")

	// Complete all level 2 tasks in batch
	ops := []Operation{
		{
			Type:   "update",
			ID:     "1.1.1",
			Status: StatusPtr(Completed),
		},
		{
			Type:   "update",
			ID:     "1.1.2",
			Status: StatusPtr(Completed),
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Both parent and grandparent should be auto-completed
	parent := tl.FindTask("1.1")
	if parent.Status != Completed {
		t.Errorf("Expected parent task to be auto-completed, but status is %v", parent.Status)
	}

	grandparent := tl.FindTask("1")
	if grandparent.Status != Completed {
		t.Errorf("Expected grandparent task to be auto-completed, but status is %v", grandparent.Status)
	}

	// Root should also be completed
	root := tl.FindTask("1")
	if root.Status != Completed {
		t.Errorf("Expected root task to be auto-completed, but status is %v", root.Status)
	}

	// Verify auto-completed tasks are tracked
	if len(response.AutoCompleted) != 2 {
		t.Errorf("Expected 2 auto-completed tasks, got %d", len(response.AutoCompleted))
	}
}

func TestExecuteBatch_AutoCompleteSimpleHierarchy(t *testing.T) {
	tl := NewTaskList("Test Tasks")

	// Set up a simple parent-child hierarchy
	tl.AddTask("", "Parent task", "")
	tl.AddTask("1", "Child 1", "")
	tl.AddTask("1", "Child 2", "")

	// Complete all children in batch
	ops := []Operation{
		{
			Type:   "update",
			ID:     "1.1",
			Status: StatusPtr(Completed),
		},
		{
			Type:   "update",
			ID:     "1.2",
			Status: StatusPtr(Completed),
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Parent should be auto-completed
	parent := tl.FindTask("1")
	if parent.Status != Completed {
		t.Errorf("Expected parent task to be auto-completed, but status is %v", parent.Status)
	}

	// Check that response indicates auto-completion
	if response.Applied != 2 {
		t.Errorf("Expected 2 applied operations, got %d", response.Applied)
	}

	// Verify auto-completed tasks are tracked
	if len(response.AutoCompleted) != 1 {
		t.Errorf("Expected 1 auto-completed task, got %d", len(response.AutoCompleted))
	}
	if len(response.AutoCompleted) > 0 && response.AutoCompleted[0] != "1" {
		t.Errorf("Expected auto-completed task ID '1', got '%s'", response.AutoCompleted[0])
	}
}
