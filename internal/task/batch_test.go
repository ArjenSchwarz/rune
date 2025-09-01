package task

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

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

func TestExecuteBatch_ValidationFailures(t *testing.T) {
	tl := NewTaskList("Test Tasks")
	tl.AddTask("", "Existing task", "")

	tests := map[string]struct {
		ops     []Operation
		wantErr bool
	}{
		"empty title": {
			ops: []Operation{
				{Type: "add", Title: ""},
			},
			wantErr: true,
		},
		"title too long": {
			ops: []Operation{
				{Type: "add", Title: string(make([]byte, 501))},
			},
			wantErr: true,
		},
		"parent not found": {
			ops: []Operation{
				{Type: "add", Parent: "999", Title: "Child"},
			},
			wantErr: true,
		},
		"remove nonexistent": {
			ops: []Operation{
				{Type: "remove", ID: "999"},
			},
			wantErr: true,
		},
		"update nonexistent": {
			ops: []Operation{
				{Type: "update", ID: "999", Status: StatusPtr(Completed)},
			},
			wantErr: true,
		},
		"unknown operation": {
			ops: []Operation{
				{Type: "unknown", ID: "1"},
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			response, err := tl.ExecuteBatch(tc.ops, false)
			if err != nil {
				t.Fatalf("ExecuteBatch returned error: %v", err)
			}

			if tc.wantErr && response.Success {
				t.Error("Expected validation failure, but got success")
			}
			if !tc.wantErr && !response.Success {
				t.Errorf("Expected success, but got errors: %v", response.Errors)
			}
		})
	}
}

func TestExecuteBatch_AtomicFailure(t *testing.T) {
	tl := NewTaskList("Test Tasks")
	tl.AddTask("", "Existing task", "")

	// Mix of valid and invalid operations
	ops := []Operation{
		{
			Type:  "add",
			Title: "Valid task",
		},
		{
			Type: "remove",
			ID:   "999", // This should fail
		},
		{
			Type:   "update",
			ID:     "1",
			Status: StatusPtr(Completed),
		},
	}

	initialTaskCount := len(tl.Tasks)

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch returned error: %v", err)
	}

	// Should fail validation and not apply any operations
	if response.Success {
		t.Error("Expected batch to fail due to invalid operation")
	}
	if response.Applied != 0 {
		t.Errorf("Expected 0 applied operations, got %d", response.Applied)
	}
	if len(tl.Tasks) != initialTaskCount {
		t.Errorf("Expected task count to remain %d, got %d", initialTaskCount, len(tl.Tasks))
	}
}

func TestExecuteBatch_DryRun(t *testing.T) {
	tl := NewTaskList("Test Tasks")
	tl.AddTask("", "Existing task", "")

	ops := []Operation{
		{
			Type:  "add",
			Title: "New task",
		},
		{
			Type:   "update",
			ID:     "1",
			Status: StatusPtr(Completed),
		},
	}

	initialTaskCount := len(tl.Tasks)
	initialStatus := tl.Tasks[0].Status

	response, err := tl.ExecuteBatch(ops, true)
	if err != nil {
		t.Fatalf("ExecuteBatch dry-run failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected dry-run success, got errors: %v", response.Errors)
	}
	if response.Applied != 2 {
		t.Errorf("Expected 2 applied operations in dry-run, got %d", response.Applied)
	}
	if response.Preview == "" {
		t.Error("Expected preview content in dry-run")
	}

	// Original should be unchanged
	if len(tl.Tasks) != initialTaskCount {
		t.Errorf("Expected task count to remain %d, got %d", initialTaskCount, len(tl.Tasks))
	}
	if tl.Tasks[0].Status != initialStatus {
		t.Errorf("Expected status to remain %v, got %v", initialStatus, tl.Tasks[0].Status)
	}
}

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

func TestBatchRequest_JSONSerialization(t *testing.T) {
	req := BatchRequest{
		File: "tasks.md",
		Operations: []Operation{
			{
				Type:  "add",
				Title: "New task",
			},
			{
				Type:   "update",
				ID:     "1",
				Status: StatusPtr(Completed),
			},
		},
		DryRun: true,
	}

	// Test JSON marshaling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	// Test JSON unmarshaling
	var req2 BatchRequest
	if err := json.Unmarshal(data, &req2); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Verify round-trip
	if req2.File != req.File {
		t.Errorf("File mismatch: %s vs %s", req2.File, req.File)
	}
	if len(req2.Operations) != len(req.Operations) {
		t.Errorf("Operations count mismatch: %d vs %d", len(req2.Operations), len(req.Operations))
	}
	if req2.DryRun != req.DryRun {
		t.Errorf("DryRun mismatch: %v vs %v", req2.DryRun, req.DryRun)
	}
}

func TestBatchResponse_JSONSerialization(t *testing.T) {
	resp := BatchResponse{
		Success: false,
		Applied: 5,
		Errors:  []string{"error 1", "error 2"},
		Preview: "# Test\n- [ ] 1. Task",
	}

	// Test JSON marshaling
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	// Test JSON unmarshaling
	var resp2 BatchResponse
	if err := json.Unmarshal(data, &resp2); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Verify round-trip
	if resp2.Success != resp.Success {
		t.Errorf("Success mismatch: %v vs %v", resp2.Success, resp.Success)
	}
	if resp2.Applied != resp.Applied {
		t.Errorf("Applied mismatch: %d vs %d", resp2.Applied, resp.Applied)
	}
	if len(resp2.Errors) != len(resp.Errors) {
		t.Errorf("Errors count mismatch: %d vs %d", len(resp2.Errors), len(resp.Errors))
	}
}

func TestValidateOperation_EdgeCases(t *testing.T) {
	tl := NewTaskList("Test")
	tl.AddTask("", "Task 1", "")

	tests := map[string]struct {
		op      Operation
		wantErr bool
	}{
		"add with empty type": {
			op:      Operation{Type: "", Title: "Test"},
			wantErr: true,
		},
		"case insensitive type": {
			op:      Operation{Type: "ADD", Title: "Test"},
			wantErr: false,
		},
		"update with empty title": {
			op:      Operation{Type: "update", ID: "1", Title: ""},
			wantErr: false, // Empty title means no update to title
		},
		"update with invalid status": {
			op:      Operation{Type: "update", ID: "1", Status: StatusPtr(Status(99))},
			wantErr: true,
		},
		"remove missing id": {
			op:      Operation{Type: "remove"},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateOperation(tl, tc.op)
			if tc.wantErr && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// Tests for batch operations with auto-completion functionality
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

func TestExecuteBatch_AutoCompleteWithMixedOperations(t *testing.T) {
	tl := NewTaskList("Test Tasks")

	// Set up initial structure
	tl.AddTask("", "Task 1", "")
	tl.AddTask("1", "Task 1.1", "")
	tl.AddTask("", "Task 2", "")
	tl.AddTask("2", "Task 2.1", "")
	tl.AddTask("2", "Task 2.2", "")

	// Mixed operations including completions that trigger auto-complete
	ops := []Operation{
		{
			Type:   "update",
			ID:     "1.1",
			Status: StatusPtr(Completed),
		},
		{
			Type:   "add",
			Parent: "2",
			Title:  "Task 2.3",
		},
		{
			Type:   "update",
			ID:     "2.1",
			Status: StatusPtr(Completed),
		},
		{
			Type:   "update",
			ID:     "2.2",
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

	// Task 1 should be auto-completed (only child is complete)
	task1 := tl.FindTask("1")
	if task1.Status != Completed {
		t.Errorf("Expected Task 1 to be auto-completed, but status is %v", task1.Status)
	}

	// Task 2 should NOT be auto-completed (new child was added)
	task2 := tl.FindTask("2")
	if task2.Status == Completed {
		t.Error("Task 2 should not be auto-completed after adding a new child")
	}

	// Verify auto-completed count
	if len(response.AutoCompleted) != 1 {
		t.Errorf("Expected 1 auto-completed task, got %d", len(response.AutoCompleted))
	}
}

func TestExecuteBatch_AutoCompleteSameParentMultipleTimes(t *testing.T) {
	tl := NewTaskList("Test Tasks")

	// Set up hierarchy
	tl.AddTask("", "Parent task", "")
	tl.AddTask("1", "Child 1", "")
	tl.AddTask("1", "Child 2", "")
	tl.AddTask("1", "Child 3", "")

	// Complete all children in batch - should trigger parent completion only once
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
		{
			Type:   "update",
			ID:     "1.3",
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

	// Parent should be completed
	parent := tl.FindTask("1")
	if parent.Status != Completed {
		t.Errorf("Expected parent to be auto-completed, but status is %v", parent.Status)
	}

	// Should only report parent as auto-completed once
	if len(response.AutoCompleted) != 1 {
		t.Errorf("Expected 1 auto-completed task, got %d", len(response.AutoCompleted))
	}
	if len(response.AutoCompleted) > 0 && response.AutoCompleted[0] != "1" {
		t.Errorf("Expected auto-completed task ID '1', got '%s'", response.AutoCompleted[0])
	}
}

func TestExecuteBatch_AutoCompleteDryRun(t *testing.T) {
	tl := NewTaskList("Test Tasks")

	// Set up hierarchy
	tl.AddTask("", "Parent task", "")
	tl.AddTask("1", "Child 1", "")
	tl.AddTask("1", "Child 2", "")

	// Complete all children in dry-run mode
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

	response, err := tl.ExecuteBatch(ops, true)
	if err != nil {
		t.Fatalf("ExecuteBatch dry-run failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// In dry-run, original should be unchanged
	parent := tl.FindTask("1")
	if parent.Status == Completed {
		t.Error("Parent should not be modified in dry-run mode")
	}

	child1 := tl.FindTask("1.1")
	if child1.Status == Completed {
		t.Error("Child 1 should not be modified in dry-run mode")
	}

	// Preview should show auto-completed tasks
	if response.Preview == "" {
		t.Error("Expected preview content in dry-run")
	}

	// Auto-completed tasks should still be reported in dry-run
	if len(response.AutoCompleted) != 1 {
		t.Errorf("Expected 1 auto-completed task in dry-run, got %d", len(response.AutoCompleted))
	}
}

func TestExecuteBatch_AutoCompleteErrorHandling(t *testing.T) {
	tl := NewTaskList("Test Tasks")

	// Set up hierarchy
	tl.AddTask("", "Parent task", "")
	tl.AddTask("1", "Child 1", "")
	tl.AddTask("1", "Child 2", "")

	// Include an invalid operation that should fail validation
	ops := []Operation{
		{
			Type:   "update",
			ID:     "1.1",
			Status: StatusPtr(Completed),
		},
		{
			Type:   "update",
			ID:     "999", // Non-existent task
			Status: StatusPtr(Completed),
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch returned error: %v", err)
	}

	// Batch should fail due to invalid operation
	if response.Success {
		t.Error("Expected batch to fail due to invalid operation")
	}

	// Nothing should be modified (atomic failure)
	child1 := tl.FindTask("1.1")
	if child1.Status == Completed {
		t.Error("Child 1 should not be modified after batch failure")
	}

	parent := tl.FindTask("1")
	if parent.Status == Completed {
		t.Error("Parent should not be auto-completed after batch failure")
	}

	// No auto-completions should be reported
	if len(response.AutoCompleted) != 0 {
		t.Errorf("Expected 0 auto-completed tasks after failure, got %d", len(response.AutoCompleted))
	}
}

func TestExecuteBatch_AutoCompleteComplexScenario(t *testing.T) {
	tl := NewTaskList("Complex Test")

	// Create a complex hierarchy
	// 1. Project (parent)
	//   1.1. Setup
	//     1.1.1. Install deps
	//     1.1.2. Configure DB
	//   1.2. Development
	//     1.2.1. Feature A
	//     1.2.2. Feature B
	//   1.3. Testing
	tl.AddTask("", "Project", "")
	tl.AddTask("1", "Setup", "")
	tl.AddTask("1.1", "Install deps", "")
	tl.AddTask("1.1", "Configure DB", "")
	tl.AddTask("1", "Development", "")
	tl.AddTask("1.2", "Feature A", "")
	tl.AddTask("1.2", "Feature B", "")
	tl.AddTask("1", "Testing", "")

	// Complete setup subtasks and one dev task
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
		{
			Type:   "update",
			ID:     "1.2.1",
			Status: StatusPtr(Completed),
		},
		{
			Type:   "update",
			ID:     "1.3",
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

	// Setup task should be auto-completed
	setup := tl.FindTask("1.1")
	if setup.Status != Completed {
		t.Errorf("Expected Setup task to be auto-completed, but status is %v", setup.Status)
	}

	// Development task should NOT be auto-completed (Feature B still pending)
	dev := tl.FindTask("1.2")
	if dev.Status == Completed {
		t.Error("Development task should not be auto-completed with pending subtask")
	}

	// Project should NOT be auto-completed (Development still has pending work)
	project := tl.FindTask("1")
	if project.Status == Completed {
		t.Error("Project should not be auto-completed with incomplete subtasks")
	}

	// Only Setup should be auto-completed
	if len(response.AutoCompleted) != 1 {
		t.Errorf("Expected 1 auto-completed task, got %d", len(response.AutoCompleted))
	}
	if len(response.AutoCompleted) > 0 && response.AutoCompleted[0] != "1.1" {
		t.Errorf("Expected auto-completed task ID '1.1', got '%s'", response.AutoCompleted[0])
	}
}

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

func TestExecuteBatch_MixedUpdateStatusOperations(t *testing.T) {
	tl := NewTaskList("Test Tasks")
	tl.AddTask("", "Task 1", "")
	tl.AddTask("", "Task 2", "")
	tl.AddTask("", "Task 3", "")

	// Mix of valid update operations and invalid update_status
	ops := []Operation{
		{
			Type:   "update",
			ID:     "1",
			Status: StatusPtr(InProgress),
		},
		{
			Type:   "update_status", // This should fail
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

	// Batch should fail due to update_status operation
	if response.Success {
		t.Error("Expected batch to fail due to update_status operation")
	}

	// Should have error for operation 2
	if len(response.Errors) == 0 {
		t.Error("Expected error for update_status operation")
	} else if !strings.Contains(response.Errors[0], "operation 2") {
		t.Errorf("Expected error to reference operation 2, got: %s", response.Errors[0])
	}

	// No operations should be applied (atomic failure)
	if response.Applied != 0 {
		t.Errorf("Expected 0 applied operations due to failure, got %d", response.Applied)
	}

	// All tasks should remain unchanged
	task1 := tl.FindTask("1")
	if task1.Status != Pending {
		t.Errorf("Task 1 should remain Pending, got %v", task1.Status)
	}
	task2 := tl.FindTask("2")
	if task2.Status != Pending {
		t.Errorf("Task 2 should remain Pending, got %v", task2.Status)
	}
	task3 := tl.FindTask("3")
	if task3.Status != Pending {
		t.Errorf("Task 3 should remain Pending, got %v", task3.Status)
	}
}

// TestExecuteBatch_UnifiedUpdateStatusOnly tests update operations with only status field
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
