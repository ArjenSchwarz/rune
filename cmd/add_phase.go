package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var addPhaseCmd = &cobra.Command{
	Use:   "add-phase [name]",
	Short: "Add a new phase to the task file",
	Long: `Add a new phase header to the task file. Phases are used to organize tasks
into logical groupings. The phase will be added as a markdown H2 header (## Phase Name)
at the end of the document.

Examples:
  rune add-phase "Planning"
  rune add-phase "Implementation"
  rune add-phase tasks.md "Testing"`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runAddPhase,
}

func init() {
	rootCmd.AddCommand(addPhaseCmd)
}

func runAddPhase(cmd *cobra.Command, args []string) error {
	var filename string
	var phaseName string

	// Handle arguments based on count
	if len(args) == 1 {
		// Only phase name provided, use git discovery for filename
		phaseName = args[0]
		resolvedFilename, err := resolveFilename([]string{})
		if err != nil {
			return err
		}
		filename = resolvedFilename
	} else {
		// Both filename and phase name provided
		filename = args[0]
		phaseName = args[1]
	}

	// Trim whitespace from phase name
	phaseName = strings.TrimSpace(phaseName)

	if verbose {
		fmt.Printf("Using task file: %s\n", filename)
		fmt.Printf("Adding phase: %s\n", phaseName)
	}

	// Check if file exists
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file %s does not exist (use 'create' command to create it first)", filename)
		}
		return fmt.Errorf("failed to access file %s: %w", filename, err)
	}

	// Dry run mode - just show what would be added
	if dryRun {
		fmt.Printf("Would add phase to file: %s\n", filename)
		fmt.Printf("Phase name: %s\n", phaseName)
		fmt.Printf("Phase header: ## %s\n", phaseName)
		return nil
	}

	// Read existing content
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Prepare the phase header
	phaseHeader := fmt.Sprintf("## %s", phaseName)

	// Ensure content ends with a newline, then append the phase
	contentStr := string(content)
	if len(contentStr) > 0 && !strings.HasSuffix(contentStr, "\n") {
		contentStr += "\n"
	}
	contentStr += phaseHeader + "\n"

	// Write back to file
	err = os.WriteFile(filename, []byte(contentStr), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Output success message
	fmt.Printf("Added phase '%s' to %s\n", phaseName, filename)

	return nil
}
