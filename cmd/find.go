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
	formatTable    = "table"
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
	// Validate status filter before doing any work
	if err := validateStatusFilter(statusFilter); err != nil {
		return err
	}

	// Resolve filename using git discovery if needed
	filename, err := resolveFilename(args)
	if err != nil {
		return err
	}

	// Get pattern from flag
	findPattern = cmd.Flag("pattern").Value.String()

	if verbose {
		verboseStderr("Using task file: %s", filename)
	}

	// Parse the task file
	taskList, err := task.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read task file: %w", err)
	}
	reportParseWarnings(taskList)

	if verbose {
		verboseStderr("Pattern: %s", findPattern)
		verboseStderr("Total tasks: %d", countAllTasks(taskList.Tasks))
	}

	// Set up query options. Parent context is not requested here: it is
	// added after filtering, so a filter cannot remove the match while
	// keeping its context (T-2070).
	opts := task.QueryOptions{
		CaseSensitive: caseSensitive,
		SearchDetails: searchDetails,
		SearchRefs:    searchRefs,
	}

	// Perform the search
	results := taskList.Find(findPattern, opts)

	// Apply additional filtering if specified
	parentFilterSet := cmd.Flags().Changed("parent")
	if statusFilter != "" || maxDepth > 0 || parentFilterSet {
		results = applyAdditionalFilters(results, statusFilter, maxDepth, parentIDFilter, parentFilterSet)
	}

	if includeParent {
		// The filters re-point a ParentID past a filtered-out parent; restore
		// the source parent so it is added back as context.
		for i := range results {
			if src := taskList.FindTask(results[i].ID); src != nil {
				results[i].ParentID = src.ParentID
			}
		}
		results = taskList.InsertParents(results)
	}

	if len(results) == 0 {
		return outputFindEmpty(fmt.Sprintf("No matching tasks found for pattern: %s", findPattern))
	}

	if verbose {
		verboseStderr("Found %d matching tasks", len(results))
	}

	// Output results based on format. The dependency index covers the whole
	// file so Blocked-by references outside the results still translate.
	switch format {
	case formatJSON:
		return outputSearchResultsJSON(results, findPattern, taskList)
	case formatMarkdown:
		return outputSearchResultsMarkdown(results, findPattern, task.BuildDependencyIndex(taskList.Tasks), taskList.RequirementsFile)
	default:
		return outputSearchResultsTable(results, findPattern, taskList.Title, task.BuildDependencyIndex(taskList.Tasks))
	}
}

func applyAdditionalFilters(results []task.Task, statusFilter string, maxDepth int, parentIDFilter string, parentFilterSet bool) []task.Task {
	// Build a set of all IDs present in the unfiltered input so we can
	// distinguish "parent was never part of the search results" (fine — the
	// ParentID is just hierarchy context) from "parent was here but got
	// filtered out" (stale reference that must be fixed per T-549).
	inputIDs := make(map[string]bool, len(results))
	for _, t := range results {
		inputIDs[t.ID] = true
	}

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

		// Apply parent ID filter — only when explicitly set via --parent flag.
		// An empty parentIDFilter means "top-level tasks" (ParentID == "").
		if parentFilterSet && t.ParentID != parentIDFilter {
			include = false
		}

		if include {
			filtered = append(filtered, t)
		}
	}

	// Fix stale ParentIDs: walk each task's ParentID up the hierarchy until
	// we find a surviving ancestor, an ancestor that was never in the input
	// (hierarchy context — not stale), or reach root (T-549).
	outputIDs := make(map[string]bool, len(filtered))
	for _, t := range filtered {
		outputIDs[t.ID] = true
	}
	for i := range filtered {
		pid := filtered[i].ParentID
		for pid != "" && inputIDs[pid] && !outputIDs[pid] {
			// When --parent was set, its value is intentional context,
			// not a stale reference — stop walking.
			if parentFilterSet && pid == parentIDFilter {
				break
			}
			if dot := strings.LastIndex(pid, "."); dot >= 0 {
				pid = pid[:dot]
			} else {
				pid = ""
			}
		}
		// Note: when the walk stops because !inputIDs[pid], the task retains
		// a ParentID referencing a task absent from the output. This is
		// hierarchy context (the parent was never in the search results),
		// not a stale reference. JSON consumers should tolerate this case.
		filtered[i].ParentID = pid
	}

	return filtered
}

func outputSearchResultsTable(results []task.Task, pattern, title string, depIndex *task.DependencyIndex) error {
	// Scheduling and requirement columns are shown only when some result
	// has that metadata, as in list output (T-2242).
	hasStream := detectNonDefaultStreams(results)
	hasBlockedBy := detectBlockedByExists(results)
	var hasOwner, hasRequirements bool
	for _, t := range results {
		hasOwner = hasOwner || t.Owner != ""
		hasRequirements = hasRequirements || len(t.Requirements) > 0
	}

	// Convert results to table data
	tableData := make([]map[string]any, 0, len(results))

	for _, t := range results {
		record := map[string]any{
			"ID":         t.ID,
			columnTitle:  t.Title,
			columnStatus: formatStatus(t.Status),
			columnLevel:  getTaskLevel(t.ID),
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

		record[columnStream] = task.GetEffectiveStream(&t)
		record[columnBlockedBy] = formatBlockedByForDisplay(&t, depIndex)
		record["Owner"] = t.Owner
		record["Requirements"] = strings.Join(t.Requirements, ", ")

		tableData = append(tableData, record)
	}

	keys := []string{"ID", columnTitle, columnStatus, columnLevel, "Parent"}
	if hasStream {
		keys = append(keys, columnStream)
	}
	if hasBlockedBy {
		keys = append(keys, columnBlockedBy)
	}
	if hasOwner {
		keys = append(keys, "Owner")
	}
	if hasRequirements {
		keys = append(keys, "Requirements")
	}
	keys = append(keys, "Details", "References")

	// Create document using go-output/v2
	doc := output.New().
		Table(fmt.Sprintf("Search Results for '%s' in %s", pattern, title), tableData,
			output.WithKeys(keys...)).
		Build()

	// Create output renderer
	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	// Render the document
	return out.Render(context.Background(), doc)
}

func outputSearchResultsJSON(results []task.Task, pattern string, source *task.TaskList) error {
	// The source file's requirements path is kept so requirement IDs can be
	// resolved (T-1995), and the source list is passed so Blocked-by
	// references to tasks outside the results still translate (T-1206).
	// Without phase markers this renders the same structure as RenderJSON.
	jsonBytes := task.RenderJSONWithPhases(&task.TaskList{
		Title:            fmt.Sprintf("Search Results for '%s'", pattern),
		Tasks:            results,
		RequirementsFile: source.RequirementsFile,
	}, nil, source)

	fmt.Print(string(jsonBytes))
	return nil
}

func outputSearchResultsMarkdown(results []task.Task, pattern string, depIndex *task.DependencyIndex, reqFile string) error {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# Search Results for '%s'\n\n", pattern)
	fmt.Fprintf(&sb, "Found %d matching tasks:\n\n", len(results))

	for _, t := range results {
		// Write task with checkbox
		checkbox := checkboxEmpty
		switch t.Status {
		case task.Completed:
			checkbox = "[x]"
		case task.InProgress:
			checkbox = "[-]"
		}

		fmt.Fprintf(&sb, "- %s %s. %s", checkbox, t.ID, t.Title)

		// Add parent context if available
		if t.ParentID != "" {
			fmt.Fprintf(&sb, " (parent: %s)", t.ParentID)
		}
		sb.WriteString("\n")

		// Add details if present
		for _, detail := range t.Details {
			fmt.Fprintf(&sb, "  - %s\n", detail)
		}

		// Scheduling and requirement metadata, in task-file order (T-2242)
		if len(t.BlockedBy) > 0 {
			fmt.Fprintf(&sb, "  - Blocked-by: %s\n", formatBlockedByForDisplay(&t, depIndex))
		}
		if t.Stream > 0 {
			fmt.Fprintf(&sb, "  - Stream: %d\n", t.Stream)
		}
		if t.Owner != "" {
			fmt.Fprintf(&sb, "  - Owner: %s\n", t.Owner)
		}
		if len(t.Requirements) > 0 {
			fmt.Fprintf(&sb, "  - Requirements: %s\n", requirementLinks(t.Requirements, reqFile))
		}

		// Add references if present
		if len(t.References) > 0 {
			fmt.Fprintf(&sb, "  - References: %s\n", strings.Join(t.References, ", "))
		}

		sb.WriteString("\n")
	}

	fmt.Print(sb.String())
	return nil
}

// FindEmptyResponse is the JSON response structure when no matching tasks are found.
type FindEmptyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Count   int    `json:"count"`
	Data    []any  `json:"data"`
}

// outputFindEmpty handles format-aware output when no matching tasks are found.
func outputFindEmpty(message string) error {
	switch format {
	case formatJSON:
		return outputJSON(FindEmptyResponse{
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
