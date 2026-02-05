package task

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

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

// PhaseTasksResult represents tasks from a phase along with the phase name
type PhaseTasksResult struct {
	PhaseName string
	Tasks     []Task
}

// FindNextPhaseTasks finds all pending tasks from the first phase that has pending tasks
func FindNextPhaseTasks(filepath string) (*PhaseTasksResult, error) {
	// Read file content to parse phases and tasks together
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	// Parse the task list normally
	taskList, err := ParseMarkdown(content)
	if err != nil {
		return nil, err
	}

	// Also parse the raw content to extract phase information
	lines := strings.Split(string(content), "\n")

	// Skip front matter if present
	if strings.HasPrefix(strings.TrimSpace(string(content)), "---") {
		inFrontMatter := false
		frontMatterCount := 0
		newLines := []string{}
		for _, line := range lines {
			if strings.TrimSpace(line) == "---" {
				frontMatterCount++
				if frontMatterCount == 2 {
					inFrontMatter = false
					continue
				} else {
					inFrontMatter = true
					continue
				}
			}
			if !inFrontMatter && frontMatterCount > 0 {
				newLines = append(newLines, line)
			}
		}
		if frontMatterCount >= 2 {
			lines = newLines
		}
	}

	// If no phases exist, return all pending tasks
	if !hasPhases(lines) {
		pendingTasks := getAllPendingTasks(taskList.Tasks)
		if len(pendingTasks) == 0 {
			return nil, nil
		}
		return &PhaseTasksResult{
			PhaseName: "",
			Tasks:     pendingTasks,
		}, nil
	}

	// Find phases and their task ranges
	phases := extractPhasesWithTaskRanges(lines, taskList.Tasks)

	// Find the first phase with pending tasks
	for _, phase := range phases {
		pendingTasks := getAllPendingTasks(phase.Tasks)
		if len(pendingTasks) > 0 {
			return &PhaseTasksResult{
				PhaseName: phase.Name,
				Tasks:     pendingTasks,
			}, nil
		}
	}

	// No phase has pending tasks
	return nil, nil
}

// PhaseWithTasks represents a phase and its associated tasks
type PhaseWithTasks struct {
	Name  string
	Tasks []Task
}

// hasPhases checks if the document contains any H2 headers (phases)
func hasPhases(lines []string) bool {
	return slices.ContainsFunc(lines, phaseHeaderPattern.MatchString)
}

// extractPhasesWithTaskRanges parses the document and associates tasks with their phases
func extractPhasesWithTaskRanges(lines []string, allTasks []Task) []PhaseWithTasks {
	var phases []PhaseWithTasks
	var currentPhase *PhaseWithTasks
	taskMap := createTaskMap(allTasks)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for phase header
		if matches := phaseHeaderPattern.FindStringSubmatch(line); matches != nil {
			// Save previous phase if exists
			if currentPhase != nil {
				phases = append(phases, *currentPhase)
			}

			// Start new phase
			currentPhase = &PhaseWithTasks{
				Name:  strings.TrimSpace(matches[1]),
				Tasks: []Task{},
			}
		} else if currentPhase != nil && taskLinePattern.MatchString(line) {
			// Extract task ID from the line
			if taskMatches := taskLinePattern.FindStringSubmatch(line); len(taskMatches) >= 4 {
				taskID := taskMatches[3]
				// Only add top-level tasks to phases
				if !strings.Contains(taskID, ".") {
					if task, exists := taskMap[taskID]; exists {
						currentPhase.Tasks = append(currentPhase.Tasks, task)
					}
				}
			}
		}
	}

	// Don't forget the last phase
	if currentPhase != nil {
		phases = append(phases, *currentPhase)
	}

	return phases
}

// createTaskMap creates a map of task ID to Task for quick lookup
func createTaskMap(tasks []Task) map[string]Task {
	taskMap := make(map[string]Task)

	var addToMap func([]Task)
	addToMap = func(taskList []Task) {
		for _, task := range taskList {
			taskMap[task.ID] = task
			// Recursively add children
			addToMap(task.Children)
		}
	}

	addToMap(tasks)
	return taskMap
}

// getAllPendingTasks recursively collects all tasks with pending work
func getAllPendingTasks(tasks []Task) []Task {
	var pending []Task
	for _, task := range tasks {
		if hasIncompleteWork(&task) {
			pending = append(pending, task)
		}
	}
	return pending
}

// hasReadyTaskInStream checks if any task in the specified stream is ready.
// A task is ready if it has Pending status, no owner assigned, and is not blocked.
func hasReadyTaskInStream(tasks []Task, stream int, index *DependencyIndex) bool {
	if index == nil {
		return false
	}

	streamTasks := FilterByStream(tasks, stream)
	for i := range streamTasks {
		task := &streamTasks[i]
		// Check if task is ready: Pending status, no owner, not blocked
		if task.Status == Pending && task.Owner == "" && !index.IsBlocked(task) {
			return true
		}
	}
	return false
}

// FindNextPhaseTasksForStream finds all tasks in the specified stream from the first phase
// that has at least one ready task in that stream.
//
// Phase selection requires at least one "ready" task:
// - Status is Pending
// - No owner assigned
// - Not blocked (all dependencies completed)
//
// Task return includes ALL stream N tasks from the phase (including blocked).
//
// Returns nil if no phase contains ready tasks in the specified stream,
// or if the document has no phases.
func FindNextPhaseTasksForStream(filepath string, stream int) (*PhaseTasksResult, error) {
	if stream <= 0 {
		return nil, fmt.Errorf("stream must be positive, got %d", stream)
	}

	// Read and parse the file
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	taskList, err := ParseMarkdown(content)
	if err != nil {
		return nil, fmt.Errorf("parsing markdown: %w", err)
	}

	// Build dependency index for blocking checks
	index := BuildDependencyIndex(taskList.Tasks)

	// Parse raw content to extract phase information
	lines := strings.Split(string(content), "\n")

	// Skip front matter if present
	if strings.HasPrefix(strings.TrimSpace(string(content)), "---") {
		inFrontMatter := false
		frontMatterCount := 0
		newLines := []string{}
		for _, line := range lines {
			if strings.TrimSpace(line) == "---" {
				frontMatterCount++
				if frontMatterCount == 2 {
					inFrontMatter = false
					continue
				} else {
					inFrontMatter = true
					continue
				}
			}
			if !inFrontMatter && frontMatterCount > 0 {
				newLines = append(newLines, line)
			}
		}
		if frontMatterCount >= 2 {
			lines = newLines
		}
	}

	// Check if document has phases
	if !hasPhases(lines) {
		return nil, nil
	}

	// Extract phases with their tasks
	phases := extractPhasesWithTaskRanges(lines, taskList.Tasks)

	// Find the first phase with ready tasks in the specified stream
	for _, phase := range phases {
		// Get all tasks in the stream from this phase
		streamTasks := FilterByStream(phase.Tasks, stream)

		// Check if any task in the stream is ready
		if hasReadyTaskInStream(phase.Tasks, stream, index) {
			// Return all stream tasks from this phase (including blocked)
			return &PhaseTasksResult{
				PhaseName: phase.Name,
				Tasks:     streamTasks,
			}, nil
		}
	}

	// No phase has ready tasks in the specified stream
	return nil, nil
}
