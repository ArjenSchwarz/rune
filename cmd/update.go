package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ArjenSchwarz/go-tasks/internal/task"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [file] [task-id]",
	Short: "Update task title, details, or references",
	Long: `Update the title, details, or references of an existing task.

Use the flags to specify what to update. If a field is not provided, it will remain unchanged.
To clear details or references, use empty values.

Examples:
  go-tasks update tasks.md 1 --title "New title"
  go-tasks update tasks.md 1.1 --details "First detail,Second detail"
  go-tasks update tasks.md 2 --references "doc.md,spec.md"
  go-tasks update tasks.md 3 --title "Updated" --details "New detail"`,
	Args: cobra.ExactArgs(2),
	RunE: runUpdate,
}

var (
	updateTitle      string
	updateDetails    string
	updateReferences string
	clearDetails     bool
	clearReferences  bool
)

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().StringVarP(&updateTitle, "title", "t", "", "new title for the task")
	updateCmd.Flags().StringVarP(&updateDetails, "details", "d", "", "comma-separated list of details")
	updateCmd.Flags().StringVarP(&updateReferences, "references", "r", "", "comma-separated list of references")
	updateCmd.Flags().BoolVar(&clearDetails, "clear-details", false, "clear all details from the task")
	updateCmd.Flags().BoolVar(&clearReferences, "clear-references", false, "clear all references from the task")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	filename := args[0]
	taskID := args[1]

	// Validate that at least one update field is provided
	if updateTitle == "" && updateDetails == "" && updateReferences == "" && !clearDetails && !clearReferences {
		return fmt.Errorf("at least one update flag must be provided (--title, --details, --references, --clear-details, or --clear-references)")
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

	// Find the task to verify it exists and get current values
	targetTask := tl.FindTask(taskID)
	if targetTask == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Prepare update values
	var newDetails, newReferences []string

	// Handle details
	if clearDetails {
		newDetails = []string{}
	} else if updateDetails != "" {
		// Split by comma and trim whitespace
		parts := strings.SplitSeq(updateDetails, ",")
		for part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				newDetails = append(newDetails, trimmed)
			}
		}
	}

	// Handle references
	if clearReferences {
		newReferences = []string{}
	} else if updateReferences != "" {
		// Split by comma and trim whitespace
		parts := strings.SplitSeq(updateReferences, ",")
		for part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				newReferences = append(newReferences, trimmed)
			}
		}
	}

	// Dry run mode - show what would be updated
	if dryRun {
		fmt.Printf("Would update task in file: %s\n", filename)
		fmt.Printf("Task ID: %s\n", taskID)
		fmt.Printf("Current title: %s\n", targetTask.Title)
		fmt.Printf("Current details: %s\n", formatDetailsForDisplay(targetTask.Details))
		fmt.Printf("Current references: %s\n", formatReferencesForDisplay(targetTask.References))
		fmt.Println()

		if updateTitle != "" {
			fmt.Printf("New title: %s\n", updateTitle)
		}
		if clearDetails {
			fmt.Printf("New details: (cleared)\n")
		} else if updateDetails != "" {
			fmt.Printf("New details: %s\n", formatDetailsForDisplay(newDetails))
		}
		if clearReferences {
			fmt.Printf("New references: (cleared)\n")
		} else if updateReferences != "" {
			fmt.Printf("New references: %s\n", formatReferencesForDisplay(newReferences))
		}
		return nil
	}

	// Update the task
	if err := tl.UpdateTask(taskID, updateTitle, newDetails, newReferences); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Write the updated file
	if err := tl.WriteFile(filename); err != nil {
		return fmt.Errorf("failed to write updated file: %w", err)
	}

	if verbose {
		fmt.Printf("Successfully updated task in file: %s\n", filename)
		fmt.Printf("Task ID: %s\n", taskID)
		if updateTitle != "" {
			fmt.Printf("Title updated to: %s\n", updateTitle)
		}
		if clearDetails {
			fmt.Printf("Details cleared\n")
		} else if updateDetails != "" {
			fmt.Printf("Details updated to: %s\n", formatDetailsForDisplay(newDetails))
		}
		if clearReferences {
			fmt.Printf("References cleared\n")
		} else if updateReferences != "" {
			fmt.Printf("References updated to: %s\n", formatReferencesForDisplay(newReferences))
		}
	} else {
		changes := []string{}
		if updateTitle != "" {
			changes = append(changes, "title")
		}
		if clearDetails || updateDetails != "" {
			changes = append(changes, "details")
		}
		if clearReferences || updateReferences != "" {
			changes = append(changes, "references")
		}
		fmt.Printf("Updated task %s (%s): %s\n", taskID, strings.Join(changes, ", "), targetTask.Title)
	}

	return nil
}

func formatDetailsForDisplay(details []string) string {
	if len(details) == 0 {
		return "(none)"
	}
	return strings.Join(details, ", ")
}

func formatReferencesForDisplay(references []string) string {
	if len(references) == 0 {
		return "(none)"
	}
	return strings.Join(references, ", ")
}
