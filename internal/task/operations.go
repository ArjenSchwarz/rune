package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

// NewTaskList creates a new TaskList with the specified title
func NewTaskList(title string) *TaskList {
	return &TaskList{
		Title:    title,
		Tasks:    []Task{},
		Modified: time.Now(),
	}
}

// WriteFile saves the TaskList to a markdown file using atomic writes
func (tl *TaskList) WriteFile(filePath string) error {
	// Validate file path
	if err := validateFilePath(filePath); err != nil {
		return err
	}

	// Generate markdown content
	content := RenderMarkdown(tl)

	// Write to temp file first for atomic operation
	tmpFile := filePath + ".tmp"
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, filePath); err != nil {
		// Clean up temp file on failure
		os.Remove(tmpFile)
		return fmt.Errorf("atomic rename: %w", err)
	}

	// Update file path in TaskList
	tl.FilePath = filePath
	return nil
}

// validateFilePath ensures the file path is safe and valid
func validateFilePath(path string) error {
	// Clean and resolve path
	cleanPath := filepath.Clean(path)

	// Get working directory for safety check
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Resolve both paths to absolute
	workDirAbs, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("resolving working directory: %w", err)
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("resolving file path: %w", err)
	}

	// Ensure resolved path is within working directory tree
	// This prevents both absolute and relative path traversal attacks
	if !strings.HasPrefix(absPath, workDirAbs+string(filepath.Separator)) && absPath != workDirAbs {
		return fmt.Errorf("path traversal attempt detected")
	}

	return nil
}
