package task

// TaskWithContext represents a task along with its incomplete children
type TaskWithContext struct {
	*Task
	IncompleteChildren []Task // Only incomplete subtasks for focused work
}

// FindNextIncompleteTask finds the first task with incomplete work
// using depth-first traversal. Returns the task along with its incomplete subtasks.
func FindNextIncompleteTask(tasks []Task) *TaskWithContext {
	for i := range tasks {
		if result := evaluateTaskForNext(&tasks[i]); result != nil {
			return result
		}
	}
	return nil
}

// evaluateTaskForNext checks if a task has incomplete work and returns it
// if so, including only its incomplete children
func evaluateTaskForNext(task *Task) *TaskWithContext {
	// If the task has incomplete work (itself or children), return it
	if hasIncompleteWork(task) {
		// Filter to only include incomplete subtasks for focused work
		incompleteChildren := filterIncompleteChildren(task.Children)

		return &TaskWithContext{
			Task:               task,
			IncompleteChildren: incompleteChildren,
		}
	}

	// Otherwise check children recursively
	for i := range task.Children {
		if result := evaluateTaskForNext(&task.Children[i]); result != nil {
			return result
		}
	}

	return nil
}

// hasIncompleteWork checks if task or any subtask is incomplete
func hasIncompleteWork(task *Task) bool {
	return hasIncompleteWorkWithDepth(task, 0, 100)
}

func hasIncompleteWorkWithDepth(task *Task, depth, maxDepth int) bool {
	if depth > maxDepth {
		// Prevent infinite recursion in case of malformed data
		return false
	}

	// Task has incomplete work if itself is not completed
	if task.Status != Completed {
		return true
	}

	// Or if any child has incomplete work
	for i := range task.Children {
		if hasIncompleteWorkWithDepth(&task.Children[i], depth+1, maxDepth) {
			return true
		}
	}

	return false
}

// filterIncompleteChildren returns only children that have incomplete work
func filterIncompleteChildren(children []Task) []Task {
	var incomplete []Task
	for _, child := range children {
		if hasIncompleteWork(&child) {
			incomplete = append(incomplete, child)
		}
	}
	return incomplete
}
