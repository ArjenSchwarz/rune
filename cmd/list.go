package cmd

import (
	"context"
	"fmt"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [file]",
	Short: "List tasks from a file",
	Long: `List all tasks from the specified task file in various output formats.

If no filename is provided and git discovery is enabled in configuration, the file
will be automatically discovered based on the current git branch using the configured
template pattern.

Supports multiple output formats:
- table: Human-readable table format (default)
- json: Structured JSON data
- markdown: Markdown format

The output includes task IDs, titles, statuses, and hierarchy information.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runList,
}

var (
	listFilter string
	showAll    bool
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVar(&listFilter, "filter", "", "filter tasks by status (pending, in-progress, completed)")
	listCmd.Flags().BoolVar(&showAll, "all", false, "show all task details including references")
}

func runList(cmd *cobra.Command, args []string) error {
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

	// Convert tasks to a flat structure for display
	taskData := flattenTasks(taskList.Tasks, listFilter)

	if len(taskData) == 0 {
		if listFilter != "" {
			fmt.Printf("No tasks found matching filter: %s\n", listFilter)
		} else {
			fmt.Printf("No tasks found in file: %s\n", filename)
		}
		return nil
	}

	// Create output document based on format
	switch format {
	case formatJSON:
		return outputJSON(taskList)
	case formatMarkdown:
		return outputMarkdown(taskList)
	default:
		return outputTable(taskList, taskData)
	}
}

// flattenTasks converts hierarchical tasks to a flat structure for table display
func flattenTasks(tasks []task.Task, statusFilter string) []map[string]any {
	var result []map[string]any

	for _, t := range tasks {
		// Apply status filter if specified
		if statusFilter != "" && !matchesStatusFilter(t.Status, statusFilter) {
			continue
		}

		// Create task record
		taskRecord := map[string]any{
			"ID":     t.ID,
			"Title":  t.Title,
			"Status": formatStatus(t.Status),
			"Level":  getTaskLevel(t.ID),
		}

		// Add optional details
		if showAll {
			if len(t.Details) > 0 {
				taskRecord["Details"] = formatDetails(t.Details)
			}
			if len(t.References) > 0 {
				taskRecord["References"] = formatReferences(t.References)
			}
		}

		result = append(result, taskRecord)

		// Recursively add children
		children := flattenTasks(t.Children, statusFilter)
		result = append(result, children...)
	}

	return result
}

func matchesStatusFilter(status task.Status, filter string) bool {
	switch filter {
	case "pending":
		return status == task.Pending
	case "in-progress", "inprogress":
		return status == task.InProgress
	case "completed":
		return status == task.Completed
	default:
		return true
	}
}

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

func formatDetails(details []string) string {
	if len(details) == 0 {
		return ""
	}
	if len(details) == 1 {
		return details[0]
	}
	return fmt.Sprintf("%d details", len(details))
}

func formatReferences(references []string) string {
	if len(references) == 0 {
		return ""
	}
	if len(references) == 1 {
		return references[0]
	}
	return fmt.Sprintf("%d references", len(references))
}

func countAllTasks(tasks []task.Task) int {
	count := len(tasks)
	for _, t := range tasks {
		count += countAllTasks(t.Children)
	}
	return count
}

func outputTable(taskList *task.TaskList, taskData []map[string]any) error {
	// Build table keys based on what we want to show
	keys := []string{"ID", "Title", "Status", "Level"}
	if showAll {
		keys = append(keys, "Details", "References")
	}

	// Create document builder
	docBuilder := output.New().
		Table(fmt.Sprintf("Tasks: %s", taskList.Title), taskData, output.WithKeys(keys...))

	// Add TaskList references section if present
	if taskList.FrontMatter != nil && len(taskList.FrontMatter.References) > 0 {
		referencesData := make([]map[string]any, len(taskList.FrontMatter.References))
		for i, ref := range taskList.FrontMatter.References {
			referencesData[i] = map[string]any{
				"Reference": ref,
			}
		}
		docBuilder = docBuilder.Table("References", referencesData, output.WithKeys("Reference"))
	}

	doc := docBuilder.Build()

	// Configure output format
	var outputFormat output.Format
	switch format {
	case "json":
		outputFormat = output.JSON
	case "markdown":
		outputFormat = output.Markdown
	default:
		outputFormat = output.Table
	}

	// Create output renderer
	out := output.NewOutput(
		output.WithFormat(outputFormat),
		output.WithWriter(output.NewStdoutWriter()),
	)

	// Render the document
	return out.Render(context.Background(), doc)
}

func outputJSON(taskList *task.TaskList) error {
	jsonOutput, err := task.RenderJSON(taskList)
	if err != nil {
		return fmt.Errorf("failed to render JSON: %w", err)
	}
	fmt.Print(string(jsonOutput))
	return nil
}

func outputMarkdown(taskList *task.TaskList) error {
	markdownOutput := task.RenderMarkdown(taskList)
	fmt.Print(string(markdownOutput))
	return nil
}
