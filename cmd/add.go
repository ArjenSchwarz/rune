package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

// AddResponse is the JSON response for the add command
type AddResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	TaskID  string `json:"task_id"`
	Title   string `json:"title"`
	Parent  string `json:"parent,omitempty"`
	Phase   string `json:"phase,omitempty"`
}

var addCmd = &cobra.Command{
	Use:   "add [file] --title [title]",
	Short: "Add a new task to a task file",
	Long: `Add a new task or subtask to the specified task file.

If no filename is provided and git discovery is enabled in configuration, the file
will be automatically discovered based on the current git branch using the configured
template pattern.

Use --parent to add the task as a subtask under an existing task.
Without --parent, the task will be added as a top-level task.

Use --position to insert the task at a specific position, causing existing
tasks at that position and beyond to be renumbered.

Use --phase to add the task to a specific phase. If the phase doesn't exist,
it will be created at the end of the document.

Use --stream to assign the task to a specific work stream for parallel execution.

Use --blocked-by to specify tasks that must complete before this task is ready.
The blocked-by flag accepts comma-separated task IDs.

Use --owner to claim the task for a specific agent.

Examples:
  rune add tasks.md --title "Write documentation"
  rune add --title "Write API docs" --parent "1"
  rune add --title "Urgent task" --position "2"
  rune add --title "Setup database" --phase "Development"
  rune add --title "Build API" --stream 2 --blocked-by "1,2"
  rune add --title "Review code" --owner "agent-1"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAdd,
}

var (
	addTitle            string
	addParent           string
	addPosition         string
	addPhase            string
	addRequirements     string
	addRequirementsFile string
	addStream           int
	addBlockedBy        string
	addOwner            string
)

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&addTitle, "title", "t", "", "title for the new task")
	addCmd.Flags().StringVarP(&addParent, "parent", "p", "", "parent task ID (optional)")
	addCmd.Flags().StringVar(&addPosition, "position", "", "position to insert task (optional)")
	addCmd.Flags().StringVar(&addPhase, "phase", "", "target phase for the new task")
	addCmd.Flags().StringVar(&addRequirements, "requirements", "", "comma-separated requirement IDs (e.g., \"1.1,1.2,2.3\")")
	addCmd.Flags().StringVar(&addRequirementsFile, "requirements-file", "", "path to requirements file (default: requirements.md)")
	addCmd.Flags().IntVar(&addStream, "stream", 0, "stream assignment for the task (positive integer)")
	addCmd.Flags().StringVar(&addBlockedBy, "blocked-by", "", "comma-separated task IDs that must complete before this task")
	addCmd.Flags().StringVar(&addOwner, "owner", "", "agent identifier claiming the task")
	addCmd.MarkFlagRequired("title")
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Resolve filename using git discovery if needed
	filename, err := resolveFilename(args)
	if err != nil {
		return err
	}

	// Use stderr for verbose when JSON requested
	if format == formatJSON {
		verboseStderr("Using task file: %s", filename)
	} else if verbose {
		fmt.Printf("Using task file: %s\n", filename)
	}

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
		if addPosition != "" {
			fmt.Printf("Position: %s\n", addPosition)
		}
		if addPhase != "" {
			fmt.Printf("Phase: %s\n", addPhase)
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

	// Check if any extended options are provided
	hasExtendedOptions := addStream != 0 || addBlockedBy != "" || addOwner != ""

	// Add the task - use phase-aware logic if phase is specified
	var newTaskID string
	switch {
	case addPhase != "":
		// Validate phase name
		if err := task.ValidatePhaseName(addPhase); err != nil {
			return err
		}
		// Use phase-aware task addition
		newTaskID, err = task.AddTaskToPhase(filename, addParent, addTitle, addPhase)
		if err != nil {
			return fmt.Errorf("failed to add task to phase: %w", err)
		}
	case hasExtendedOptions:
		// Use extended task addition with dependencies/streams support
		opts := task.AddOptions{
			Position: addPosition,
			Stream:   addStream,
			Owner:    addOwner,
		}

		// Parse blocked-by references
		if addBlockedBy != "" {
			opts.BlockedBy = parseRequirementIDs(addBlockedBy)
		}

		newTaskID, err = tl.AddTaskWithOptions(addParent, addTitle, opts)
		if err != nil {
			return fmt.Errorf("failed to add task: %w", err)
		}

		// Handle requirements if provided
		if addRequirements != "" {
			reqIDs := parseRequirementIDs(addRequirements)
			for _, reqID := range reqIDs {
				if !task.IsValidID(reqID) {
					return fmt.Errorf("invalid requirement ID format: %s", reqID)
				}
			}
			if newTask := tl.FindTask(newTaskID); newTask != nil {
				newTask.Requirements = reqIDs
			}
		}

		// Set requirements file path if provided, otherwise use default
		if addRequirementsFile != "" {
			tl.RequirementsFile = addRequirementsFile
		} else if tl.RequirementsFile == "" && addRequirements != "" {
			tl.RequirementsFile = task.DefaultRequirementsFile
		}

		// Write the updated file
		if err := tl.WriteFile(filename); err != nil {
			return fmt.Errorf("failed to write updated file: %w", err)
		}
	default:
		// Use regular task addition
		newTaskID, err = tl.AddTask(addParent, addTitle, addPosition)
		if err != nil {
			return fmt.Errorf("failed to add task: %w", err)
		}

		// Handle requirements if provided
		if addRequirements != "" {
			// Parse comma-separated IDs
			reqIDs := parseRequirementIDs(addRequirements)

			// Validate format using existing validation
			for _, reqID := range reqIDs {
				if !task.IsValidID(reqID) {
					return fmt.Errorf("invalid requirement ID format: %s", reqID)
				}
			}

			// Update task with requirements
			if newTask := tl.FindTask(newTaskID); newTask != nil {
				newTask.Requirements = reqIDs
			}
		}

		// Set requirements file path if provided, otherwise use default
		if addRequirementsFile != "" {
			tl.RequirementsFile = addRequirementsFile
		} else if tl.RequirementsFile == "" && addRequirements != "" {
			tl.RequirementsFile = task.DefaultRequirementsFile
		}

		// Write the updated file
		if err := tl.WriteFile(filename); err != nil {
			return fmt.Errorf("failed to write updated file: %w", err)
		}
	}

	// Format-aware output
	switch format {
	case formatJSON:
		return outputJSON(AddResponse{
			Success: true,
			Message: fmt.Sprintf("Added task %s", newTaskID),
			TaskID:  newTaskID,
			Title:   addTitle,
			Parent:  addParent,
			Phase:   addPhase,
		})
	case formatMarkdown:
		if addParent != "" {
			fmt.Printf("**Added:** %s - %s (under %s)\n", newTaskID, addTitle, addParent)
		} else {
			fmt.Printf("**Added:** %s - %s\n", newTaskID, addTitle)
		}
		return nil
	default: // table
		if verbose {
			// Find the newly added task to get its details
			var newTask *task.Task
			if newTaskID != "" {
				newTask = tl.FindTask(newTaskID)
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
			fmt.Printf("Added task %s: %s\n", newTaskID, addTitle)
		}
		return nil
	}
}

// parseRequirementIDs parses comma-separated requirement IDs from a string
func parseRequirementIDs(input string) []string {
	parts := strings.Split(input, ",")
	ids := make([]string, 0)
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	return ids
}
