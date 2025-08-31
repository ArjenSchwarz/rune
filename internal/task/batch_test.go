package task

import (
	"encoding/json"
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
			Type:   "update_status",
			ID:     "1",
			Status: InProgress,
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
	tl.AddTask("", "Existing task")

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
		"update_status nonexistent": {
			ops: []Operation{
				{Type: "update_status", ID: "999", Status: Completed},
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
	tl.AddTask("", "Existing task")

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
			Type:   "update_status",
			ID:     "1",
			Status: Completed,
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
	tl.AddTask("", "Existing task")

	ops := []Operation{
		{
			Type:  "add",
			Title: "New task",
		},
		{
			Type:   "update_status",
			ID:     "1",
			Status: Completed,
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
	tl.AddTask("", "Task 1")
	tl.AddTask("1", "Task 1.1")
	tl.AddTask("1", "Task 1.2")
	tl.AddTask("", "Task 2")

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
			Type:   "update_status",
			ID:     "2",
			Status: InProgress,
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
				Type:   "update_status",
				ID:     "1",
				Status: Completed,
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
	tl.AddTask("", "Task 1")

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
		"update_status with invalid status": {
			op:      Operation{Type: "update_status", ID: "1", Status: Status(99)},
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
	tl.AddTask("", "Parent task")
	tl.AddTask("1", "Child 1")
	tl.AddTask("1", "Child 2")

	// Complete all children in batch
	ops := []Operation{
		{
			Type:   "update_status",
			ID:     "1.1",
			Status: Completed,
		},
		{
			Type:   "update_status",
			ID:     "1.2",
			Status: Completed,
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
	tl.AddTask("", "Root task")
	tl.AddTask("1", "Level 1 task")
	tl.AddTask("1.1", "Level 2 task 1")
	tl.AddTask("1.1", "Level 2 task 2")

	// Complete all level 2 tasks in batch
	ops := []Operation{
		{
			Type:   "update_status",
			ID:     "1.1.1",
			Status: Completed,
		},
		{
			Type:   "update_status",
			ID:     "1.1.2",
			Status: Completed,
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
	tl.AddTask("", "Parent task")
	tl.AddTask("1", "Child 1")
	tl.AddTask("1", "Child 2")
	tl.AddTask("1", "Child 3")

	// Mark first child as already complete
	tl.UpdateStatus("1.1", Completed)

	// Complete only one more child (not all)
	ops := []Operation{
		{
			Type:   "update_status",
			ID:     "1.2",
			Status: Completed,
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
	tl.AddTask("", "Task 1")
	tl.AddTask("1", "Task 1.1")
	tl.AddTask("", "Task 2")
	tl.AddTask("2", "Task 2.1")
	tl.AddTask("2", "Task 2.2")

	// Mixed operations including completions that trigger auto-complete
	ops := []Operation{
		{
			Type:   "update_status",
			ID:     "1.1",
			Status: Completed,
		},
		{
			Type:   "add",
			Parent: "2",
			Title:  "Task 2.3",
		},
		{
			Type:   "update_status",
			ID:     "2.1",
			Status: Completed,
		},
		{
			Type:   "update_status",
			ID:     "2.2",
			Status: Completed,
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
	tl.AddTask("", "Parent task")
	tl.AddTask("1", "Child 1")
	tl.AddTask("1", "Child 2")
	tl.AddTask("1", "Child 3")

	// Complete all children in batch - should trigger parent completion only once
	ops := []Operation{
		{
			Type:   "update_status",
			ID:     "1.1",
			Status: Completed,
		},
		{
			Type:   "update_status",
			ID:     "1.2",
			Status: Completed,
		},
		{
			Type:   "update_status",
			ID:     "1.3",
			Status: Completed,
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
	tl.AddTask("", "Parent task")
	tl.AddTask("1", "Child 1")
	tl.AddTask("1", "Child 2")

	// Complete all children in dry-run mode
	ops := []Operation{
		{
			Type:   "update_status",
			ID:     "1.1",
			Status: Completed,
		},
		{
			Type:   "update_status",
			ID:     "1.2",
			Status: Completed,
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
	tl.AddTask("", "Parent task")
	tl.AddTask("1", "Child 1")
	tl.AddTask("1", "Child 2")

	// Include an invalid operation that should fail validation
	ops := []Operation{
		{
			Type:   "update_status",
			ID:     "1.1",
			Status: Completed,
		},
		{
			Type:   "update_status",
			ID:     "999", // Non-existent task
			Status: Completed,
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
	tl.AddTask("", "Project")
	tl.AddTask("1", "Setup")
	tl.AddTask("1.1", "Install deps")
	tl.AddTask("1.1", "Configure DB")
	tl.AddTask("1", "Development")
	tl.AddTask("1.2", "Feature A")
	tl.AddTask("1.2", "Feature B")
	tl.AddTask("1", "Testing")

	// Complete setup subtasks and one dev task
	ops := []Operation{
		{
			Type:   "update_status",
			ID:     "1.1.1",
			Status: Completed,
		},
		{
			Type:   "update_status",
			ID:     "1.1.2",
			Status: Completed,
		},
		{
			Type:   "update_status",
			ID:     "1.2.1",
			Status: Completed,
		},
		{
			Type:   "update_status",
			ID:     "1.3",
			Status: Completed,
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
