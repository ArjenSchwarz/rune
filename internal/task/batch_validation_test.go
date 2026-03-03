package task

import (
	"encoding/json"
	"testing"
)

func TestBatchRequest_RequirementsFile(t *testing.T) {
	req := BatchRequest{
		File:             "tasks.md",
		RequirementsFile: "specs/requirements.md",
		Operations: []Operation{
			{
				Type:         "add",
				Title:        "Task",
				Requirements: []string{"1.1"},
			},
		},
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal BatchRequest: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled BatchRequest
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal BatchRequest: %v", err)
	}

	if unmarshaled.RequirementsFile != "specs/requirements.md" {
		t.Errorf("Expected requirements_file 'specs/requirements.md', got '%s'", unmarshaled.RequirementsFile)
	}
}
func TestExecuteBatch_AtomicBehaviorWithInvalidRequirements(t *testing.T) {
	tl := NewTaskList("Test Tasks")
	tl.AddTask("", "Existing task", "")

	// Mix of valid operation and operation with invalid requirements
	ops := []Operation{
		{
			Type:  "add",
			Title: "Valid task",
		},
		{
			Type:         "add",
			Title:        "Task with invalid requirements",
			Requirements: []string{"invalid-id"},
		},
	}

	initialTaskCount := len(tl.Tasks)

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch returned error: %v", err)
	}

	// Should fail validation and not apply any operations
	if response.Success {
		t.Error("Expected batch to fail due to invalid requirements")
	}
	if response.Applied != 0 {
		t.Errorf("Expected 0 applied operations, got %d", response.Applied)
	}
	if len(tl.Tasks) != initialTaskCount {
		t.Errorf("Expected task count to remain %d, got %d", initialTaskCount, len(tl.Tasks))
	}
}

func TestExecuteBatch_RequirementsValidation(t *testing.T) {
	tests := map[string]struct {
		ops     []Operation
		wantErr bool
	}{
		"invalid requirement ID in add": {
			ops: []Operation{
				{
					Type:         "add",
					Title:        "Task",
					Requirements: []string{"invalid"},
				},
			},
			wantErr: true,
		},
		"invalid requirement ID in update": {
			ops: []Operation{
				{
					Type:         "update",
					ID:           "1",
					Requirements: []string{"abc.xyz"},
				},
			},
			wantErr: true,
		},
		"valid hierarchical requirements": {
			ops: []Operation{
				{
					Type:         "add",
					Title:        "Task",
					Requirements: []string{"1", "1.1", "1.2.3", "10.20.30.40"},
				},
			},
			wantErr: false,
		},
		"mixed valid and invalid requirements": {
			ops: []Operation{
				{
					Type:         "add",
					Title:        "Task",
					Requirements: []string{"1.1", "invalid", "2.3"},
				},
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create fresh task list for each test
			testTL := NewTaskList("Test Tasks")
			testTL.AddTask("", "Existing task", "")

			response, err := testTL.ExecuteBatch(tc.ops, false)
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

// TestExecuteBatch_UpdateStatusNotAppliedBeforeInvalidDetails verifies that a batch
// update with status + invalid details fails atomically. The bug was that status was
// applied via UpdateStatus before UpdateTask/UpdateTaskWithOptions validated details,
// so a failure on details would leave the status partially applied.
func TestExecuteBatch_UpdateStatusNotAppliedBeforeInvalidDetails(t *testing.T) {
	tests := map[string]struct {
		ops  []Operation
		desc string
	}{
		"status + overlong detail": {
			ops: []Operation{
				{
					Type:    "update",
					ID:      "1",
					Status:  StatusPtr(Completed),
					Details: []string{string(make([]byte, MaxDetailLength+1))},
				},
			},
			desc: "detail exceeding max length",
		},
		"status + overlong reference": {
			ops: []Operation{
				{
					Type:       "update",
					ID:         "1",
					Status:     StatusPtr(Completed),
					References: []string{string(make([]byte, 501))},
				},
			},
			desc: "reference exceeding max length",
		},
		"status + detail with control chars": {
			ops: []Operation{
				{
					Type:    "update",
					ID:      "1",
					Status:  StatusPtr(InProgress),
					Details: []string{"has \x00 null byte"},
				},
			},
			desc: "detail containing null byte",
		},
		"status + reference with control chars": {
			ops: []Operation{
				{
					Type:       "update",
					ID:         "1",
					Status:     StatusPtr(InProgress),
					References: []string{"has \x00 null byte"},
				},
			},
			desc: "reference containing null byte",
		},
		"status + valid detail + overlong reference": {
			ops: []Operation{
				{
					Type:       "update",
					ID:         "1",
					Status:     StatusPtr(Completed),
					Details:    []string{"valid detail"},
					References: []string{string(make([]byte, 501))},
				},
			},
			desc: "valid detail but overlong reference",
		},
		"status + extended fields + overlong detail": {
			ops: []Operation{
				{
					Type:    "update",
					ID:      "1",
					Status:  StatusPtr(Completed),
					Details: []string{string(make([]byte, MaxDetailLength+1))},
					Owner:   strPtr("agent-1"),
				},
			},
			desc: "overlong detail with extended fields (owner)",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl := NewTaskList("Test Tasks")
			tl.AddTask("", "Existing task", "")

			originalStatus := tl.Tasks[0].Status

			response, err := tl.ExecuteBatch(tc.ops, false)
			if err != nil {
				t.Fatalf("ExecuteBatch returned unexpected error: %v", err)
			}

			// Should fail validation
			if response.Success {
				t.Errorf("Expected batch to fail due to %s", tc.desc)
			}

			// Original task status must NOT have changed (atomicity)
			task := tl.FindTask("1")
			if task.Status != originalStatus {
				t.Errorf("Status was partially applied before failure on %s: expected %v, got %v",
					tc.desc, originalStatus, task.Status)
			}

			// No operations should be applied
			if response.Applied != 0 {
				t.Errorf("Expected 0 applied operations, got %d", response.Applied)
			}
		})
	}
}

// strPtr returns a pointer to the given string value for use in Operation structs
func strPtr(s string) *string {
	return &s
}

// TestValidateOperation_RejectsInvalidDetailsAndReferences verifies that validateOperation
// catches invalid detail/reference content upfront for update operations, rather than
// deferring validation to UpdateTask/UpdateTaskWithOptions where status may already be applied.
func TestValidateOperation_RejectsInvalidDetailsAndReferences(t *testing.T) {
	tl := NewTaskList("Test")
	tl.AddTask("", "Task 1", "")

	tests := map[string]struct {
		op      Operation
		wantErr bool
	}{
		"update with overlong detail": {
			op: Operation{
				Type:    "update",
				ID:      "1",
				Details: []string{string(make([]byte, MaxDetailLength+1))},
			},
			wantErr: true,
		},
		"update with overlong reference": {
			op: Operation{
				Type:       "update",
				ID:         "1",
				References: []string{string(make([]byte, 501))},
			},
			wantErr: true,
		},
		"update with null byte in detail": {
			op: Operation{
				Type:    "update",
				ID:      "1",
				Details: []string{"has \x00 null"},
			},
			wantErr: true,
		},
		"update with null byte in reference": {
			op: Operation{
				Type:       "update",
				ID:         "1",
				References: []string{"has \x00 null"},
			},
			wantErr: true,
		},
		"update with valid detail": {
			op: Operation{
				Type:    "update",
				ID:      "1",
				Details: []string{"valid detail"},
			},
			wantErr: false,
		},
		"update with valid reference": {
			op: Operation{
				Type:       "update",
				ID:         "1",
				References: []string{"valid-ref.md"},
			},
			wantErr: false,
		},
		"add with overlong detail": {
			op: Operation{
				Type:    "add",
				Title:   "New task",
				Details: []string{string(make([]byte, MaxDetailLength+1))},
			},
			wantErr: true,
		},
		"add with overlong reference": {
			op: Operation{
				Type:       "add",
				Title:      "New task",
				References: []string{string(make([]byte, 501))},
			},
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

// TestApplyUpdateOperation_NoPartialStatusApply verifies that applyUpdateOperation
// does not apply a status change if other fields (details, references) are invalid.
// This is a defense-in-depth check -- even if validateOperation is strengthened,
// the apply function itself should not partially mutate the task.
func TestApplyUpdateOperation_NoPartialStatusApply(t *testing.T) {
	tests := map[string]struct {
		op   Operation
		desc string
	}{
		"status + overlong detail without extended fields": {
			op: Operation{
				Type:    "update",
				ID:      "1",
				Status:  StatusPtr(Completed),
				Details: []string{string(make([]byte, MaxDetailLength+1))},
			},
			desc: "overlong detail on standard path",
		},
		"status + overlong reference without extended fields": {
			op: Operation{
				Type:       "update",
				ID:         "1",
				Status:     StatusPtr(Completed),
				References: []string{string(make([]byte, 501))},
			},
			desc: "overlong reference on standard path",
		},
		"status + overlong detail with extended fields": {
			op: Operation{
				Type:    "update",
				ID:      "1",
				Status:  StatusPtr(Completed),
				Details: []string{string(make([]byte, MaxDetailLength+1))},
				Owner:   strPtr("agent-1"),
			},
			desc: "overlong detail on extended path",
		},
		"status + overlong reference with extended fields": {
			op: Operation{
				Type:       "update",
				ID:         "1",
				Status:     StatusPtr(Completed),
				References: []string{string(make([]byte, 501))},
				Owner:      strPtr("agent-1"),
			},
			desc: "overlong reference on extended path",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl := NewTaskList("Test Tasks")
			tl.AddTask("", "Existing task", "")

			originalStatus := tl.Tasks[0].Status

			err := applyUpdateOperation(tl, tc.op)
			if err == nil {
				t.Fatalf("Expected error for %s, got nil", tc.desc)
			}

			// Status must NOT have been partially applied
			task := tl.FindTask("1")
			if task.Status != originalStatus {
				t.Errorf("Status was partially applied before failure on %s: expected %v, got %v",
					tc.desc, originalStatus, task.Status)
			}
		})
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
