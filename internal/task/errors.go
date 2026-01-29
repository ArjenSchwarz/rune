package task

import (
	"errors"
	"fmt"
	"strings"
)

// Error types for dependencies and streams
var (
	// Stable ID errors
	ErrNoStableID        = errors.New("task does not have a stable ID (legacy task)")
	ErrStableIDNotFound  = errors.New("stable ID not found")
	ErrDuplicateStableID = errors.New("duplicate stable ID detected")

	// Dependency errors
	ErrCircularDependency = errors.New("circular dependency detected")
	ErrInvalidBlockedBy   = errors.New("invalid blocked-by reference")

	// Stream errors
	ErrInvalidStream = errors.New("stream must be a positive integer")

	// Owner errors
	ErrInvalidOwner = errors.New("owner contains invalid characters")
)

// CircularDependencyError provides detailed cycle information
type CircularDependencyError struct {
	Path []string // Cycle path: [A, B, C, A] for A→B→C→A
}

func (e *CircularDependencyError) Error() string {
	if len(e.Path) == 2 && e.Path[0] == e.Path[1] {
		return fmt.Sprintf("task cannot depend on itself: %s", e.Path[0])
	}
	return fmt.Sprintf("circular dependency detected: %s", strings.Join(e.Path, " → "))
}

// Warning represents a non-fatal issue encountered during operation
type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	TaskID  string `json:"taskId,omitempty"` // Hierarchical ID if applicable
}

// Warning code constants
const (
	// WarnInvalidStableID indicates a stable ID has an invalid format
	WarnInvalidStableID = "invalid_stable_id"

	// WarnMissingDependency indicates a blocked-by reference points to a non-existent stable ID
	WarnMissingDependency = "missing_dependency"

	// WarnDuplicateStableID indicates multiple tasks share the same stable ID
	WarnDuplicateStableID = "duplicate_stable_id"

	// WarnInvalidStreamValue indicates a stream value is invalid
	WarnInvalidStreamValue = "invalid_stream_value"

	// WarnDependentsRemoved indicates a task was removed that other tasks depend on
	WarnDependentsRemoved = "dependents_removed"

	// WarnDependentsExist indicates a task being removed has other tasks depending on it
	WarnDependentsExist = "dependents_exist"
)
