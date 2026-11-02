package task

import (
	"fmt"
	"strings"
)

// AutoCompleteParents checks and completes parent tasks when all children are done
// and the parent is not blocked by an unresolved dependency.
// Returns a slice of task IDs that were auto-completed
func (tl *TaskList) AutoCompleteParents(taskID string) ([]string, error) {
	var completedParents []string
	visited := make(map[string]bool) // Prevent cycles
	maxDepth := 100                  // Prevent infinite recursion
	depth := 0
	index := BuildDependencyIndex(tl.Tasks)

	parentID := getParentID(taskID)
	for parentID != "" && depth < maxDepth {
		if visited[parentID] {
			return completedParents, fmt.Errorf("cycle detected at task %s", parentID)
		}
		visited[parentID] = true

		parent := tl.FindTask(parentID)
		if parent == nil {
			// Parent doesn't exist, break the chain
			break
		}

		// Complete the parent only when all its children are complete and its
		// own dependencies are resolved; a blocked parent stays open so its
		// dependents remain blocked.
		if allChildrenComplete(parent) && parent.Status != Completed && !index.IsBlocked(parent) {
			parent.Status = Completed
			completedParents = append(completedParents, parentID)
		}

		// Continue checking up the chain regardless of whether this parent was completed
		// as grandparents might have all their other children completed
		parentID = getParentID(parentID)
		depth++
	}

	if depth >= maxDepth {
		return completedParents, fmt.Errorf("max depth exceeded while checking parent tasks")
	}

	return completedParents, nil
}

// ReopenCompletedAncestors sets every completed ancestor of taskID back to
// pending, so a parent never stays completed while a descendant is not.
// It is the counterpart of AutoCompleteParents for status rollbacks.
// Returns the IDs of the reopened ancestors, nearest first.
func (tl *TaskList) ReopenCompletedAncestors(taskID string) []string {
	var reopened []string
	for parentID := getParentID(taskID); parentID != ""; parentID = getParentID(parentID) {
		parent := tl.FindTask(parentID)
		if parent == nil {
			break
		}
		if parent.Status == Completed {
			parent.Status = Pending
			reopened = append(reopened, parentID)
		}
	}
	return reopened
}

// allChildrenComplete checks if all direct children of a task are complete
func allChildrenComplete(task *Task) bool {
	// A task without children is considered to have all children complete
	if len(task.Children) == 0 {
		return true
	}

	for _, child := range task.Children {
		if child.Status != Completed {
			return false
		}
		// Recursively check if all grandchildren are also complete
		if !allChildrenComplete(&child) {
			return false
		}
	}
	return true
}

// getParentID extracts the parent ID from a task ID
// Example: "1.2.3" -> "1.2", "1" -> ""
func getParentID(taskID string) string {
	parts := strings.Split(taskID, ".")
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[:len(parts)-1], ".")
}
