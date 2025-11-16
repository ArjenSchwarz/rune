package task

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

const (
	// Checkbox markers for task status
	checkboxPending    = "[ ]"
	checkboxInProgress = "[-]"
	checkboxCompleted  = "[x]"

	// DefaultRequirementsFile is the default filename for requirements
	DefaultRequirementsFile = "requirements.md"
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

// UnmarshalJSON implements custom JSON unmarshaling for Status
// Accepts both integer values (0, 1, 2) and string names ("Pending", "InProgress", "Completed")
func (s *Status) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as integer first
	var intVal int
	if err := json.Unmarshal(data, &intVal); err == nil {
		switch intVal {
		case 0:
			*s = Pending
		case 1:
			*s = InProgress
		case 2:
			*s = Completed
		default:
			return fmt.Errorf("invalid status value: %d (must be 0-2)", intVal)
		}
		return nil
	}

	// Try to unmarshal as string
	var strVal string
	if err := json.Unmarshal(data, &strVal); err != nil {
		return fmt.Errorf("status must be an integer (0-2) or string: %w", err)
	}

	switch strVal {
	case "Pending", "pending":
		*s = Pending
	case "InProgress", "inprogress", "in-progress", "in_progress":
		*s = InProgress
	case "Completed", "completed":
		*s = Completed
	default:
		return fmt.Errorf("invalid status string: %s (must be Pending, InProgress, or Completed)", strVal)
	}
	return nil
}

// MarshalJSON implements custom JSON marshaling for Status
// Always outputs as integer for consistency
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(s))
}

// Task represents a single task in a hierarchical task list
type Task struct {
	ID           string
	Title        string
	Status       Status
	Details      []string
	References   []string
	Requirements []string `json:"requirements,omitempty"`
	Children     []Task
	ParentID     string
}

var taskIDPattern = regexp.MustCompile(`^[1-9]\d*(\.[1-9]\d*)*$`)

// Validate checks if the task has valid data
func (t *Task) Validate() error {
	if t.Title == "" {
		return fmt.Errorf("task title cannot be empty")
	}
	if len(t.Title) > 500 {
		return fmt.Errorf("task title exceeds 500 characters")
	}
	if !IsValidID(t.ID) {
		return fmt.Errorf("invalid task ID format: %s", t.ID)
	}
	// Validate requirement IDs match hierarchical pattern
	for _, reqID := range t.Requirements {
		if !IsValidID(reqID) {
			return fmt.Errorf("invalid requirement ID format: %s", reqID)
		}
	}
	return nil
}

// IsValidID checks if an ID matches the hierarchical pattern
func IsValidID(id string) bool {
	return taskIDPattern.MatchString(id)
}

// TaskList represents a collection of tasks with metadata
type TaskList struct {
	Title            string
	Tasks            []Task
	FrontMatter      *FrontMatter
	FilePath         string
	RequirementsFile string `json:"requirements_file,omitempty"`
	Modified         time.Time
}

// Stats represents aggregate statistics for a task list
type Stats struct {
	Total      int `json:"Total"`
	Pending    int `json:"Pending"`
	InProgress int `json:"InProgress"`
	Completed  int `json:"Completed"`
}

// CalculateStats computes aggregate statistics for all tasks in the list
func (tl *TaskList) CalculateStats() Stats {
	stats := Stats{}

	var countTasks func(tasks []Task)
	countTasks = func(tasks []Task) {
		for _, t := range tasks {
			stats.Total++
			switch t.Status {
			case Pending:
				stats.Pending++
			case InProgress:
				stats.InProgress++
			case Completed:
				stats.Completed++
			}
			// Recursively count children
			if len(t.Children) > 0 {
				countTasks(t.Children)
			}
		}
	}

	countTasks(tl.Tasks)
	return stats
}
