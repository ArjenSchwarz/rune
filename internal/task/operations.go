package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Resource limits for security
const (
	MaxTaskCount      = 10000 // Maximum number of tasks
	MaxHierarchyDepth = 10    // Maximum hierarchy depth
	MaxDetailLength   = 1000  // Maximum characters per detail line
)

// AddTask adds a new task to the task list under the specified parent
func (tl *TaskList) AddTask(parentID, title string) error {
	// Validate input
	if err := validateTaskInput(title); err != nil {
		return err
	}

	// Check resource limits
	if err := tl.checkResourceLimits(parentID); err != nil {
		return err
	}
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

	// Validate input
	if title != "" {
		if err := validateTaskInput(title); err != nil {
			return err
		}
		task.Title = title
	}
	if details != nil {
		if err := validateDetails(details); err != nil {
			return err
		}
		task.Details = details
	}
	if refs != nil {
		if err := validateReferences(refs); err != nil {
			return err
		}
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
	// Check for null bytes and control characters
	if containsNullByte(path) {
		return fmt.Errorf("path contains null bytes or control characters")
	}

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

// validateTaskInput sanitizes and validates task input
func validateTaskInput(input string) error {
	// Check for null bytes and control characters
	if containsNullByte(input) {
		return fmt.Errorf("input contains null bytes or control characters")
	}

	// Length validation is handled by Task.Validate()
	return nil
}

// validateDetails validates task details
func validateDetails(details []string) error {
	for i, detail := range details {
		if containsNullByte(detail) {
			return fmt.Errorf("detail %d contains null bytes or control characters", i+1)
		}
		if len(detail) > MaxDetailLength {
			return fmt.Errorf("detail %d exceeds maximum length of %d characters", i+1, MaxDetailLength)
		}
	}
	return nil
}

// validateReferences validates task references
func validateReferences(refs []string) error {
	for i, ref := range refs {
		if containsNullByte(ref) {
			return fmt.Errorf("reference %d contains null bytes or control characters", i+1)
		}
		if len(ref) > 500 {
			return fmt.Errorf("reference %d exceeds maximum length of 500 characters", i+1)
		}
	}
	return nil
}

// containsNullByte checks for null bytes and dangerous control characters
func containsNullByte(s string) bool {
	for _, r := range s {
		if r == 0 || (r < 32 && r != '\t' && r != '\n' && r != '\r') {
			return true
		}
	}
	return false
}

// checkResourceLimits enforces resource limits
func (tl *TaskList) checkResourceLimits(parentID string) error {
	// Count total tasks
	totalTasks := tl.countTotalTasks()
	if totalTasks >= MaxTaskCount {
		return fmt.Errorf("maximum task limit of %d reached", MaxTaskCount)
	}

	// Check hierarchy depth if adding to a parent
	if parentID != "" {
		depth := tl.getTaskDepth(parentID)
		if depth >= MaxHierarchyDepth {
			return fmt.Errorf("maximum hierarchy depth of %d reached", MaxHierarchyDepth)
		}
	}

	return nil
}

// countTotalTasks counts all tasks in the hierarchy
func (tl *TaskList) countTotalTasks() int {
	count := 0
	for _, task := range tl.Tasks {
		count += 1 + countTasksRecursive(&task)
	}
	return count
}

// countTasksRecursive counts tasks recursively
func countTasksRecursive(task *Task) int {
	count := 0
	for _, child := range task.Children {
		count += 1 + countTasksRecursive(&child)
	}
	return count
}

// getTaskDepth calculates the depth of a task in the hierarchy
func (tl *TaskList) getTaskDepth(taskID string) int {
	parts := strings.Split(taskID, ".")
	return len(parts)
}
