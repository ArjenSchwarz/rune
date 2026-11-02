package cmd

import (
	"context"
	"fmt"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

var streamsCmd = &cobra.Command{
	Use:   "streams [file]",
	Short: "Display stream status",
	Long: `Shows the status of all work streams including ready, blocked, and active task counts.

Streams are used to partition work across agents for parallel execution.
Each stream shows:
- Ready: Tasks that can be started (all dependencies met, not owned)
- Blocked: Tasks waiting on dependencies
- Active: Tasks currently in progress

If no filename is provided and git discovery is enabled in configuration, the file
will be automatically discovered based on the current git branch.

Examples:
  # Show all streams status
  rune streams tasks.md

  # Show only available streams (those with ready tasks)
  rune streams tasks.md --available

  # Output as JSON for machine processing
  rune streams tasks.md --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStreams,
}

var (
	streamsAvailable bool
	streamsJSON      bool
)

func init() {
	rootCmd.AddCommand(streamsCmd)
	streamsCmd.Flags().BoolVarP(&streamsAvailable, "available", "a", false, "show only available streams (with ready tasks)")
	streamsCmd.Flags().BoolVarP(&streamsJSON, "json", "j", false, "output as JSON")
}

func runStreams(cmd *cobra.Command, args []string) error {
	// Resolve filename using git discovery if needed
	filename, err := resolveFilename(args)
	if err != nil {
		return err
	}

	if verbose {
		verboseStderr("Using task file: %s", filename)
	}

	// Parse the task file
	taskList, err := task.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read task file: %w", err)
	}
	reportParseWarnings(taskList)

	// Build dependency index and analyze streams
	index := task.BuildDependencyIndex(taskList.Tasks)
	result := task.AnalyzeStreams(taskList.Tasks, index)

	// Filter out empty streams (all tasks completed)
	result = filterEmptyStreams(result)

	// Filter to available streams if requested
	if streamsAvailable {
		result = filterAvailableStreams(result)
	}

	// Output based on format. The global --format json is honoured as well as
	// the command's own --json flag (T-1409).
	if streamsJSON || format == formatJSON {
		return outputJSON(result)
	}

	return outputStreamsTable(result)
}

// filterAvailableStreams returns only streams that have ready tasks.
// Available is recomputed from the surviving streams.
func filterAvailableStreams(result *task.StreamsResult) *task.StreamsResult {
	filtered := &task.StreamsResult{
		Streams:   make([]task.StreamStatus, 0),
		Available: []int{},
	}

	for _, stream := range result.Streams {
		if len(stream.Ready) > 0 {
			filtered.Streams = append(filtered.Streams, stream)
			filtered.Available = append(filtered.Available, stream.ID)
		}
	}

	return filtered
}

// filterEmptyStreams removes streams that have no pending tasks (all completed).
// Available is recomputed from the surviving streams.
func filterEmptyStreams(result *task.StreamsResult) *task.StreamsResult {
	filtered := &task.StreamsResult{
		Streams:   make([]task.StreamStatus, 0),
		Available: []int{},
	}

	for _, stream := range result.Streams {
		// Claimed (owned, not started) tasks are pending work too (T-1626)
		if len(stream.Ready) > 0 || len(stream.Blocked) > 0 || len(stream.Active) > 0 || len(stream.Claimed) > 0 {
			filtered.Streams = append(filtered.Streams, stream)
			if len(stream.Ready) > 0 {
				filtered.Available = append(filtered.Available, stream.ID)
			}
		}
	}

	return filtered
}

// outputStreamsTable renders streams as a table
func outputStreamsTable(result *task.StreamsResult) error {
	if len(result.Streams) == 0 {
		outputMessage("No streams with pending tasks")
		return nil
	}

	// Build table data
	tableData := make([]map[string]any, len(result.Streams))
	for i, stream := range result.Streams {
		available := "No"
		if len(stream.Ready) > 0 {
			available = "Yes"
		}

		tableData[i] = map[string]any{
			columnStream: stream.ID,
			"Ready":      len(stream.Ready),
			"Blocked":    len(stream.Blocked),
			"Active":     len(stream.Active),
			"Claimed":    len(stream.Claimed),
			"Available":  available,
		}
	}

	// Create table output
	doc := output.New().
		Table("Stream Status", tableData, output.WithKeys(columnStream, "Ready", "Blocked", "Active", "Claimed", "Available")).
		Build()

	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	return out.Render(context.Background(), doc)
}
