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

// FindTask searches for a task by ID in the task hierarchy
func (tl *TaskList) FindTask(taskID string) *Task {
	if taskID == "" {
		return nil
	}
	return findTaskRecursive(tl.Tasks, taskID)
}

func findTaskRecursive(tasks []Task, taskID string) *Task {
	for i := range tasks {
		if tasks[i].ID == taskID {
			return &tasks[i]
		}
		if found := findTaskRecursive(tasks[i].Children, taskID); found != nil {
			return found
		}
	}
	return nil
}

// AddTask adds a new task to the task list under the specified parent
func (tl *TaskList) AddTask(parentID, title string) error {
	if parentID != "" {
		parent := tl.FindTask(parentID)
		if parent == nil {
			return fmt.Errorf("parent task %s not found", parentID)
		}

		newTask := Task{
			ID:       fmt.Sprintf("%s.%d", parentID, len(parent.Children)+1),
			Title:    title,
			Status:   Pending,
			ParentID: parentID,
		}
		parent.Children = append(parent.Children, newTask)
	} else {
		newTask := Task{
			ID:     fmt.Sprintf("%d", len(tl.Tasks)+1),
			Title:  title,
			Status: Pending,
		}
		tl.Tasks = append(tl.Tasks, newTask)
	}

	tl.Modified = time.Now()
	return nil
}

// RemoveTask removes a task from the task list and renumbers remaining tasks
func (tl *TaskList) RemoveTask(taskID string) error {
	if removed := tl.removeTaskRecursive(&tl.Tasks, taskID, ""); removed {
		tl.renumberTasks()
		tl.Modified = time.Now()
		return nil
	}
	return fmt.Errorf("task %s not found", taskID)
}

func (tl *TaskList) removeTaskRecursive(tasks *[]Task, taskID string, _ string) bool {
	for i := 0; i < len(*tasks); i++ {
		if (*tasks)[i].ID == taskID {
			*tasks = append((*tasks)[:i], (*tasks)[i+1:]...)
			return true
		}
		if tl.removeTaskRecursive(&(*tasks)[i].Children, taskID, (*tasks)[i].ID) {
			return true
		}
	}
	return false
}

func (tl *TaskList) renumberTasks() {
	for i := range tl.Tasks {
		tl.Tasks[i].ID = fmt.Sprintf("%d", i+1)
		renumberChildren(&tl.Tasks[i])
	}
}

func renumberChildren(parent *Task) {
	for i := range parent.Children {
		parent.Children[i].ID = fmt.Sprintf("%s.%d", parent.ID, i+1)
		parent.Children[i].ParentID = parent.ID
		renumberChildren(&parent.Children[i])
	}
}

// UpdateStatus changes the status of a task
func (tl *TaskList) UpdateStatus(taskID string, status Status) error {
	task := tl.FindTask(taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}
	task.Status = status
	tl.Modified = time.Now()
	return nil
}

// UpdateTask modifies the title, details, and references of a task
func (tl *TaskList) UpdateTask(taskID, title string, details, refs []string) error {
	task := tl.FindTask(taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	if title != "" {
		task.Title = title
	}
	if details != nil {
		task.Details = details
	}
	if refs != nil {
		task.References = refs
	}

	tl.Modified = time.Now()
	return nil
}
