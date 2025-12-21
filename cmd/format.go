package cmd

import (
	"encoding/json"
	"fmt"
	"os"
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
