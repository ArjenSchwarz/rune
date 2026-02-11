package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

// RenumberResponse represents the JSON output for the renumber command.
type RenumberResponse struct {
	Success    bool   `json:"success"`
	TaskCount  int    `json:"task_count"`
	BackupFile string `json:"backup_file"`
}

var renumberCmd = &cobra.Command{
	Use:   "renumber [file]",
	Short: "Fix task numbering in a file",
	Long: `Renumber recalculates all task IDs to create sequential numbering.

This command is useful when tasks have been manually reordered and the
hierarchical IDs need to be recalculated.

Features:
- Creates automatic backups (.bak extension) before making any changes
- Uses global sequential numbering (1, 2, 3...) across the entire file
- Preserves task hierarchy and parent-child relationships
- Preserves task statuses, details, and references
- Preserves stable IDs, blocked-by dependencies, streams, and owners
- Preserves phase markers and YAML front matter
- Uses atomic file operations to prevent corruption

Usage Examples:
  # Renumber tasks with default table output
  rune renumber tasks.md

  # Renumber with JSON output
  rune renumber tasks.md --format json

  # Renumber file with phases
  rune renumber project.md --format markdown

How It Works:
  1. Validates file path and checks resource limits
  2. Parses the task file and phase markers
  3. Creates backup file (.bak extension)
  4. Renumbers all tasks sequentially (fills gaps)
  5. Updates phase markers to reflect new task IDs
  6. Writes changes atomically (temp file → rename)
  7. Displays summary with task count and backup location

Important Notes:
  - Requirement links in task details are NOT updated automatically
  - Backup file is always created for safety
  - If interrupted (Ctrl+C), original file remains intact
  - Use backup file to restore if needed

Use Cases:
  - After manually reordering tasks in the file
  - Fixing gaps in numbering (1, 2, 5 → 1, 2, 3)
  - Cleaning up IDs after complex editing
  - Standardizing numbering after merging sources`,
	Args: cobra.ExactArgs(1),
	RunE: runRenumber,
}

func init() {
	rootCmd.AddCommand(renumberCmd)
}

func runRenumber(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// Phase 1: Fast validation (before expensive operations)
	if err := task.ValidateFilePath(filePath); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	if fileInfo.Size() > task.MaxFileSize {
		return fmt.Errorf("file exceeds 10MB limit")
	}

	// Phase 2: Read file content to extract task IDs in file order (before parsing renumbers them)
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	fileTaskIDOrder := extractTaskIDOrder(string(fileContent))

	// Phase 2.5: Parse file (note: parser renumbers tasks automatically)
	taskList, phaseMarkers, err := task.ParseFileWithPhases(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse task file: %w", err)
	}

	// Phase 3: Validate resource limits
	totalTasks := taskList.CountTotalTasks()
	if totalTasks >= task.MaxTaskCount {
		return fmt.Errorf("task count (%d) exceeds limit of %d",
			totalTasks, task.MaxTaskCount)
	}

	// Phase 4: Create backup BEFORE any modifications
	backupPath, err := createBackup(filePath, fileInfo)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Phase 5: Convert phase markers from ID-based to position-based using file task order
	phasePositions := convertPhaseMarkersToPositions(phaseMarkers, fileTaskIDOrder)

	// Phase 5.5: Renumber tasks
	taskList.RenumberTasks()

	// Phase 5.75: Convert phase positions back to ID-based using new IDs
	phaseMarkers = convertPhasePositionsToMarkers(phasePositions, taskList)

	// Phase 6: Write file (atomic operation)
	if len(phaseMarkers) > 0 {
		err = task.WriteFileWithPhases(taskList, phaseMarkers, filePath)
	} else {
		err = taskList.WriteFile(filePath)
	}

	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Phase 7: Display summary
	return displaySummary(taskList, backupPath, format)
}

// createBackup creates a backup file with .bak extension
func createBackup(filePath string, fileInfo os.FileInfo) (string, error) {
	backupPath := filePath + ".bak"

	// Read original file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("reading original file: %w", err)
	}

	// Write backup (overwrites existing .bak)
	if err := os.WriteFile(backupPath, content, fileInfo.Mode().Perm()); err != nil {
		return "", fmt.Errorf("writing backup: %w", err)
	}

	return backupPath, nil
}

// displaySummary outputs the renumbering results in the specified format
func displaySummary(tl *task.TaskList, backupPath, format string) error {
	totalTasks := tl.CountTotalTasks()

	switch format {
	case formatJSON:
		return outputJSON(RenumberResponse{
			Success:    true,
			TaskCount:  totalTasks,
			BackupFile: backupPath,
		})

	case formatMarkdown:
		fmt.Println("# Renumbering Summary")
		fmt.Println()
		fmt.Printf("- **Total Tasks**: %d\n", totalTasks)
		fmt.Printf("- **Backup File**: %s\n", backupPath)
		fmt.Println("- **Status**: ✓ Success")
		return nil

	case "table":
		fallthrough
	default:
		// Use go-output library for consistent formatting
		data := []map[string]any{
			{"Field": "Total Tasks", "Value": fmt.Sprintf("%d", totalTasks)},
			{"Field": "Backup File", "Value": backupPath},
			{"Field": "Status", "Value": "✓ Success"},
		}

		doc := output.New().
			Table("Renumbering Summary", data, output.WithKeys("Field", "Value")).
			Build()

		out := output.NewOutput(
			output.WithFormat(output.Table()),
			output.WithWriter(output.NewStdoutWriter()),
		)

		return out.Render(context.Background(), doc)
	}
}

// phasePosition represents a phase marker by its position in the task list
// rather than by task ID. This allows phases to maintain their position
// even when task IDs change during renumbering.
type phasePosition struct {
	Name          string
	AfterPosition int // -1 means before all tasks, 0+ means after task at that index
}

// extractTaskIDOrder extracts root-level task IDs in the order they appear in the file.
// This is needed because the parser automatically renumbers tasks, but phase markers
// reference the original IDs from the file.
func extractTaskIDOrder(content string) []string {
	var taskIDs []string
	lines := strings.Split(content, "\n")

	taskLinePattern := regexp.MustCompile(`^- \[[ \-xX]\] (\d+(?:\.\d+)*)\. `)

	for _, line := range lines {
		if matches := taskLinePattern.FindStringSubmatch(line); len(matches) >= 2 {
			taskID := matches[1]
			// Only track root-level tasks (no dots in ID)
			if !strings.Contains(taskID, ".") {
				taskIDs = append(taskIDs, taskID)
			}
		}
	}

	return taskIDs
}

// convertPhaseMarkersToPositions converts phase markers from ID-based to position-based.
// Uses fileTaskIDOrder to map original file task IDs to their position in the file.
func convertPhaseMarkersToPositions(markers []task.PhaseMarker, fileTaskIDOrder []string) []phasePosition {
	// Build a map from file task ID to position
	idToPosition := make(map[string]int)
	for i, id := range fileTaskIDOrder {
		idToPosition[id] = i
	}

	positions := make([]phasePosition, len(markers))

	for i, marker := range markers {
		positions[i].Name = marker.Name

		if marker.AfterTaskID == "" {
			// Phase at the beginning
			positions[i].AfterPosition = -1
			continue
		}

		// Look up the position where this task ID appeared in the file
		if pos, exists := idToPosition[marker.AfterTaskID]; exists {
			positions[i].AfterPosition = pos
		} else {
			// Task ID not found, default to -1 (shouldn't happen with valid files)
			positions[i].AfterPosition = -1
		}
	}

	return positions
}

// convertPhasePositionsToMarkers converts position-based phase data back to
// ID-based phase markers using the renumbered task IDs.
func convertPhasePositionsToMarkers(positions []phasePosition, tl *task.TaskList) []task.PhaseMarker {
	markers := make([]task.PhaseMarker, len(positions))

	for i, pos := range positions {
		markers[i].Name = pos.Name

		if pos.AfterPosition == -1 {
			// Phase at the beginning
			markers[i].AfterTaskID = ""
		} else if pos.AfterPosition < len(tl.Tasks) {
			// Use the new ID of the task at this position
			markers[i].AfterTaskID = tl.Tasks[pos.AfterPosition].ID
		}
		// If position is out of bounds, leave AfterTaskID empty (shouldn't happen)
	}

	return markers
}

// getRootTaskID extracts the root task ID from a hierarchical ID.
// Examples: "3" -> "3", "3.2.1" -> "3", "15.4" -> "15"
func getRootTaskID(taskID string) string {
	parts := strings.Split(taskID, ".")
	return parts[0]
}
