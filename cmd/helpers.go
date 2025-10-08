package cmd

import "github.com/arjenschwarz/rune/internal/task"

// Checkbox markers for markdown status formatting
const (
	checkboxPending    = "[ ]"
	checkboxInProgress = "[-]"
	checkboxCompleted  = "[x]"
)

// formatStatus converts a task.Status to a human-readable string
func formatStatus(status task.Status) string {
	switch status {
	case task.Pending:
		return "Pending"
	case task.InProgress:
		return "In Progress"
	case task.Completed:
		return "Completed"
	default:
		return "Unknown"
	}
}

// formatStatusMarkdown converts a task.Status to markdown checkbox format
func formatStatusMarkdown(status task.Status) string {
	switch status {
	case task.Pending:
		return checkboxPending
	case task.InProgress:
		return checkboxInProgress
	case task.Completed:
		return checkboxCompleted
	default:
		return checkboxPending
	}
}

// getTaskLevel calculates the nesting level from a hierarchical task ID
// Example: "1" = 1, "1.2" = 2, "1.2.3" = 3
func getTaskLevel(id string) int {
	if id == "" {
		return 0
	}
	level := 0
	for _, char := range id {
		if char == '.' {
			level++
		}
	}
	return level + 1
}

// countAllTasks recursively counts all tasks including children
func countAllTasks(tasks []task.Task) int {
	count := len(tasks)
	for _, t := range tasks {
		count += countAllTasks(t.Children)
	}
	return count
}
