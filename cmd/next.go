package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

var (
	phaseFlag  bool
	streamFlag int
	claimFlag  string
	oneTask    bool
)

var nextCmd = &cobra.Command{
	Use:   "next [file]",
	Short: "Get the next incomplete task",
	Long: `Get the next incomplete task from the specified task file.

This command finds the first task that has incomplete work (either the task itself
or any of its subtasks are not marked as completed) using depth-first traversal.

With the --phase flag, the command returns all pending tasks from the next phase
(the first phase in document order containing pending tasks) instead of a single task.

With the --one flag, the command shows only the first incomplete subtask at each
level of the hierarchy, creating a single path from the parent to the first
incomplete leaf task.

Stream and Claim Support:
- --stream N: Filter tasks to only those in stream N
- --claim AGENT_ID: Claim the task(s) by setting status to in-progress and owner
- --stream N --claim AGENT_ID: Claim ALL ready tasks in stream N
- --claim AGENT_ID (without --stream): Claim only the single next ready task

If no filename is provided and git discovery is enabled in configuration, the file
will be automatically discovered based on the current git branch using the configured
template pattern.

Supports multiple output formats:
- table: Human-readable table format (default)
- json: Structured JSON data
- markdown: Markdown format

The output includes the incomplete task and its subtasks, along with any reference
documents defined in the task file's front matter.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runNext,
}

func init() {
	rootCmd.AddCommand(nextCmd)
	nextCmd.Flags().BoolVar(&phaseFlag, "phase", false, "get all tasks from next phase")
	nextCmd.Flags().IntVarP(&streamFlag, "stream", "s", 0, "filter to specific stream")
	nextCmd.Flags().StringVarP(&claimFlag, "claim", "c", "", "claim task(s) with agent ID")
	nextCmd.Flags().BoolVarP(&oneTask, "one", "1", false, "show only first incomplete subtask at each level")
}

func runNext(cmd *cobra.Command, args []string) error {
	// Resolve filename using git discovery if needed
	filename, err := resolveFilename(args)
	if err != nil {
		return err
	}

	if verbose {
		verboseStderr("Using task file: %s", filename)
	}

	// Validate stream flag early
	if streamFlag < 0 {
		return fmt.Errorf("stream must be non-negative, got %d", streamFlag)
	}

	// Handle claim mode (with or without phase/stream)
	if claimFlag != "" {
		return runNextWithClaim(filename)
	}

	// Handle phase mode (without claim)
	if phaseFlag {
		return runNextPhase(filename)
	}

	// Handle stream filter mode (without claim or phase)
	if streamFlag > 0 {
		return runNextWithStream(filename)
	}

	// Parse the task file
	taskList, err := task.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read task file: %w", err)
	}

	if verbose {
		verboseStderr("Title: %s", taskList.Title)
		verboseStderr("Total tasks: %d", countAllTasks(taskList.Tasks))
	}

	// Find the next incomplete task
	nextTask := task.FindNextIncompleteTask(taskList.Tasks)
	if nextTask == nil {
		return outputNextEmpty("All tasks are complete!")
	}

	// Apply --one filter if requested
	if oneTask {
		task.FilterToFirstIncompletePath(nextTask)
	}

	if verbose {
		verboseStderr("Found next task: %s (with %d incomplete subtasks)",
			nextTask.Title, len(nextTask.IncompleteChildren))
	}

	// Create output based on format
	switch format {
	case formatJSON:
		return outputNextTaskJSON(nextTask, taskList.FrontMatter)
	case formatMarkdown:
		return outputNextTaskMarkdown(nextTask, taskList.FrontMatter)
	default:
		return outputNextTaskTable(nextTask, taskList.FrontMatter)
	}
}

// runNextWithStream handles the --stream flag without claiming
func runNextWithStream(filename string) error {
	// Parse the task file
	taskList, err := task.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read task file: %w", err)
	}

	// Build dependency index
	index := task.BuildDependencyIndex(taskList.Tasks)

	// Find ready tasks (pending, unblocked, unclaimed)
	readyTasks := getReadyTasks(taskList.Tasks, index)

	// Filter by stream
	streamTasks := task.FilterByStream(readyTasks, streamFlag)

	if len(streamTasks) == 0 {
		return outputNextStreamEmpty(streamFlag)
	}

	// Return the first ready task in the stream
	nextTask := &task.TaskWithContext{
		Task:               &streamTasks[0],
		IncompleteChildren: filterIncompleteChildren(streamTasks[0].Children),
	}

	switch format {
	case formatJSON:
		return outputNextTaskJSONWithStream(nextTask, taskList.FrontMatter, index)
	case formatMarkdown:
		return outputNextTaskMarkdown(nextTask, taskList.FrontMatter)
	default:
		return outputNextTaskTable(nextTask, taskList.FrontMatter)
	}
}

// runNextWithClaim handles the --claim flag (with or without --stream)
func runNextWithClaim(filename string) error {
	// Parse the task file
	taskList, err := task.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read task file: %w", err)
	}

	// Build dependency index
	index := task.BuildDependencyIndex(taskList.Tasks)

	var taskIDsToClaim []string

	switch {
	case phaseFlag && streamFlag > 0:
		// Handle --phase --stream --claim combination
		phaseResult, err := task.FindNextPhaseTasksForStream(filename, streamFlag)
		if err != nil {
			return fmt.Errorf("failed to find next phase tasks for stream %d: %w", streamFlag, err)
		}

		if phaseResult == nil {
			return outputClaimEmpty(streamFlag)
		}

		// Filter to only ready tasks (pending, no owner, not blocked)
		for i := range phaseResult.Tasks {
			t := &phaseResult.Tasks[i]
			if t.Status == task.Pending && t.Owner == "" && !index.IsBlocked(t) {
				taskIDsToClaim = append(taskIDsToClaim, t.ID)
			}
		}
	case streamFlag > 0:
		// Claim all ready tasks in the specified stream
		readyTasks := getReadyTasks(taskList.Tasks, index)
		filteredTasks := task.FilterByStream(readyTasks, streamFlag)
		for _, t := range filteredTasks {
			taskIDsToClaim = append(taskIDsToClaim, t.ID)
		}
	default:
		// Claim only the single next ready task
		readyTasks := getReadyTasks(taskList.Tasks, index)
		if len(readyTasks) > 0 {
			taskIDsToClaim = append(taskIDsToClaim, readyTasks[0].ID)
		}
	}

	if len(taskIDsToClaim) == 0 {
		return outputClaimEmpty(streamFlag)
	}

	// Claim the tasks (set status to in-progress and owner)
	for _, taskID := range taskIDsToClaim {
		taskPtr := taskList.FindTask(taskID)
		if taskPtr != nil {
			taskPtr.Status = task.InProgress
			taskPtr.Owner = claimFlag
		}
	}

	// Write the updated task list back to file
	if err := taskList.WriteFile(filename); err != nil {
		return fmt.Errorf("failed to write task file: %w", err)
	}

	// Rebuild index after modification
	index = task.BuildDependencyIndex(taskList.Tasks)

	// Re-fetch the claimed tasks with updated data
	var claimedTasks []task.Task
	for _, taskID := range taskIDsToClaim {
		t := taskList.FindTask(taskID)
		if t != nil {
			claimedTasks = append(claimedTasks, *t)
		}
	}

	// Output the claimed tasks
	switch format {
	case formatJSON:
		return outputClaimJSON(claimedTasks, taskList.FrontMatter, index, streamFlag)
	case formatMarkdown:
		return outputClaimMarkdown(claimedTasks, taskList.FrontMatter)
	default:
		return outputClaimTable(claimedTasks, taskList.FrontMatter)
	}
}

// getReadyTasks finds all tasks that are ready to be worked on:
// - Status is Pending (not InProgress or Completed)
// - All blockers are completed
// - No owner assigned
func getReadyTasks(tasks []task.Task, index *task.DependencyIndex) []task.Task {
	var ready []task.Task

	var findReady func(tasks []task.Task)
	findReady = func(taskList []task.Task) {
		for i := range taskList {
			t := &taskList[i]
			// Check if task is ready:
			// - Pending status
			// - Not blocked (all dependencies completed)
			// - No owner
			if t.Status == task.Pending && !index.IsBlocked(t) && t.Owner == "" {
				ready = append(ready, *t)
			}
			// Check children
			if len(t.Children) > 0 {
				findReady(t.Children)
			}
		}
	}

	findReady(tasks)
	return ready
}

// filterIncompleteChildren returns children that have incomplete work
func filterIncompleteChildren(children []task.Task) []task.Task {
	var incomplete []task.Task
	for _, child := range children {
		if child.Status != task.Completed {
			incomplete = append(incomplete, child)
		}
	}
	return incomplete
}

// outputNextTaskTable renders the next task in table format
func outputNextTaskTable(nextTask *task.TaskWithContext, frontMatter *task.FrontMatter) error {
	// Build task data for display
	var taskData []map[string]any

	// Add the main task
	taskRecord := map[string]any{
		"ID":     nextTask.ID,
		"Title":  nextTask.Title,
		"Status": formatStatus(nextTask.Status),
		"Level":  getTaskLevel(nextTask.ID),
	}
	taskData = append(taskData, taskRecord)

	// Add incomplete children
	for _, child := range nextTask.IncompleteChildren {
		childRecord := map[string]any{
			"ID":     child.ID,
			"Title":  child.Title,
			"Status": formatStatus(child.Status),
			"Level":  getTaskLevel(child.ID),
		}
		taskData = append(taskData, childRecord)

		// Recursively add incomplete grandchildren
		addIncompleteChildrenToData(&child, &taskData)
	}

	// Create table document
	builder := output.New().
		Table("Next Task", taskData, output.WithKeys("ID", "Title", "Status", "Level"))

	// Add task details if present
	if len(nextTask.Details) > 0 {
		detailData := make([]map[string]any, len(nextTask.Details))
		for i, detail := range nextTask.Details {
			detailData[i] = map[string]any{
				"Detail": detail,
			}
		}
		builder = builder.Table("Task Details", detailData, output.WithKeys("Detail"))
	}

	// Add task-level references if present
	if len(nextTask.References) > 0 {
		taskRefData := make([]map[string]any, len(nextTask.References))
		for i, ref := range nextTask.References {
			taskRefData[i] = map[string]any{
				"Path": ref,
			}
		}
		builder = builder.Table("Task References", taskRefData, output.WithKeys("Path"))
	}

	// Add front matter references if present
	if frontMatter != nil && len(frontMatter.References) > 0 {
		refData := make([]map[string]any, len(frontMatter.References))
		for i, ref := range frontMatter.References {
			refData[i] = map[string]any{
				"Path": ref,
			}
		}
		builder = builder.Table("Reference Documents", refData, output.WithKeys("Path"))
	}

	doc := builder.Build()

	// Render the document
	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	return out.Render(context.Background(), doc)
}

// outputNextTaskMarkdown renders the next task in markdown format
func outputNextTaskMarkdown(nextTask *task.TaskWithContext, frontMatter *task.FrontMatter) error {
	var result strings.Builder

	// Add main task
	result.WriteString("# Next Task\n\n")
	result.WriteString(fmt.Sprintf("- %s %s. %s\n",
		formatStatusMarkdown(nextTask.Status), nextTask.ID, nextTask.Title))

	// Add task details if present
	if len(nextTask.Details) > 0 {
		for _, detail := range nextTask.Details {
			result.WriteString(fmt.Sprintf("  %s\n", detail))
		}
	}

	// Add incomplete children
	for _, child := range nextTask.IncompleteChildren {
		result.WriteString(renderTaskMarkdown(&child, "  "))
	}

	// Add task-level references if present
	if len(nextTask.References) > 0 {
		result.WriteString("\n## Task References\n\n")
		for _, ref := range nextTask.References {
			result.WriteString(fmt.Sprintf("- %s\n", ref))
		}
	}

	// Add front matter references if present
	if frontMatter != nil && len(frontMatter.References) > 0 {
		result.WriteString("\n## References\n\n")
		for _, ref := range frontMatter.References {
			result.WriteString(fmt.Sprintf("- %s\n", ref))
		}
	}

	fmt.Print(result.String())
	return nil
}

// outputNextTaskJSON renders the next task in JSON format
func outputNextTaskJSON(nextTask *task.TaskWithContext, frontMatter *task.FrontMatter) error {
	// Create a simplified structure for JSON output
	type TaskJSON struct {
		ID         string     `json:"id"`
		Title      string     `json:"title"`
		Status     string     `json:"status"`
		Details    []string   `json:"details,omitempty"`
		References []string   `json:"references,omitempty"`
		Children   []TaskJSON `json:"children,omitempty"`
	}

	type OutputJSON struct {
		Success               bool     `json:"success"`
		NextTask              TaskJSON `json:"next_task"`
		TaskReferences        []string `json:"task_references,omitempty"`
		FrontMatterReferences []string `json:"front_matter_references,omitempty"`
	}

	// Convert main task
	var convertTask func(t *task.Task, children []task.Task) TaskJSON
	convertTask = func(t *task.Task, children []task.Task) TaskJSON {
		tj := TaskJSON{
			ID:     t.ID,
			Title:  t.Title,
			Status: formatStatus(t.Status),
		}
		if len(t.Details) > 0 {
			tj.Details = t.Details
		}
		if len(t.References) > 0 {
			tj.References = t.References
		}
		for i := range children {
			tj.Children = append(tj.Children, convertTask(&children[i], children[i].Children))
		}
		return tj
	}

	// Use IncompleteChildren if --one flag is set, otherwise use all Children for context
	childrenToShow := nextTask.Task.Children
	if oneTask {
		childrenToShow = nextTask.IncompleteChildren
	}

	output := OutputJSON{
		Success:  true,
		NextTask: convertTask(nextTask.Task, childrenToShow),
	}

	// Add task-level references if present
	if len(nextTask.References) > 0 {
		output.TaskReferences = nextTask.References
	}

	// Add front matter references if present
	if frontMatter != nil && len(frontMatter.References) > 0 {
		output.FrontMatterReferences = frontMatter.References
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Print(string(jsonData))
	return nil
}

// Helper functions

// addIncompleteChildrenToData recursively adds incomplete children to table data
func addIncompleteChildrenToData(parentTask *task.Task, taskData *[]map[string]any) {
	for _, child := range parentTask.Children {
		if child.Status != task.Completed {
			childRecord := map[string]any{
				"ID":     child.ID,
				"Title":  child.Title,
				"Status": formatStatus(child.Status),
				"Level":  getTaskLevel(child.ID),
			}
			*taskData = append(*taskData, childRecord)

			// Recursively add its incomplete children
			addIncompleteChildrenToData(&child, taskData)
		}
	}
}

// renderTaskMarkdown recursively renders a task in markdown format
func renderTaskMarkdown(t *task.Task, indent string) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("%s- %s %s. %s\n",
		indent, formatStatusMarkdown(t.Status), t.ID, t.Title))

	// Add task details if present
	if len(t.Details) > 0 {
		for _, detail := range t.Details {
			result.WriteString(fmt.Sprintf("%s  %s\n", indent, detail))
		}
	}

	// Add task references if present (for individual tasks)
	if len(t.References) > 0 {
		refList := strings.Join(t.References, ", ")
		result.WriteString(fmt.Sprintf("%s  References: %s\n", indent, refList))
	}

	for _, child := range t.Children {
		if child.Status != task.Completed {
			result.WriteString(renderTaskMarkdown(&child, indent+"  "))
		}
	}

	return result.String()
}

// runNextPhase handles the --phase flag functionality
func runNextPhase(filename string) error {
	var phaseResult *task.PhaseTasksResult
	var err error

	// Use stream-aware phase discovery if stream is specified
	if streamFlag > 0 {
		phaseResult, err = task.FindNextPhaseTasksForStream(filename, streamFlag)
		if err != nil {
			return fmt.Errorf("failed to find next phase tasks for stream %d: %w", streamFlag, err)
		}

		if phaseResult == nil {
			return outputNextPhaseEmpty(fmt.Sprintf("No ready tasks found in stream %d", streamFlag))
		}
	} else {
		// Use existing behavior for backward compatibility
		phaseResult, err = task.FindNextPhaseTasks(filename)
		if err != nil {
			return fmt.Errorf("failed to find next phase tasks: %w", err)
		}

		if phaseResult == nil {
			return outputNextPhaseEmpty("No pending tasks found in any phase!")
		}
	}

	// Parse file for front matter and building dependency index
	taskList, err := task.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read task file for front matter: %w", err)
	}

	// Build dependency index
	index := task.BuildDependencyIndex(taskList.Tasks)

	if verbose {
		if phaseResult.PhaseName != "" {
			if streamFlag > 0 {
				verboseStderr("Found %d tasks in stream %d from phase '%s'", len(phaseResult.Tasks), streamFlag, phaseResult.PhaseName)
			} else {
				verboseStderr("Found %d pending tasks in phase '%s'", len(phaseResult.Tasks), phaseResult.PhaseName)
			}
		} else {
			verboseStderr("Found %d pending tasks (no phases in document)", len(phaseResult.Tasks))
		}
	}

	// Create output based on format
	switch format {
	case formatJSON:
		return outputPhaseTasksJSONWithStreams(phaseResult, taskList.FrontMatter, index, taskList.Tasks)
	case formatMarkdown:
		return outputPhaseTasksMarkdown(phaseResult, taskList.FrontMatter, index)
	default:
		return outputPhaseTasksTable(phaseResult, taskList.FrontMatter, index)
	}
}

// outputPhaseTasksTable renders phase tasks in table format
func outputPhaseTasksTable(phaseResult *task.PhaseTasksResult, frontMatter *task.FrontMatter, index *task.DependencyIndex) error {
	// Build task data for display
	var taskData []map[string]any

	// Add all tasks from the phase
	for _, t := range phaseResult.Tasks {
		taskRecord := map[string]any{
			"ID":     t.ID,
			"Title":  t.Title,
			"Status": formatStatusWithBlocking(&t, index),
			"Level":  getTaskLevel(t.ID),
		}
		if phaseResult.PhaseName != "" {
			taskRecord["Phase"] = phaseResult.PhaseName
		}
		taskData = append(taskData, taskRecord)

		// Add all children (incomplete and complete for context)
		addAllChildrenToData(&t, &taskData, phaseResult.PhaseName, index)
	}

	// Create table document
	keys := []string{"ID", "Title", "Status", "Level"}
	if phaseResult.PhaseName != "" {
		keys = append(keys, "Phase")
	}

	tableTitle := "Next Phase Tasks"
	if phaseResult.PhaseName != "" {
		tableTitle = fmt.Sprintf("Next Phase Tasks (%s)", phaseResult.PhaseName)
	}

	builder := output.New().
		Table(tableTitle, taskData, output.WithKeys(keys...))

	// Add front matter references if present
	if frontMatter != nil && len(frontMatter.References) > 0 {
		refData := make([]map[string]any, len(frontMatter.References))
		for i, ref := range frontMatter.References {
			refData[i] = map[string]any{
				"Path": ref,
			}
		}
		builder = builder.Table("Reference Documents", refData, output.WithKeys("Path"))
	}

	doc := builder.Build()

	// Render the document
	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	return out.Render(context.Background(), doc)
}

// outputPhaseTasksMarkdown renders phase tasks in markdown format
func outputPhaseTasksMarkdown(phaseResult *task.PhaseTasksResult, frontMatter *task.FrontMatter, index *task.DependencyIndex) error {
	var result strings.Builder

	// Add header
	if phaseResult.PhaseName != "" {
		fmt.Fprintf(&result, "# Next Phase Tasks (%s)\n\n", phaseResult.PhaseName)
	} else {
		result.WriteString("# Next Phase Tasks\n\n")
	}

	// Add all tasks from the phase
	for _, t := range phaseResult.Tasks {
		result.WriteString(renderPhaseTaskMarkdownWithBlocking(&t, "", index))
		result.WriteString("\n")
	}

	// Add front matter references if present
	if frontMatter != nil && len(frontMatter.References) > 0 {
		result.WriteString("\n## References\n\n")
		for _, ref := range frontMatter.References {
			result.WriteString("- ")
			result.WriteString(ref)
			result.WriteString("\n")
		}
	}

	fmt.Print(result.String())
	return nil
}

// renderPhaseTaskMarkdownWithBlocking renders a task with blocking info using strings.Builder
func renderPhaseTaskMarkdownWithBlocking(t *task.Task, indent string, index *task.DependencyIndex) string {
	var b strings.Builder

	b.WriteString(indent)
	b.WriteString("- ")
	b.WriteString(formatStatusMarkdown(t.Status))
	b.WriteString(" ")
	b.WriteString(t.ID)
	b.WriteString(". ")
	b.WriteString(t.Title)

	if t.Status == task.Pending && index != nil && index.IsBlocked(t) {
		blockedBy := index.TranslateToHierarchical(t.BlockedBy)
		if len(blockedBy) > 0 {
			b.WriteString(" (blocked by: ")
			b.WriteString(strings.Join(blockedBy, ", "))
			b.WriteString(")")
		} else {
			b.WriteString(" (blocked)")
		}
	}
	b.WriteString("\n")

	if len(t.Details) > 0 {
		for _, detail := range t.Details {
			b.WriteString(indent)
			b.WriteString("  ")
			b.WriteString(detail)
			b.WriteString("\n")
		}
	}

	if len(t.References) > 0 {
		b.WriteString(indent)
		b.WriteString("  References: ")
		b.WriteString(strings.Join(t.References, ", "))
		b.WriteString("\n")
	}

	for i := range t.Children {
		b.WriteString(renderPhaseTaskMarkdownWithBlocking(&t.Children[i], indent+"  ", index))
	}

	return b.String()
}

// PhaseTaskJSONWithStreams is a task with stream and dependency info for phase output
type PhaseTaskJSONWithStreams struct {
	ID         string                     `json:"id"`
	Title      string                     `json:"title"`
	Status     string                     `json:"status"`
	Stream     int                        `json:"stream"`
	Owner      string                     `json:"owner,omitempty"`
	Blocked    bool                       `json:"blocked"`
	BlockedBy  []string                   `json:"blockedBy,omitempty"`
	Details    []string                   `json:"details,omitempty"`
	References []string                   `json:"references,omitempty"`
	Children   []PhaseTaskJSONWithStreams `json:"children,omitempty"`
}

// StreamsSummary provides summary info about streams in a phase
type StreamsSummary struct {
	ID        int      `json:"id"`
	Ready     []string `json:"ready"`
	Blocked   []string `json:"blocked"`
	Active    []string `json:"active"`
	Available bool     `json:"available"`
}

// PhaseOutputJSONWithStreams is the phase output with stream info
type PhaseOutputJSONWithStreams struct {
	Success               bool                       `json:"success"`
	Count                 int                        `json:"count"`
	PhaseName             string                     `json:"phase_name,omitempty"`
	Tasks                 []PhaseTaskJSONWithStreams `json:"tasks"`
	StreamsSummary        []StreamsSummary           `json:"streams_summary,omitempty"`
	FrontMatterReferences []string                   `json:"front_matter_references,omitempty"`
}

// outputPhaseTasksJSONWithStreams renders phase tasks with stream/dependency info
func outputPhaseTasksJSONWithStreams(phaseResult *task.PhaseTasksResult, frontMatter *task.FrontMatter, index *task.DependencyIndex, allTasks []task.Task) error {
	// Convert tasks with stream info
	var convertTask func(t *task.Task) PhaseTaskJSONWithStreams
	convertTask = func(t *task.Task) PhaseTaskJSONWithStreams {
		tj := PhaseTaskJSONWithStreams{
			ID:        t.ID,
			Title:     t.Title,
			Status:    formatStatus(t.Status),
			Stream:    task.GetEffectiveStream(t),
			Owner:     t.Owner,
			Blocked:   index.IsBlocked(t),
			BlockedBy: index.TranslateToHierarchical(t.BlockedBy),
		}
		if len(t.Details) > 0 {
			tj.Details = t.Details
		}
		if len(t.References) > 0 {
			tj.References = t.References
		}
		for _, child := range t.Children {
			tj.Children = append(tj.Children, convertTask(&child))
		}
		return tj
	}

	outputData := PhaseOutputJSONWithStreams{
		Success: true,
		Count:   len(phaseResult.Tasks),
		Tasks:   []PhaseTaskJSONWithStreams{},
	}

	if phaseResult.PhaseName != "" {
		outputData.PhaseName = phaseResult.PhaseName
	}

	for _, t := range phaseResult.Tasks {
		outputData.Tasks = append(outputData.Tasks, convertTask(&t))
	}

	// Calculate streams summary
	streamsResult := task.AnalyzeStreams(allTasks, index)
	if len(streamsResult.Streams) > 0 {
		outputData.StreamsSummary = make([]StreamsSummary, len(streamsResult.Streams))
		for i, s := range streamsResult.Streams {
			outputData.StreamsSummary[i] = StreamsSummary{
				ID:        s.ID,
				Ready:     s.Ready,
				Blocked:   s.Blocked,
				Active:    s.Active,
				Available: len(s.Ready) > 0,
			}
		}
	}

	// Add front matter references if present
	if frontMatter != nil && len(frontMatter.References) > 0 {
		outputData.FrontMatterReferences = frontMatter.References
	}

	jsonData, err := json.MarshalIndent(outputData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Print(string(jsonData))
	return nil
}

// addAllChildrenToData recursively adds all children (complete and incomplete) to table data.
// When index is non-nil, blocking status is included in the status field.
func addAllChildrenToData(parentTask *task.Task, taskData *[]map[string]any, phaseName string, index *task.DependencyIndex) {
	for _, child := range parentTask.Children {
		childRecord := map[string]any{
			"ID":     child.ID,
			"Title":  child.Title,
			"Status": formatStatusWithBlocking(&child, index),
			"Level":  getTaskLevel(child.ID),
		}
		if phaseName != "" {
			childRecord["Phase"] = phaseName
		}
		*taskData = append(*taskData, childRecord)

		// Recursively add its children
		addAllChildrenToData(&child, taskData, phaseName, index)
	}
}

// formatStatusWithBlocking returns the status string with a blocking/ready indicator
// for pending tasks when a dependency index is available.
func formatStatusWithBlocking(t *task.Task, index *task.DependencyIndex) string {
	status := formatStatus(t.Status)
	if t.Status != task.Pending || index == nil {
		return status
	}
	if index.IsBlocked(t) {
		return status + " (blocked)"
	}
	return status + " (ready)"
}

// NextEmptyResponse is the JSON response structure when no next task exists.
type NextEmptyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// outputNextEmpty handles format-aware output when all tasks are complete.
func outputNextEmpty(message string) error {
	switch format {
	case formatJSON:
		return outputJSON(NextEmptyResponse{
			Success: true,
			Message: message,
			Data:    nil,
		})
	case formatMarkdown:
		outputMarkdownMessage(message)
		return nil
	default:
		outputMessage(message)
		return nil
	}
}

// NextPhaseEmptyResponse is the JSON response structure when no pending phase tasks exist.
type NextPhaseEmptyResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	PhaseName string `json:"phase_name"`
	Tasks     []any  `json:"tasks"`
}

// outputNextPhaseEmpty handles format-aware output when no pending tasks exist in any phase.
func outputNextPhaseEmpty(message string) error {
	switch format {
	case formatJSON:
		return outputJSON(NextPhaseEmptyResponse{
			Success:   true,
			Message:   message,
			PhaseName: "",
			Tasks:     []any{},
		})
	case formatMarkdown:
		outputMarkdownMessage(message)
		return nil
	default:
		outputMessage(message)
		return nil
	}
}

// ============================================================================
// Stream and Claim Output Functions
// ============================================================================

// NextStreamEmptyResponse is the JSON response when no tasks exist in the specified stream.
type NextStreamEmptyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Stream  int    `json:"stream"`
}

// outputNextStreamEmpty handles format-aware output when no tasks in stream.
func outputNextStreamEmpty(stream int) error {
	message := fmt.Sprintf("No ready tasks found in stream %d", stream)
	switch format {
	case formatJSON:
		return outputJSON(NextStreamEmptyResponse{
			Success: true,
			Message: message,
			Stream:  stream,
		})
	case formatMarkdown:
		outputMarkdownMessage(message)
		return nil
	default:
		outputMessage(message)
		return nil
	}
}

// ClaimEmptyResponse is the JSON response when no tasks are available to claim.
type ClaimEmptyResponse struct {
	Success bool       `json:"success"`
	Message string     `json:"message"`
	Stream  int        `json:"stream,omitempty"`
	Claimed []struct{} `json:"claimed"`
}

// outputClaimEmpty handles format-aware output when no tasks to claim.
func outputClaimEmpty(stream int) error {
	var message string
	if stream > 0 {
		message = fmt.Sprintf("No ready tasks to claim in stream %d", stream)
	} else {
		message = "No ready tasks to claim"
	}

	switch format {
	case formatJSON:
		resp := ClaimEmptyResponse{
			Success: true,
			Message: message,
			Claimed: []struct{}{},
		}
		if stream > 0 {
			resp.Stream = stream
		}
		return outputJSON(resp)
	case formatMarkdown:
		outputMarkdownMessage(message)
		return nil
	default:
		outputMessage(message)
		return nil
	}
}

// ClaimTaskJSON represents a claimed task in JSON output
type ClaimTaskJSON struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	Stream    int      `json:"stream"`
	Owner     string   `json:"owner"`
	BlockedBy []string `json:"blockedBy,omitempty"`
}

// ClaimResponse is the JSON response after claiming tasks
type ClaimResponse struct {
	Success bool            `json:"success"`
	Count   int             `json:"count"`
	Stream  int             `json:"stream,omitempty"`
	Claimed []ClaimTaskJSON `json:"claimed"`
}

// outputClaimJSON outputs claimed tasks in JSON format
func outputClaimJSON(claimed []task.Task, frontMatter *task.FrontMatter, index *task.DependencyIndex, stream int) error {
	claimedJSON := make([]ClaimTaskJSON, len(claimed))
	for i, t := range claimed {
		claimedJSON[i] = ClaimTaskJSON{
			ID:        t.ID,
			Title:     t.Title,
			Status:    formatStatus(t.Status),
			Stream:    task.GetEffectiveStream(&t),
			Owner:     t.Owner,
			BlockedBy: index.TranslateToHierarchical(t.BlockedBy),
		}
	}

	resp := ClaimResponse{
		Success: true,
		Count:   len(claimed),
		Claimed: claimedJSON,
	}
	if stream > 0 {
		resp.Stream = stream
	}

	return outputJSON(resp)
}

// outputClaimMarkdown outputs claimed tasks in markdown format
func outputClaimMarkdown(claimed []task.Task, _ *task.FrontMatter) error {
	fmt.Println("# Claimed Tasks")
	fmt.Println()
	for _, t := range claimed {
		fmt.Printf("- [-] %s. %s\n", t.ID, t.Title)
		fmt.Printf("  - Owner: %s\n", t.Owner)
		if t.Stream > 0 {
			fmt.Printf("  - Stream: %d\n", t.Stream)
		}
	}
	return nil
}

// outputClaimTable outputs claimed tasks in table format
func outputClaimTable(claimed []task.Task, _ *task.FrontMatter) error {
	var taskData []map[string]any
	for _, t := range claimed {
		record := map[string]any{
			"ID":     t.ID,
			"Title":  t.Title,
			"Status": formatStatus(t.Status),
			"Owner":  t.Owner,
			"Stream": task.GetEffectiveStream(&t),
		}
		taskData = append(taskData, record)
	}

	builder := output.New().
		Table("Claimed Tasks", taskData, output.WithKeys("ID", "Title", "Status", "Owner", "Stream"))

	doc := builder.Build()
	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	return out.Render(context.Background(), doc)
}

// NextTaskJSONWithStream is the JSON output for a single task with stream/dependency info
type NextTaskJSONWithStream struct {
	Success               bool                   `json:"success"`
	NextTask              TaskJSONWithStreamInfo `json:"next_task"`
	FrontMatterReferences []string               `json:"front_matter_references,omitempty"`
}

// TaskJSONWithStreamInfo is a task with stream and dependency info
type TaskJSONWithStreamInfo struct {
	ID         string                   `json:"id"`
	Title      string                   `json:"title"`
	Status     string                   `json:"status"`
	Stream     int                      `json:"stream"`
	Owner      string                   `json:"owner,omitempty"`
	BlockedBy  []string                 `json:"blockedBy,omitempty"`
	Details    []string                 `json:"details,omitempty"`
	References []string                 `json:"references,omitempty"`
	Children   []TaskJSONWithStreamInfo `json:"children,omitempty"`
}

// outputNextTaskJSONWithStream outputs a single task with stream/dependency info
func outputNextTaskJSONWithStream(nextTask *task.TaskWithContext, frontMatter *task.FrontMatter, index *task.DependencyIndex) error {
	var convertTask func(t *task.Task) TaskJSONWithStreamInfo
	convertTask = func(t *task.Task) TaskJSONWithStreamInfo {
		tj := TaskJSONWithStreamInfo{
			ID:        t.ID,
			Title:     t.Title,
			Status:    formatStatus(t.Status),
			Stream:    task.GetEffectiveStream(t),
			Owner:     t.Owner,
			BlockedBy: index.TranslateToHierarchical(t.BlockedBy),
		}
		if len(t.Details) > 0 {
			tj.Details = t.Details
		}
		if len(t.References) > 0 {
			tj.References = t.References
		}
		for _, child := range t.Children {
			tj.Children = append(tj.Children, convertTask(&child))
		}
		return tj
	}

	resp := NextTaskJSONWithStream{
		Success:  true,
		NextTask: convertTask(nextTask.Task),
	}

	if frontMatter != nil && len(frontMatter.References) > 0 {
		resp.FrontMatterReferences = frontMatter.References
	}

	return outputJSON(resp)
}
