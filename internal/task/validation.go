package task

import (
	"fmt"
	"strings"
)

// ValidatePhaseName validates a phase name for use in task files
// Returns an error if the phase name is empty after trimming whitespace
func ValidatePhaseName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("phase name cannot be empty")
	}
	return nil
}
