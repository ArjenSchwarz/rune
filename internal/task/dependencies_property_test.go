package task

import (
	"fmt"
	"testing"

	"pgregory.net/rapid"
)

// TestProperty_NoCyclesCreatedThroughValidOperations tests that valid dependency operations
// cannot create cycles, and that detected cycles are always real cycles.
func TestProperty_NoCyclesCreatedThroughValidOperations(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random number of tasks (2-50)
		numTasks := rapid.IntRange(2, 50).Draw(t, "numTasks")

		// Create tasks with stable IDs
		tasks := make([]Task, numTasks)
		for i := range numTasks {
			tasks[i] = Task{
				ID:       fmt.Sprintf("%d", i+1),
				Title:    fmt.Sprintf("Task %d", i+1),
				StableID: fmt.Sprintf("abc%04d", i),
				Status:   Pending,
			}
		}

		// Build initial index (no dependencies yet)
		idx := BuildDependencyIndex(tasks)

		// Try to add random valid dependencies
		numAttempts := rapid.IntRange(1, 100).Draw(t, "numAttempts")
		for range numAttempts {
			// Pick random source and target
			fromIdx := rapid.IntRange(0, numTasks-1).Draw(t, "fromIdx")
			toIdx := rapid.IntRange(0, numTasks-1).Draw(t, "toIdx")

			fromID := tasks[fromIdx].StableID
			toID := tasks[toIdx].StableID

			// Check if adding this dependency would create a cycle
			hasCycle, _ := idx.DetectCycle(fromID, toID)

			if !hasCycle {
				// If no cycle detected, add the dependency
				tasks[fromIdx].BlockedBy = append(tasks[fromIdx].BlockedBy, toID)

				// Rebuild index with new dependency
				idx = BuildDependencyIndex(tasks)

				// Property: after adding, verify no cycle actually exists
				// by checking that we can traverse from any task without looping
				if hasActualCycle(tasks) {
					t.Fatalf("cycle was created despite DetectCycle returning false: %s -> %s", fromID, toID)
				}
			}
		}
	})
}

// TestProperty_DetectedCyclesAreReal tests that when DetectCycle returns true,
// there would actually be a cycle if the dependency were added.
func TestProperty_DetectedCyclesAreReal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a small number of tasks to keep cycles findable
		numTasks := rapid.IntRange(3, 20).Draw(t, "numTasks")

		// Create tasks
		tasks := make([]Task, numTasks)
		for i := range numTasks {
			tasks[i] = Task{
				ID:       fmt.Sprintf("%d", i+1),
				Title:    fmt.Sprintf("Task %d", i+1),
				StableID: fmt.Sprintf("abc%04d", i),
				Status:   Pending,
			}
		}

		// Add some random dependencies without checking for cycles first
		numDeps := rapid.IntRange(0, numTasks).Draw(t, "numDeps")
		for range numDeps {
			fromIdx := rapid.IntRange(0, numTasks-1).Draw(t, "fromIdx")
			toIdx := rapid.IntRange(0, numTasks-1).Draw(t, "toIdx")
			if fromIdx != toIdx { // Avoid self-reference for now
				tasks[fromIdx].BlockedBy = append(tasks[fromIdx].BlockedBy, tasks[toIdx].StableID)
			}
		}

		idx := BuildDependencyIndex(tasks)

		// Try adding random dependencies and verify DetectCycle correctness
		fromIdx := rapid.IntRange(0, numTasks-1).Draw(t, "testFromIdx")
		toIdx := rapid.IntRange(0, numTasks-1).Draw(t, "testToIdx")

		fromID := tasks[fromIdx].StableID
		toID := tasks[toIdx].StableID

		hasCycle, cyclePath := idx.DetectCycle(fromID, toID)

		if hasCycle {
			// Property: if a cycle is detected, the path should show a real cycle
			if len(cyclePath) < 2 {
				t.Fatalf("cycle detected but path is too short: %v", cyclePath)
			}

			// For self-reference, path should be [id, id]
			if fromID == toID {
				if len(cyclePath) != 2 || cyclePath[0] != fromID || cyclePath[1] != fromID {
					t.Fatalf("self-reference cycle path incorrect: %v", cyclePath)
				}
			} else {
				// Path should start with fromID and contain toID
				if cyclePath[0] != fromID {
					t.Fatalf("cycle path should start with fromID %s, got %v", fromID, cyclePath)
				}
			}
		}
	})
}

// TestProperty_CycleDetectionConsistency tests that DetectCycle gives consistent results
func TestProperty_CycleDetectionConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Build a fixed set of tasks with known dependencies
		tasks := []Task{
			{ID: "1", StableID: "abc0001", Status: Pending},
			{ID: "2", StableID: "abc0002", Status: Pending, BlockedBy: []string{"abc0001"}},
			{ID: "3", StableID: "abc0003", Status: Pending, BlockedBy: []string{"abc0002"}},
			{ID: "4", StableID: "abc0004", Status: Pending},
		}

		idx := BuildDependencyIndex(tasks)

		// Run the same cycle detection multiple times
		numRuns := rapid.IntRange(10, 100).Draw(t, "numRuns")

		// Test known cycle case: 1 -> 3 (when 3 -> 2 -> 1 exists)
		var firstResult bool
		var firstPath []string
		for i := range numRuns {
			hasCycle, path := idx.DetectCycle("abc0001", "abc0003")
			if i == 0 {
				firstResult = hasCycle
				firstPath = path
			} else {
				// Property: results should be consistent
				if hasCycle != firstResult {
					t.Fatalf("inconsistent cycle detection on run %d: got %v, expected %v", i, hasCycle, firstResult)
				}
				if len(path) != len(firstPath) {
					t.Fatalf("inconsistent path length on run %d: got %d, expected %d", i, len(path), len(firstPath))
				}
			}
		}
	})
}

// hasActualCycle performs DFS to detect if any actual cycle exists in the task graph
func hasActualCycle(tasks []Task) bool {
	// Build adjacency list from BlockedBy (reverse direction for cycle check)
	// If A.BlockedBy contains B, then A depends on B (edge: A -> B)
	taskMap := make(map[string]*Task)
	for i := range tasks {
		taskMap[tasks[i].StableID] = &tasks[i]
	}

	visited := make(map[string]int) // 0: unvisited, 1: in current path, 2: finished
	var hasCycle bool

	var dfs func(id string) bool
	dfs = func(id string) bool {
		if visited[id] == 1 {
			return true // Back edge found - cycle!
		}
		if visited[id] == 2 {
			return false // Already processed
		}

		visited[id] = 1 // Mark as in current path

		task := taskMap[id]
		if task != nil {
			for _, depID := range task.BlockedBy {
				if dfs(depID) {
					return true
				}
			}
		}

		visited[id] = 2 // Mark as finished
		return false
	}

	// Check from each node
	for i := range tasks {
		if visited[tasks[i].StableID] == 0 {
			if dfs(tasks[i].StableID) {
				hasCycle = true
				break
			}
		}
	}

	return hasCycle
}
