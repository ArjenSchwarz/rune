package task

import (
	"fmt"
	"regexp"
	"time"
)

const (
	// Checkbox markers for task status
	checkboxPending    = "[ ]"
	checkboxInProgress = "[-]"
	checkboxCompleted  = "[x]"
)

// Status represents the state of a task
type Status int

const (
	// Pending indicates a task that has not been started
	Pending Status = iota
	// InProgress indicates a task that is currently being worked on
	InProgress
	// Completed indicates a task that has been finished
	Completed
)

// String returns the checkbox representation of the status
func (s Status) String() string {
	switch s {
	case Pending:
		return checkboxPending
	case InProgress:
		return checkboxInProgress
	case Completed:
		return checkboxCompleted
	default:
		return checkboxPending
	}
}

// ParseStatus converts a checkbox string to a Status
func ParseStatus(s string) (Status, error) {
	switch s {
	case checkboxPending:
		return Pending, nil
	case checkboxInProgress:
		return InProgress, nil
	case checkboxCompleted, "[X]":
		return Completed, nil
	default:
		return Pending, fmt.Errorf("invalid status: %s", s)
	}
}

// Task represents a single task in a hierarchical task list
type Task struct {
	ID         string
	Title      string
	Status     Status
	Details    []string
	References []string
	Children   []Task
	ParentID   string
}

var taskIDPattern = regexp.MustCompile(`^\d+(\.\d+)*$`)

// Validate checks if the task has valid data
func (t *Task) Validate() error {
	if t.Title == "" {
		return fmt.Errorf("task title cannot be empty")
	}
	if len(t.Title) > 500 {
		return fmt.Errorf("task title exceeds 500 characters")
	}
	if !isValidID(t.ID) {
		return fmt.Errorf("invalid task ID format: %s", t.ID)
	}
	return nil
}

func isValidID(id string) bool {
	return taskIDPattern.MatchString(id)
}

// TaskList represents a collection of tasks with metadata
type TaskList struct {
	Title    string
	Tasks    []Task
	FilePath string
	Modified time.Time
}
