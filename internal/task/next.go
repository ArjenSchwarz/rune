package task

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

const frontMatterDelimiter = "---"

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

// FindNextReadyTask finds the first top-level task that has ready work,
// skipping tasks whose remaining work is blocked by unresolved dependencies
// or claimed by an owner. The returned task is a copy without its
// unavailable subtasks (incomplete work that is blocked or claimed), so they
// are never presented as available. Fully completed subtasks are kept as
// context, as next-task-workflow requirement 1.6 asks.
func FindNextReadyTask(tasks []Task, index *DependencyIndex) *TaskWithContext {
	for i := range tasks {
		if hasReadyWork(&tasks[i], index) {
			pruned := pruneToReadyWork(tasks[i], index)
			return &TaskWithContext{
				Task:               &pruned,
				IncompleteChildren: filterIncompleteChildren(pruned.Children),
			}
		}
	}
	return nil
}

// pruneToReadyWork returns a copy of task keeping only descendants that are
// fully completed or have ready work.
func pruneToReadyWork(task Task, index *DependencyIndex) Task {
	var children []Task
	for i := range task.Children {
		child := &task.Children[i]
		if !hasIncompleteWork(child) || hasReadyWork(child, index) {
			children = append(children, pruneToReadyWork(task.Children[i], index))
		}
	}
	task.Children = children
	return task
}

// hasReadyWork reports whether a task, or a descendant reachable without
// passing through a blocked or claimed task, is incomplete and available.
func hasReadyWork(task *Task, index *DependencyIndex) bool {
	if task.Status != Completed {
		return task.Owner == "" && !index.IsBlocked(task)
	}
	for i := range task.Children {
		if hasReadyWork(&task.Children[i], index) {
			return true
		}
	}
	return false
}

// FilterToFirstReadyPath is like FilterToFirstIncompletePath but follows the
// first child with ready work at each level.
func FilterToFirstReadyPath(taskCtx *TaskWithContext, index *DependencyIndex) {
	if taskCtx == nil {
		return
	}
	taskCtx.IncompleteChildren = filterToFirstReadyPath(taskCtx.IncompleteChildren, index)
}

func filterToFirstReadyPath(children []Task, index *DependencyIndex) []Task {
	for i := range children {
		if hasReadyWork(&children[i], index) {
			filtered := children[i]
			filtered.Children = filterToFirstReadyPath(filtered.Children, index)
			return []Task{filtered}
		}
	}
	return []Task{}
}

// HasIncompleteWork checks if task or any subtask is incomplete.
// A task has incomplete work if its own status is not Completed,
// or if any descendant has incomplete work (recursive).
func HasIncompleteWork(task *Task) bool {
	return hasIncompleteWork(task)
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

// FilterToFirstIncompletePath modifies a TaskWithContext to show only the first
// incomplete child at each level, creating a single path through the hierarchy
func FilterToFirstIncompletePath(taskCtx *TaskWithContext) {
	if taskCtx == nil {
		return
	}
	taskCtx.IncompleteChildren = filterToFirstIncompletePath(taskCtx.IncompleteChildren)
}

// filterToFirstIncompletePath returns a slice containing only the first child
// with incomplete work, recursively filtering its children as well
func filterToFirstIncompletePath(children []Task) []Task {
	for i := range children {
		if hasIncompleteWork(&children[i]) {
			// Found first incomplete child, filter its children recursively
			filtered := children[i]
			filtered.Children = filterToFirstIncompletePath(filtered.Children)
			return []Task{filtered}
		}
	}
	return []Task{}
}

// PhaseTasksResult represents tasks from a phase along with the phase name
type PhaseTasksResult struct {
	PhaseName string
	Tasks     []Task
}

// skipFrontMatter removes YAML front matter from lines if present.
// Returns the lines without front matter delimiters and content.
func skipFrontMatter(content string, lines []string) []string {
	if !strings.HasPrefix(strings.TrimSpace(content), frontMatterDelimiter) {
		return lines
	}

	inFrontMatter := false
	frontMatterClosed := false
	newLines := []string{}
	for _, line := range lines {
		if !frontMatterClosed && strings.TrimSpace(line) == frontMatterDelimiter {
			if inFrontMatter {
				// Closing delimiter
				inFrontMatter = false
				frontMatterClosed = true
				continue
			}
			// Opening delimiter
			inFrontMatter = true
			continue
		}
		if !inFrontMatter && frontMatterClosed {
			newLines = append(newLines, line)
		}
	}
	if frontMatterClosed {
		return newLines
	}
	return lines
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
	lines := splitLines(string(content))
	lines = skipFrontMatter(string(content), lines)

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

// extractPhasesWithTaskRanges parses the document and associates top-level
// tasks with their phases. It maps raw markdown task lines to parsed tasks by
// position, because the parser assigns sequential IDs regardless of what IDs
// appear in the markdown. Tasks before the first phase header are not included.
func extractPhasesWithTaskRanges(lines []string, allTasks []Task) []PhaseWithTasks {
	var phases []PhaseWithTasks
	walkPhaseLines(lines,
		func(name string) {
			phases = append(phases, PhaseWithTasks{Name: name, Tasks: []Task{}})
		},
		func(position int) {
			if len(phases) > 0 && position <= len(allTasks) {
				current := &phases[len(phases)-1]
				current.Tasks = append(current.Tasks, allTasks[position-1])
			}
		})
	return phases
}

// getAllPendingTasks returns the tasks that have pending work, with each
// task's descendant tree pruned to only the subtrees that still have
// incomplete work. Completed descendants without incomplete work below them
// are dropped so phase output only contains actionable work.
func getAllPendingTasks(tasks []Task) []Task {
	var pending []Task
	for _, task := range tasks {
		if hasIncompleteWork(&task) {
			pending = append(pending, withIncompleteDescendants(task))
		}
	}
	return pending
}

// withIncompleteDescendants returns a copy of task whose Children (at every
// depth) are limited to subtasks that have incomplete work. The original task
// tree is not modified.
func withIncompleteDescendants(task Task) Task {
	return withIncompleteDescendantsDepth(task, 0, 100)
}

func withIncompleteDescendantsDepth(task Task, depth, maxDepth int) Task {
	if len(task.Children) == 0 {
		return task
	}
	if depth >= maxDepth {
		// Prevent runaway recursion on malformed data
		task.Children = nil
		return task
	}
	var children []Task
	for i := range task.Children {
		if hasIncompleteWork(&task.Children[i]) {
			children = append(children, withIncompleteDescendantsDepth(task.Children[i], depth+1, maxDepth))
		}
	}
	task.Children = children
	return task
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
// Each returned task's children are limited to stream N.
//
// A document without phase headers is treated as a single unnamed phase,
// matching FindNextPhaseTasks. Returns nil if no phase contains ready tasks
// in the specified stream.
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
	lines := splitLines(string(content))
	lines = skipFrontMatter(string(content), lines)

	// Without phase headers the whole document is one unnamed phase
	phases := []PhaseWithTasks{{Name: "", Tasks: taskList.Tasks}}
	if hasPhases(lines) {
		phases = extractPhasesWithTaskRanges(lines, taskList.Tasks)
	}

	// Find the first phase with ready tasks in the specified stream
	for _, phase := range phases {
		// Get all tasks in the stream from this phase
		streamTasks := FilterByStream(phase.Tasks, stream)

		// Check if any task in the stream is ready
		if hasReadyTaskInStream(phase.Tasks, stream, index) {
			// Return all non-completed stream tasks from this phase
			var result []Task
			for _, t := range streamTasks {
				if t.Status != Completed {
					result = append(result, PruneToStream(withIncompleteDescendants(t), stream))
				}
			}
			if len(result) == 0 {
				continue
			}
			return &PhaseTasksResult{
				PhaseName: phase.Name,
				Tasks:     result,
			}, nil
		}
	}

	// No phase has ready tasks in the specified stream
	return nil, nil
}
