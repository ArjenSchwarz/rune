package cmd

import (
	"fmt"
	"os"

	"github.com/ArjenSchwarz/go-tasks/internal/task"
	"github.com/spf13/cobra"
)

var completeCmd = &cobra.Command{
	Use:   "complete [file] [task-id]",
	Short: "Mark a task as completed",
	Long: `Mark the specified task as completed by changing its status to [x].

Examples:
  go-tasks complete tasks.md 1
  go-tasks complete tasks.md 1.2.3`,
	Args: cobra.ExactArgs(2),
	RunE: runComplete,
}

func init() {
	rootCmd.AddCommand(completeCmd)
}

func runComplete(cmd *cobra.Command, args []string) error {
	filename := args[0]
	taskID := args[1]

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

	// Write the updated file
	if err := tl.WriteFile(filename); err != nil {
		return fmt.Errorf("failed to write updated file: %w", err)
	}

	if verbose {
		fmt.Printf("Successfully marked task as completed in file: %s\n", filename)
		fmt.Printf("Task ID: %s\n", taskID)
		fmt.Printf("Title: %s\n", targetTask.Title)
		fmt.Printf("Status: completed [x]\n")
	} else {
		fmt.Printf("Completed task %s: %s\n", taskID, targetTask.Title)
	}

	return nil
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
