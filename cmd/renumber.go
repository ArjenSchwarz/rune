package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

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

	// Phase 2: Parse file
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

	// Phase 5: Renumber tasks
	taskList.RenumberTasks()

	// Phase 5.5: Update phase markers to reflect new task IDs
	// Note: renumberTasks() changes all task IDs, so phase markers need adjustment
	if len(phaseMarkers) > 0 {
		phaseMarkers = adjustPhaseMarkersAfterRenumber(phaseMarkers)
	}

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
		data := map[string]any{
			"task_count":  totalTasks,
			"backup_file": backupPath,
			"success":     true,
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)

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
			output.WithFormat(output.Table),
			output.WithWriter(output.NewStdoutWriter()),
		)

		return out.Render(context.Background(), doc)
	}
}

// adjustPhaseMarkersAfterRenumber updates phase marker AfterTaskID values
// to reflect the new task IDs after renumbering.
//
// Phase markers are positional - they mark boundaries between sections of tasks.
// After renumbering, tasks maintain their order, so phase positions remain valid.
// We extract the root task number from each AfterTaskID and reformat it.
func adjustPhaseMarkersAfterRenumber(markers []task.PhaseMarker) []task.PhaseMarker {
	adjustedMarkers := make([]task.PhaseMarker, len(markers))

	for i, marker := range markers {
		adjustedMarkers[i] = marker

		if marker.AfterTaskID == "" {
			// Phase at beginning - no adjustment needed
			continue
		}

		// Get the root task number from the ID
		// e.g., "3" -> 3, "3.2.1" -> 3 (all reference root task 3)
		rootTaskNum := getRootTaskNumber(marker.AfterTaskID)

		// After renumbering, root tasks are numbered 1, 2, 3...
		// So the phase is still after root task N
		adjustedMarkers[i].AfterTaskID = fmt.Sprintf("%d", rootTaskNum)
	}

	return adjustedMarkers
}

// getRootTaskNumber extracts the root task number from a hierarchical ID.
// Examples: "3" -> 3, "3.2.1" -> 3, "15.4" -> 15
func getRootTaskNumber(taskID string) int {
	parts := strings.Split(taskID, ".")
	num, _ := strconv.Atoi(parts[0])
	return num
}
