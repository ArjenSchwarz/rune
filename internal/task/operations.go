package task

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strconv"
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
// If position is provided, the task will be inserted at that position, otherwise appended
// Returns the ID of the newly created task
func (tl *TaskList) AddTask(parentID, title, position string) (string, error) {
	// Validate input
	if err := validateTaskInput(title); err != nil {
		return "", err
	}

	// Check resource limits
	if err := tl.checkResourceLimits(parentID); err != nil {
		return "", err
	}

	// If position is specified, use position-based insertion
	if position != "" {
		return tl.addTaskAtPosition(parentID, title, position)
	}

	// Default append behavior
	var newTaskID string
	if parentID != "" {
		// Cache parent task lookup to avoid redundant calls
		parent := tl.FindTask(parentID)
		if parent == nil {
			return "", fmt.Errorf("parent task %s not found", parentID)
		}

		newTaskID = fmt.Sprintf("%s.%d", parentID, len(parent.Children)+1)
		newTask := Task{
			ID:       newTaskID,
			Title:    title,
			Status:   Pending,
			ParentID: parentID,
		}
		parent.Children = append(parent.Children, newTask)
	} else {
		newTaskID = fmt.Sprintf("%d", len(tl.Tasks)+1)
		newTask := Task{
			ID:     newTaskID,
			Title:  title,
			Status: Pending,
		}
		tl.Tasks = append(tl.Tasks, newTask)
	}

	tl.Modified = time.Now()
	return newTaskID, nil
}

// addTaskAtPosition inserts a task at the specified position and returns the new task ID
func (tl *TaskList) addTaskAtPosition(parentID, title, position string) (string, error) {
	// Validate position format
	if !isValidID(position) {
		return "", fmt.Errorf("invalid position format: %s", position)
	}

	// Parse position to get insertion index
	parts := strings.Split(position, ".")
	lastPart := parts[len(parts)-1]
	targetIndex, err := strconv.Atoi(lastPart)
	if err != nil {
		return "", fmt.Errorf("invalid position format: %s", position)
	}
	if targetIndex < 1 {
		return "", fmt.Errorf("invalid position: positions must be >= 1")
	}
	targetIndex-- // Convert to 0-based index

	// Cache parent task lookup to avoid redundant FindTask() calls
	var parent *Task
	if parentID != "" {
		parent = tl.FindTask(parentID)
		if parent == nil {
			return "", fmt.Errorf("parent task %s not found", parentID)
		}

		// Bounds check - if beyond end, append
		if targetIndex > len(parent.Children) {
			targetIndex = len(parent.Children)
		}

		// Create new task
		newTask := Task{
			ID:       "temp", // Will be renumbered
			Title:    title,
			Status:   Pending,
			ParentID: parentID,
		}

		// Insert at position
		parent.Children = append(parent.Children[:targetIndex],
			append([]Task{newTask}, parent.Children[targetIndex:]...)...)
	} else {
		// Insert at root level
		if targetIndex > len(tl.Tasks) {
			targetIndex = len(tl.Tasks)
		}

		// Create new task
		newTask := Task{
			ID:     "temp", // Will be renumbered
			Title:  title,
			Status: Pending,
		}

		// Insert at position
		tl.Tasks = append(tl.Tasks[:targetIndex],
			append([]Task{newTask}, tl.Tasks[targetIndex:]...)...)
	}

	// Renumber all tasks to maintain consistency
	tl.renumberTasks()
	tl.Modified = time.Now()

	// After renumbering, use cached parent reference to avoid redundant FindTask() call
	var newTaskID string
	if parent != nil && targetIndex < len(parent.Children) {
		newTaskID = parent.Children[targetIndex].ID
	} else if parentID == "" && targetIndex < len(tl.Tasks) {
		newTaskID = tl.Tasks[targetIndex].ID
	}

	return newTaskID, nil
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

// NewTaskList creates a new TaskList with the specified title and optional FrontMatter
// The FrontMatter parameter is optional to maintain backward compatibility
func NewTaskList(title string, frontMatter ...*FrontMatter) *TaskList {
	tl := &TaskList{
		Title:    title,
		Tasks:    []Task{},
		Modified: time.Now(),
	}

	// If FrontMatter is provided, attach it to the TaskList
	if len(frontMatter) > 0 && frontMatter[0] != nil {
		tl.FrontMatter = frontMatter[0]
	}

	return tl
}

// WriteFile saves the TaskList to a markdown file using atomic writes
// This method includes front matter if present
func (tl *TaskList) WriteFile(filePath string) error {
	// Validate file path
	if err := validateFilePath(filePath); err != nil {
		return err
	}

	// Generate markdown content with front matter if present
	var content []byte
	if tl.FrontMatter != nil && (len(tl.FrontMatter.References) > 0 || len(tl.FrontMatter.Metadata) > 0) {
		// Render markdown without front matter
		markdownContent := RenderMarkdown(tl)
		// Combine with front matter
		fullContent := SerializeWithFrontMatter(tl.FrontMatter, string(markdownContent))
		content = []byte(fullContent)
	} else {
		// No front matter, just render markdown
		content = RenderMarkdown(tl)
	}

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

// AddFrontMatterContent adds or merges front matter content into the TaskList
func (tl *TaskList) AddFrontMatterContent(references []string, metadata map[string]string) error {
	// If both are nil, this is a no-op
	if references == nil && metadata == nil {
		return nil
	}

	// Initialize front matter if it doesn't exist
	if tl.FrontMatter == nil {
		tl.FrontMatter = &FrontMatter{}
	}

	// Merge references
	if references != nil {
		tl.FrontMatter.References = append(tl.FrontMatter.References, references...)
	}

	// Merge metadata with simple replacement strategy
	if metadata != nil {
		if tl.FrontMatter.Metadata == nil {
			tl.FrontMatter.Metadata = make(map[string]string)
		}
		// Simple replacement - last wins
		maps.Copy(tl.FrontMatter.Metadata, metadata)
	}

	return nil
}

// AddTaskToPhase adds a task to a specific phase, creating the phase if it doesn't exist
// This function handles phase-aware task addition by finding the correct position within a phase
func AddTaskToPhase(filepath, parentID, title, phaseName string) (string, error) {
	// Parse file with phase information
	tl, phaseMarkers, err := ParseFileWithPhases(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to parse file with phases: %w", err)
	}

	// Validate input
	if err := validateTaskInput(title); err != nil {
		return "", err
	}

	// Check resource limits
	if err := tl.checkResourceLimits(parentID); err != nil {
		return "", err
	}

	var newTaskID string
	var insertPosition int = -1

	// Find the phase position
	phaseFound := false
	for _, marker := range phaseMarkers {
		if marker.Name == phaseName {
			phaseFound = true
			// Find the position to insert the task
			if marker.AfterTaskID == "" {
				// Phase is at the beginning, insert after phase header
				insertPosition = 0
			} else {
				// Find the task after which this phase starts
				for i, task := range tl.Tasks {
					if task.ID == marker.AfterTaskID {
						insertPosition = i + 1
						break
					}
				}
			}
			break
		}
	}

	if !phaseFound {
		// Phase doesn't exist, create it at the end and add task there
		insertPosition = len(tl.Tasks)
		
		// Add phase marker to the list (will be rendered when file is written)
		afterTaskID := ""
		if len(tl.Tasks) > 0 {
			afterTaskID = tl.Tasks[len(tl.Tasks)-1].ID
		}
		phaseMarkers = append(phaseMarkers, PhaseMarker{
			Name:        phaseName,
			AfterTaskID: afterTaskID,
		})
	} else {
		// Phase exists, find where to insert the task within this phase
		
		// Now find where the next phase starts (this is where current phase ends)
		phaseEndPos := len(tl.Tasks) // Default to end of list
		
		// Look for the next phase marker in document order
		for i, marker := range phaseMarkers {
			// Skip until we find our target phase
			if marker.Name != phaseName {
				continue
			}
			
			// Look for the next phase after this one
			if i+1 < len(phaseMarkers) {
				nextMarker := phaseMarkers[i+1]
				if nextMarker.AfterTaskID == "" {
					phaseEndPos = 0 // Next phase is at the beginning (shouldn't happen in practice)
				} else {
					// Find where the next phase starts
					// The next phase starts after the specified task, so we want to insert before that
					for j, task := range tl.Tasks {
						if task.ID == nextMarker.AfterTaskID {
							phaseEndPos = j + 1 // Insert after this task (which is where next phase starts)
							break
						}
					}
				}
			}
			break
		}
		
		// Insert at the end of the current phase
		insertPosition = phaseEndPos
	}

	// Handle parentID if specified
	if parentID != "" {
		// For subtasks, use existing AddTask logic
		newTaskID, err = tl.AddTask(parentID, title, "")
		if err != nil {
			return "", fmt.Errorf("failed to add subtask: %w", err)
		}
	} else {
		// Insert task at the calculated position
		newTaskID = fmt.Sprintf("%d", insertPosition+1)
		newTask := Task{
			ID:     "temp", // Will be renumbered
			Title:  title,
			Status: Pending,
		}

		// Insert at position
		if insertPosition >= len(tl.Tasks) {
			tl.Tasks = append(tl.Tasks, newTask)
		} else {
			tl.Tasks = append(tl.Tasks[:insertPosition],
				append([]Task{newTask}, tl.Tasks[insertPosition:]...)...)
		}

		// Renumber all tasks
		tl.renumberTasks()
		
		// Get the actual new task ID after renumbering
		if insertPosition < len(tl.Tasks) {
			newTaskID = tl.Tasks[insertPosition].ID
		}
		
		// Update phase markers to account for the insertion
		// We need to update the phase marker that comes AFTER the phase we're adding to
		// This marker should now point to the last task in our target phase
		if phaseFound {
			// Find the next phase marker after our target phase
			for i, marker := range phaseMarkers {
				if marker.Name == phaseName {
					// Look for the next phase marker
					if i+1 < len(phaseMarkers) {
						nextMarker := &phaseMarkers[i+1]
						// The next phase should now start after the newly inserted task
						// Since we inserted at position insertPosition and it got renumbered,
						// the next phase should start after the task at insertPosition
						if insertPosition < len(tl.Tasks) {
							nextMarker.AfterTaskID = tl.Tasks[insertPosition].ID
						}
					}
					break
				}
			}
		}
	}

	// Write the file with phases preserved
	if err := WriteFileWithPhases(tl, phaseMarkers, filepath); err != nil {
		return "", fmt.Errorf("failed to write file with phases: %w", err)
	}

	tl.Modified = time.Now()
	return newTaskID, nil
}

// WriteFileWithPhases saves the TaskList to a markdown file preserving phase markers
func WriteFileWithPhases(tl *TaskList, phaseMarkers []PhaseMarker, filePath string) error {
	// Validate file path
	if err := validateFilePath(filePath); err != nil {
		return err
	}

	// Generate markdown content with phases
	var content []byte
	if tl.FrontMatter != nil && (len(tl.FrontMatter.References) > 0 || len(tl.FrontMatter.Metadata) > 0) {
		// Render markdown with phases but without front matter
		markdownContent := RenderMarkdownWithPhases(tl, phaseMarkers)
		// Combine with front matter
		fullContent := SerializeWithFrontMatter(tl.FrontMatter, string(markdownContent))
		content = []byte(fullContent)
	} else {
		// No front matter, just render markdown with phases
		content = RenderMarkdownWithPhases(tl, phaseMarkers)
	}

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
