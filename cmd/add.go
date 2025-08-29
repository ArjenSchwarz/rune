package cmd

import (
	"fmt"
	"os"

	"github.com/ArjenSchwarz/go-tasks/internal/task"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [file] --title [title]",
	Short: "Add a new task to a task file",
	Long: `Add a new task or subtask to the specified task file.

Use --parent to add the task as a subtask under an existing task.
Without --parent, the task will be added as a top-level task.

Examples:
  go-tasks add tasks.md --title "Write documentation"
  go-tasks add tasks.md --title "Write API docs" --parent "1"`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

var (
	addTitle  string
	addParent string
)

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&addTitle, "title", "t", "", "title for the new task")
	addCmd.Flags().StringVarP(&addParent, "parent", "p", "", "parent task ID (optional)")
	addCmd.MarkFlagRequired("title")
}

func runAdd(cmd *cobra.Command, args []string) error {
	filename := args[0]

	// Check if file exists
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file %s does not exist (use 'create' command to create it first)", filename)
		}
		return fmt.Errorf("failed to access file %s: %w", filename, err)
	}

	// Parse existing file
	tl, err := task.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to parse task file: %w", err)
	}

	// Validate parent ID if provided
	if addParent != "" {
		if parent := tl.FindTask(addParent); parent == nil {
			return fmt.Errorf("parent task %s not found", addParent)
		}
	}

	// Dry run mode - just show what would be added
	if dryRun {
		fmt.Printf("Would add task to file: %s\n", filename)
		fmt.Printf("Title: %s\n", addTitle)
		if addParent != "" {
			if parent := tl.FindTask(addParent); parent != nil {
				fmt.Printf("Parent: %s (%s)\n", addParent, parent.Title)
			}
		} else {
			fmt.Printf("Location: Top-level task\n")
		}

		// Calculate what the new task ID would be
		var newID string
		if addParent != "" {
			if parent := tl.FindTask(addParent); parent != nil {
				newID = fmt.Sprintf("%s.%d", addParent, len(parent.Children)+1)
			}
		} else {
			newID = fmt.Sprintf("%d", len(tl.Tasks)+1)
		}
		fmt.Printf("New task ID would be: %s\n", newID)
		return nil
	}

	// Add the task
	if err := tl.AddTask(addParent, addTitle); err != nil {
		return fmt.Errorf("failed to add task: %w", err)
	}

	// Write the updated file
	if err := tl.WriteFile(filename); err != nil {
		return fmt.Errorf("failed to write updated file: %w", err)
	}

	if verbose {
		// Find the newly added task to get its ID
		var newTask *task.Task
		if addParent != "" {
			if parent := tl.FindTask(addParent); parent != nil && len(parent.Children) > 0 {
				newTask = &parent.Children[len(parent.Children)-1]
			}
		} else if len(tl.Tasks) > 0 {
			newTask = &tl.Tasks[len(tl.Tasks)-1]
		}

		fmt.Printf("Successfully added task to file: %s\n", filename)
		if newTask != nil {
			fmt.Printf("Task ID: %s\n", newTask.ID)
			fmt.Printf("Title: %s\n", newTask.Title)
			if addParent != "" {
				if parent := tl.FindTask(addParent); parent != nil {
					fmt.Printf("Parent: %s (%s)\n", addParent, parent.Title)
				}
			}
		}
	} else {
		// Find the newly added task for simple output
		var newTaskID string
		if addParent != "" {
			if parent := tl.FindTask(addParent); parent != nil && len(parent.Children) > 0 {
				newTaskID = parent.Children[len(parent.Children)-1].ID
			}
		} else if len(tl.Tasks) > 0 {
			newTaskID = tl.Tasks[len(tl.Tasks)-1].ID
		}
		fmt.Printf("Added task %s: %s\n", newTaskID, addTitle)
	}

	return nil
}
