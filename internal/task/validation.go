package task

import (
	"fmt"
	"strings"
	"unicode"
)

// ValidatePhaseName validates a phase name for use in task files.
// Returns an error if the phase name is empty after trimming whitespace, or
// if it contains newlines or other control characters. Phase names are
// rendered verbatim into a markdown header ("## {name}"), so an unvalidated
// control character (e.g. a newline) would let the name inject arbitrary
// markdown/task lines into the file.
func ValidatePhaseName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("phase name cannot be empty")
	}
	if strings.ContainsFunc(name, unicode.IsControl) {
		return fmt.Errorf("phase name cannot contain control characters (e.g. newlines)")
	}
	return nil
}
