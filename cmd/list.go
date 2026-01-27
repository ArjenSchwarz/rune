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

Filtering Options:
- --filter STATUS: Filter by status (pending, in-progress, completed)
- --stream N: Filter to tasks in stream N
- --owner AGENT_ID: Filter to tasks owned by a specific agent
- --owner "": Filter to unowned tasks only

Column Display:
The table output adapts based on the data present:
- Stream column appears when any task has a non-default stream assignment
- BlockedBy column appears when any task has dependencies
- Owner column appears when any task has an owner assigned

The output includes task IDs, titles, statuses, dependencies, and hierarchy information.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runList,
}

var (
	listFilter       string
	showAll          bool
	listStreamFilter int
	listOwnerFilter  string
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVar(&listFilter, "filter", "", "filter tasks by status (pending, in-progress, completed)")
	listCmd.Flags().BoolVar(&showAll, "all", false, "show all task details including references")
	listCmd.Flags().IntVar(&listStreamFilter, "stream", 0, "filter tasks by stream number")
	listCmd.Flags().StringVar(&listOwnerFilter, "owner", "", "filter tasks by owner")
}

func runList(cmd *cobra.Command, args []string) error {
	// Resolve filename using git discovery if needed
	filename, err := resolveFilename(args)
	if err != nil {
		return err
	}

	if verbose {
		verboseStderr("Using task file: %s", filename)
	}

	// Parse the task file with phase information
	taskList, phaseMarkers, err := task.ParseFileWithPhases(filename)
	if err != nil {
		return fmt.Errorf("failed to read task file: %w", err)
	}

	if verbose {
		verboseStderr("Title: %s", taskList.Title)
		verboseStderr("Total tasks: %d", countAllTasks(taskList.Tasks))
		if len(phaseMarkers) > 0 {
			verboseStderr("Phases found: %d", len(phaseMarkers))

			// Check for duplicate phase names
			duplicates := findDuplicatePhases(phaseMarkers)
			if len(duplicates) > 0 {
				verboseStderr("⚠️  Warning: Duplicate phase names detected: %v", duplicates)
				verboseStderr("   Operations on these phases will use the first occurrence.")
			}
		}
	}

	// Build dependency index for blocked-by translation
	depIndex := task.BuildDependencyIndex(taskList.Tasks)

	// Detect if non-default streams exist (for conditional display)
	hasNonDefaultStreams := detectNonDefaultStreams(taskList.Tasks)

	// Create filter options
	filterOpts := listFilterOptions{
		statusFilter: listFilter,
		streamFilter: listStreamFilter,
		ownerFilter:  listOwnerFilter,
		ownerSet:     cmd.Flags().Changed("owner"), // Track if --owner was explicitly set
	}

	// Convert tasks to a flat structure for display
	taskData := flattenTasksWithFilters(taskList, phaseMarkers, depIndex, hasNonDefaultStreams, filterOpts)

	if len(taskData) == 0 {
		message := buildEmptyMessage(filename, filterOpts)
		return outputListEmpty(message)
	}

	// Create output document based on format
	switch format {
	case formatJSON:
		return outputJSONWithFilters(taskList, phaseMarkers, depIndex, filterOpts)
	case formatMarkdown:
		return outputMarkdownWithPhases(taskList, phaseMarkers)
	default:
		return outputTableWithFilters(taskList, taskData, phaseMarkers, hasNonDefaultStreams, depIndex)
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

// ListEmptyResponse is the JSON response structure when no tasks are found.
type ListEmptyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Count   int    `json:"count"`
	Data    []any  `json:"data"`
}

// outputListEmpty handles format-aware output when no tasks are found.
func outputListEmpty(message string) error {
	switch format {
	case formatJSON:
		return outputJSON(ListEmptyResponse{
			Success: true,
			Message: message,
			Count:   0,
			Data:    []any{},
		})
	case formatMarkdown:
		outputMarkdownMessage(message)
		return nil
	default:
		outputMessage(message)
		return nil
	}
}

// listFilterOptions contains all filter parameters for list command
type listFilterOptions struct {
	statusFilter string
	streamFilter int
	ownerFilter  string
	ownerSet     bool // True if --owner flag was explicitly provided
}

// detectNonDefaultStreams checks if any task has a stream other than 1
func detectNonDefaultStreams(tasks []task.Task) bool {
	for _, t := range tasks {
		if task.GetEffectiveStream(&t) != 1 {
			return true
		}
		if detectNonDefaultStreams(t.Children) {
			return true
		}
	}
	return false
}

// detectBlockedByExists checks if any task has blocked-by references
func detectBlockedByExists(tasks []task.Task) bool {
	for _, t := range tasks {
		if len(t.BlockedBy) > 0 {
			return true
		}
		if detectBlockedByExists(t.Children) {
			return true
		}
	}
	return false
}

// buildEmptyMessage constructs appropriate message when no tasks match filters
func buildEmptyMessage(filename string, opts listFilterOptions) string {
	var filters []string
	if opts.statusFilter != "" {
		filters = append(filters, fmt.Sprintf("status=%s", opts.statusFilter))
	}
	if opts.streamFilter > 0 {
		filters = append(filters, fmt.Sprintf("stream=%d", opts.streamFilter))
	}
	if opts.ownerSet {
		if opts.ownerFilter == "" {
			filters = append(filters, "owner=(unowned)")
		} else {
			filters = append(filters, fmt.Sprintf("owner=%s", opts.ownerFilter))
		}
	}

	if len(filters) > 0 {
		return fmt.Sprintf("No tasks found matching filters: %s", strings.Join(filters, ", "))
	}
	return fmt.Sprintf("No tasks found in file: %s", filename)
}

// matchesFilters checks if a task matches all specified filters
func matchesFilters(t *task.Task, opts listFilterOptions) bool {
	// Status filter
	if opts.statusFilter != "" && !matchesStatusFilter(t.Status, opts.statusFilter) {
		return false
	}
	// Stream filter
	if opts.streamFilter > 0 && task.GetEffectiveStream(t) != opts.streamFilter {
		return false
	}
	// Owner filter (only if explicitly set)
	if opts.ownerSet && t.Owner != opts.ownerFilter {
		return false
	}
	return true
}

// flattenTasksWithFilters converts hierarchical tasks to a flat structure with filtering
func flattenTasksWithFilters(taskList *task.TaskList, phaseMarkers []task.PhaseMarker, depIndex *task.DependencyIndex, hasNonDefaultStreams bool, opts listFilterOptions) []map[string]any {
	var result []map[string]any
	hasPhases := len(phaseMarkers) > 0
	hasBlockedBy := detectBlockedByExists(taskList.Tasks)

	var flattenRecursive func(tasks []task.Task)
	flattenRecursive = func(tasks []task.Task) {
		for _, t := range tasks {
			// Apply all filters
			if !matchesFilters(&t, opts) {
				// Still process children even if parent doesn't match
				flattenRecursive(t.Children)
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

			// Add stream column if non-default streams exist
			if hasNonDefaultStreams {
				taskRecord["Stream"] = task.GetEffectiveStream(&t)
			}

			// Add blocked-by as hierarchical IDs if any task has dependencies
			if hasBlockedBy {
				blockedByHierarchical := depIndex.TranslateToHierarchical(t.BlockedBy)
				if len(blockedByHierarchical) > 0 {
					taskRecord["BlockedBy"] = strings.Join(blockedByHierarchical, ", ")
				} else {
					taskRecord["BlockedBy"] = ""
				}
			}

			// Add owner column if task has an owner
			if t.Owner != "" {
				taskRecord["Owner"] = t.Owner
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
			flattenRecursive(t.Children)
		}
	}

	flattenRecursive(taskList.Tasks)
	return result
}

// outputTableWithFilters renders table with stream and blocked-by columns
func outputTableWithFilters(taskList *task.TaskList, taskData []map[string]any, phaseMarkers []task.PhaseMarker, hasNonDefaultStreams bool, depIndex *task.DependencyIndex) error {
	// Build table keys based on what we want to show
	keys := []string{"ID"}

	// Add Phase column if phases exist
	if len(phaseMarkers) > 0 {
		keys = append(keys, "Phase")
	}

	keys = append(keys, "Title", "Status")

	// Add Stream column conditionally
	if hasNonDefaultStreams {
		keys = append(keys, "Stream")
	}

	// Add BlockedBy column if any task has dependencies
	hasBlockedBy := detectBlockedByExists(taskList.Tasks)
	if hasBlockedBy {
		keys = append(keys, "BlockedBy")
	}

	// Add Owner column if any task has owner (check task data for any owner)
	hasOwner := false
	for _, record := range taskData {
		if _, ok := record["Owner"]; ok {
			hasOwner = true
			break
		}
	}
	if hasOwner {
		keys = append(keys, "Owner")
	}

	keys = append(keys, "Level")

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

	// Create output renderer
	out := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	// Render the document
	return out.Render(context.Background(), doc)
}

// outputJSONWithFilters renders JSON with filtered tasks and new fields
func outputJSONWithFilters(taskList *task.TaskList, phaseMarkers []task.PhaseMarker, depIndex *task.DependencyIndex, opts listFilterOptions) error {
	// Apply filters to the task list for JSON output
	filteredTasks := filterTasksRecursive(taskList.Tasks, opts)

	// Create a copy of the task list with filtered tasks
	filteredList := &task.TaskList{
		Title:            taskList.Title,
		Tasks:            filteredTasks,
		FrontMatter:      taskList.FrontMatter,
		RequirementsFile: taskList.RequirementsFile,
	}

	jsonOutput := task.RenderJSONWithPhases(filteredList, phaseMarkers)
	fmt.Print(string(jsonOutput))
	return nil
}

// filterTasksRecursive filters tasks recursively
func filterTasksRecursive(tasks []task.Task, opts listFilterOptions) []task.Task {
	var result []task.Task
	for _, t := range tasks {
		// Filter children first
		filteredChildren := filterTasksRecursive(t.Children, opts)

		if matchesFilters(&t, opts) {
			taskCopy := t
			taskCopy.Children = filteredChildren
			result = append(result, taskCopy)
		} else if len(filteredChildren) > 0 {
			// Include parent if any children match
			taskCopy := t
			taskCopy.Children = filteredChildren
			result = append(result, taskCopy)
		}
	}
	return result
}
