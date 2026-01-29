package task

import (
	"testing"
)

func TestDependencyIndex_DetectCycle_SelfReference(t *testing.T) {
	// Test A → A (self-reference)
	tasks := []Task{
		{ID: "1", Title: "Task A", StableID: "abc0001"},
	}
	idx := BuildDependencyIndex(tasks)

	hasCycle, path := idx.DetectCycle("abc0001", "abc0001")
	if !hasCycle {
		t.Errorf("DetectCycle(abc0001, abc0001) = false, want true (self-reference)")
	}
	if len(path) != 2 || path[0] != "abc0001" || path[1] != "abc0001" {
		t.Errorf("DetectCycle path = %v, want [abc0001, abc0001]", path)
	}
}

func TestDependencyIndex_DetectCycle_DirectCycle(t *testing.T) {
	// Test A → B → A (direct cycle)
	// Currently B depends on nothing, we want to add A → B
	// which would create A → B → (nothing yet)
	// But if B already has A in its blockedBy, then A → B would create cycle
	tasks := []Task{
		{ID: "1", Title: "Task A", StableID: "abc0001"},
		{ID: "2", Title: "Task B", StableID: "abc0002", BlockedBy: []string{"abc0001"}},
	}
	idx := BuildDependencyIndex(tasks)

	// Adding A → B (A depends on B) when B already depends on A
	// This would create: A → B → A
	hasCycle, path := idx.DetectCycle("abc0001", "abc0002")
	if !hasCycle {
		t.Errorf("DetectCycle(abc0001, abc0002) = false, want true (B already depends on A)")
	}
	// Path should be: abc0001 → abc0002 → abc0001
	if len(path) < 2 {
		t.Errorf("DetectCycle path = %v, want path showing cycle", path)
	}
}

func TestDependencyIndex_DetectCycle_IndirectCycle(t *testing.T) {
	// Test A → B → C → A (indirect cycle)
	// C depends on B, B depends on A
	// Adding A → C would create: A → C → B → A (cycle through existing deps)
	tasks := []Task{
		{ID: "1", Title: "Task A", StableID: "abc0001"},
		{ID: "2", Title: "Task B", StableID: "abc0002", BlockedBy: []string{"abc0001"}},
		{ID: "3", Title: "Task C", StableID: "abc0003", BlockedBy: []string{"abc0002"}},
	}
	idx := BuildDependencyIndex(tasks)

	// Adding A → C (A depends on C) when the chain C → B → A exists
	hasCycle, path := idx.DetectCycle("abc0001", "abc0003")
	if !hasCycle {
		t.Errorf("DetectCycle(abc0001, abc0003) = false, want true (indirect cycle)")
	}
	// Path should show the cycle: abc0001 → abc0003 → abc0002 → abc0001
	if len(path) < 3 {
		t.Errorf("DetectCycle path = %v, want path showing indirect cycle", path)
	}
}

func TestDependencyIndex_DetectCycle_ValidChain(t *testing.T) {
	// Test valid chain (no false positives)
	// A → B → C is valid when adding D → A (D depends on A)
	tasks := []Task{
		{ID: "1", Title: "Task A", StableID: "abc0001"},
		{ID: "2", Title: "Task B", StableID: "abc0002", BlockedBy: []string{"abc0001"}},
		{ID: "3", Title: "Task C", StableID: "abc0003", BlockedBy: []string{"abc0002"}},
		{ID: "4", Title: "Task D", StableID: "abc0004"},
	}
	idx := BuildDependencyIndex(tasks)

	// Adding D → A (D depends on A) should be valid - no cycle
	hasCycle, _ := idx.DetectCycle("abc0004", "abc0001")
	if hasCycle {
		t.Errorf("DetectCycle(abc0004, abc0001) = true, want false (valid dependency)")
	}

	// Adding D → C (D depends on C) should be valid - no cycle
	hasCycle, _ = idx.DetectCycle("abc0004", "abc0003")
	if hasCycle {
		t.Errorf("DetectCycle(abc0004, abc0003) = true, want false (valid dependency)")
	}
}

func TestDependencyIndex_DetectCycle_NoExistingDependencies(t *testing.T) {
	// Test adding dependency to tasks with no existing dependencies
	tasks := []Task{
		{ID: "1", Title: "Task A", StableID: "abc0001"},
		{ID: "2", Title: "Task B", StableID: "abc0002"},
	}
	idx := BuildDependencyIndex(tasks)

	// A → B should be valid
	hasCycle, _ := idx.DetectCycle("abc0001", "abc0002")
	if hasCycle {
		t.Errorf("DetectCycle(abc0001, abc0002) = true, want false (no existing deps)")
	}

	// B → A should be valid
	hasCycle, _ = idx.DetectCycle("abc0002", "abc0001")
	if hasCycle {
		t.Errorf("DetectCycle(abc0002, abc0001) = true, want false (no existing deps)")
	}
}

func TestDependencyIndex_DetectCycle_DeepChain(t *testing.T) {
	// Test a longer chain to ensure depth doesn't cause false positives
	// Chain: A → B → C → D → E
	tasks := []Task{
		{ID: "1", Title: "Task A", StableID: "abc0001"},
		{ID: "2", Title: "Task B", StableID: "abc0002", BlockedBy: []string{"abc0001"}},
		{ID: "3", Title: "Task C", StableID: "abc0003", BlockedBy: []string{"abc0002"}},
		{ID: "4", Title: "Task D", StableID: "abc0004", BlockedBy: []string{"abc0003"}},
		{ID: "5", Title: "Task E", StableID: "abc0005", BlockedBy: []string{"abc0004"}},
	}
	idx := BuildDependencyIndex(tasks)

	// Adding A → E would create cycle: A → E → D → C → B → A
	hasCycle, path := idx.DetectCycle("abc0001", "abc0005")
	if !hasCycle {
		t.Errorf("DetectCycle(abc0001, abc0005) = false, want true (deep cycle)")
	}
	if len(path) < 5 {
		t.Errorf("DetectCycle path = %v, want path showing deep cycle", path)
	}

	// Adding F → A should be valid (F doesn't exist, but treating as new task)
	// Actually, let's add a new task F
	tasks = append(tasks, Task{ID: "6", Title: "Task F", StableID: "abc0006"})
	idx = BuildDependencyIndex(tasks)

	hasCycle, _ = idx.DetectCycle("abc0006", "abc0001")
	if hasCycle {
		t.Errorf("DetectCycle(abc0006, abc0001) = true, want false (valid dependency)")
	}
}

func TestDependencyIndex_DetectCycle_NonExistentTask(t *testing.T) {
	tasks := []Task{
		{ID: "1", Title: "Task A", StableID: "abc0001"},
	}
	idx := BuildDependencyIndex(tasks)

	// Checking cycle with non-existent task should not panic and return no cycle
	hasCycle, _ := idx.DetectCycle("abc0001", "xyz9999")
	if hasCycle {
		t.Errorf("DetectCycle with non-existent target should return false")
	}

	hasCycle, _ = idx.DetectCycle("xyz9999", "abc0001")
	if hasCycle {
		t.Errorf("DetectCycle with non-existent source should return false")
	}
}

func TestDependencyIndex_DetectCycle_DiamondPattern(t *testing.T) {
	// Diamond dependency pattern:
	//     A
	//    / \
	//   B   C
	//    \ /
	//     D
	// D depends on both B and C, B and C both depend on A
	tasks := []Task{
		{ID: "1", Title: "Task A", StableID: "abc0001"},
		{ID: "2", Title: "Task B", StableID: "abc0002", BlockedBy: []string{"abc0001"}},
		{ID: "3", Title: "Task C", StableID: "abc0003", BlockedBy: []string{"abc0001"}},
		{ID: "4", Title: "Task D", StableID: "abc0004", BlockedBy: []string{"abc0002", "abc0003"}},
	}
	idx := BuildDependencyIndex(tasks)

	// No cycles in diamond pattern - verify no false positives
	// Adding E → D should be valid
	tasks = append(tasks, Task{ID: "5", Title: "Task E", StableID: "abc0005"})
	idx = BuildDependencyIndex(tasks)

	hasCycle, _ := idx.DetectCycle("abc0005", "abc0004")
	if hasCycle {
		t.Errorf("DetectCycle(abc0005, abc0004) = true, want false (diamond pattern)")
	}

	// But adding A → D would create a cycle
	hasCycle, _ = idx.DetectCycle("abc0001", "abc0004")
	if !hasCycle {
		t.Errorf("DetectCycle(abc0001, abc0004) = false, want true (diamond cycle)")
	}
}

func TestDependencyIndex_DetectCycle_BranchingPaths(t *testing.T) {
	// Test that algorithm doesn't get confused by multiple paths to same node
	// that don't form cycles
	//     A
	//    /|\
	//   B C D
	//    \|/
	//     E
	tasks := []Task{
		{ID: "1", Title: "Task A", StableID: "abc0001"},
		{ID: "2", Title: "Task B", StableID: "abc0002", BlockedBy: []string{"abc0001"}},
		{ID: "3", Title: "Task C", StableID: "abc0003", BlockedBy: []string{"abc0001"}},
		{ID: "4", Title: "Task D", StableID: "abc0004", BlockedBy: []string{"abc0001"}},
		{ID: "5", Title: "Task E", StableID: "abc0005", BlockedBy: []string{"abc0002", "abc0003", "abc0004"}},
	}
	idx := BuildDependencyIndex(tasks)

	// Adding F → E should be valid
	tasks = append(tasks, Task{ID: "6", Title: "Task F", StableID: "abc0006"})
	idx = BuildDependencyIndex(tasks)

	hasCycle, _ := idx.DetectCycle("abc0006", "abc0005")
	if hasCycle {
		t.Errorf("DetectCycle(abc0006, abc0005) = true, want false")
	}
}

func TestDependencyIndex_DetectCycle_CyclePathContent(t *testing.T) {
	// Verify the cycle path contains the correct IDs
	tasks := []Task{
		{ID: "1", Title: "Task A", StableID: "abc0001"},
		{ID: "2", Title: "Task B", StableID: "abc0002", BlockedBy: []string{"abc0001"}},
	}
	idx := BuildDependencyIndex(tasks)

	// A → B when B → A creates cycle: A → B → A
	hasCycle, path := idx.DetectCycle("abc0001", "abc0002")
	if !hasCycle {
		t.Fatalf("DetectCycle should detect cycle")
	}

	// Verify the path makes sense
	if len(path) < 2 {
		t.Errorf("Cycle path too short: %v", path)
	}

	// First element should be the source (fromStableID)
	if path[0] != "abc0001" {
		t.Errorf("Cycle path should start with abc0001, got %v", path)
	}

	// Second element should be the target (toStableID)
	if len(path) > 1 && path[1] != "abc0002" {
		t.Errorf("Cycle path second element should be abc0002, got %v", path)
	}
}
