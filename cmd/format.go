package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/arjenschwarz/rune/internal/task"
)

// Format utility functions for consistent output across commands.
// These utilities ensure commands respect the --format flag properly.

// outputJSON marshals any struct to stdout as JSON with indentation.
// This should be used by all commands when producing JSON output.
func outputJSON(data any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// outputMarkdownMessage writes a message as a markdown blockquote.
// Used for informational messages when markdown format is requested.
func outputMarkdownMessage(message string) {
	fmt.Printf("> %s\n", message)
}

// outputMessage writes a plain text message to stdout.
// Used for informational messages when table format is requested.
func outputMessage(message string) {
	fmt.Println(message)
}

// verboseStderr writes verbose output to stderr when verbose mode is enabled.
// This preserves stdout for clean JSON output when --format json is used.
func verboseStderr(formatStr string, args ...any) {
	if verbose {
		fmt.Fprintf(os.Stderr, formatStr+"\n", args...)
	}
}

// reportParseWarnings writes non-fatal parse warnings (such as duplicate
// stable IDs) to stderr so they never corrupt JSON output on stdout.
func reportParseWarnings(tl *task.TaskList) {
	for _, w := range tl.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w.Message)
	}
}

// printWarnings writes operation warnings to stderr for table and markdown
// output; JSON output carries them in a "warnings" array instead.
func printWarnings(warnings []task.Warning) {
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w.Message)
	}
}
