package cmd

import (
	"fmt"
	"os"

	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

// RemoveResponse is the JSON response for the remove command
type RemoveResponse struct {
	Success         bool   `json:"success"`
	Message         string `json:"message"`
	TaskID          string `json:"task_id"`
	Title           string `json:"title"`
	ChildrenRemoved int    `json:"children_removed"`
}

var removeCmd = &cobra.Command{
	Use:   "remove [file] [task-id]",
	Short: "Remove a task and all its subtasks",
	Long: `Remove the specified task and all its subtasks from the file.

All remaining tasks will be automatically renumbered to maintain consistency.
This operation cannot be undone, so use --dry-run to preview changes first.

Examples:
  rune remove tasks.md 1
  rune remove tasks.md 2.1
  rune remove tasks.md 3 --dry-run`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runRemove,
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	var filename, taskID string

	// Handle different argument patterns
	if len(args) == 2 {
		// Traditional: remove [file] [task-id]
		filename = args[0]
		taskID = args[1]
	} else {
		// New: remove [task-id] with git discovery
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
		verboseStderr("Removing task %s", taskID)
	} else if verbose {
		fmt.Printf("Using task file: %s\n", filename)
		fmt.Printf("Removing task %s\n", taskID)
	}

	// Check if file exists
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file %s does not exist", filename)
		}
		return fmt.Errorf("failed to access file %s: %w", filename, err)
	}

	// Read original file content for phase-aware operations
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	// Parse existing file
	tl, err := task.ParseMarkdown(content)
	if err != nil {
		return fmt.Errorf("failed to parse task file: %w", err)
	}
	tl.FilePath = filename

	// Find the task to verify it exists and get info for output
	targetTask := tl.FindTask(taskID)
	if targetTask == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Count children recursively for informational output
	childCount := countTaskChildren(targetTask)

	// Dry run mode - show what would be removed
	if dryRun {
		fmt.Printf("Would remove task from file: %s\n", filename)
		fmt.Printf("Task ID: %s\n", taskID)
		fmt.Printf("Title: %s\n", targetTask.Title)
		if childCount > 0 {
			fmt.Printf("This task has %d subtask(s) that will also be removed\n", childCount)
			fmt.Println("Subtasks to be removed:")
			printTaskHierarchy(targetTask, "  ")
		}
		fmt.Printf("All remaining tasks will be renumbered\n")
		return nil
	}

	// Remove the task (phase-aware - handles both phase and non-phase files)
	if err := tl.RemoveTaskWithPhases(taskID, content); err != nil {
		return fmt.Errorf("failed to remove task: %w", err)
	}

	// Format-aware output
	switch format {
	case formatJSON:
		return outputJSON(RemoveResponse{
			Success:         true,
			Message:         fmt.Sprintf("Removed task %s", taskID),
			TaskID:          taskID,
			Title:           targetTask.Title,
			ChildrenRemoved: childCount,
		})
	case formatMarkdown:
		if childCount > 0 {
			fmt.Printf("**Removed:** %s - %s (and %d subtasks)\n", taskID, targetTask.Title, childCount)
		} else {
			fmt.Printf("**Removed:** %s - %s\n", taskID, targetTask.Title)
		}
		return nil
	default: // table
		if verbose {
			fmt.Printf("Successfully removed task from file: %s\n", filename)
			fmt.Printf("Removed task ID: %s\n", taskID)
			fmt.Printf("Title: %s\n", targetTask.Title)
			if childCount > 0 {
				fmt.Printf("Also removed %d subtask(s)\n", childCount)
			}
			fmt.Printf("Remaining tasks have been renumbered\n")
		} else {
			if childCount > 0 {
				fmt.Printf("Removed task %s and %d subtask(s): %s\n", taskID, childCount, targetTask.Title)
			} else {
				fmt.Printf("Removed task %s: %s\n", taskID, targetTask.Title)
			}
		}
		return nil
	}
}

// countTaskChildren recursively counts all children of a task
func countTaskChildren(task *task.Task) int {
	count := len(task.Children)
	for i := range task.Children {
		count += countTaskChildren(&task.Children[i])
	}
	return count
}

// printTaskHierarchy recursively prints task hierarchy for dry-run preview
func printTaskHierarchy(task *task.Task, indent string) {
	for i := range task.Children {
		child := &task.Children[i]
		fmt.Printf("%s- %s %s. %s\n", indent, statusToCheckbox(child.Status), child.ID, child.Title)
		if len(child.Children) > 0 {
			printTaskHierarchy(child, indent+"  ")
		}
	}
}

// statusToCheckbox converts a task status to its checkbox representation
func statusToCheckbox(status task.Status) string {
	switch status {
	case task.Pending:
		return "[ ]"
	case task.InProgress:
		return "[-]"
	case task.Completed:
		return "[x]"
	default:
		return "[ ]"
	}
}
