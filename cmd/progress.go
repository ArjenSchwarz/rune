package cmd

import (
	"fmt"
	"os"

	"github.com/ArjenSchwarz/go-tasks/internal/task"
	"github.com/spf13/cobra"
)

var progressCmd = &cobra.Command{
	Use:   "progress [file] [task-id]",
	Short: "Mark a task as in-progress",
	Long: `Mark the specified task as in-progress by changing its status to [-].

This indicates that work on the task has started but is not yet complete.

Examples:
  go-tasks progress tasks.md 1
  go-tasks progress tasks.md 1.2.3`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runProgress,
}

func init() {
	rootCmd.AddCommand(progressCmd)
}

func runProgress(cmd *cobra.Command, args []string) error {
	var filename, taskID string

	// Handle different argument patterns
	if len(args) == 2 {
		// Traditional: progress [file] [task-id]
		filename = args[0]
		taskID = args[1]
	} else {
		// New: progress [task-id] with git discovery
		taskID = args[0]
		var err error
		filename, err = resolveFilename([]string{})
		if err != nil {
			return err
		}
	}

	if verbose {
		fmt.Printf("Using task file: %s\n", filename)
		fmt.Printf("Marking task %s as in-progress\n", taskID)
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

	// Dry run mode - just show what would be set to in-progress
	if dryRun {
		fmt.Printf("Would mark task as in-progress in file: %s\n", filename)
		fmt.Printf("Task ID: %s\n", taskID)
		fmt.Printf("Current status: %s\n", statusToString(targetTask.Status))
		fmt.Printf("New status: in-progress\n")
		fmt.Printf("Title: %s\n", targetTask.Title)
		return nil
	}

	// Update the task status
	if err := tl.UpdateStatus(taskID, task.InProgress); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Write the updated file
	if err := tl.WriteFile(filename); err != nil {
		return fmt.Errorf("failed to write updated file: %w", err)
	}

	if verbose {
		fmt.Printf("Successfully marked task as in-progress in file: %s\n", filename)
		fmt.Printf("Task ID: %s\n", taskID)
		fmt.Printf("Title: %s\n", targetTask.Title)
		fmt.Printf("Status: in-progress [-]\n")
	} else {
		fmt.Printf("Started task %s: %s\n", taskID, targetTask.Title)
	}

	return nil
}
