package cmd

import (
	"context"
	"fmt"
	"strings"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

const (
	formatJSON     = "json"
	formatMarkdown = "markdown"
	checkboxEmpty  = "[ ]"
)

var findCmd = &cobra.Command{
	Use:   "find [file] --pattern [pattern]",
	Short: "Find tasks matching a search pattern",
	Long: `Search for tasks in the specified file that match the given pattern.

If no filename is provided and git discovery is enabled in configuration, the file
will be automatically discovered based on the current git branch using the configured
template pattern.

The search can be performed across:
- Task titles (default)
- Task details (with --search-details)
- Task references (with --search-refs)

Results can be filtered by status and hierarchy level. The command returns
hierarchical context for search results when requested.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runFind,
}

var (
	findPattern    string
	searchDetails  bool
	searchRefs     bool
	caseSensitive  bool
	includeParent  bool
	statusFilter   string
	maxDepth       int
	parentIDFilter string
)

func init() {
	rootCmd.AddCommand(findCmd)

	// Search configuration flags
	findCmd.Flags().StringP("pattern", "p", "", "search pattern to match against tasks")
	findCmd.Flags().BoolVar(&searchDetails, "search-details", false, "search in task details")
	findCmd.Flags().BoolVar(&searchRefs, "search-refs", false, "search in task references")
	findCmd.Flags().BoolVar(&caseSensitive, "case-sensitive", false, "perform case-sensitive search")
	findCmd.Flags().BoolVar(&includeParent, "include-parent", false, "include parent context in results")

	// Filtering flags
	findCmd.Flags().StringVar(&statusFilter, "status", "", "filter by task status (pending, in-progress, completed)")
	findCmd.Flags().IntVar(&maxDepth, "max-depth", 0, "maximum hierarchy depth (0 means no limit)")
	findCmd.Flags().StringVar(&parentIDFilter, "parent", "", "filter by parent task ID (empty string for top-level tasks)")

	// Make pattern flag required
	findCmd.MarkFlagRequired("pattern")
}

func runFind(cmd *cobra.Command, args []string) error {
	// Resolve filename using git discovery if needed
	filename, err := resolveFilename(args)
	if err != nil {
		return err
	}

	// Get pattern from flag
	findPattern = cmd.Flag("pattern").Value.String()

	if verbose {
		fmt.Printf("Using task file: %s\n", filename)
	}

	// Parse the task file
	taskList, err := task.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read task file: %w", err)
	}

	if verbose {
		fmt.Printf("Pattern: %s\n", findPattern)
		fmt.Printf("Total tasks: %d\n", countAllTasks(taskList.Tasks))
	}

	// Set up query options
	opts := task.QueryOptions{
		CaseSensitive: caseSensitive,
		SearchDetails: searchDetails,
		SearchRefs:    searchRefs,
		IncludeParent: includeParent,
	}

	// Perform the search
	results := taskList.Find(findPattern, opts)

	// Apply additional filtering if specified
	if statusFilter != "" || maxDepth > 0 || parentIDFilter != "" {
		results = applyAdditionalFilters(results, statusFilter, maxDepth, parentIDFilter)
	}

	if len(results) == 0 {
		fmt.Printf("No tasks found matching pattern: %s\n", findPattern)
		return nil
	}

	if verbose {
		fmt.Printf("Found %d matching tasks\n", len(results))
	}

	// Output results based on format
	switch format {
	case formatJSON:
		return outputSearchResultsJSON(results, findPattern)
	case formatMarkdown:
		return outputSearchResultsMarkdown(results, findPattern)
	default:
		return outputSearchResultsTable(results, findPattern, taskList.Title)
	}
}

func applyAdditionalFilters(results []task.Task, statusFilter string, maxDepth int, parentIDFilter string) []task.Task {
	var filtered []task.Task

	for _, t := range results {
		include := true

		// Apply status filter
		if statusFilter != "" && !matchesStatusFilter(t.Status, statusFilter) {
			include = false
		}

		// Apply depth filter
		if maxDepth > 0 && getTaskLevel(t.ID) > maxDepth {
			include = false
		}

		// Apply parent ID filter
		if parentIDFilter != "" && t.ParentID != parentIDFilter {
			include = false
		}

		if include {
			filtered = append(filtered, t)
		}
	}

	return filtered
}

func outputSearchResultsTable(results []task.Task, pattern, title string) error {
	// Convert results to table data
	tableData := make([]map[string]any, 0, len(results))

	for _, t := range results {
		record := map[string]any{
			"ID":     t.ID,
			"Title":  t.Title,
			"Status": formatStatus(t.Status),
			"Level":  getTaskLevel(t.ID),
		}

		// Add parent context if available
		if t.ParentID != "" {
			record["Parent"] = t.ParentID
		} else {
			record["Parent"] = "root"
		}

		// Add details count if task has details
		if len(t.Details) > 0 {
			record["Details"] = fmt.Sprintf("%d details", len(t.Details))
		} else {
			record["Details"] = ""
		}

		// Add references count if task has references
		if len(t.References) > 0 {
			record["References"] = fmt.Sprintf("%d refs", len(t.References))
		} else {
			record["References"] = ""
		}

		tableData = append(tableData, record)
	}

	// Create document using go-output/v2
	doc := output.New().
		Table(fmt.Sprintf("Search Results for '%s' in %s", pattern, title), tableData,
			output.WithKeys("ID", "Title", "Status", "Level", "Parent", "Details", "References")).
		Build()

	// Create output renderer
	out := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	// Render the document
	return out.Render(context.Background(), doc)
}

func outputSearchResultsJSON(results []task.Task, pattern string) error {
	// Use the existing JSON renderer from task package
	jsonBytes, err := task.RenderJSON(&task.TaskList{
		Title: fmt.Sprintf("Search Results for '%s'", pattern),
		Tasks: results,
	})
	if err != nil {
		return fmt.Errorf("failed to render search results as JSON: %w", err)
	}

	fmt.Print(string(jsonBytes))
	return nil
}

func outputSearchResultsMarkdown(results []task.Task, pattern string) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Search Results for '%s'\n\n", pattern))
	sb.WriteString(fmt.Sprintf("Found %d matching tasks:\n\n", len(results)))

	for _, t := range results {
		// Write task with checkbox
		checkbox := checkboxEmpty
		switch t.Status {
		case task.Completed:
			checkbox = "[x]"
		case task.InProgress:
			checkbox = "[-]"
		}

		sb.WriteString(fmt.Sprintf("- %s %s. %s", checkbox, t.ID, t.Title))

		// Add parent context if available
		if t.ParentID != "" {
			sb.WriteString(fmt.Sprintf(" (parent: %s)", t.ParentID))
		}
		sb.WriteString("\n")

		// Add details if present
		for _, detail := range t.Details {
			sb.WriteString(fmt.Sprintf("  - %s\n", detail))
		}

		// Add references if present
		if len(t.References) > 0 {
			sb.WriteString(fmt.Sprintf("  - References: %s\n", strings.Join(t.References, ", ")))
		}

		sb.WriteString("\n")
	}

	fmt.Print(sb.String())
	return nil
}
