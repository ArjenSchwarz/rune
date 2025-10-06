package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

var (
	hasPhasesVerbose bool
)

var hasPhasesCmd = &cobra.Command{
	Use:   "has-phases [file]",
	Short: "Check if the task file contains phases",
	Long: `Check if the task file contains phase headers (H2 markdown headers).
Returns JSON output with phase detection results and uses exit codes for scripting:
  - Exit code 0: File contains phases
  - Exit code 1: File does not contain phases or error occurred

The JSON output includes:
  - hasPhases: boolean indicating if phases exist
  - count: number of phases found
  - phases: array of phase names (included when --verbose is used)

Examples:
  rune has-phases
  rune has-phases tasks.md
  rune has-phases --verbose tasks.md`,
	Args:          cobra.MaximumNArgs(1),
	RunE:          runHasPhases,
	SilenceErrors: true, // We output JSON errors ourselves
	SilenceUsage:  true, // Don't show usage on errors
}

func init() {
	rootCmd.AddCommand(hasPhasesCmd)
	hasPhasesCmd.Flags().BoolVarP(&hasPhasesVerbose, "verbose", "v", false, "include phase names in output")
}

func runHasPhases(cmd *cobra.Command, args []string) error {
	// Resolve filename
	filename, err := resolveFilename(args)
	if err != nil {
		// Return error as JSON
		outputError(err.Error())
		return err
	}

	// Read file content
	content, err := os.ReadFile(filename)
	if err != nil {
		errMsg := fmt.Sprintf("failed to read file %s: %v", filename, err)
		outputError(errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// Extract phase markers
	lines := strings.Split(string(content), "\n")
	markers := task.ExtractPhaseMarkers(lines)

	// Build result
	hasPhases := len(markers) > 0
	result := HasPhasesOutput{
		HasPhases: hasPhases,
		Count:     len(markers),
	}

	// Include phase names if verbose flag is set
	if hasPhasesVerbose {
		result.Phases = make([]string, len(markers))
		for i, marker := range markers {
			result.Phases[i] = marker.Name
		}
	} else {
		result.Phases = []string{}
	}

	// Output JSON result
	jsonOutput, err := json.Marshal(result)
	if err != nil {
		errMsg := fmt.Sprintf("failed to marshal JSON: %v", err)
		outputError(errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	fmt.Println(string(jsonOutput))

	// Return error if no phases found to set exit code 1
	// Error printing is suppressed via command SilenceErrors flag
	if !hasPhases {
		return fmt.Errorf("no phases found")
	}

	return nil
}

// HasPhasesOutput represents the JSON output structure for has-phases command
type HasPhasesOutput struct {
	HasPhases bool     `json:"hasPhases"`
	Count     int      `json:"count"`
	Phases    []string `json:"phases"`
}

// outputError outputs an error message in JSON format
func outputError(message string) {
	errorOutput := map[string]string{
		"error": message,
	}
	jsonOutput, _ := json.Marshal(errorOutput)
	fmt.Println(string(jsonOutput))
}
