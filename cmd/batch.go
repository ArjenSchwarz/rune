package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

var (
	batchInput string
)

var batchCmd = &cobra.Command{
	Use:   "batch [file]",
	Short: "Execute multiple operations from JSON input",
	Long: `Execute multiple task operations in a single atomic transaction.
Operations are specified as JSON input either from stdin, a file, or a string.

The JSON format should be:
{
  "file": "tasks.md",
  "operations": [
    {
      "type": "add",
      "title": "New task",
      "parent": "1",
      "phase": "Planning"
    },
    {
      "type": "update",
      "id": "2",
      "status": 2
    }
  ],
  "dry_run": false
}

Operation types:
- add: Add a new task (requires title, optional parent, phase)
- remove: Remove a task (requires id)
- update: Update task fields (requires id, optional title, status, details, references)

Phase support:
- Add "phase" field to "add" operations to specify target phase
- If phase doesn't exist, it will be created automatically
- Duplicate phase names use first occurrence
- Mixed operations (some with phases, some without) are supported

All operations are atomic - either all succeed or none are applied.`,
	RunE: runBatch,
	Args: cobra.MaximumNArgs(1),
}

func runBatch(cmd *cobra.Command, args []string) error {
	// Read JSON input
	var jsonData []byte
	var err error

	switch {
	case batchInput != "":
		// Input provided as flag
		jsonData = []byte(batchInput)
	case len(args) > 0 && args[0] != "":
		// Input from file
		jsonData, err = os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("reading input file: %w", err)
		}
	default:
		// Input from stdin
		jsonData, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading from stdin: %w", err)
		}
	}

	// Parse JSON request
	var req task.BatchRequest
	if err := json.Unmarshal(jsonData, &req); err != nil {
		return fmt.Errorf("parsing JSON input: %w", err)
	}

	// Validate request
	if req.File == "" {
		return fmt.Errorf("file field is required in batch request")
	}
	if len(req.Operations) == 0 {
		return fmt.Errorf("at least one operation is required")
	}
	if len(req.Operations) > 100 {
		return fmt.Errorf("maximum of 100 operations allowed per batch")
	}

	// Override dry-run from command flag if set
	if dryRun {
		req.DryRun = true
	}

	// Check if any operations use phases
	hasPhaseOps := slices.ContainsFunc(req.Operations, func(op task.Operation) bool {
		return op.Phase != ""
	})

	var response *task.BatchResponse
	var taskList *task.TaskList

	// Use phase-aware execution if needed
	if hasPhaseOps {
		// Parse file with phases
		var phaseMarkers []task.PhaseMarker
		taskList, phaseMarkers, err = task.ParseFileWithPhases(req.File)
		if err != nil {
			return fmt.Errorf("loading task file with phases: %w", err)
		}

		// Execute batch operations with phase support
		response, err = taskList.ExecuteBatchWithPhases(req.Operations, req.DryRun, phaseMarkers, req.File)
		if err != nil {
			return fmt.Errorf("executing batch operations with phases: %w", err)
		}

		// File is already saved by ExecuteBatchWithPhases if not a dry run
	} else {
		// Load task list without phases
		taskList, err = task.ParseFile(req.File)
		if err != nil {
			return fmt.Errorf("loading task file: %w", err)
		}

		// Execute batch operations
		response, err = taskList.ExecuteBatch(req.Operations, req.DryRun)
		if err != nil {
			return fmt.Errorf("executing batch operations: %w", err)
		}

		// Save the file if not a dry run and operations succeeded
		if !req.DryRun && response.Success {
			if err := taskList.WriteFile(req.File); err != nil {
				return fmt.Errorf("saving updated file: %w", err)
			}
		}
	}

	// Handle output based on format
	switch strings.ToLower(format) {
	case formatJSON:
		return outputBatchJSON(cmd, response)
	case "table", formatMarkdown:
		return outputBatchText(cmd, response, req.DryRun, req.File, taskList)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func outputBatchJSON(cmd *cobra.Command, response *task.BatchResponse) error {
	output, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(output))
	return nil
}

func outputBatchText(cmd *cobra.Command, response *task.BatchResponse, isDryRun bool, filename string, taskList *task.TaskList) error {
	out := cmd.OutOrStdout()

	if !response.Success {
		fmt.Fprintf(out, "❌ Batch operation failed:\n")
		for i, err := range response.Errors {
			fmt.Fprintf(out, "  %d. %s\n", i+1, err)
		}
		return nil
	}

	if isDryRun {
		fmt.Fprintf(out, "✅ Dry run successful - %d operations validated\n", response.Applied)

		// Show auto-completed tasks if any
		if len(response.AutoCompleted) > 0 {
			fmt.Fprintf(out, "\n🎯 Auto-completed parent tasks:\n")
			for _, taskID := range response.AutoCompleted {
				fmt.Fprintf(out, "  - Task %s\n", taskID)
			}
		}

		fmt.Fprintf(out, "\nPreview of changes:\n")
		fmt.Fprintf(out, "---\n")
		fmt.Fprint(out, response.Preview)
		fmt.Fprintf(out, "---\n")
		fmt.Fprintf(out, "\nUse --dry-run=false to apply these changes.\n")
	} else {
		fmt.Fprintf(out, "✅ Batch operation successful - %d operations applied\n", response.Applied)

		// Show auto-completed tasks if any
		if len(response.AutoCompleted) > 0 {
			fmt.Fprintf(out, "\n🎯 Auto-completed parent tasks:\n")
			for _, taskID := range response.AutoCompleted {
				fmt.Fprintf(out, "  - Task %s\n", taskID)
			}
		}

		// File is already saved in runBatch function

		if verbose {
			fmt.Fprintf(out, "\nUpdated file contents:\n")
			fmt.Fprintf(out, "---\n")
			content := task.RenderMarkdown(taskList)
			fmt.Fprint(out, string(content))
			fmt.Fprintf(out, "---\n")
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(batchCmd)

	// Add batch-specific flags
	batchCmd.Flags().StringVarP(&batchInput, "input", "i", "",
		"JSON input as string (alternative to file or stdin)")

	// Add usage examples
	batchCmd.Example = `  # Execute operations from file
  rune batch operations.json

  # Execute operations from stdin
  echo '{"file":"tasks.md","operations":[{"type":"add","title":"New task"}]}' | rune batch

  # Execute operations from string input
  rune batch --input '{"file":"tasks.md","operations":[{"type":"add","title":"New task"}]}'

  # Dry run to preview changes
  rune batch operations.json --dry-run

  # Get JSON output
  rune batch operations.json --format json`
}
