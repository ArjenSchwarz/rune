package task

import (
	"testing"
)

// ============================================================================
// Tests for extended AddTask operation
// ============================================================================

func TestAddTaskWithOptions(t *testing.T) {
	t.Run("AddTask generates stable ID", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		id, err := tl.AddTaskWithOptions("", "First task", AddOptions{})
		if err != nil {
			t.Fatalf("AddTaskWithOptions failed: %v", err)
		}

		task := tl.FindTask(id)
		if task == nil {
			t.Fatal("task not found")
		}

		// Task should have a stable ID
		if task.StableID == "" {
			t.Error("expected task to have a stable ID")
		}

		// Stable ID should be valid format (7 lowercase alphanumeric)
		if !IsValidStableID(task.StableID) {
			t.Errorf("stable ID %q is not valid format", task.StableID)
		}
	})

	t.Run("AddTask with stream option", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		id, err := tl.AddTaskWithOptions("", "Task with stream", AddOptions{
			Stream: 3,
		})
		if err != nil {
			t.Fatalf("AddTaskWithOptions failed: %v", err)
		}

		task := tl.FindTask(id)
		if task == nil {
			t.Fatal("task not found")
		}

		if task.Stream != 3 {
			t.Errorf("expected stream 3, got %d", task.Stream)
		}
	})

	t.Run("AddTask with blocked-by option", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		// First add a task that will be the blocker
		blockerId, err := tl.AddTaskWithOptions("", "Blocker task", AddOptions{})
		if err != nil {
			t.Fatalf("AddTaskWithOptions failed: %v", err)
		}

		blockerTask := tl.FindTask(blockerId)
		if blockerTask == nil {
			t.Fatal("blocker task not found")
		}

		// Add a task blocked by the first task
		id, err := tl.AddTaskWithOptions("", "Blocked task", AddOptions{
			BlockedBy: []string{blockerId}, // Using hierarchical ID
		})
		if err != nil {
			t.Fatalf("AddTaskWithOptions failed: %v", err)
		}

		task := tl.FindTask(id)
		if task == nil {
			t.Fatal("blocked task not found")
		}

		// Task should have the blocker's stable ID in BlockedBy
		if len(task.BlockedBy) != 1 {
			t.Fatalf("expected 1 blocker, got %d", len(task.BlockedBy))
		}

		if task.BlockedBy[0] != blockerTask.StableID {
			t.Errorf("expected blocker stable ID %q, got %q", blockerTask.StableID, task.BlockedBy[0])
		}
	})

	t.Run("AddTask with owner option", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		id, err := tl.AddTaskWithOptions("", "Task with owner", AddOptions{
			Owner: "agent-1",
		})
		if err != nil {
			t.Fatalf("AddTaskWithOptions failed: %v", err)
		}

		task := tl.FindTask(id)
		if task == nil {
			t.Fatal("task not found")
		}

		if task.Owner != "agent-1" {
			t.Errorf("expected owner 'agent-1', got %q", task.Owner)
		}
	})

	t.Run("AddTask blocked-by auto-assigns stable ID to target", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		// Create a legacy task (without stable ID) by directly adding to Tasks slice
		legacyTask := Task{
			ID:     "1",
			Title:  "Legacy task",
			Status: Pending,
			// StableID is empty - this is a legacy task
		}
		tl.Tasks = append(tl.Tasks, legacyTask)

		// Add a task blocked by the legacy task — should auto-assign a stable ID
		newID, err := tl.AddTaskWithOptions("", "New task", AddOptions{
			BlockedBy: []string{"1"},
		})
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}

		// The legacy task should now have a stable ID
		target := tl.FindTask("1")
		if target.StableID == "" {
			t.Error("expected legacy task to get a stable ID, but it's still empty")
		}

		// The new task should reference the auto-assigned stable ID
		newTask := tl.FindTask(newID)
		if newTask == nil {
			t.Fatal("new task not found")
		}
		if len(newTask.BlockedBy) != 1 || newTask.BlockedBy[0] != target.StableID {
			t.Errorf("expected blocked-by [%s], got %v", target.StableID, newTask.BlockedBy)
		}
	})

	t.Run("AddTask with multiple options", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		// Create a blocker task first
		blockerId, err := tl.AddTaskWithOptions("", "Blocker", AddOptions{})
		if err != nil {
			t.Fatalf("AddTaskWithOptions failed: %v", err)
		}

		// Add task with all options
		id, err := tl.AddTaskWithOptions("", "Full options task", AddOptions{
			Stream:    2,
			BlockedBy: []string{blockerId},
			Owner:     "agent-2",
		})
		if err != nil {
			t.Fatalf("AddTaskWithOptions failed: %v", err)
		}

		task := tl.FindTask(id)
		if task == nil {
			t.Fatal("task not found")
		}

		if task.Stream != 2 {
			t.Errorf("expected stream 2, got %d", task.Stream)
		}
		if len(task.BlockedBy) != 1 {
			t.Errorf("expected 1 blocker, got %d", len(task.BlockedBy))
		}
		if task.Owner != "agent-2" {
			t.Errorf("expected owner 'agent-2', got %q", task.Owner)
		}
		if task.StableID == "" {
			t.Error("expected task to have stable ID")
		}
	})

	t.Run("AddTask invalid stream value", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		// Stream 0 is invalid
		_, err := tl.AddTaskWithOptions("", "Task", AddOptions{
			Stream: 0,
		})
		// Stream 0 should be treated as "not set" and default to 1, not an error
		// Actually per requirements, 0 means "not explicitly set", only negative is error
		if err != nil {
			t.Logf("Stream 0 behavior: %v", err)
		}

		// Negative stream should be an error
		_, err = tl.AddTaskWithOptions("", "Task", AddOptions{
			Stream: -1,
		})
		if err == nil {
			t.Error("expected error for negative stream, got nil")
		}
	})

	t.Run("AddTask blocked-by to non-existent task", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		// Try to add a task blocked by a non-existent task
		_, err := tl.AddTaskWithOptions("", "New task", AddOptions{
			BlockedBy: []string{"999"}, // Non-existent task
		})

		// Should fail because target doesn't exist
		if err == nil {
			t.Error("expected error when blocking on non-existent task, got nil")
		}
	})

	t.Run("AddTask blocked-by with cycle detection", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		// Create task A
		taskAId, err := tl.AddTaskWithOptions("", "Task A", AddOptions{})
		if err != nil {
			t.Fatalf("AddTaskWithOptions failed: %v", err)
		}

		// Try to create task A blocked by itself (self-reference)
		_, err = tl.AddTaskWithOptions("", "Task B", AddOptions{
			BlockedBy: []string{taskAId},
		})
		if err != nil {
			t.Fatalf("Creating Task B blocked by A should work: %v", err)
		}

		// Note: Self-reference on creation isn't possible since the task doesn't exist yet
		// Cycles through update operations are tested in the update tests
	})
}

// ============================================================================
// Tests for extended UpdateTask operation
// ============================================================================

func TestUpdateTaskWithOptions(t *testing.T) {
	t.Run("UpdateTask with stream modification", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		id, err := tl.AddTaskWithOptions("", "Task 1", AddOptions{Stream: 1})
		if err != nil {
			t.Fatalf("AddTaskWithOptions failed: %v", err)
		}

		stream := 3
		err = tl.UpdateTaskWithOptions(id, UpdateOptions{
			Stream: &stream,
		})
		if err != nil {
			t.Fatalf("UpdateTaskWithOptions failed: %v", err)
		}

		task := tl.FindTask(id)
		if task.Stream != 3 {
			t.Errorf("expected stream 3, got %d", task.Stream)
		}
	})

	t.Run("UpdateTask with blocked-by modification", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		// Create blocker tasks
		blocker1Id, _ := tl.AddTaskWithOptions("", "Blocker 1", AddOptions{})
		blocker2Id, _ := tl.AddTaskWithOptions("", "Blocker 2", AddOptions{})

		// Create task without dependencies
		taskId, _ := tl.AddTaskWithOptions("", "Main task", AddOptions{})

		// Update to add blockers
		err := tl.UpdateTaskWithOptions(taskId, UpdateOptions{
			BlockedBy: []string{blocker1Id, blocker2Id},
		})
		if err != nil {
			t.Fatalf("UpdateTaskWithOptions failed: %v", err)
		}

		task := tl.FindTask(taskId)
		if len(task.BlockedBy) != 2 {
			t.Errorf("expected 2 blockers, got %d", len(task.BlockedBy))
		}
	})

	t.Run("UpdateTask with owner modification", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		id, _ := tl.AddTaskWithOptions("", "Task 1", AddOptions{})

		owner := "agent-new"
		err := tl.UpdateTaskWithOptions(id, UpdateOptions{
			Owner: &owner,
		})
		if err != nil {
			t.Fatalf("UpdateTaskWithOptions failed: %v", err)
		}

		task := tl.FindTask(id)
		if task.Owner != "agent-new" {
			t.Errorf("expected owner 'agent-new', got %q", task.Owner)
		}
	})

	t.Run("UpdateTask with release flag", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		id, _ := tl.AddTaskWithOptions("", "Task 1", AddOptions{Owner: "agent-1"})

		err := tl.UpdateTaskWithOptions(id, UpdateOptions{
			Release: true,
		})
		if err != nil {
			t.Fatalf("UpdateTaskWithOptions failed: %v", err)
		}

		task := tl.FindTask(id)
		if task.Owner != "" {
			t.Errorf("expected owner to be cleared, got %q", task.Owner)
		}
	})

	t.Run("UpdateTask cycle detection on blocked-by update", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		// Create A -> B chain (A blocks B, meaning B depends on A)
		taskAId, _ := tl.AddTaskWithOptions("", "Task A", AddOptions{})
		taskBId, _ := tl.AddTaskWithOptions("", "Task B", AddOptions{
			BlockedBy: []string{taskAId},
		})

		// Try to make A depend on B (creating a cycle: A -> B -> A)
		err := tl.UpdateTaskWithOptions(taskAId, UpdateOptions{
			BlockedBy: []string{taskBId},
		})

		// Should fail with cycle detection
		if err == nil {
			t.Error("expected cycle detection error, got nil")
		}

		// Check it's a circular dependency error
		if _, ok := err.(*CircularDependencyError); !ok {
			t.Errorf("expected CircularDependencyError, got %T: %v", err, err)
		}
	})

	t.Run("UpdateTask self-reference detection", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		taskId, _ := tl.AddTaskWithOptions("", "Task A", AddOptions{})

		// Try to make task depend on itself
		err := tl.UpdateTaskWithOptions(taskId, UpdateOptions{
			BlockedBy: []string{taskId},
		})

		// Should fail with self-reference detection
		if err == nil {
			t.Error("expected self-reference error, got nil")
		}

		if _, ok := err.(*CircularDependencyError); !ok {
			t.Errorf("expected CircularDependencyError, got %T: %v", err, err)
		}
	})

	t.Run("UpdateTask invalid stream value rejection", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		id, _ := tl.AddTaskWithOptions("", "Task 1", AddOptions{})

		// Negative stream should be rejected
		stream := -1
		err := tl.UpdateTaskWithOptions(id, UpdateOptions{
			Stream: &stream,
		})

		if err == nil {
			t.Error("expected error for negative stream, got nil")
		}
		if err != ErrInvalidStream {
			t.Errorf("expected ErrInvalidStream, got %v", err)
		}
	})

	t.Run("UpdateTask invalid owner rejection", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		id, _ := tl.AddTaskWithOptions("", "Task 1", AddOptions{})

		// Owner with newlines should be rejected
		owner := "agent\nwith\nnewlines"
		err := tl.UpdateTaskWithOptions(id, UpdateOptions{
			Owner: &owner,
		})

		if err == nil {
			t.Error("expected error for invalid owner, got nil")
		}
		if err != ErrInvalidOwner {
			t.Errorf("expected ErrInvalidOwner, got %v", err)
		}
	})

	t.Run("UpdateTask clearing blocked-by", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		blockerId, _ := tl.AddTaskWithOptions("", "Blocker", AddOptions{})
		taskId, _ := tl.AddTaskWithOptions("", "Task", AddOptions{
			BlockedBy: []string{blockerId},
		})

		// Clear blocked-by with empty slice
		err := tl.UpdateTaskWithOptions(taskId, UpdateOptions{
			BlockedBy: []string{},
		})
		if err != nil {
			t.Fatalf("UpdateTaskWithOptions failed: %v", err)
		}

		task := tl.FindTask(taskId)
		if len(task.BlockedBy) != 0 {
			t.Errorf("expected empty blocked-by, got %v", task.BlockedBy)
		}
	})
}

// ============================================================================
// Tests for extended RemoveTask operation
// ============================================================================

func TestRemoveTaskWithDependents(t *testing.T) {
	t.Run("RemoveTask with dependents generates warning", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		// Create a task that others will depend on
		blockerId, _ := tl.AddTaskWithOptions("", "Blocker", AddOptions{})
		blockerTask := tl.FindTask(blockerId)
		blockerStableID := blockerTask.StableID

		// Create tasks that depend on it
		tl.AddTaskWithOptions("", "Dependent 1", AddOptions{
			BlockedBy: []string{blockerId},
		})
		tl.AddTaskWithOptions("", "Dependent 2", AddOptions{
			BlockedBy: []string{blockerId},
		})

		// Remove the blocker task (it's at ID "1", so after removal IDs will shift)
		warnings, err := tl.RemoveTaskWithDependents(blockerId)
		if err != nil {
			t.Fatalf("RemoveTaskWithDependents failed: %v", err)
		}

		// Should have warnings about dependents
		if len(warnings) == 0 {
			t.Error("expected warnings about dependents, got none")
		}

		// Verify task was removed by checking no task has the stable ID
		found := false
		for _, task := range tl.Tasks {
			if task.StableID == blockerStableID {
				found = true
				break
			}
		}
		if found {
			t.Error("expected task to be removed (stable ID still found)")
		}

		// Verify we have 2 tasks remaining
		if len(tl.Tasks) != 2 {
			t.Errorf("expected 2 tasks remaining, got %d", len(tl.Tasks))
		}
	})

	t.Run("RemoveTask dependent references are cleaned up", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		// Create blocker and dependent
		blockerId, _ := tl.AddTaskWithOptions("", "Blocker", AddOptions{})
		blockerTask := tl.FindTask(blockerId)

		depId, _ := tl.AddTaskWithOptions("", "Dependent", AddOptions{
			BlockedBy: []string{blockerId},
		})

		// Verify dependent has the blocker in its BlockedBy
		depTask := tl.FindTask(depId)
		if len(depTask.BlockedBy) != 1 {
			t.Fatalf("expected 1 blocker, got %d", len(depTask.BlockedBy))
		}
		if depTask.BlockedBy[0] != blockerTask.StableID {
			t.Errorf("expected blocker stable ID %q in BlockedBy", blockerTask.StableID)
		}

		// Remove the blocker
		_, err := tl.RemoveTaskWithDependents(blockerId)
		if err != nil {
			t.Fatalf("RemoveTaskWithDependents failed: %v", err)
		}

		// Dependent should no longer have the reference (after renumbering, task ID changes)
		// Need to find the task again by its new ID (should be "1" now)
		newDepTask := tl.FindTask("1") // After removal and renumbering
		if newDepTask == nil {
			t.Fatal("dependent task not found after removal")
		}

		if len(newDepTask.BlockedBy) != 0 {
			t.Errorf("expected blocker reference to be cleaned up, got %v", newDepTask.BlockedBy)
		}
	})

	t.Run("RemoveTask stable ID is not reused", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		// Add and remove a task
		id1, _ := tl.AddTaskWithOptions("", "Task 1", AddOptions{})
		task1 := tl.FindTask(id1)
		stableID1 := task1.StableID

		_, err := tl.RemoveTaskWithDependents(id1)
		if err != nil {
			t.Fatalf("RemoveTaskWithDependents failed: %v", err)
		}

		// Add more tasks and verify the stable ID isn't reused
		for range 10 {
			newId, _ := tl.AddTaskWithOptions("", "New Task", AddOptions{})
			newTask := tl.FindTask(newId)
			if newTask.StableID == stableID1 {
				t.Errorf("stable ID %q was reused", stableID1)
			}
		}
	})

	t.Run("RemoveTask without dependents no warning", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		// Create a task without dependents
		id, _ := tl.AddTaskWithOptions("", "Standalone task", AddOptions{})

		warnings, err := tl.RemoveTaskWithDependents(id)
		if err != nil {
			t.Fatalf("RemoveTaskWithDependents failed: %v", err)
		}

		// Should have no warnings
		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %v", warnings)
		}
	})
}

// ============================================================================
// Helper function tests
// ============================================================================

func TestResolveToStableIDs(t *testing.T) {
	t.Run("resolve hierarchical IDs to stable IDs", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		id1, _ := tl.AddTaskWithOptions("", "Task 1", AddOptions{})
		id2, _ := tl.AddTaskWithOptions("", "Task 2", AddOptions{})

		task1 := tl.FindTask(id1)
		task2 := tl.FindTask(id2)

		stableIDs, err := tl.resolveToStableIDs([]string{id1, id2})
		if err != nil {
			t.Fatalf("resolveToStableIDs failed: %v", err)
		}

		if len(stableIDs) != 2 {
			t.Fatalf("expected 2 stable IDs, got %d", len(stableIDs))
		}

		if stableIDs[0] != task1.StableID {
			t.Errorf("expected stable ID %q, got %q", task1.StableID, stableIDs[0])
		}
		if stableIDs[1] != task2.StableID {
			t.Errorf("expected stable ID %q, got %q", task2.StableID, stableIDs[1])
		}
	})

	t.Run("resolve fails for non-existent task", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		_, err := tl.resolveToStableIDs([]string{"999"})
		if err == nil {
			t.Error("expected error for non-existent task, got nil")
		}
	})

	t.Run("resolve auto-assigns stable ID to legacy task", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		// Create legacy task directly
		tl.Tasks = append(tl.Tasks, Task{
			ID:     "1",
			Title:  "Legacy",
			Status: Pending,
		})

		stableIDs, err := tl.resolveToStableIDs([]string{"1"})
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if len(stableIDs) != 1 {
			t.Fatalf("expected 1 stable ID, got %d", len(stableIDs))
		}
		if stableIDs[0] == "" {
			t.Error("expected non-empty stable ID")
		}

		// The task should now have the stable ID persisted
		task := tl.FindTask("1")
		if task.StableID != stableIDs[0] {
			t.Errorf("expected task stable ID %q, got %q", stableIDs[0], task.StableID)
		}
	})
}

func TestValidateOwner(t *testing.T) {
	tests := map[string]struct {
		owner   string
		wantErr bool
	}{
		"valid simple owner":       {"agent-1", false},
		"valid owner with spaces":  {"My Agent", false},
		"valid owner with numbers": {"agent123", false},
		"valid owner with special": {"agent-1_v2", false},
		"invalid with newline":     {"agent\nagent", true},
		"invalid with carriage":    {"agent\ragent", true},
		"invalid with tab":         {"agent\tagent", true},
		"invalid with null":        {"agent\x00agent", true},
		"empty owner":              {"", false}, // Empty is valid (means no owner)
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateOwner(tc.owner)
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestCollectStableIDs(t *testing.T) {
	t.Run("collect from flat list", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		tl.AddTaskWithOptions("", "Task 1", AddOptions{})
		tl.AddTaskWithOptions("", "Task 2", AddOptions{})
		tl.AddTaskWithOptions("", "Task 3", AddOptions{})

		ids := tl.collectStableIDs()

		if len(ids) != 3 {
			t.Errorf("expected 3 stable IDs, got %d", len(ids))
		}
	})

	t.Run("collect from hierarchy", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		parentId, _ := tl.AddTaskWithOptions("", "Parent", AddOptions{})
		tl.AddTaskWithOptions(parentId, "Child 1", AddOptions{})
		tl.AddTaskWithOptions(parentId, "Child 2", AddOptions{})

		ids := tl.collectStableIDs()

		if len(ids) != 3 {
			t.Errorf("expected 3 stable IDs, got %d", len(ids))
		}
	})

	t.Run("excludes legacy tasks without stable IDs", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		// Add a task with stable ID
		tl.AddTaskWithOptions("", "Task with ID", AddOptions{})

		// Add legacy task directly
		tl.Tasks = append(tl.Tasks, Task{
			ID:     "2",
			Title:  "Legacy",
			Status: Pending,
		})

		ids := tl.collectStableIDs()

		if len(ids) != 1 {
			t.Errorf("expected 1 stable ID (excluding legacy), got %d", len(ids))
		}
	})
}
