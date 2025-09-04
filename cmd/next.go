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

var nextCmd = &cobra.Command{
	Use:   "next [file]",
	Short: "Get the next incomplete task",
	Long: `Get the next incomplete task from the specified task file.

This command finds the first task that has incomplete work (either the task itself
or any of its subtasks are not marked as completed) using depth-first traversal.

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
}

func runNext(cmd *cobra.Command, args []string) error {
	// Resolve filename using git discovery if needed
	filename, err := resolveFilename(args)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("Using task file: %s\n", filename)
	}

	// Parse the task file
	taskList, err := task.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read task file: %w", err)
	}

	if verbose {
		fmt.Printf("Title: %s\n", taskList.Title)
		fmt.Printf("Total tasks: %d\n", countAllTasks(taskList.Tasks))
	}

	// Find the next incomplete task
	nextTask := task.FindNextIncompleteTask(taskList.Tasks)
	if nextTask == nil {
		fmt.Println("All tasks are complete!")
		return nil
	}

	if verbose {
		fmt.Printf("Found next task: %s (with %d incomplete subtasks)\n",
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

const (
	checkboxPending    = "[ ]"
	checkboxInProgress = "[-]"
	checkboxCompleted  = "[x]"
)

// formatStatusMarkdown formats status for markdown display
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
