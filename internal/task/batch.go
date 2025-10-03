package task

import (
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"
)

const (
	updateOperation = "update"
	addOperation    = "add"
	removeOperation = "remove"
)

// StatusPtr returns a pointer to the given status value for use in Operation structs
func StatusPtr(s Status) *Status {
	return &s
}

// hasStatusField returns true if the operation has a status field provided
func hasStatusField(op Operation) bool {
	return op.Status != nil
}

// validateTaskIDFormat validates that a string follows the task ID format
func validateTaskIDFormat(id string) bool {
	return isValidID(id)
}

// sortOperationsForPositionInsertions sorts operations to process position insertions in reverse order
// This ensures that position references remain valid to the original pre-batch state
func sortOperationsForPositionInsertions(ops []Operation) []Operation {
	// Separate position insertions from other operations
	var positionInsertions []Operation
	var otherOps []Operation

	for _, op := range ops {
		if strings.ToLower(op.Type) == addOperation && op.Position != "" {
			positionInsertions = append(positionInsertions, op)
		} else {
			otherOps = append(otherOps, op)
		}
	}

	// Sort position insertions in reverse order (higher positions first)
	sort.Slice(positionInsertions, func(i, j int) bool {
		return comparePositions(positionInsertions[i].Position, positionInsertions[j].Position) > 0
	})

	// Combine: position insertions first (in reverse order), then other operations
	result := make([]Operation, 0, len(ops))
	result = append(result, positionInsertions...)
	result = append(result, otherOps...)

	return result
}

// comparePositions compares two position strings numerically
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func comparePositions(a, b string) int {
	// Split positions into parts
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	// Compare part by part
	maxLen := max(len(partsB), len(partsA))

	for i := range maxLen {
		var numA, numB int

		if i < len(partsA) {
			numA, _ = strconv.Atoi(partsA[i])
		}
		if i < len(partsB) {
			numB, _ = strconv.Atoi(partsB[i])
		}

		if numA < numB {
			return -1
		} else if numA > numB {
			return 1
		}
	}

	return 0
}

// Operation represents a single operation in a batch
type Operation struct {
	Type       string   `json:"type"`
	ID         string   `json:"id,omitempty"`
	Parent     string   `json:"parent,omitempty"`
	Title      string   `json:"title,omitempty"`
	Status     *Status  `json:"status,omitempty"`
	Details    []string `json:"details,omitempty"`
	References []string `json:"references,omitempty"`
	Position   string   `json:"position,omitempty"`
	Phase      string   `json:"phase,omitempty"`
}

// BatchRequest represents a request for multiple operations
type BatchRequest struct {
	File       string      `json:"file"`
	Operations []Operation `json:"operations"`
	DryRun     bool        `json:"dry_run"`
}

// BatchResponse represents the response from a batch operation
type BatchResponse struct {
	Success       bool     `json:"success"`
	Applied       int      `json:"applied"`
	Errors        []string `json:"errors,omitempty"`
	Preview       string   `json:"preview,omitempty"`
	AutoCompleted []string `json:"auto_completed,omitempty"`
}

// ExecuteBatch validates and executes a batch of operations atomically
func (tl *TaskList) ExecuteBatch(ops []Operation, dryRun bool) (*BatchResponse, error) {
	response := &BatchResponse{
		Success:       true,
		Applied:       0,
		Errors:        []string{},
		AutoCompleted: []string{},
	}

	// Sort operations to process position insertions in reverse order
	// This ensures position references use original pre-batch state
	sortedOps := sortOperationsForPositionInsertions(ops)

	// Create a deep copy to test all operations first (per Decision #12)
	testList, err := tl.deepCopy()
	if err != nil {
		return nil, fmt.Errorf("creating test copy: %w", err)
	}

	// Track auto-completed tasks for test copy
	testAutoCompleted := make(map[string]bool)

	// Validate and apply operations to test copy sequentially
	for i, op := range sortedOps {
		if err := validateOperation(testList, op); err != nil {
			response.Success = false
			response.Errors = append(response.Errors, fmt.Sprintf("operation %d: %v", i+1, err))
			return response, nil
		}

		// Apply operation to test copy for subsequent validations
		if err := applyOperationWithAutoComplete(testList, op, testAutoCompleted); err != nil {
			response.Success = false
			response.Errors = append(response.Errors, fmt.Sprintf("operation %d: %v", i+1, err))
			return response, nil
		}
	}

	// Convert test auto-completed map to slice for response
	for taskID := range testAutoCompleted {
		response.AutoCompleted = append(response.AutoCompleted, taskID)
	}

	// For dry run, return preview without applying to original
	if dryRun {
		response.Preview = string(RenderMarkdown(testList))
		response.Applied = len(sortedOps)
		return response, nil
	}

	// Track auto-completed tasks for actual operations
	autoCompleted := make(map[string]bool)

	// Apply all operations to original (atomic - all succeed or all fail)
	for _, op := range sortedOps {
		if err := applyOperationWithAutoComplete(tl, op, autoCompleted); err != nil {
			return nil, fmt.Errorf("applying operation: %w", err)
		}
		response.Applied++
	}

	// Clear and rebuild auto-completed list from actual operations
	response.AutoCompleted = []string{}
	for taskID := range autoCompleted {
		response.AutoCompleted = append(response.AutoCompleted, taskID)
	}

	return response, nil
}

// validateOperation checks if an operation is valid without applying it
func validateOperation(tl *TaskList, op Operation) error {
	switch strings.ToLower(op.Type) {
	case addOperation:
		if op.Title == "" {
			return fmt.Errorf("add operation requires title")
		}
		if len(op.Title) > 500 {
			return fmt.Errorf("title exceeds 500 characters")
		}
		if op.Parent != "" && tl.FindTask(op.Parent) == nil {
			return fmt.Errorf("parent task %s not found", op.Parent)
		}
		if op.Position != "" {
			if !validateTaskIDFormat(op.Position) {
				return fmt.Errorf("invalid position format: %s", op.Position)
			}
		}
	case removeOperation:
		if op.ID == "" {
			return fmt.Errorf("remove operation requires id")
		}
		if tl.FindTask(op.ID) == nil {
			return fmt.Errorf("task %s not found", op.ID)
		}
	case updateOperation:
		if op.ID == "" {
			return fmt.Errorf("update operation requires id")
		}
		if tl.FindTask(op.ID) == nil {
			return fmt.Errorf("task %s not found", op.ID)
		}
		if op.Title != "" && len(op.Title) > 500 {
			return fmt.Errorf("title exceeds 500 characters")
		}
		// Validate status only when status field is provided
		if hasStatusField(op) && (*op.Status < Pending || *op.Status > Completed) {
			return fmt.Errorf("invalid status value: %d", *op.Status)
		}
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
	return nil
}

// applyOperation executes a single operation
func applyOperation(tl *TaskList, op Operation) error {
	switch strings.ToLower(op.Type) {
	case addOperation:
		// Add the task (now supports position parameter)
		newTaskID, err := tl.AddTask(op.Parent, op.Title, op.Position)
		if err != nil {
			return err
		}
		// If details or references are provided, update the newly added task
		if len(op.Details) > 0 || len(op.References) > 0 {
			if newTaskID != "" {
				return tl.UpdateTask(newTaskID, "", op.Details, op.References)
			}
		}
		return nil
	case removeOperation:
		return tl.RemoveTask(op.ID)
	case updateOperation:
		// Handle status update only when status field is provided
		if hasStatusField(op) {
			if err := tl.UpdateStatus(op.ID, *op.Status); err != nil {
				return err
			}
		}
		// Handle other field updates (title, details, references) if provided
		return tl.UpdateTask(op.ID, op.Title, op.Details, op.References)
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
}

// applyOperationWithAutoComplete executes a single operation and tracks auto-completed tasks
func applyOperationWithAutoComplete(tl *TaskList, op Operation, autoCompleted map[string]bool) error {
	// Apply the operation
	if err := applyOperation(tl, op); err != nil {
		return err
	}

	// Check for auto-completion on update operations that mark tasks as completed
	if strings.ToLower(op.Type) == updateOperation && hasStatusField(op) && *op.Status == Completed {
		// Check and auto-complete parent tasks
		completed, err := tl.AutoCompleteParents(op.ID)
		if err != nil {
			// Log error but don't fail the operation
			// Auto-completion is a bonus feature, not critical to the operation
			return nil
		}

		// Track auto-completed tasks (avoid duplicates)
		for _, taskID := range completed {
			autoCompleted[taskID] = true
		}
	}

	return nil
}

// deepCopy creates a deep copy of the TaskList for dry-run operations
func (tl *TaskList) deepCopy() (*TaskList, error) {
	// Simple approach: render to markdown and parse back
	content := RenderMarkdown(tl)
	copyList, err := ParseMarkdown(content)
	if err != nil {
		return nil, fmt.Errorf("creating deep copy: %w", err)
	}
	copyList.FilePath = tl.FilePath
	// Preserve front matter
	if tl.FrontMatter != nil {
		copyList.FrontMatter = &FrontMatter{
			References: make([]string, len(tl.FrontMatter.References)),
			Metadata:   make(map[string]string),
		}
		copy(copyList.FrontMatter.References, tl.FrontMatter.References)
		maps.Copy(copyList.FrontMatter.Metadata, tl.FrontMatter.Metadata)
	}
	return copyList, nil
}

// ExecuteBatchWithPhases validates and executes a batch of operations atomically with phase support
func (tl *TaskList) ExecuteBatchWithPhases(ops []Operation, dryRun bool, phaseMarkers []PhaseMarker, filePath string) (*BatchResponse, error) {
	response := &BatchResponse{
		Success:       true,
		Applied:       0,
		Errors:        []string{},
		AutoCompleted: []string{},
	}

	// Check if any operations use phases
	hasPhaseOps := false
	for _, op := range ops {
		if op.Phase != "" {
			hasPhaseOps = true
			break
		}
	}

	// If no phase operations, use regular batch execution
	if !hasPhaseOps {
		return tl.ExecuteBatch(ops, dryRun)
	}

	// Sort operations to process position insertions in reverse order
	sortedOps := sortOperationsForPositionInsertions(ops)

	// Create a deep copy to test all operations first
	testList, err := tl.deepCopyWithPhases(phaseMarkers)
	if err != nil {
		return nil, fmt.Errorf("creating test copy: %w", err)
	}

	// Track auto-completed tasks for test copy
	testAutoCompleted := make(map[string]bool)

	// Track phase markers during test execution
	testPhaseMarkers := make([]PhaseMarker, len(phaseMarkers))
	copy(testPhaseMarkers, phaseMarkers)

	// Validate and apply operations to test copy sequentially
	for i, op := range sortedOps {
		if err := validateOperation(testList, op); err != nil {
			response.Success = false
			response.Errors = append(response.Errors, fmt.Sprintf("operation %d: %v", i+1, err))
			return response, nil
		}

		// Apply operation to test copy
		if err := applyOperationWithPhases(testList, op, testAutoCompleted, &testPhaseMarkers); err != nil {
			response.Success = false
			response.Errors = append(response.Errors, fmt.Sprintf("operation %d: %v", i+1, err))
			return response, nil
		}
	}

	// Convert test auto-completed map to slice for response
	for taskID := range testAutoCompleted {
		response.AutoCompleted = append(response.AutoCompleted, taskID)
	}

	// For dry run, return preview without applying to original
	if dryRun {
		response.Preview = string(RenderMarkdownWithPhases(testList, testPhaseMarkers))
		response.Applied = len(sortedOps)
		return response, nil
	}

	// Track auto-completed tasks for actual operations
	autoCompleted := make(map[string]bool)

	// Track phase markers during actual execution
	actualPhaseMarkers := make([]PhaseMarker, len(phaseMarkers))
	copy(actualPhaseMarkers, phaseMarkers)

	// Apply all operations to original (atomic - all succeed or all fail)
	for _, op := range sortedOps {
		if err := applyOperationWithPhases(tl, op, autoCompleted, &actualPhaseMarkers); err != nil {
			return nil, fmt.Errorf("applying operation: %w", err)
		}
		response.Applied++
	}

	// Clear and rebuild auto-completed list from actual operations
	response.AutoCompleted = []string{}
	for taskID := range autoCompleted {
		response.AutoCompleted = append(response.AutoCompleted, taskID)
	}

	// Write the file with phases preserved
	if err := WriteFileWithPhases(tl, actualPhaseMarkers, filePath); err != nil {
		return nil, fmt.Errorf("writing file with phases: %w", err)
	}

	return response, nil
}

// deepCopyWithPhases creates a deep copy preserving phase markers
func (tl *TaskList) deepCopyWithPhases(phaseMarkers []PhaseMarker) (*TaskList, error) {
	// Render with phases and parse back
	content := RenderMarkdownWithPhases(tl, phaseMarkers)
	copyList, err := ParseMarkdown(content)
	if err != nil {
		return nil, fmt.Errorf("creating deep copy: %w", err)
	}
	copyList.FilePath = tl.FilePath
	// Preserve front matter
	if tl.FrontMatter != nil {
		copyList.FrontMatter = &FrontMatter{
			References: make([]string, len(tl.FrontMatter.References)),
			Metadata:   make(map[string]string),
		}
		copy(copyList.FrontMatter.References, tl.FrontMatter.References)
		maps.Copy(copyList.FrontMatter.Metadata, tl.FrontMatter.Metadata)
	}
	return copyList, nil
}

// applyOperationWithPhases executes a single operation with phase support and tracks auto-completed tasks
func applyOperationWithPhases(tl *TaskList, op Operation, autoCompleted map[string]bool, phaseMarkers *[]PhaseMarker) error {
	switch strings.ToLower(op.Type) {
	case addOperation:
		if op.Phase != "" {
			// Phase-aware add operation
			return addTaskWithPhaseMarkers(tl, op, phaseMarkers)
		}
		// Regular add operation
		newTaskID, err := tl.AddTask(op.Parent, op.Title, op.Position)
		if err != nil {
			return err
		}
		// If details or references are provided, update the newly added task
		if len(op.Details) > 0 || len(op.References) > 0 {
			if newTaskID != "" {
				return tl.UpdateTask(newTaskID, "", op.Details, op.References)
			}
		}
		return nil
	case removeOperation:
		return tl.RemoveTask(op.ID)
	case updateOperation:
		// Handle status update only when status field is provided
		if hasStatusField(op) {
			if err := tl.UpdateStatus(op.ID, *op.Status); err != nil {
				return err
			}

			// Check for auto-completion on update operations that mark tasks as completed
			if *op.Status == Completed {
				completed, err := tl.AutoCompleteParents(op.ID)
				if err == nil {
					// Track auto-completed tasks (avoid duplicates)
					for _, taskID := range completed {
						autoCompleted[taskID] = true
					}
				}
			}
		}
		// Handle other field updates (title, details, references) if provided
		return tl.UpdateTask(op.ID, op.Title, op.Details, op.References)
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
}

// addTaskWithPhaseMarkers adds a task to a specific phase, updating phase markers as needed
func addTaskWithPhaseMarkers(tl *TaskList, op Operation, phaseMarkers *[]PhaseMarker) error {
	// Validate input
	if err := validateTaskInput(op.Title); err != nil {
		return err
	}

	// Check resource limits
	if err := tl.checkResourceLimits(op.Parent); err != nil {
		return err
	}

	var insertPosition int
	phaseFound := false

	// Find the phase position
	for _, marker := range *phaseMarkers {
		if marker.Name == op.Phase {
			phaseFound = true
			break
		}
	}

	if !phaseFound {
		// Phase doesn't exist, create it at the end and add task there
		insertPosition = len(tl.Tasks)

		// Add phase marker to the list
		afterTaskID := ""
		if len(tl.Tasks) > 0 {
			afterTaskID = tl.Tasks[len(tl.Tasks)-1].ID
		}
		*phaseMarkers = append(*phaseMarkers, PhaseMarker{
			Name:        op.Phase,
			AfterTaskID: afterTaskID,
		})
	} else {
		// Phase exists, find where to insert the task within this phase
		phaseEndPos := len(tl.Tasks) // Default to end of list

		// Look for the next phase marker in document order
		for i, marker := range *phaseMarkers {
			// Skip until we find our target phase
			if marker.Name != op.Phase {
				continue
			}

			// Look for the next phase after this one
			if i+1 < len(*phaseMarkers) {
				nextMarker := (*phaseMarkers)[i+1]
				if nextMarker.AfterTaskID != "" {
					// Find where the next phase starts
					for j, task := range tl.Tasks {
						if task.ID == nextMarker.AfterTaskID {
							phaseEndPos = j + 1
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
	if op.Parent != "" {
		// For subtasks, use existing AddTask logic
		newTaskID, err := tl.AddTask(op.Parent, op.Title, "")
		if err != nil {
			return err
		}
		// If details or references are provided, update the newly added task
		if len(op.Details) > 0 || len(op.References) > 0 {
			if newTaskID != "" {
				return tl.UpdateTask(newTaskID, "", op.Details, op.References)
			}
		}
		return nil
	}

	// Insert task at the calculated position
	newTask := Task{
		ID:     "temp", // Will be renumbered
		Title:  op.Title,
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

	// Update phase markers to account for the insertion
	if phaseFound {
		// Find the next phase marker after our target phase
		for i, marker := range *phaseMarkers {
			if marker.Name == op.Phase {
				// Look for the next phase marker
				if i+1 < len(*phaseMarkers) {
					nextMarker := &(*phaseMarkers)[i+1]
					// The next phase should now start after the newly inserted task
					if insertPosition < len(tl.Tasks) {
						nextMarker.AfterTaskID = tl.Tasks[insertPosition].ID
					}
				}
				break
			}
		}
	}

	// If details or references are provided, update the newly added task
	if len(op.Details) > 0 || len(op.References) > 0 {
		if insertPosition < len(tl.Tasks) {
			newTaskID := tl.Tasks[insertPosition].ID
			return tl.UpdateTask(newTaskID, "", op.Details, op.References)
		}
	}

	return nil
}
