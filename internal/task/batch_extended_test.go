package task

import (
	"strings"
	"testing"
)

// TestExecuteBatch_AddWithStream tests batch add operations with stream assignment
func TestExecuteBatch_AddWithStream(t *testing.T) {
	tl := NewTaskList("Stream Test")

	// First add a task to get a stable ID for dependency testing
	_, err := tl.AddTaskWithOptions("", "Prerequisite task", AddOptions{})
	if err != nil {
		t.Fatalf("Failed to add prerequisite task: %v", err)
	}

	ops := []Operation{
		{
			Type:   "add",
			Title:  "Stream 2 task",
			Stream: intPtr(2),
		},
		{
			Type:   "add",
			Title:  "Stream 3 task",
			Stream: intPtr(3),
		},
		{
			Type:  "add",
			Title: "Default stream task",
			// No stream specified - should default to 1
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

	// Verify stream assignments
	task2 := findTaskByTitleRecursive(tl, "Stream 2 task")
	if task2 == nil {
		t.Fatal("Stream 2 task not found")
	}
	if task2.Stream != 2 {
		t.Errorf("Expected stream 2, got %d", task2.Stream)
	}

	task3 := findTaskByTitleRecursive(tl, "Stream 3 task")
	if task3 == nil {
		t.Fatal("Stream 3 task not found")
	}
	if task3.Stream != 3 {
		t.Errorf("Expected stream 3, got %d", task3.Stream)
	}

	defaultTask := findTaskByTitleRecursive(tl, "Default stream task")
	if defaultTask == nil {
		t.Fatal("Default stream task not found")
	}
	// Stream should be 0 (not explicitly set) but effective stream is 1
	if GetEffectiveStream(defaultTask) != 1 {
		t.Errorf("Expected effective stream 1, got %d", GetEffectiveStream(defaultTask))
	}
}

// TestExecuteBatch_AddWithBlockedBy tests batch add operations with blocked-by dependencies
func TestExecuteBatch_AddWithBlockedBy(t *testing.T) {
	tl := NewTaskList("Dependency Test")

	// Add prerequisite tasks with stable IDs
	id1, err := tl.AddTaskWithOptions("", "Task A", AddOptions{})
	if err != nil {
		t.Fatalf("Failed to add Task A: %v", err)
	}
	id2, err := tl.AddTaskWithOptions("", "Task B", AddOptions{})
	if err != nil {
		t.Fatalf("Failed to add Task B: %v", err)
	}

	// Add task with dependencies via batch
	ops := []Operation{
		{
			Type:      "add",
			Title:     "Dependent task",
			BlockedBy: []string{id1, id2}, // Hierarchical IDs
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Verify dependent task has blocked-by references
	depTask := findTaskByTitleRecursive(tl, "Dependent task")
	if depTask == nil {
		t.Fatal("Dependent task not found")
	}

	if len(depTask.BlockedBy) != 2 {
		t.Errorf("Expected 2 blocked-by references, got %d", len(depTask.BlockedBy))
	}

	// The BlockedBy field should contain stable IDs (resolved from hierarchical IDs)
	taskA := tl.FindTask(id1)
	taskB := tl.FindTask(id2)
	if taskA == nil || taskB == nil {
		t.Fatal("Could not find prerequisite tasks")
	}

	// Verify the stable IDs are in the BlockedBy list
	foundA := false
	foundB := false
	for _, stableID := range depTask.BlockedBy {
		if stableID == taskA.StableID {
			foundA = true
		}
		if stableID == taskB.StableID {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Errorf("BlockedBy should contain stable IDs of Task A and B, got %v", depTask.BlockedBy)
	}
}

// TestExecuteBatch_AddWithOwner tests batch add operations with owner assignment
func TestExecuteBatch_AddWithOwner(t *testing.T) {
	tl := NewTaskList("Owner Test")

	ops := []Operation{
		{
			Type:  "add",
			Title: "Owned task",
			Owner: stringPtr("agent-1"),
		},
		{
			Type:  "add",
			Title: "Unowned task",
			// No owner specified
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Verify owner assignments
	ownedTask := findTaskByTitleRecursive(tl, "Owned task")
	if ownedTask == nil {
		t.Fatal("Owned task not found")
	}
	if ownedTask.Owner != "agent-1" {
		t.Errorf("Expected owner 'agent-1', got '%s'", ownedTask.Owner)
	}

	unownedTask := findTaskByTitleRecursive(tl, "Unowned task")
	if unownedTask == nil {
		t.Fatal("Unowned task not found")
	}
	if unownedTask.Owner != "" {
		t.Errorf("Expected empty owner, got '%s'", unownedTask.Owner)
	}
}

// TestExecuteBatch_UpdateWithStream tests batch update operations with stream changes
func TestExecuteBatch_UpdateWithStream(t *testing.T) {
	tl := NewTaskList("Update Stream Test")

	// Add initial tasks
	tl.AddTaskWithOptions("", "Task 1", AddOptions{Stream: 1})
	tl.AddTaskWithOptions("", "Task 2", AddOptions{Stream: 2})

	ops := []Operation{
		{
			Type:   "update",
			ID:     "1",
			Stream: intPtr(3), // Change stream from 1 to 3
		},
		{
			Type:   "update",
			ID:     "2",
			Stream: intPtr(1), // Change stream from 2 to 1
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Verify stream updates
	task1 := tl.FindTask("1")
	if task1.Stream != 3 {
		t.Errorf("Task 1: expected stream 3, got %d", task1.Stream)
	}

	task2 := tl.FindTask("2")
	if task2.Stream != 1 {
		t.Errorf("Task 2: expected stream 1, got %d", task2.Stream)
	}
}

// TestExecuteBatch_UpdateWithBlockedBy tests batch update operations with dependency changes
func TestExecuteBatch_UpdateWithBlockedBy(t *testing.T) {
	tl := NewTaskList("Update Dependency Test")

	// Add tasks with stable IDs
	tl.AddTaskWithOptions("", "Task A", AddOptions{})
	tl.AddTaskWithOptions("", "Task B", AddOptions{})
	tl.AddTaskWithOptions("", "Task C", AddOptions{})

	// Update Task C to depend on A and B
	ops := []Operation{
		{
			Type:      "update",
			ID:        "3",
			BlockedBy: []string{"1", "2"}, // Hierarchical IDs
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Verify dependency update
	taskC := tl.FindTask("3")
	if len(taskC.BlockedBy) != 2 {
		t.Errorf("Expected 2 blocked-by references, got %d", len(taskC.BlockedBy))
	}
}

// TestExecuteBatch_UpdateWithOwner tests batch update operations with owner changes
func TestExecuteBatch_UpdateWithOwner(t *testing.T) {
	tl := NewTaskList("Update Owner Test")

	// Add initial task with owner
	tl.AddTaskWithOptions("", "Task 1", AddOptions{Owner: "agent-1"})

	ops := []Operation{
		{
			Type:  "update",
			ID:    "1",
			Owner: stringPtr("agent-2"), // Change owner
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Verify owner update
	task1 := tl.FindTask("1")
	if task1.Owner != "agent-2" {
		t.Errorf("Expected owner 'agent-2', got '%s'", task1.Owner)
	}
}

// TestExecuteBatch_UpdateWithRelease tests batch update operations with release flag
func TestExecuteBatch_UpdateWithRelease(t *testing.T) {
	tl := NewTaskList("Release Owner Test")

	// Add initial task with owner
	tl.AddTaskWithOptions("", "Task 1", AddOptions{Owner: "agent-1"})

	ops := []Operation{
		{
			Type:    "update",
			ID:      "1",
			Release: true, // Clear owner
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Verify owner was cleared
	task1 := tl.FindTask("1")
	if task1.Owner != "" {
		t.Errorf("Expected owner to be cleared, got '%s'", task1.Owner)
	}
}

// TestExecuteBatch_CycleDetection tests that batch operations detect circular dependencies
func TestExecuteBatch_CycleDetection(t *testing.T) {
	tl := NewTaskList("Cycle Detection Test")

	// Add initial tasks with stable IDs
	tl.AddTaskWithOptions("", "Task A", AddOptions{})
	tl.AddTaskWithOptions("", "Task B", AddOptions{})

	// Set up A depends on B
	taskA := tl.FindTask("1")
	taskB := tl.FindTask("2")
	taskA.BlockedBy = []string{taskB.StableID}

	// Try to make B depend on A (creates cycle)
	ops := []Operation{
		{
			Type:      "update",
			ID:        "2",
			BlockedBy: []string{"1"}, // B depends on A - creates A→B→A cycle
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch returned error: %v", err)
	}

	// Should fail due to circular dependency
	if response.Success {
		t.Error("Expected batch to fail due to circular dependency")
	}

	if len(response.Errors) == 0 {
		t.Error("Expected error message about circular dependency")
	} else if !strings.Contains(response.Errors[0], "circular dependency") {
		t.Errorf("Expected circular dependency error, got: %s", response.Errors[0])
	}
}

// TestExecuteBatch_AtomicityWithDependencyError tests that batch fails atomically on dependency errors
func TestExecuteBatch_AtomicityWithDependencyError(t *testing.T) {
	tl := NewTaskList("Atomicity Test")

	// Add initial tasks
	tl.AddTaskWithOptions("", "Task A", AddOptions{})
	tl.AddTaskWithOptions("", "Task B", AddOptions{})

	// Create a batch with valid operation followed by invalid dependency
	ops := []Operation{
		{
			Type:   "update",
			ID:     "1",
			Stream: intPtr(5), // Valid operation
		},
		{
			Type:      "update",
			ID:        "2",
			BlockedBy: []string{"99"}, // Invalid: task 99 doesn't exist
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch returned error: %v", err)
	}

	// Batch should fail
	if response.Success {
		t.Error("Expected batch to fail due to invalid dependency reference")
	}

	// First operation should NOT be applied (atomic failure)
	task1 := tl.FindTask("1")
	if task1.Stream == 5 {
		t.Error("Task 1 stream should not be updated due to atomic failure")
	}
}

// TestExecuteBatch_InvalidStreamValue tests validation of stream values
func TestExecuteBatch_InvalidStreamValue(t *testing.T) {
	tl := NewTaskList("Invalid Stream Test")

	// Add a task
	tl.AddTaskWithOptions("", "Task 1", AddOptions{})

	tests := map[string]struct {
		stream     int
		shouldFail bool
	}{
		"negative stream": {
			stream:     -1,
			shouldFail: true,
		},
		"zero stream": {
			stream:     0,
			shouldFail: false, // 0 means "not set", which is valid
		},
		"valid positive stream": {
			stream:     5,
			shouldFail: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset task
			tl.FindTask("1").Stream = 0

			ops := []Operation{
				{
					Type:   "update",
					ID:     "1",
					Stream: intPtr(tc.stream),
				},
			}

			response, err := tl.ExecuteBatch(ops, false)
			if err != nil {
				t.Fatalf("ExecuteBatch returned error: %v", err)
			}

			if tc.shouldFail && response.Success {
				t.Errorf("Expected batch to fail for stream value %d", tc.stream)
			}
			if !tc.shouldFail && !response.Success {
				t.Errorf("Expected batch to succeed for stream value %d, got errors: %v", tc.stream, response.Errors)
			}
		})
	}
}

// TestExecuteBatch_InvalidOwner tests validation of owner strings
func TestExecuteBatch_InvalidOwner(t *testing.T) {
	tl := NewTaskList("Invalid Owner Test")

	// Add a task
	tl.AddTaskWithOptions("", "Task 1", AddOptions{})

	// Try to set owner with newline (invalid)
	ops := []Operation{
		{
			Type:  "update",
			ID:    "1",
			Owner: stringPtr("agent\nwith-newline"),
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch returned error: %v", err)
	}

	if response.Success {
		t.Error("Expected batch to fail due to invalid owner with newline")
	}
}

// TestExecuteBatch_DependencyOnLegacyTask tests error when depending on task without stable ID
func TestExecuteBatch_DependencyOnLegacyTask(t *testing.T) {
	tl := NewTaskList("Legacy Task Test")

	// Add a legacy task (manually, without stable ID)
	tl.Tasks = append(tl.Tasks, Task{
		ID:     "1",
		Title:  "Legacy task",
		Status: Pending,
		// No StableID - this is a legacy task
	})

	// Add a new task with stable ID
	tl.AddTaskWithOptions("", "New task", AddOptions{})

	// Try to add dependency on legacy task
	ops := []Operation{
		{
			Type:      "update",
			ID:        "2",
			BlockedBy: []string{"1"}, // Task 1 has no stable ID
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch returned error: %v", err)
	}

	if response.Success {
		t.Error("Expected batch to fail when depending on legacy task")
	}

	if len(response.Errors) == 0 {
		t.Error("Expected error message about legacy task")
	}
}

// TestExecuteBatch_AllNewFields tests batch with all new fields combined
func TestExecuteBatch_AllNewFields(t *testing.T) {
	tl := NewTaskList("Combined Test")

	// Add prerequisite task
	tl.AddTaskWithOptions("", "Prerequisite", AddOptions{})

	// Add task with all new fields
	ops := []Operation{
		{
			Type:      "add",
			Title:     "Full featured task",
			Stream:    intPtr(2),
			BlockedBy: []string{"1"},
			Owner:     stringPtr("agent-1"),
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Verify all fields
	task := findTaskByTitleRecursive(tl, "Full featured task")
	if task == nil {
		t.Fatal("Task not found")
	}

	if task.Stream != 2 {
		t.Errorf("Expected stream 2, got %d", task.Stream)
	}
	if len(task.BlockedBy) != 1 {
		t.Errorf("Expected 1 blocked-by reference, got %d", len(task.BlockedBy))
	}
	if task.Owner != "agent-1" {
		t.Errorf("Expected owner 'agent-1', got '%s'", task.Owner)
	}
	if task.StableID == "" {
		t.Error("Expected task to have stable ID")
	}
}

// TestExecuteBatch_DryRunWithNewFields tests that dry run works with new fields
func TestExecuteBatch_DryRunWithNewFields(t *testing.T) {
	tl := NewTaskList("Dry Run Test")

	// Add prerequisite task
	tl.AddTaskWithOptions("", "Prerequisite", AddOptions{})

	initialTaskCount := len(tl.Tasks)

	ops := []Operation{
		{
			Type:      "add",
			Title:     "New task",
			Stream:    intPtr(2),
			BlockedBy: []string{"1"},
			Owner:     stringPtr("agent-1"),
		},
	}

	response, err := tl.ExecuteBatch(ops, true) // dry-run
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Original should be unchanged
	if len(tl.Tasks) != initialTaskCount {
		t.Errorf("Expected task count to remain %d after dry-run, got %d", initialTaskCount, len(tl.Tasks))
	}

	// Preview should contain the task
	if !strings.Contains(response.Preview, "New task") {
		t.Error("Preview should contain 'New task'")
	}
	if !strings.Contains(response.Preview, "Stream: 2") {
		t.Error("Preview should contain stream assignment")
	}
}

// Helper functions

func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

// findTaskByTitleRecursive is a local helper to avoid conflicts with other test files
func findTaskByTitleRecursive(tl *TaskList, title string) *Task {
	var find func(tasks []Task) *Task
	find = func(tasks []Task) *Task {
		for i := range tasks {
			if tasks[i].Title == title {
				return &tasks[i]
			}
			if found := find(tasks[i].Children); found != nil {
				return found
			}
		}
		return nil
	}
	return find(tl.Tasks)
}
