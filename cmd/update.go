package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

// UpdateResponse is the JSON response for the update command
type UpdateResponse struct {
	Success       bool     `json:"success"`
	Message       string   `json:"message"`
	TaskID        string   `json:"task_id"`
	Title         string   `json:"title"`
	FieldsUpdated []string `json:"fields_updated"`
}

var updateCmd = &cobra.Command{
	Use:   "update [file] [task-id]",
	Short: "Update task title, details, references, or dependencies",
	Long: `Update the title, details, references, or dependencies of an existing task.

Use the flags to specify what to update. If a field is not provided, it will remain unchanged.
To clear details or references, use the --clear-* flags.

Use --stream to change the task's work stream assignment.
Use --blocked-by to set task dependencies (comma-separated task IDs).
Use --owner to claim the task for an agent.
Use --release to clear the owner (release the task).

Examples:
  rune update tasks.md 1 --title "New title"
  rune update tasks.md 1.1 --details "First detail,Second detail"
  rune update tasks.md 2 --references "doc.md,spec.md"
  rune update tasks.md 3 --title "Updated" --details "New detail"
  rune update tasks.md 1 --stream 2
  rune update tasks.md 2 --blocked-by "1"
  rune update tasks.md 1 --owner "agent-1"
  rune update tasks.md 1 --release`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runUpdate,
}

const noneDisplay = "(none)"

var (
	updateTitle        string
	updateDetails      string
	updateReferences   string
	updateRequirements string
	clearDetails       bool
	clearReferences    bool
	clearRequirements  bool
	updateStream       int
	updateStreamSet    bool
	updateBlockedBy    string
	updateOwner        string
	updateOwnerSet     bool
	updateRelease      bool
)

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().StringVarP(&updateTitle, "title", "t", "", "new title for the task")
	updateCmd.Flags().StringVarP(&updateDetails, "details", "d", "", "comma-separated list of details")
	updateCmd.Flags().StringVarP(&updateReferences, "references", "r", "", "comma-separated list of references")
	updateCmd.Flags().StringVar(&updateRequirements, "requirements", "", "comma-separated list of requirement IDs")
	updateCmd.Flags().BoolVar(&clearDetails, "clear-details", false, "clear all details from the task")
	updateCmd.Flags().BoolVar(&clearReferences, "clear-references", false, "clear all references from the task")
	updateCmd.Flags().BoolVar(&clearRequirements, "clear-requirements", false, "clear all requirements from the task")
	updateCmd.Flags().IntVar(&updateStream, "stream", 0, "stream assignment for the task (positive integer)")
	updateCmd.Flags().StringVar(&updateBlockedBy, "blocked-by", "", "comma-separated task IDs that must complete before this task")
	updateCmd.Flags().StringVar(&updateOwner, "owner", "", "agent identifier claiming the task")
	updateCmd.Flags().BoolVar(&updateRelease, "release", false, "clear the owner (release the task)")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	var filename, taskID string

	// Handle different argument patterns
	if len(args) == 2 {
		// Traditional: update [file] [task-id]
		filename = args[0]
		taskID = args[1]
	} else {
		// New: update [task-id] with git discovery
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
		verboseStderr("Updating task %s", taskID)
	} else if verbose {
		fmt.Printf("Using task file: %s\n", filename)
		fmt.Printf("Updating task %s\n", taskID)
	}

	// Check if stream flag was explicitly set (only if not already set by tests)
	if cmd.Flags().Lookup("stream") != nil {
		updateStreamSet = cmd.Flags().Changed("stream")
	}
	// Check if owner flag was explicitly set (only if not already set by tests)
	if cmd.Flags().Lookup("owner") != nil {
		updateOwnerSet = cmd.Flags().Changed("owner")
	}

	// Validate that at least one update field is provided
	if updateTitle == "" && updateDetails == "" && updateReferences == "" && updateRequirements == "" &&
		!clearDetails && !clearReferences && !clearRequirements &&
		!updateStreamSet && updateBlockedBy == "" && !updateOwnerSet && !updateRelease {
		return fmt.Errorf("at least one update flag must be provided (--title, --details, --references, --requirements, --clear-details, --clear-references, --clear-requirements, --stream, --blocked-by, --owner, or --release)")
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
	var newDetails, newReferences, newRequirements []string

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

	// Handle requirements
	if clearRequirements {
		newRequirements = []string{}
	} else if updateRequirements != "" {
		// Parse comma-separated IDs
		newRequirements = parseRequirementIDs(updateRequirements)

		// Validate format
		for _, reqID := range newRequirements {
			if !task.IsValidID(reqID) {
				return fmt.Errorf("invalid requirement ID format: %s", reqID)
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
		fmt.Printf("Current requirements: %s\n", formatRequirementsForDisplay(targetTask.Requirements))
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
		if clearRequirements {
			fmt.Printf("New requirements: (cleared)\n")
		} else if updateRequirements != "" {
			fmt.Printf("New requirements: %s\n", formatRequirementsForDisplay(newRequirements))
		}
		return nil
	}

	// Check if any extended options are being used
	hasExtendedOptions := updateStreamSet || updateBlockedBy != "" || updateOwnerSet || updateRelease

	// Update the task
	if hasExtendedOptions {
		// Use extended update with dependencies/streams support
		opts := task.UpdateOptions{
			Release: updateRelease,
		}

		// Set title if provided
		if updateTitle != "" {
			opts.Title = &updateTitle
		}

		// Set details if updating
		if clearDetails || updateDetails != "" {
			opts.Details = newDetails
		}

		// Set references if updating
		if clearReferences || updateReferences != "" {
			opts.References = newReferences
		}

		// Set requirements if updating
		if clearRequirements || updateRequirements != "" {
			opts.Requirements = newRequirements
		}

		// Set stream if flag was used
		if updateStreamSet {
			opts.Stream = &updateStream
		}

		// Set blocked-by if provided
		if updateBlockedBy != "" {
			opts.BlockedBy = parseRequirementIDs(updateBlockedBy)
		}

		// Set owner if flag was used
		if updateOwnerSet {
			opts.Owner = &updateOwner
		}

		if err := tl.UpdateTaskWithOptions(taskID, opts); err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}
	} else {
		// Use regular update
		if err := tl.UpdateTask(taskID, updateTitle, newDetails, newReferences, newRequirements); err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}
	}

	// Write the updated file
	if err := tl.WriteFile(filename); err != nil {
		return fmt.Errorf("failed to write updated file: %w", err)
	}

	// Collect changes for output
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
	if clearRequirements || updateRequirements != "" {
		changes = append(changes, "requirements")
	}
	if updateStreamSet {
		changes = append(changes, "stream")
	}
	if updateBlockedBy != "" {
		changes = append(changes, "blocked-by")
	}
	if updateOwnerSet {
		changes = append(changes, "owner")
	}
	if updateRelease {
		changes = append(changes, "release")
	}

	// Format-aware output
	switch format {
	case formatJSON:
		return outputJSON(UpdateResponse{
			Success:       true,
			Message:       fmt.Sprintf("Updated task %s", taskID),
			TaskID:        taskID,
			Title:         targetTask.Title,
			FieldsUpdated: changes,
		})
	case formatMarkdown:
		fmt.Printf("**Updated:** %s - %s (%s)\n", taskID, targetTask.Title, strings.Join(changes, ", "))
		return nil
	default: // table
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
			if clearRequirements {
				fmt.Printf("Requirements cleared\n")
			} else if updateRequirements != "" {
				fmt.Printf("Requirements updated to: %s\n", formatRequirementsForDisplay(newRequirements))
			}
		} else {
			fmt.Printf("Updated task %s (%s): %s\n", taskID, strings.Join(changes, ", "), targetTask.Title)
		}
		return nil
	}
}

func formatDetailsForDisplay(details []string) string {
	if len(details) == 0 {
		return noneDisplay
	}
	return strings.Join(details, ", ")
}

func formatReferencesForDisplay(references []string) string {
	if len(references) == 0 {
		return noneDisplay
	}
	return strings.Join(references, ", ")
}

func formatRequirementsForDisplay(requirements []string) string {
	if len(requirements) == 0 {
		return noneDisplay
	}
	return strings.Join(requirements, ", ")
}
