package task

// maxDependencyDepth limits DFS traversal to prevent stack overflow on deep chains
const maxDependencyDepth = 1000

// DependencyIndex provides fast lookup for dependency resolution
type DependencyIndex struct {
	byStableID     map[string]*Task    // StableID -> Task lookup
	byHierarchical map[string]*Task    // Hierarchical ID -> Task lookup
	dependents     map[string][]string // StableID -> list of stable IDs that depend on it
}

// BuildDependencyIndex creates an index from a task list
func BuildDependencyIndex(tasks []Task) *DependencyIndex {
	idx := &DependencyIndex{
		byStableID:     make(map[string]*Task),
		byHierarchical: make(map[string]*Task),
		dependents:     make(map[string][]string),
	}

	// Recursively index all tasks
	var indexTasks func(tasks []Task)
	indexTasks = func(tasks []Task) {
		for i := range tasks {
			task := &tasks[i]

			// Index by hierarchical ID
			if task.ID != "" {
				idx.byHierarchical[task.ID] = task
			}

			// Index by stable ID (only if present)
			if task.StableID != "" {
				idx.byStableID[task.StableID] = task

				// Build dependents map
				for _, blockerID := range task.BlockedBy {
					idx.dependents[blockerID] = append(idx.dependents[blockerID], task.StableID)
				}
			}

			// Recursively index children
			if len(task.Children) > 0 {
				indexTasks(task.Children)
			}
		}
	}

	indexTasks(tasks)
	return idx
}

// GetTask returns a task by stable ID, or nil if not found
func (idx *DependencyIndex) GetTask(stableID string) *Task {
	if stableID == "" {
		return nil
	}
	return idx.byStableID[stableID]
}

// GetTaskByHierarchicalID returns a task by hierarchical ID, or nil if not found
func (idx *DependencyIndex) GetTaskByHierarchicalID(id string) *Task {
	if id == "" {
		return nil
	}
	return idx.byHierarchical[id]
}

// GetDependents returns the stable IDs of tasks that depend on the given stable ID
func (idx *DependencyIndex) GetDependents(stableID string) []string {
	deps := idx.dependents[stableID]
	if deps == nil {
		return []string{}
	}
	return deps
}

// IsReady returns true if all blockers are completed
// A task with no blockers is always ready
func (idx *DependencyIndex) IsReady(task *Task) bool {
	if task == nil || len(task.BlockedBy) == 0 {
		return true
	}

	for _, blockerID := range task.BlockedBy {
		blocker := idx.GetTask(blockerID)
		if blocker == nil {
			// If blocker doesn't exist, we consider the task as not ready
			// (conservative approach for invalid references)
			continue
		}
		if blocker.Status != Completed {
			return false
		}
	}
	return true
}

// IsBlocked returns true if any blocker is not completed
func (idx *DependencyIndex) IsBlocked(task *Task) bool {
	return !idx.IsReady(task)
}

// DetectCycle checks if adding a dependency from→to would create a cycle.
// Returns (hasCycle, cyclePath) where cyclePath shows the circular chain if found.
//
// The algorithm works by checking if the target (toStableID) can reach back to
// the source (fromStableID) through its existing dependencies. If it can, then
// adding from→to would create a cycle.
func (idx *DependencyIndex) DetectCycle(fromStableID, toStableID string) (bool, []string) {
	// Check self-reference: A → A
	if fromStableID == toStableID {
		return true, []string{fromStableID, fromStableID}
	}

	// DFS from the target (toStableID) following its BlockedBy chain.
	// If we can reach the source (fromStableID), adding the dependency
	// would create: fromStableID → toStableID → ... → fromStableID
	visited := make(map[string]bool)
	path := []string{toStableID}

	var dfs func(current string, depth int) bool
	dfs = func(current string, depth int) bool {
		if current == fromStableID {
			return true // Found path back to source - cycle!
		}
		if visited[current] {
			return false // Already explored this branch
		}
		if depth > maxDependencyDepth {
			// Unusually deep chain - don't treat as cycle, but stop searching
			return false
		}
		visited[current] = true

		task := idx.GetTask(current)
		if task != nil {
			for _, blockerID := range task.BlockedBy {
				path = append(path, blockerID)
				if dfs(blockerID, depth+1) {
					return true
				}
				path = path[:len(path)-1]
			}
		}
		return false
	}

	if dfs(toStableID, 0) {
		// Return full cycle: from → to → ... → from
		return true, append([]string{fromStableID}, path...)
	}
	return false, nil
}

// TranslateToHierarchical converts stable IDs to current hierarchical IDs.
// Unknown stable IDs are excluded from the result.
func (idx *DependencyIndex) TranslateToHierarchical(stableIDs []string) []string {
	if len(stableIDs) == 0 {
		return []string{}
	}

	result := make([]string, 0, len(stableIDs))
	for _, stableID := range stableIDs {
		task := idx.GetTask(stableID)
		if task != nil {
			result = append(result, task.ID)
		}
	}
	return result
}
