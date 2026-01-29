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

// AddOptions contains optional parameters for adding a task with extended features
type AddOptions struct {
	Position  string   // Position to insert at (e.g., "1", "2.1")
	Phase     string   // Phase name (for phase-aware insertion)
	Stream    int      // Stream assignment (positive integer, 0 = not set)
	BlockedBy []string // Hierarchical IDs of blocking tasks
	Owner     string   // Agent identifier
}

// UpdateOptions contains optional parameters for updating a task with extended features
type UpdateOptions struct {
	Title        *string  // New title (nil = no change)
	Details      []string // New details (nil = no change)
	References   []string // New references (nil = no change)
	Requirements []string // New requirements (nil = no change)
	Stream       *int     // Stream assignment (nil = no change)
	BlockedBy    []string // Hierarchical IDs of blockers (nil = no change, empty = clear)
	Owner        *string  // Owner string (nil = no change)
	Release      bool     // Clear owner if true
}

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
	if !IsValidID(position) {
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
	tl.RenumberTasks()
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
		tl.RenumberTasks()
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

// RenumberTasks recalculates all task IDs using sequential hierarchical numbering
func (tl *TaskList) RenumberTasks() {
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

// UpdateTask modifies the title, details, references, and requirements of a task
// If requirements is nil, the requirements are not modified
// If requirements is an empty slice, the requirements are cleared
func (tl *TaskList) UpdateTask(taskID, title string, details, refs, requirements []string) error {
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
	if requirements != nil {
		if err := validateRequirements(requirements); err != nil {
			return err
		}
		task.Requirements = requirements
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
	if err := ValidateFilePath(filePath); err != nil {
		return err
	}

	// Check if the original file has phase markers (if file exists)
	var phaseMarkers []PhaseMarker
	if existingContent, err := os.ReadFile(filePath); err == nil {
		// File exists, check for phases
		lines := strings.Split(string(existingContent), "\n")
		phaseMarkers = ExtractPhaseMarkers(lines)
	}

	// If phases exist, use phase-aware write
	if len(phaseMarkers) > 0 {
		return WriteFileWithPhases(tl, phaseMarkers, filePath)
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

	// Get original file permissions if file exists, otherwise use default 0644
	perm := os.FileMode(0644)
	if fileInfo, err := os.Stat(filePath); err == nil {
		perm = fileInfo.Mode().Perm()
	}

	// Write to temp file first for atomic operation
	tmpFile := filePath + ".tmp"
	if err := os.WriteFile(tmpFile, content, perm); err != nil {
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

// ValidateFilePath ensures the file path is safe and valid
func ValidateFilePath(path string) error {
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

// validateRequirements validates task requirements
func validateRequirements(requirements []string) error {
	for i, req := range requirements {
		if containsNullByte(req) {
			return fmt.Errorf("requirement %d contains null bytes or control characters", i+1)
		}
		if !IsValidID(req) {
			return fmt.Errorf("requirement %d has invalid format: %s (must match pattern ^\\d+(\\.\\d+)*$)", i+1, req)
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
	totalTasks := tl.CountTotalTasks()
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

// CountTotalTasks counts all tasks in the hierarchy
func (tl *TaskList) CountTotalTasks() int {
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
		return "", fmt.Errorf("failed to parse file '%s' with phases: %w", filepath, err)
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
	var insertPosition int

	// Find the phase position
	phaseFound := false
	for _, marker := range phaseMarkers {
		if marker.Name == phaseName {
			phaseFound = true
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
			return "", fmt.Errorf("failed to add subtask to '%s': %w", filepath, err)
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
		tl.RenumberTasks()

		// Get the actual new task ID after renumbering
		if insertPosition < len(tl.Tasks) {
			newTaskID = tl.Tasks[insertPosition].ID
		}

		// Update phase markers to account for the insertion
		// IMPORTANT: Since we ALWAYS insert at the END of the phase (insertPosition = phaseEndPos),
		// the newly inserted task becomes the last task in the current phase. Therefore, the next
		// phase marker must be updated to point to this newly inserted task's ID.
		// This maintains the invariant that phase markers always point to the last task in the
		// preceding phase.
		if phaseFound {
			// Find the next phase marker after our target phase
			for i, marker := range phaseMarkers {
				if marker.Name == phaseName {
					// Look for the next phase marker
					if i+1 < len(phaseMarkers) {
						nextMarker := &phaseMarkers[i+1]
						// Update the next phase to start after the newly inserted task
						// (which is now the last task in the current phase)
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
		return "", fmt.Errorf("failed to write file '%s' with phases: %w", filepath, err)
	}

	tl.Modified = time.Now()
	return newTaskID, nil
}

// WriteFileWithPhases saves the TaskList to a markdown file preserving phase markers
func WriteFileWithPhases(tl *TaskList, phaseMarkers []PhaseMarker, filePath string) error {
	// Validate file path
	if err := ValidateFilePath(filePath); err != nil {
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

	// Get original file permissions if file exists, otherwise use default 0644
	perm := os.FileMode(0644)
	if fileInfo, err := os.Stat(filePath); err == nil {
		perm = fileInfo.Mode().Perm()
	}

	// Write to temp file first for atomic operation
	tmpFile := filePath + ".tmp"
	if err := os.WriteFile(tmpFile, content, perm); err != nil {
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

// Phase-aware operation wrappers that preserve phase markers

// RemoveTaskWithPhases removes a task while preserving phase structure
func (tl *TaskList) RemoveTaskWithPhases(taskID string, originalContent []byte) error {
	// Extract phase markers from the original content
	lines := strings.Split(string(originalContent), "\n")
	phaseMarkers := ExtractPhaseMarkers(lines)

	// If no phases, just use regular operations
	if len(phaseMarkers) == 0 {
		if err := tl.RemoveTask(taskID); err != nil {
			return err
		}
		return tl.WriteFile(tl.FilePath)
	}

	// Remove the task
	if err := tl.RemoveTask(taskID); err != nil {
		return err
	}

	// Adjust phase markers for the removed task
	adjustPhaseMarkersForRemoval(taskID, &phaseMarkers)

	// Write with phases preserved
	return WriteFileWithPhases(tl, phaseMarkers, tl.FilePath)
}

// UpdateTaskWithPhases updates a task while preserving phase structure
func (tl *TaskList) UpdateTaskWithPhases(taskID, title string, details, refs []string, originalContent []byte) error {
	// Extract phase markers from the original content
	lines := strings.Split(string(originalContent), "\n")
	phaseMarkers := ExtractPhaseMarkers(lines)

	// Update the task
	if err := tl.UpdateTask(taskID, title, details, refs, nil); err != nil {
		return err
	}

	// If phases were present, write with phases preserved
	if len(phaseMarkers) > 0 {
		return WriteFileWithPhases(tl, phaseMarkers, tl.FilePath)
	}

	// No phases, use regular write
	return tl.WriteFile(tl.FilePath)
}

// UpdateStatusWithPhases updates a task status while preserving phase structure
func (tl *TaskList) UpdateStatusWithPhases(taskID string, status Status, originalContent []byte) error {
	// Extract phase markers from the original content
	lines := strings.Split(string(originalContent), "\n")
	phaseMarkers := ExtractPhaseMarkers(lines)

	// Update the status
	if err := tl.UpdateStatus(taskID, status); err != nil {
		return err
	}

	// If phases were present, write with phases preserved
	if len(phaseMarkers) > 0 {
		return WriteFileWithPhases(tl, phaseMarkers, tl.FilePath)
	}

	// No phases, use regular write
	return tl.WriteFile(tl.FilePath)
}

// Helper function to extract the main task number from a task ID
func getTaskNumber(taskID string) int {
	parts := strings.Split(taskID, ".")
	if len(parts) == 0 {
		return -1
	}

	num, err := strconv.Atoi(parts[0])
	if err != nil {
		return -1
	}
	return num
}

// adjustPhaseMarkersForRemoval updates phase markers after a top-level task is removed.
// Only top-level task removal (IDs without ".") affects phase markers since subtask
// removal does not change the numbering of top-level tasks.
func adjustPhaseMarkersForRemoval(taskID string, phaseMarkers *[]PhaseMarker) {
	// Only adjust for top-level tasks
	if strings.Contains(taskID, ".") {
		return
	}

	removedTaskNum := getTaskNumber(taskID)
	if removedTaskNum == -1 {
		return
	}

	for i := range *phaseMarkers {
		if (*phaseMarkers)[i].AfterTaskID == "" {
			continue
		}
		afterTaskNum := getTaskNumber((*phaseMarkers)[i].AfterTaskID)
		if afterTaskNum == removedTaskNum {
			// This phase was positioned after the removed task
			// Move it to be positioned after the previous task
			if removedTaskNum > 1 {
				(*phaseMarkers)[i].AfterTaskID = fmt.Sprintf("%d", removedTaskNum-1)
			} else {
				// Removing task 1, so phase goes to beginning
				(*phaseMarkers)[i].AfterTaskID = ""
			}
		} else if afterTaskNum > removedTaskNum {
			// This phase marker comes after the removed task, so decrement the ID
			// to account for the fact that all subsequent tasks are renumbered
			(*phaseMarkers)[i].AfterTaskID = fmt.Sprintf("%d", afterTaskNum-1)
		}
		// If afterTaskNum < removedTaskNum, no adjustment needed
	}
}

// ============================================================================
// Extended operations with dependencies and streams support
// ============================================================================

// AddTaskWithOptions adds a new task with extended options (stream, blocked-by, owner).
// Returns the hierarchical ID of the newly created task.
func (tl *TaskList) AddTaskWithOptions(parentID, title string, opts AddOptions) (string, error) {
	// Validate input
	if err := validateTaskInput(title); err != nil {
		return "", err
	}

	// Check resource limits
	if err := tl.checkResourceLimits(parentID); err != nil {
		return "", err
	}

	// Validate stream (negative is invalid)
	if opts.Stream < 0 {
		return "", ErrInvalidStream
	}

	// Validate owner
	if opts.Owner != "" {
		if err := validateOwner(opts.Owner); err != nil {
			return "", err
		}
	}

	// Resolve blocked-by references to stable IDs
	var blockedByStableIDs []string
	if len(opts.BlockedBy) > 0 {
		var err error
		blockedByStableIDs, err = tl.resolveToStableIDs(opts.BlockedBy)
		if err != nil {
			return "", err
		}
	}

	// Generate stable ID for the new task
	existingIDs := tl.collectStableIDs()
	idGen := NewStableIDGenerator(existingIDs)
	stableID, err := idGen.Generate()
	if err != nil {
		return "", fmt.Errorf("generating stable ID: %w", err)
	}

	// Create the task
	newTask := Task{
		Title:     title,
		Status:    Pending,
		StableID:  stableID,
		Stream:    opts.Stream,
		BlockedBy: blockedByStableIDs,
		Owner:     opts.Owner,
	}

	// Add task to appropriate location
	var newTaskID string
	switch {
	case opts.Position != "":
		var posErr error
		newTaskID, posErr = tl.addTaskWithOptionsAtPosition(parentID, &newTask, opts.Position)
		if posErr != nil {
			return "", posErr
		}
	case parentID != "":
		parent := tl.FindTask(parentID)
		if parent == nil {
			return "", fmt.Errorf("parent task %s not found", parentID)
		}
		newTaskID = fmt.Sprintf("%s.%d", parentID, len(parent.Children)+1)
		newTask.ID = newTaskID
		newTask.ParentID = parentID
		parent.Children = append(parent.Children, newTask)
	default:
		newTaskID = fmt.Sprintf("%d", len(tl.Tasks)+1)
		newTask.ID = newTaskID
		tl.Tasks = append(tl.Tasks, newTask)
	}

	tl.Modified = time.Now()
	return newTaskID, nil
}

// addTaskWithOptionsAtPosition inserts a task with options at a specific position
func (tl *TaskList) addTaskWithOptionsAtPosition(parentID string, newTask *Task, position string) (string, error) {
	// Validate position format
	if !IsValidID(position) {
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

	var parent *Task
	if parentID != "" {
		parent = tl.FindTask(parentID)
		if parent == nil {
			return "", fmt.Errorf("parent task %s not found", parentID)
		}

		if targetIndex > len(parent.Children) {
			targetIndex = len(parent.Children)
		}

		newTask.ID = "temp"
		newTask.ParentID = parentID
		parent.Children = append(parent.Children[:targetIndex],
			append([]Task{*newTask}, parent.Children[targetIndex:]...)...)
	} else {
		if targetIndex > len(tl.Tasks) {
			targetIndex = len(tl.Tasks)
		}

		newTask.ID = "temp"
		tl.Tasks = append(tl.Tasks[:targetIndex],
			append([]Task{*newTask}, tl.Tasks[targetIndex:]...)...)
	}

	// Renumber all tasks
	tl.RenumberTasks()

	// Find the new task ID after renumbering
	var newTaskID string
	if parent != nil && targetIndex < len(parent.Children) {
		newTaskID = parent.Children[targetIndex].ID
	} else if parentID == "" && targetIndex < len(tl.Tasks) {
		newTaskID = tl.Tasks[targetIndex].ID
	}

	return newTaskID, nil
}

// UpdateTaskWithOptions updates a task with extended options
func (tl *TaskList) UpdateTaskWithOptions(taskID string, opts UpdateOptions) error {
	task := tl.FindTask(taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Update title if provided
	if opts.Title != nil && *opts.Title != "" {
		if err := validateTaskInput(*opts.Title); err != nil {
			return err
		}
		task.Title = *opts.Title
	}

	// Update details if provided
	if opts.Details != nil {
		if err := validateDetails(opts.Details); err != nil {
			return err
		}
		task.Details = opts.Details
	}

	// Update references if provided
	if opts.References != nil {
		if err := validateReferences(opts.References); err != nil {
			return err
		}
		task.References = opts.References
	}

	// Update requirements if provided
	if opts.Requirements != nil {
		if err := validateRequirements(opts.Requirements); err != nil {
			return err
		}
		task.Requirements = opts.Requirements
	}

	// Update stream if provided
	if opts.Stream != nil {
		if *opts.Stream < 0 {
			return ErrInvalidStream
		}
		task.Stream = *opts.Stream
	}

	// Update blocked-by if provided
	if opts.BlockedBy != nil {
		if len(opts.BlockedBy) > 0 {
			// Resolve hierarchical IDs to stable IDs
			stableIDs, err := tl.resolveToStableIDs(opts.BlockedBy)
			if err != nil {
				return err
			}

			// Check for cycles
			if task.StableID != "" {
				index := BuildDependencyIndex(tl.Tasks)
				for _, toStableID := range stableIDs {
					if hasCycle, path := index.DetectCycle(task.StableID, toStableID); hasCycle {
						return &CircularDependencyError{Path: path}
					}
				}
			}

			task.BlockedBy = stableIDs
		} else {
			// Empty slice means clear blocked-by
			task.BlockedBy = []string{}
		}
	}

	// Update owner if provided
	if opts.Owner != nil {
		if err := validateOwner(*opts.Owner); err != nil {
			return err
		}
		task.Owner = *opts.Owner
	}

	// Handle release flag (clears owner)
	if opts.Release {
		task.Owner = ""
	}

	tl.Modified = time.Now()
	return nil
}

// RemoveTaskWithDependents removes a task and cleans up dependent references.
// Returns warnings about any dependents that had references cleaned up.
func (tl *TaskList) RemoveTaskWithDependents(taskID string) ([]string, error) {
	task := tl.FindTask(taskID)
	if task == nil {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	var warnings []string

	// If task has a stable ID, check for dependents
	if task.StableID != "" {
		index := BuildDependencyIndex(tl.Tasks)
		dependents := index.GetDependents(task.StableID)

		if len(dependents) > 0 {
			// Remove from all blocked-by lists
			tl.removeFromBlockedByLists(task.StableID)
			warnings = append(warnings,
				fmt.Sprintf("removed dependency references from %d task(s)", len(dependents)))
		}
	}

	// Remove the task using existing logic
	if removed := tl.removeTaskRecursive(&tl.Tasks, taskID, ""); removed {
		tl.RenumberTasks()
		tl.Modified = time.Now()
		return warnings, nil
	}

	return nil, fmt.Errorf("task %s not found", taskID)
}

// removeFromBlockedByLists removes a stable ID from all BlockedBy lists
func (tl *TaskList) removeFromBlockedByLists(stableID string) {
	var removeRecursive func(tasks []Task)
	removeRecursive = func(tasks []Task) {
		for i := range tasks {
			task := &tasks[i]
			// Remove the stable ID from BlockedBy if present
			newBlockedBy := make([]string, 0, len(task.BlockedBy))
			for _, blockerID := range task.BlockedBy {
				if blockerID != stableID {
					newBlockedBy = append(newBlockedBy, blockerID)
				}
			}
			task.BlockedBy = newBlockedBy

			// Process children
			if len(task.Children) > 0 {
				removeRecursive(task.Children)
			}
		}
	}

	removeRecursive(tl.Tasks)
}

// collectStableIDs collects all stable IDs from the task list
func (tl *TaskList) collectStableIDs() []string {
	var ids []string
	var collectRecursive func(tasks []Task)
	collectRecursive = func(tasks []Task) {
		for _, task := range tasks {
			if task.StableID != "" {
				ids = append(ids, task.StableID)
			}
			if len(task.Children) > 0 {
				collectRecursive(task.Children)
			}
		}
	}

	collectRecursive(tl.Tasks)
	return ids
}

// resolveToStableIDs converts hierarchical IDs to stable IDs.
// Returns an error if any task doesn't exist or doesn't have a stable ID.
func (tl *TaskList) resolveToStableIDs(hierarchicalIDs []string) ([]string, error) {
	stableIDs := make([]string, 0, len(hierarchicalIDs))

	for _, hid := range hierarchicalIDs {
		task := tl.FindTask(hid)
		if task == nil {
			return nil, fmt.Errorf("task %s not found", hid)
		}
		if task.StableID == "" {
			return nil, ErrNoStableID
		}
		stableIDs = append(stableIDs, task.StableID)
	}

	return stableIDs, nil
}

// validateOwner checks if an owner string contains valid characters.
// Owner strings must not contain newlines or other control characters.
func validateOwner(owner string) error {
	if owner == "" {
		return nil // Empty owner is valid
	}

	for _, r := range owner {
		// Reject newlines and control characters (except spaces)
		if r == '\n' || r == '\r' || r == '\t' || r == 0 || (r < 32 && r != ' ') {
			return ErrInvalidOwner
		}
	}

	return nil
}
