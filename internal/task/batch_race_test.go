package task

import (
	"fmt"
	"testing"
)

// TestBatchRace_AgentCreatesTasksWithDeps simulates an AI agent creating
// multiple tasks in a single batch where later tasks depend on earlier ones.
// This is the scenario described in T-23.
func TestBatchRace_AgentCreatesTasksWithDeps(t *testing.T) {
	t.Run("chain dependency all new tasks", func(t *testing.T) {
		// Agent creates 3 tasks: C depends on B, B depends on A
		tl := NewTaskList("Test")

		ops := []Operation{
			{Type: "add", Title: "Task A"},
			{Type: "add", Title: "Task B", BlockedBy: []string{"1"}},
			{Type: "add", Title: "Task C", BlockedBy: []string{"2"}},
		}

		response, err := tl.ExecuteBatch(ops, false)
		if err != nil {
			t.Fatalf("ExecuteBatch error: %v", err)
		}
		if !response.Success {
			t.Fatalf("Batch failed: %v", response.Errors)
		}

		// Verify chain
		taskA := tl.FindTask("1")
		taskB := tl.FindTask("2")
		taskC := tl.FindTask("3")

		if taskA == nil || taskB == nil || taskC == nil {
			t.Fatal("Not all tasks found")
		}

		// Task A should have a stable ID (auto-assigned as dep target)
		if taskA.StableID == "" {
			t.Error("Task A should have auto-assigned stable ID")
		}

		// Task B should depend on Task A
		if len(taskB.BlockedBy) != 1 || taskB.BlockedBy[0] != taskA.StableID {
			t.Errorf("Task B blocked_by = %v, want [%s]", taskB.BlockedBy, taskA.StableID)
		}

		// Task C should depend on Task B
		if len(taskC.BlockedBy) != 1 || taskC.BlockedBy[0] != taskB.StableID {
			t.Errorf("Task C blocked_by = %v, want [%s]", taskC.BlockedBy, taskB.StableID)
		}
	})

	t.Run("multiple deps on same task", func(t *testing.T) {
		// Tasks B and C both depend on Task A (all new in same batch)
		tl := NewTaskList("Test")

		ops := []Operation{
			{Type: "add", Title: "Task A"},
			{Type: "add", Title: "Task B", BlockedBy: []string{"1"}},
			{Type: "add", Title: "Task C", BlockedBy: []string{"1"}},
		}

		response, err := tl.ExecuteBatch(ops, false)
		if err != nil {
			t.Fatalf("ExecuteBatch error: %v", err)
		}
		if !response.Success {
			t.Fatalf("Batch failed: %v", response.Errors)
		}

		taskA := tl.FindTask("1")
		taskB := tl.FindTask("2")
		taskC := tl.FindTask("3")

		// Both B and C should reference A's stable ID
		if taskB.BlockedBy[0] != taskA.StableID {
			t.Errorf("Task B should depend on Task A's stable ID %s, got %v", taskA.StableID, taskB.BlockedBy)
		}
		if taskC.BlockedBy[0] != taskA.StableID {
			t.Errorf("Task C should depend on Task A's stable ID %s, got %v", taskA.StableID, taskC.BlockedBy)
		}
	})

	t.Run("parent with subtask dependencies", func(t *testing.T) {
		// Agent creates parent task, subtasks, with deps between subtasks
		tl := NewTaskList("Test")

		ops := []Operation{
			{Type: "add", Title: "Phase 1"},
			{Type: "add", Title: "Design", Parent: "1"},
			{Type: "add", Title: "Implement", Parent: "1", BlockedBy: []string{"1.1"}},
			{Type: "add", Title: "Phase 2", BlockedBy: []string{"1"}},
			{Type: "add", Title: "Deploy", Parent: "2", BlockedBy: []string{"1.2"}},
		}

		response, err := tl.ExecuteBatch(ops, false)
		if err != nil {
			t.Fatalf("ExecuteBatch error: %v", err)
		}
		if !response.Success {
			t.Fatalf("Batch failed: %v", response.Errors)
		}

		design := tl.FindTask("1.1")
		implement := tl.FindTask("1.2")
		phase1 := tl.FindTask("1")
		phase2 := tl.FindTask("2")
		deploy := tl.FindTask("2.1")

		if design == nil || implement == nil || phase1 == nil || phase2 == nil || deploy == nil {
			t.Fatalf("Missing tasks: design=%v implement=%v phase1=%v phase2=%v deploy=%v",
				design, implement, phase1, phase2, deploy)
		}

		// Implement depends on Design
		if len(implement.BlockedBy) != 1 || implement.BlockedBy[0] != design.StableID {
			t.Errorf("Implement blocked_by = %v, want [%s]", implement.BlockedBy, design.StableID)
		}

		// Phase 2 depends on Phase 1
		if len(phase2.BlockedBy) != 1 || phase2.BlockedBy[0] != phase1.StableID {
			t.Errorf("Phase 2 blocked_by = %v, want [%s]", phase2.BlockedBy, phase1.StableID)
		}

		// Deploy depends on Implement
		if len(deploy.BlockedBy) != 1 || deploy.BlockedBy[0] != implement.StableID {
			t.Errorf("Deploy blocked_by = %v, want [%s]", deploy.BlockedBy, implement.StableID)
		}
	})

	t.Run("existing tasks with new deps", func(t *testing.T) {
		// File already has tasks; agent adds new tasks that depend on existing ones
		tl := NewTaskList("Test")
		tl.AddTask("", "Existing task 1", "")
		tl.AddTask("", "Existing task 2", "")

		ops := []Operation{
			{Type: "add", Title: "New task A", BlockedBy: []string{"1"}},
			{Type: "add", Title: "New task B", BlockedBy: []string{"2", "3"}},
		}

		response, err := tl.ExecuteBatch(ops, false)
		if err != nil {
			t.Fatalf("ExecuteBatch error: %v", err)
		}
		if !response.Success {
			t.Fatalf("Batch failed: %v", response.Errors)
		}

		existing1 := tl.FindTask("1")
		newA := tl.FindTask("3")
		newB := tl.FindTask("4")

		// New task A depends on existing task 1
		if len(newA.BlockedBy) != 1 || newA.BlockedBy[0] != existing1.StableID {
			t.Errorf("New A blocked_by = %v, want [%s]", newA.BlockedBy, existing1.StableID)
		}

		// New task B depends on existing task 2 AND new task A
		if len(newB.BlockedBy) != 2 {
			t.Errorf("New B should have 2 deps, got %d: %v", len(newB.BlockedBy), newB.BlockedBy)
		}
	})

	t.Run("dry run then actual matches", func(t *testing.T) {
		// Verify dry run produces valid output (even though stable IDs may differ)
		tl := NewTaskList("Test")

		ops := []Operation{
			{Type: "add", Title: "Task A"},
			{Type: "add", Title: "Task B", BlockedBy: []string{"1"}},
		}

		// Dry run
		dryResponse, err := tl.ExecuteBatch(ops, true)
		if err != nil {
			t.Fatalf("Dry run error: %v", err)
		}
		if !dryResponse.Success {
			t.Fatalf("Dry run failed: %v", dryResponse.Errors)
		}
		if dryResponse.Preview == "" {
			t.Error("Dry run should have preview")
		}

		// Actual run
		actualResponse, err := tl.ExecuteBatch(ops, false)
		if err != nil {
			t.Fatalf("Actual run error: %v", err)
		}
		if !actualResponse.Success {
			t.Fatalf("Actual run failed: %v", actualResponse.Errors)
		}
	})

	t.Run("add with stream and blocked_by", func(t *testing.T) {
		// Agent creates tasks with both stream and blocked_by
		tl := NewTaskList("Test")
		stream1 := 1
		stream2 := 2

		ops := []Operation{
			{Type: "add", Title: "Task A", Stream: &stream1},
			{Type: "add", Title: "Task B", Stream: &stream2, BlockedBy: []string{"1"}},
		}

		response, err := tl.ExecuteBatch(ops, false)
		if err != nil {
			t.Fatalf("ExecuteBatch error: %v", err)
		}
		if !response.Success {
			t.Fatalf("Batch failed: %v", response.Errors)
		}

		taskA := tl.FindTask("1")
		taskB := tl.FindTask("2")

		if taskA.Stream != 1 {
			t.Errorf("Task A stream = %d, want 1", taskA.Stream)
		}
		if taskB.Stream != 2 {
			t.Errorf("Task B stream = %d, want 2", taskB.Stream)
		}
		if len(taskB.BlockedBy) != 1 || taskB.BlockedBy[0] != taskA.StableID {
			t.Errorf("Task B blocked_by = %v, want [%s]", taskB.BlockedBy, taskA.StableID)
		}
	})

	t.Run("large batch agent task creation", func(t *testing.T) {
		// Simulates a realistic agent session creating a task plan with many deps
		tl := NewTaskList("Feature Implementation")
		stream1 := 1
		stream2 := 2

		ops := []Operation{
			// Phase 1
			{Type: "add", Title: "Research requirements"},
			{Type: "add", Title: "Write design doc", BlockedBy: []string{"1"}},
			{Type: "add", Title: "Review design", BlockedBy: []string{"2"}},
			// Phase 2 - parallel streams
			{Type: "add", Title: "Implement core module", BlockedBy: []string{"3"}, Stream: &stream1},
			{Type: "add", Title: "Implement UI", BlockedBy: []string{"3"}, Stream: &stream2},
			{Type: "add", Title: "Write core tests", BlockedBy: []string{"4"}, Stream: &stream1},
			{Type: "add", Title: "Write UI tests", BlockedBy: []string{"5"}, Stream: &stream2},
			// Phase 3 - convergence
			{Type: "add", Title: "Integration testing", BlockedBy: []string{"6", "7"}},
			{Type: "add", Title: "Deploy to staging", BlockedBy: []string{"8"}},
			{Type: "add", Title: "Production release", BlockedBy: []string{"9"}},
		}

		response, err := tl.ExecuteBatch(ops, false)
		if err != nil {
			t.Fatalf("ExecuteBatch error: %v", err)
		}
		if !response.Success {
			t.Fatalf("Batch failed: %v", response.Errors)
		}

		// Verify all 10 tasks created
		if len(tl.Tasks) != 10 {
			t.Fatalf("Expected 10 tasks, got %d", len(tl.Tasks))
		}

		// Verify dependency chain
		for i := 2; i <= 10; i++ {
			task := tl.FindTask(fmt.Sprintf("%d", i))
			if task == nil {
				t.Fatalf("Task %d not found", i)
			}
			if len(task.BlockedBy) == 0 {
				t.Errorf("Task %d (%s) should have dependencies", i, task.Title)
			}
		}

		// Verify integration testing depends on both test tasks
		integrationTask := tl.FindTask("8")
		if len(integrationTask.BlockedBy) != 2 {
			t.Errorf("Integration testing should have 2 deps, got %d", len(integrationTask.BlockedBy))
		}
	})
}

// TestBatchRace_UpdateWithDeps tests update operations that add dependencies
// to tasks created in the same batch.
func TestBatchRace_UpdateWithDeps(t *testing.T) {
	t.Run("add then update with blocked_by", func(t *testing.T) {
		// Agent creates tasks, then updates one to add a dependency
		tl := NewTaskList("Test")

		ops := []Operation{
			{Type: "add", Title: "Task A"},
			{Type: "add", Title: "Task B"},
			{Type: "update", ID: "2", BlockedBy: []string{"1"}},
		}

		response, err := tl.ExecuteBatch(ops, false)
		if err != nil {
			t.Fatalf("ExecuteBatch error: %v", err)
		}
		if !response.Success {
			t.Fatalf("Batch failed: %v", response.Errors)
		}

		taskA := tl.FindTask("1")
		taskB := tl.FindTask("2")

		if taskA.StableID == "" {
			t.Error("Task A should have auto-assigned stable ID")
		}
		if len(taskB.BlockedBy) != 1 || taskB.BlockedBy[0] != taskA.StableID {
			t.Errorf("Task B blocked_by = %v, want [%s]", taskB.BlockedBy, taskA.StableID)
		}
	})

	t.Run("update task without stable ID to add blocked_by", func(t *testing.T) {
		// Task created without extended fields, then updated with blocked_by
		// The task being updated should get a stable ID for proper dependency tracking
		tl := NewTaskList("Test")
		tl.AddTask("", "Task A", "")
		tl.AddTask("", "Task B", "")

		// Neither task has a stable ID yet
		taskA := tl.FindTask("1")
		taskB := tl.FindTask("2")
		if taskA.StableID != "" || taskB.StableID != "" {
			t.Skip("Tasks already have stable IDs")
		}

		ops := []Operation{
			{Type: "update", ID: "2", BlockedBy: []string{"1"}},
		}

		response, err := tl.ExecuteBatch(ops, false)
		if err != nil {
			t.Fatalf("ExecuteBatch error: %v", err)
		}
		if !response.Success {
			t.Fatalf("Batch failed: %v", response.Errors)
		}

		taskA = tl.FindTask("1")
		taskB = tl.FindTask("2")

		// Task A (dependency target) should have stable ID
		if taskA.StableID == "" {
			t.Error("Task A should have auto-assigned stable ID as dependency target")
		}

		// Task B should reference Task A's stable ID
		if len(taskB.BlockedBy) != 1 || taskB.BlockedBy[0] != taskA.StableID {
			t.Errorf("Task B blocked_by = %v, want [%s]", taskB.BlockedBy, taskA.StableID)
		}

		// KEY CHECK: Task B should also have a stable ID for proper dependency tracking
		// Without it, Task B is invisible to the dependency index and cycle detection fails
		if taskB.StableID == "" {
			t.Error("Task B should have a stable ID after being given blocked_by dependencies")
		}
	})
}

// TestBatchRace_CycleDetectionWithoutStableIDs tests that cycle detection works
// even when tasks don't have pre-existing stable IDs.
func TestBatchRace_CycleDetectionWithoutStableIDs(t *testing.T) {
	t.Run("mutual dependency cycle detected", func(t *testing.T) {
		tl := NewTaskList("Test")
		tl.AddTask("", "Task A", "")
		tl.AddTask("", "Task B", "")

		ops := []Operation{
			{Type: "update", ID: "1", BlockedBy: []string{"2"}},
			{Type: "update", ID: "2", BlockedBy: []string{"1"}},
		}

		response, err := tl.ExecuteBatch(ops, false)
		if err != nil {
			t.Fatalf("ExecuteBatch returned hard error: %v", err)
		}
		if response.Success {
			t.Error("Batch should fail due to circular dependency")
		}
	})

	t.Run("three way cycle detected", func(t *testing.T) {
		tl := NewTaskList("Test")
		tl.AddTask("", "Task A", "")
		tl.AddTask("", "Task B", "")
		tl.AddTask("", "Task C", "")

		ops := []Operation{
			{Type: "update", ID: "1", BlockedBy: []string{"2"}},
			{Type: "update", ID: "2", BlockedBy: []string{"3"}},
			{Type: "update", ID: "3", BlockedBy: []string{"1"}},
		}

		response, err := tl.ExecuteBatch(ops, false)
		if err != nil {
			t.Fatalf("ExecuteBatch returned hard error: %v", err)
		}
		if response.Success {
			t.Error("Batch should fail due to circular dependency")
		}
	})
}
