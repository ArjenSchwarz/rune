package task

import (
	"fmt"
	"strings"
)

// PhaseMarker represents a phase header found during parsing
// This is a transient structure used only during parsing/rendering
type PhaseMarker struct {
	Name        string // Phase name from H2 header
	AfterTaskID string // ID of task that precedes this phase (empty if at start)
}

// addPhase creates a markdown H2 header line for a new phase
// The phase name is not trimmed or validated - it's preserved exactly as provided
func addPhase(name string) string {
	return fmt.Sprintf("## %s\n", name)
}

// getTaskPhase returns the name of the phase that contains the specified task
// It determines phase membership by document position - a task belongs to the
// most recent phase header that precedes it in the document.
// For subtasks (IDs with dots), it finds the parent task's phase.
// Returns empty string if the task is not within any phase or doesn't exist.
// Uses positional counting to map raw markdown IDs to parsed sequential IDs.
func getTaskPhase(taskID string, content []byte) string {
	if taskID == "" {
		return ""
	}

	// For subtasks, find the parent task ID
	parentID := taskID
	if before, _, ok := strings.Cut(taskID, "."); ok {
		parentID = before
	}

	lines := splitLines(string(content))
	currentPhase := ""
	topLevelTaskCount := 0

	for _, line := range lines {
		// Check if this is a phase header
		if matches := phaseHeaderPattern.FindStringSubmatch(line); matches != nil {
			currentPhase = strings.TrimSpace(matches[1])
			continue
		}

		// Check if this line contains a top-level task
		if taskMatches := taskLinePattern.FindStringSubmatch(line); len(taskMatches) >= 4 {
			rawID := taskMatches[3]
			if !strings.Contains(rawID, ".") {
				topLevelTaskCount++
				sequentialID := fmt.Sprintf("%d", topLevelTaskCount)
				if sequentialID == parentID {
					return currentPhase
				}
			}
		}
	}

	return ""
}

// buildTaskPhaseMap creates a map from sequential parsed task IDs to phase names
// in a single pass through the document lines. This is more efficient than calling
// getTaskPhase for each task individually.
// Uses positional counting to map raw markdown IDs to parsed sequential IDs.
func buildTaskPhaseMap(lines []string) map[string]string {
	taskPhaseMap := make(map[string]string)
	currentPhase := ""
	topLevelTaskCount := 0

	for _, line := range lines {
		// Check if this is a phase header
		if matches := phaseHeaderPattern.FindStringSubmatch(line); matches != nil {
			currentPhase = strings.TrimSpace(matches[1])
			continue
		}

		// Check if this line contains a top-level task
		if taskMatches := taskLinePattern.FindStringSubmatch(line); len(taskMatches) >= 4 {
			rawID := taskMatches[3]
			if !strings.Contains(rawID, ".") {
				topLevelTaskCount++
				if currentPhase != "" {
					sequentialID := fmt.Sprintf("%d", topLevelTaskCount)
					taskPhaseMap[sequentialID] = currentPhase
				}
			}
		}
	}

	return taskPhaseMap
}

// getNextPhaseTasks returns all pending/in-progress tasks from the first phase
// that contains non-completed tasks. It scans phases in document order and returns
// tasks from the first phase with pending work.
// Returns empty slice and empty phase name if no phases have pending tasks.
func getNextPhaseTasks(content []byte) ([]Task, string) {
	taskList, err := ParseMarkdown(content)
	if err != nil {
		return nil, ""
	}

	lines := splitLines(string(content))
	markers := ExtractPhaseMarkers(lines)

	// If no phases exist, return empty
	if len(markers) == 0 {
		return nil, ""
	}

	// Build a map to track which phase each top-level task belongs to (single pass)
	topLevelPhaseMap := buildTaskPhaseMap(lines)

	// Extend map to include all tasks (children inherit parent's phase)
	taskPhaseMap := make(map[string]string)
	for _, task := range taskList.Tasks {
		// Get parent ID for subtasks
		parentID := task.ID
		if idx := strings.Index(task.ID, "."); idx != -1 {
			parentID = task.ID[:idx]
		}
		// Look up parent's phase
		if phase, exists := topLevelPhaseMap[parentID]; exists {
			taskPhaseMap[task.ID] = phase
		}
	}

	// Find the first phase (in document order) with pending tasks
	for _, marker := range markers {
		var pendingTasks []Task
		for _, task := range taskList.Tasks {
			// Check if task belongs to this phase and has incomplete work
			// (either the task itself is not completed, or any descendant is incomplete)
			if taskPhaseMap[task.ID] == marker.Name && hasIncompleteWork(&task) {
				pendingTasks = append(pendingTasks, task)
			}
		}

		if len(pendingTasks) > 0 {
			return pendingTasks, marker.Name
		}
	}

	return nil, ""
}

// findPhasePosition locates a phase in the document and returns whether it was found
// and the ID of the task that precedes it (empty string if phase is at document start).
// When duplicate phase names exist, returns the first occurrence.
// Phase names are case-sensitive.
func findPhasePosition(phaseName string, content []byte) (found bool, afterTaskID string) {
	lines := splitLines(string(content))
	markers := ExtractPhaseMarkers(lines)

	for _, marker := range markers {
		if marker.Name == phaseName {
			return true, marker.AfterTaskID
		}
	}

	return false, ""
}
