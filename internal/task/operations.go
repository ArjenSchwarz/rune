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
	MaxTitleLength    = 500   // Maximum characters per task title
	MaxDetailLength   = 1000  // Maximum characters per detail line
)

// tempTaskID is the placeholder ID assigned to newly inserted tasks before renumbering
const tempTaskID = "temp"

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
			ID:       tempTaskID, // Will be renumbered
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
			ID:     tempTaskID, // Will be renumbered
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

// RemoveTask removes a task from the task list and renumbers remaining tasks.
//
// Deprecated: Use RemoveTaskWithDependents to ensure BlockedBy references
// are cleaned up. RemoveTask only removes the task and renumbers.
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
//
// If the file on disk has phase headers, WriteFile re-reads them and keeps
// them at the same top-level task positions. That is only correct when the
// top-level structure is unchanged since the file was read (status, title or
// metadata edits, appends, subtask changes). A caller that inserts, removes
// or reorders top-level tasks must track the markers itself (for example via
// AddTaskWithPhases or adjustPhaseMarkersForRemoval) and write with
// WriteFileWithPhases instead, or phase boundaries shift onto the wrong tasks.
func (tl *TaskList) WriteFile(filePath string) error {
	// Validate file path
	if err := ValidateFilePath(filePath); err != nil {
		return err
	}

	// Check if the original file has phase markers (if file exists)
	var phaseMarkers []PhaseMarker
	if existingContent, err := os.ReadFile(filePath); err == nil {
		phaseMarkers = phaseMarkersFromContent(existingContent)
	}

	// If phases exist, use phase-aware write
	if len(phaseMarkers) > 0 {
		return WriteFileWithPhases(tl, phaseMarkers, filePath)
	}

	// Generate markdown content with front matter if present
	var content []byte
	if !tl.FrontMatter.IsEmpty() {
		// Render markdown without front matter
		markdownContent := RenderMarkdown(tl)
		// Combine with front matter
		fullContent := SerializeWithFrontMatter(tl.FrontMatter, string(markdownContent))
		content = []byte(fullContent)
	} else {
		// No front matter, just render markdown
		content = RenderMarkdown(tl)
	}

	if err := WriteFileAtomic(filePath, content); err != nil {
		return err
	}

	// Update file path in TaskList
	tl.FilePath = filePath
	return nil
}

// WriteFileAtomic writes content to filePath without ever truncating or
// otherwise modifying the existing file in place: it writes the full content
// to a temporary file (filePath + ".tmp") in the same directory first, then
// replaces the target with a single atomic rename. If the write to the temp
// file fails partway (disk full, quota exceeded, file-size limits, etc.), the
// temp file is removed and filePath is left completely untouched, unlike
// os.WriteFile, which opens the target with O_TRUNC and can leave it
// truncated if the write fails after that.
//
// Callers are responsible for validating filePath (e.g. via ValidateFilePath)
// before calling this.
func WriteFileAtomic(filePath string, content []byte) error {
	// Get original file permissions if file exists, otherwise use default 0644
	perm := os.FileMode(0644)
	if fileInfo, err := os.Stat(filePath); err == nil {
		perm = fileInfo.Mode().Perm()
	}

	// Write to temp file first for atomic operation
	tmpFile := filePath + ".tmp"
	if err := os.WriteFile(tmpFile, content, perm); err != nil {
		// Clean up a partially written temp file on failure (e.g. disk full,
		// quota exceeded, file-size limits stopped the write partway through).
		// Best-effort: the write error is what the caller needs to see, so a
		// failure to remove the leftover temp file is deliberately ignored.
		_ = os.Remove(tmpFile)
		return fmt.Errorf("writing temp file: %w", err)
	}

	// os.WriteFile creates the temp file through open(2), which filters the
	// mode through the process umask, so group- and other-writable bits are
	// dropped (0664 becomes 0644 under the common umask 022). Writing in place
	// over an existing file did not have this problem, because open(2) ignores
	// the mode argument when the file already exists. Set the mode explicitly
	// so the renamed file keeps the permissions the original had.
	if err := os.Chmod(tmpFile, perm); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("setting temp file permissions: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, filePath); err != nil {
		// Clean up temp file on failure. Best-effort for the same reason as
		// above: the rename error is the one worth reporting.
		_ = os.Remove(tmpFile)
		return fmt.Errorf("atomic rename: %w", err)
	}

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

	// Lexical containment check first (fast path)
	if !strings.HasPrefix(absPath, workDirAbs+string(filepath.Separator)) && absPath != workDirAbs {
		return fmt.Errorf("path traversal attempt detected")
	}

	// Resolve symlinks to prevent escaping the working directory via symlinks.
	// EvalSymlinks requires the path (or its parent) to exist, so we walk up
	// until we find an existing ancestor and resolve from there.
	realWorkDir, err := filepath.EvalSymlinks(workDirAbs)
	if err != nil {
		return fmt.Errorf("resolving working directory symlinks: %w", err)
	}

	realPath, err := resolveExistingPrefix(absPath)
	if err != nil {
		return fmt.Errorf("resolving file path symlinks: %w", err)
	}

	// Re-check containment after symlink resolution
	if !strings.HasPrefix(realPath, realWorkDir+string(filepath.Separator)) && realPath != realWorkDir {
		return fmt.Errorf("path traversal attempt detected: symlink resolves outside working directory")
	}

	return nil
}

// resolveExistingPrefix resolves symlinks for the longest existing prefix of
// path, then appends the remaining unresolved tail. This handles the case
// where the file itself does not yet exist (e.g. create operations) but an
// ancestor directory contains a symlink.
func resolveExistingPrefix(absPath string) (string, error) {
	// Try resolving the full path first (common case: file exists)
	resolved, err := filepath.EvalSymlinks(absPath)
	if err == nil {
		return resolved, nil
	}

	// Walk up to find the deepest existing ancestor
	dir := filepath.Dir(absPath)
	tail := filepath.Base(absPath)
	cursor := absPath

	for dir != cursor {
		resolved, err = filepath.EvalSymlinks(dir)
		if err == nil {
			return filepath.Join(resolved, tail), nil
		}
		tail = filepath.Join(filepath.Base(dir), tail)
		cursor = dir
		dir = filepath.Dir(dir)
	}

	// Filesystem root could not be resolved; return as-is (safe: will fail containment check)
	return cursor, nil
}

// validateTaskInput sanitizes and validates task title input. It is the
// single choke point for every operation that sets a task's title (AddTask,
// AddTaskToPhase, AddTaskWithOptions, UpdateTask, UpdateTaskWithOptions, and
// the batch equivalents), as well as TaskList titles via ValidateTaskListTitle.
//
// An empty title is rejected because it renders as a bullet with no title
// text (e.g. "- [ ] 1. ") or, for a TaskList title, a bare "# " heading —
// neither of which the parser recognises, producing a file Rune cannot read
// back. A whitespace-only title is rejected for consistency rather than
// necessity: "- [ ] 1.    " does parse and round-trips as a task titled with
// blank spaces, though the TaskList equivalent "#    " does not. See T-1561
// (task titles) and T-1500 (TaskList titles).
func validateTaskInput(input string) error {
	if strings.TrimSpace(input) == "" {
		return fmt.Errorf("task title cannot be empty")
	}

	// Check for null bytes and control characters
	if containsNullByte(input) {
		return fmt.Errorf("input contains null bytes or control characters")
	}

	if len(input) > MaxTitleLength {
		return fmt.Errorf("task title exceeds %d characters", MaxTitleLength)
	}

	return nil
}

// ValidateTaskListTitle validates a TaskList title before it is rendered into
// the markdown H1 heading. It applies the same rules as task titles
// (validateTaskInput): non-empty (and not whitespace-only), free of null
// bytes and control characters (including \n and \r, which would split the
// single-line heading across multiple lines), and within the length limit.
func ValidateTaskListTitle(title string) error {
	return validateTaskInput(title)
}

// validateDetails validates task details.
//
// Empty and whitespace-only details are rejected. renderTask writes every
// detail as a "- <content>" bullet, so a detail with no content renders as a
// bare "  - " line, which the parser refuses to read back (T-2041) — rune
// would be able to write a file it cannot open. Rejecting here rather than
// skipping the detail at render time keeps the round-trip honest: silently
// dropping input is exactly the behaviour T-2041 removed from the parser, and
// a detail without content carries no meaning to drop.
//
// Details that the parser would read back as metadata are rejected for the
// same reason: they would not survive the round trip as details.
func validateDetails(details []string) error {
	for i, detail := range details {
		if strings.TrimSpace(detail) == "" {
			return fmt.Errorf("detail %d is empty: details must have content", i+1)
		}
		if containsNullByte(detail) {
			return fmt.Errorf("detail %d contains null bytes or control characters", i+1)
		}
		if len(detail) > MaxDetailLength {
			return fmt.Errorf("detail %d exceeds maximum length of %d characters", i+1, MaxDetailLength)
		}
		// A detail that looks like a metadata line ("Stream: 3", "Owner: x",
		// "References: a", ...) is rendered as a plain "- " bullet but parsed
		// back as that metadata, silently turning the detail into data.
		if field := detailMetadataField(detail); field != "" {
			return fmt.Errorf("detail %d would be read back as %s metadata, not as a detail: %q", i+1, field, detail)
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

// containsNullByte checks for null bytes and dangerous control characters.
// Rejects all control characters below U+0020 except horizontal tab (\t).
// Notably, \n and \r are rejected because they break markdown line structure
// when embedded in titles, details, or references.
func containsNullByte(s string) bool {
	for _, r := range s {
		if r == 0 || (r < 32 && r != '\t') {
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

// AddTaskToPhase adds a task to a specific phase of the file at filepath,
// creating the phase if it doesn't exist, and writes the file. A subtask
// (parentID set) stays under its parent and never creates a phase. This is a
// file-level convenience wrapper around TaskList.AddTaskWithPhases.
func AddTaskToPhase(filepath, parentID, title, phaseName string) (string, error) {
	tl, phaseMarkers, err := ParseFileWithPhases(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to parse file '%s' with phases: %w", filepath, err)
	}

	// An empty phase name would make AddTaskWithPhases fall back to a plain
	// add, so reject it here: this API always targets a named phase.
	if _, err := NormalizePhaseName(phaseName); err != nil {
		return "", err
	}

	newTaskID, err := tl.AddTaskWithPhases(Operation{
		Type:   addOperation,
		Parent: parentID,
		Title:  title,
		Phase:  phaseName,
	}, &phaseMarkers)
	if err != nil {
		return "", err
	}

	if err := WriteFileWithPhases(tl, phaseMarkers, filepath); err != nil {
		return "", fmt.Errorf("failed to write file '%s' with phases: %w", filepath, err)
	}
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
	if !tl.FrontMatter.IsEmpty() {
		// Render markdown with phases but without front matter
		markdownContent := RenderMarkdownWithPhases(tl, phaseMarkers, nil)
		// Combine with front matter
		fullContent := SerializeWithFrontMatter(tl.FrontMatter, string(markdownContent))
		content = []byte(fullContent)
	} else {
		// No front matter, just render markdown with phases
		content = RenderMarkdownWithPhases(tl, phaseMarkers, nil)
	}

	if err := WriteFileAtomic(filePath, content); err != nil {
		return err
	}

	// Update file path in TaskList
	tl.FilePath = filePath
	return nil
}

// Phase-aware operation wrappers that preserve phase markers

// RemoveTaskWithPhases removes a task while preserving phase structure
func (tl *TaskList) RemoveTaskWithPhases(taskID string, originalContent []byte) error {
	// Extract phase markers from the original content
	phaseMarkers := phaseMarkersFromContent(originalContent)

	// If no phases, just use regular operations
	if len(phaseMarkers) == 0 {
		if _, err := tl.RemoveTaskWithDependents(taskID); err != nil {
			return err
		}
		return tl.WriteFile(tl.FilePath)
	}

	// Remove the task (cleans up stale BlockedBy references)
	if _, err := tl.RemoveTaskWithDependents(taskID); err != nil {
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
	phaseMarkers := phaseMarkersFromContent(originalContent)

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
	phaseMarkers := phaseMarkersFromContent(originalContent)

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

// adjustPhaseMarkersForInsertion updates phase markers after a top-level task is inserted.
// insertPosition is the 0-based index where the new task was inserted. All markers whose
// AfterTaskID references a task at or after this position (1-based: insertPosition+1 or
// higher) are incremented by one to account for the shift in task numbering.
func adjustPhaseMarkersForInsertion(insertPosition int, phaseMarkers *[]PhaseMarker) {
	for i := range *phaseMarkers {
		if (*phaseMarkers)[i].AfterTaskID == "" {
			continue
		}
		afterTaskNum := getTaskNumber((*phaseMarkers)[i].AfterTaskID)
		// Tasks at position insertPosition or later (1-based: insertPosition+1) have
		// shifted up by one due to the insertion.
		if afterTaskNum >= insertPosition+1 {
			(*phaseMarkers)[i].AfterTaskID = fmt.Sprintf("%d", afterTaskNum+1)
		}
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

		newTask.ID = tempTaskID
		newTask.ParentID = parentID
		parent.Children = append(parent.Children[:targetIndex],
			append([]Task{*newTask}, parent.Children[targetIndex:]...)...)
	} else {
		if targetIndex > len(tl.Tasks) {
			targetIndex = len(tl.Tasks)
		}

		newTask.ID = tempTaskID
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

			// Auto-assign stable ID to the task being updated if it doesn't have one.
			// A task with BlockedBy must have a StableID so that it appears in the
			// dependency index and can be referenced by other tasks.
			if task.StableID == "" {
				existingIDs := tl.collectStableIDs()
				idGen := NewStableIDGenerator(existingIDs)
				newID, genErr := idGen.Generate()
				if genErr != nil {
					return fmt.Errorf("generating stable ID for task %s: %w", taskID, genErr)
				}
				task.StableID = newID
			}

			// Check for cycles
			index := BuildDependencyIndex(tl.Tasks)
			for _, toStableID := range stableIDs {
				if hasCycle, path := index.DetectCycle(task.StableID, toStableID); hasCycle {
					return &CircularDependencyError{Path: path}
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

	// Collect the stable IDs of the task and all its descendants: the whole
	// subtree is deleted, so references to any of them would be left dangling.
	removedIDs := make(map[string]bool)
	var collect func(t *Task)
	collect = func(t *Task) {
		if t.StableID != "" {
			removedIDs[t.StableID] = true
		}
		for i := range t.Children {
			collect(&t.Children[i])
		}
	}
	collect(task)

	if !tl.removeTaskRecursive(&tl.Tasks, taskID, "") {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	// Clean up blocked-by references in the surviving tasks. The tree walk is
	// cheap and catches all dependents regardless of whether they have their
	// own StableID.
	var warnings []string
	if count := tl.removeFromBlockedByLists(removedIDs); count > 0 {
		warnings = append(warnings,
			fmt.Sprintf("removed dependency references from %d task(s)", count))
	}

	tl.RenumberTasks()
	tl.Modified = time.Now()
	return warnings, nil
}

// removeFromBlockedByLists removes the given stable IDs from all BlockedBy
// lists. Returns the number of tasks that had at least one reference removed.
func (tl *TaskList) removeFromBlockedByLists(stableIDs map[string]bool) int {
	if len(stableIDs) == 0 {
		return 0
	}
	count := 0
	var removeRecursive func(tasks []Task)
	removeRecursive = func(tasks []Task) {
		for i := range tasks {
			task := &tasks[i]
			// Remove the stable IDs from BlockedBy if present
			newBlockedBy := make([]string, 0, len(task.BlockedBy))
			for _, blockerID := range task.BlockedBy {
				if !stableIDs[blockerID] {
					newBlockedBy = append(newBlockedBy, blockerID)
				}
			}
			if len(newBlockedBy) < len(task.BlockedBy) {
				count++
			}
			task.BlockedBy = newBlockedBy

			// Process children
			if len(task.Children) > 0 {
				removeRecursive(task.Children)
			}
		}
	}

	removeRecursive(tl.Tasks)
	return count
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
// If a task doesn't have a stable ID, one is auto-assigned.
// Returns an error if any task doesn't exist.
func (tl *TaskList) resolveToStableIDs(hierarchicalIDs []string) ([]string, error) {
	stableIDs := make([]string, 0, len(hierarchicalIDs))

	// Created on first use and reused, rather than re-collecting every stable
	// ID in the list for each task that needs one
	var idGen *StableIDGenerator
	for _, hid := range hierarchicalIDs {
		task := tl.FindTask(hid)
		if task == nil {
			return nil, fmt.Errorf("task %s not found", hid)
		}
		if task.StableID == "" {
			if idGen == nil {
				idGen = NewStableIDGenerator(tl.collectStableIDs())
			}
			newID, err := idGen.Generate()
			if err != nil {
				return nil, fmt.Errorf("generating stable ID for task %s: %w", hid, err)
			}
			task.StableID = newID
		}
		stableIDs = append(stableIDs, task.StableID)
	}

	return stableIDs, nil
}

// ValidateOwner validates an owner string using the same rules
// AddTaskWithOptions and UpdateTaskWithOptions already apply via the
// unexported validateOwner: control characters (including newlines) are
// rejected, everything else — including spaces — is allowed, and an empty
// owner is valid (it means "unclaimed"). Exported so command entry points
// that assign Owner directly, without going through AddTaskWithOptions or
// UpdateTaskWithOptions (such as `next --claim`), can validate the value
// before writing it into task metadata rather than duplicating the rules
// or skipping validation entirely (T-1565).
func ValidateOwner(owner string) error {
	return validateOwner(owner)
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
