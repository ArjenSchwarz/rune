package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

// CompleteResponse is the JSON response for the complete command
type CompleteResponse struct {
	Success       bool     `json:"success"`
	Message       string   `json:"message"`
	TaskID        string   `json:"task_id"`
	Title         string   `json:"title"`
	AutoCompleted []string `json:"auto_completed,omitempty"`
}

var completeCmd = &cobra.Command{
	Use:   "complete [file] [task-id]",
	Short: "Mark a task as completed",
	Long: `Mark the specified task as completed by changing its status to [x].

If only a task-id is provided and git discovery is enabled in configuration, the file
will be automatically discovered based on the current git branch using the configured
template pattern.

Examples:
  rune complete tasks.md 1
  rune complete 1.2.3`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runComplete,
}

func init() {
	rootCmd.AddCommand(completeCmd)
}

func runComplete(cmd *cobra.Command, args []string) error {
	var filename, taskID string

	// Handle different argument patterns
	if len(args) == 2 {
		// Traditional: complete [file] [task-id]
		filename = args[0]
		taskID = args[1]
	} else {
		// New: complete [task-id] with git discovery
		taskID = args[0]
		var err error
		filename, err = resolveFilename([]string{})
		if err != nil {
			return err
		}
	}

	// Use stderr for verbose when JSON requested
	if format == formatJSON {
		verboseStderr("Using task file: %s", filename)
		verboseStderr("Marking task %s as complete", taskID)
	} else if verbose {
		fmt.Printf("Using task file: %s\n", filename)
		fmt.Printf("Marking task %s as complete\n", taskID)
	}

	// Check if file exists
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file %s does not exist", filename)
		}
		return fmt.Errorf("failed to access file %s: %w", filename, err)
	}

	// Parse existing file
	tl, err := task.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to parse task file: %w", err)
	}

	// Find the task to verify it exists
	targetTask := tl.FindTask(taskID)
	if targetTask == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Dry run mode - just show what would be completed
	if dryRun {
		fmt.Printf("Would mark task as completed in file: %s\n", filename)
		fmt.Printf("Task ID: %s\n", taskID)
		fmt.Printf("Current status: %s\n", statusToString(targetTask.Status))
		fmt.Printf("New status: completed\n")
		fmt.Printf("Title: %s\n", targetTask.Title)
		return nil
	}

	// Update the task status
	if err := tl.UpdateStatus(taskID, task.Completed); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Auto-complete parent tasks if all their children are now complete
	autoCompleted, err := tl.AutoCompleteParents(taskID)
	if err != nil {
		return fmt.Errorf("failed to auto-complete parent tasks: %w", err)
	}

	// Write the updated file
	if err := tl.WriteFile(filename); err != nil {
		return fmt.Errorf("failed to write updated file: %w", err)
	}

	// Format-aware output
	switch format {
	case formatJSON:
		return outputJSON(CompleteResponse{
			Success:       true,
			Message:       fmt.Sprintf("Completed task %s", taskID),
			TaskID:        taskID,
			Title:         targetTask.Title,
			AutoCompleted: autoCompleted,
		})
	case formatMarkdown:
		fmt.Printf("**Completed:** %s - %s\n", taskID, targetTask.Title)
		if len(autoCompleted) > 0 {
			fmt.Printf("**Auto-completed:** %s\n", strings.Join(autoCompleted, ", "))
		}
		return nil
	default: // table
		if verbose {
			fmt.Printf("Successfully marked task as completed in file: %s\n", filename)
			fmt.Printf("Task ID: %s\n", taskID)
			fmt.Printf("Title: %s\n", targetTask.Title)
			fmt.Printf("Status: completed [x]\n")
			if len(autoCompleted) > 0 {
				fmt.Printf("Auto-completed parent tasks: %s\n", strings.Join(autoCompleted, ", "))
			}
		} else {
			fmt.Printf("Completed task %s: %s\n", taskID, targetTask.Title)
			if len(autoCompleted) > 0 {
				fmt.Printf("Auto-completed parent tasks: %s\n", strings.Join(autoCompleted, ", "))
			}
		}
		return nil
	}
}

// statusToString converts a task status to a readable string
func statusToString(status task.Status) string {
	switch status {
	case task.Pending:
		return "pending"
	case task.InProgress:
		return "in-progress"
	case task.Completed:
		return "completed"
	default:
		return "unknown"
	}
}
