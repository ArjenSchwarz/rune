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
	phaseFlag bool
)

var nextCmd = &cobra.Command{
	Use:   "next [file]",
	Short: "Get the next incomplete task",
	Long: `Get the next incomplete task from the specified task file.

This command finds the first task that has incomplete work (either the task itself
or any of its subtasks are not marked as completed) using depth-first traversal.

With the --phase flag, the command returns all pending tasks from the next phase
(the first phase in document order containing pending tasks) instead of a single task.

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

	// Handle phase mode
	if phaseFlag {
		return runNextPhase(filename)
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
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	return out.Render(context.Background(), doc)
}

// outputNextTaskMarkdown renders the next task in markdown format
func outputNextTaskMarkdown(nextTask *task.TaskWithContext, frontMatter *task.FrontMatter) error {
	var result string

	// Add main task
	result += "# Next Task\n\n"
	result += fmt.Sprintf("- %s %s. %s\n",
		formatStatusMarkdown(nextTask.Status), nextTask.ID, nextTask.Title)

	// Add task details if present
	if len(nextTask.Details) > 0 {
		for _, detail := range nextTask.Details {
			result += fmt.Sprintf("  %s\n", detail)
		}
	}

	// Add incomplete children
	for _, child := range nextTask.IncompleteChildren {
		result += renderTaskMarkdown(&child, "  ")
	}

	// Add task-level references if present
	if len(nextTask.References) > 0 {
		result += "\n## Task References\n\n"
		for _, ref := range nextTask.References {
			result += fmt.Sprintf("- %s\n", ref)
		}
	}

	// Add front matter references if present
	if frontMatter != nil && len(frontMatter.References) > 0 {
		result += "\n## References\n\n"
		for _, ref := range frontMatter.References {
			result += fmt.Sprintf("- %s\n", ref)
		}
	}

	fmt.Print(result)
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
	var convertTask func(t *task.Task) TaskJSON
	convertTask = func(t *task.Task) TaskJSON {
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
		for _, child := range t.Children {
			tj.Children = append(tj.Children, convertTask(&child))
		}
		return tj
	}

	output := OutputJSON{
		Success:  true,
		NextTask: convertTask(nextTask.Task),
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
	result := fmt.Sprintf("%s- %s %s. %s\n",
		indent, formatStatusMarkdown(t.Status), t.ID, t.Title)

	// Add task details if present
	if len(t.Details) > 0 {
		for _, detail := range t.Details {
			result += fmt.Sprintf("%s  %s\n", indent, detail)
		}
	}

	// Add task references if present (for individual tasks)
	if len(t.References) > 0 {
		refList := strings.Join(t.References, ", ")
		result += fmt.Sprintf("%s  References: %s\n", indent, refList)
	}

	for _, child := range t.Children {
		if child.Status != task.Completed {
			result += renderTaskMarkdown(&child, indent+"  ")
		}
	}

	return result
}

// runNextPhase handles the --phase flag functionality
func runNextPhase(filename string) error {
	// Find next phase tasks
	phaseResult, err := task.FindNextPhaseTasks(filename)
	if err != nil {
		return fmt.Errorf("failed to find next phase tasks: %w", err)
	}

	if phaseResult == nil {
		return outputNextPhaseEmpty("No pending tasks found in any phase!")
	}

	// Parse file for front matter
	taskList, err := task.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read task file for front matter: %w", err)
	}

	if verbose {
		if phaseResult.PhaseName != "" {
			verboseStderr("Found %d pending tasks in phase '%s'", len(phaseResult.Tasks), phaseResult.PhaseName)
		} else {
			verboseStderr("Found %d pending tasks (no phases in document)", len(phaseResult.Tasks))
		}
	}

	// Create output based on format
	switch format {
	case formatJSON:
		return outputPhaseTasksJSON(phaseResult, taskList.FrontMatter)
	case formatMarkdown:
		return outputPhaseTasksMarkdown(phaseResult, taskList.FrontMatter)
	default:
		return outputPhaseTasksTable(phaseResult, taskList.FrontMatter)
	}
}

// outputPhaseTasksTable renders phase tasks in table format
func outputPhaseTasksTable(phaseResult *task.PhaseTasksResult, frontMatter *task.FrontMatter) error {
	// Build task data for display
	var taskData []map[string]any

	// Add all tasks from the phase
	for _, task := range phaseResult.Tasks {
		taskRecord := map[string]any{
			"ID":     task.ID,
			"Title":  task.Title,
			"Status": formatStatus(task.Status),
			"Level":  getTaskLevel(task.ID),
		}
		if phaseResult.PhaseName != "" {
			taskRecord["Phase"] = phaseResult.PhaseName
		}
		taskData = append(taskData, taskRecord)

		// Add all children (incomplete and complete for context)
		addAllChildrenToData(&task, &taskData, phaseResult.PhaseName)
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
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	return out.Render(context.Background(), doc)
}

// outputPhaseTasksMarkdown renders phase tasks in markdown format
func outputPhaseTasksMarkdown(phaseResult *task.PhaseTasksResult, frontMatter *task.FrontMatter) error {
	var result string

	// Add header
	if phaseResult.PhaseName != "" {
		result += fmt.Sprintf("# Next Phase Tasks (%s)\n\n", phaseResult.PhaseName)
	} else {
		result += "# Next Phase Tasks\n\n"
	}

	// Add all tasks from the phase
	for _, task := range phaseResult.Tasks {
		result += fmt.Sprintf("- %s %s. %s\n",
			formatStatusMarkdown(task.Status), task.ID, task.Title)

		// Add task details if present
		if len(task.Details) > 0 {
			for _, detail := range task.Details {
				result += fmt.Sprintf("  %s\n", detail)
			}
		}

		// Add task references if present
		if len(task.References) > 0 {
			refList := strings.Join(task.References, ", ")
			result += fmt.Sprintf("  References: %s\n", refList)
		}

		// Add all children
		for _, child := range task.Children {
			result += renderTaskMarkdown(&child, "  ")
		}
	}

	// Add front matter references if present
	if frontMatter != nil && len(frontMatter.References) > 0 {
		result += "\n## References\n\n"
		for _, ref := range frontMatter.References {
			result += fmt.Sprintf("- %s\n", ref)
		}
	}

	fmt.Print(result)
	return nil
}

// outputPhaseTasksJSON renders phase tasks in JSON format
func outputPhaseTasksJSON(phaseResult *task.PhaseTasksResult, frontMatter *task.FrontMatter) error {
	// Create a structure for JSON output
	type TaskJSON struct {
		ID         string     `json:"id"`
		Title      string     `json:"title"`
		Status     string     `json:"status"`
		Details    []string   `json:"details,omitempty"`
		References []string   `json:"references,omitempty"`
		Children   []TaskJSON `json:"children,omitempty"`
	}

	type OutputJSON struct {
		Success               bool       `json:"success"`
		Count                 int        `json:"count"`
		PhaseName             string     `json:"phase_name,omitempty"`
		Tasks                 []TaskJSON `json:"tasks"`
		FrontMatterReferences []string   `json:"front_matter_references,omitempty"`
	}

	// Convert tasks
	var convertTask func(t *task.Task) TaskJSON
	convertTask = func(t *task.Task) TaskJSON {
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
		for _, child := range t.Children {
			tj.Children = append(tj.Children, convertTask(&child))
		}
		return tj
	}

	output := OutputJSON{
		Success: true,
		Count:   len(phaseResult.Tasks),
		Tasks:   []TaskJSON{},
	}

	if phaseResult.PhaseName != "" {
		output.PhaseName = phaseResult.PhaseName
	}

	for _, task := range phaseResult.Tasks {
		output.Tasks = append(output.Tasks, convertTask(&task))
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

// addAllChildrenToData recursively adds all children (complete and incomplete) to table data
func addAllChildrenToData(parentTask *task.Task, taskData *[]map[string]any, phaseName string) {
	for _, child := range parentTask.Children {
		childRecord := map[string]any{
			"ID":     child.ID,
			"Title":  child.Title,
			"Status": formatStatus(child.Status),
			"Level":  getTaskLevel(child.ID),
		}
		if phaseName != "" {
			childRecord["Phase"] = phaseName
		}
		*taskData = append(*taskData, childRecord)

		// Recursively add its children
		addAllChildrenToData(&child, taskData, phaseName)
	}
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
