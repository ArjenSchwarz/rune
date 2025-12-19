package cmd

import (
	"context"
	"fmt"
	"strings"

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

	// Parse the task file with phase information
	taskList, phaseMarkers, err := task.ParseFileWithPhases(filename)
	if err != nil {
		return fmt.Errorf("failed to read task file: %w", err)
	}

	if verbose {
		fmt.Printf("Title: %s\n", taskList.Title)
		fmt.Printf("Total tasks: %d\n", countAllTasks(taskList.Tasks))
		if len(phaseMarkers) > 0 {
			fmt.Printf("Phases found: %d\n", len(phaseMarkers))

			// Check for duplicate phase names
			duplicates := findDuplicatePhases(phaseMarkers)
			if len(duplicates) > 0 {
				fmt.Printf("⚠️  Warning: Duplicate phase names detected: %v\n", duplicates)
				fmt.Printf("   Operations on these phases will use the first occurrence.\n")
			}
		}
	}

	// Convert tasks to a flat structure for display
	taskData := flattenTasksWithPhases(taskList, phaseMarkers, listFilter)

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
		return outputJSONWithPhases(taskList, phaseMarkers)
	case formatMarkdown:
		return outputMarkdownWithPhases(taskList, phaseMarkers)
	default:
		return outputTableWithPhases(taskList, taskData, phaseMarkers)
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

// findDuplicatePhases identifies duplicate phase names in the phase markers
func findDuplicatePhases(markers []task.PhaseMarker) []string {
	seen := make(map[string]int)
	var duplicates []string

	for _, marker := range markers {
		seen[marker.Name]++
	}

	for name, count := range seen {
		if count > 1 {
			duplicates = append(duplicates, name)
		}
	}

	return duplicates
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

func outputMarkdown(taskList *task.TaskList) error {
	var buf strings.Builder

	// Add front matter references if present and --all flag is used
	if showAll && taskList.FrontMatter != nil && len(taskList.FrontMatter.References) > 0 {
		buf.WriteString("## Document References\n\n")
		for _, ref := range taskList.FrontMatter.References {
			buf.WriteString(fmt.Sprintf("- %s\n", ref))
		}
		buf.WriteString("\n")
	}

	// Add tasks
	markdownOutput := task.RenderMarkdown(taskList)
	buf.Write(markdownOutput)

	fmt.Print(buf.String())
	return nil
}

// flattenTasksWithPhases converts hierarchical tasks to a flat structure with phase information
func flattenTasksWithPhases(taskList *task.TaskList, phaseMarkers []task.PhaseMarker, statusFilter string) []map[string]any {
	var result []map[string]any
	hasPhases := len(phaseMarkers) > 0

	var flattenRecursive func(tasks []task.Task, statusFilter string)
	flattenRecursive = func(tasks []task.Task, statusFilter string) {
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

			// Add phase column if phases exist
			if hasPhases {
				phase := task.GetTaskPhase(taskList, phaseMarkers, t.ID)
				taskRecord["Phase"] = phase
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
			flattenRecursive(t.Children, statusFilter)
		}
	}

	flattenRecursive(taskList.Tasks, statusFilter)
	return result
}

func outputTableWithPhases(taskList *task.TaskList, taskData []map[string]any, phaseMarkers []task.PhaseMarker) error {
	// Build table keys based on what we want to show
	keys := []string{"ID"}

	// Add Phase column if phases exist
	if len(phaseMarkers) > 0 {
		keys = append(keys, "Phase")
	}

	keys = append(keys, "Title", "Status", "Level")

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

func outputJSONWithPhases(taskList *task.TaskList, phaseMarkers []task.PhaseMarker) error {
	jsonOutput := task.RenderJSONWithPhases(taskList, phaseMarkers)
	fmt.Print(string(jsonOutput))
	return nil
}

func outputMarkdownWithPhases(taskList *task.TaskList, phaseMarkers []task.PhaseMarker) error {
	var buf strings.Builder

	// Add front matter references if present and --all flag is used
	if showAll && taskList.FrontMatter != nil && len(taskList.FrontMatter.References) > 0 {
		buf.WriteString("## Document References\n\n")
		for _, ref := range taskList.FrontMatter.References {
			buf.WriteString(fmt.Sprintf("- %s\n", ref))
		}
		buf.WriteString("\n")
	}

	// Add tasks with phases
	markdownOutput := task.RenderMarkdownWithPhases(taskList, phaseMarkers)
	buf.Write(markdownOutput)

	fmt.Print(buf.String())
	return nil
}
