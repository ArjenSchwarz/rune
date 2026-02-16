package task

import (
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"
)

const (
	updateOperation   = "update"
	addOperation      = "add"
	removeOperation   = "remove"
	addPhaseOperation = "add-phase"
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
	return IsValidID(id)
}

// sortOperationsForExecution sorts operations to ensure correct execution order:
// 1. Position insertions are sorted in reverse order (highest position first) so position references remain valid
// 2. Remove operations are sorted in reverse order (highest ID first) so users can specify original task IDs
// The sorted operations maintain their relative order: position insertions first, then all other operations
// (with removes sorted in reverse among themselves but in their original position relative to non-removes)
func sortOperationsForExecution(ops []Operation) []Operation {
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

	// Sort remove operations in reverse order (higher IDs first) while preserving their relative
	// position to non-remove operations. This allows users to specify original task IDs when
	// doing multiple removes - removing highest first preserves lower IDs.
	// We achieve this by collecting removes, sorting them, and reinserting at their block position.
	var removes []Operation
	removeIndices := []int{}

	for i, op := range otherOps {
		if strings.ToLower(op.Type) == removeOperation {
			removes = append(removes, op)
			removeIndices = append(removeIndices, i)
		}
	}

	// Sort removes in reverse order
	sort.Slice(removes, func(i, j int) bool {
		return comparePositions(removes[i].ID, removes[j].ID) > 0
	})

	// Replace removes back at their original indices (in sorted order)
	for i, idx := range removeIndices {
		otherOps[idx] = removes[i]
	}

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
	Type         string   `json:"type"`
	ID           string   `json:"id,omitempty"`
	Parent       string   `json:"parent,omitempty"`
	Title        string   `json:"title,omitempty"`
	Status       *Status  `json:"status,omitempty"`
	Details      []string `json:"details,omitempty"`
	References   []string `json:"references,omitempty"`
	Requirements []string `json:"requirements,omitempty"`
	Position     string   `json:"position,omitempty"`
	// Phase has context-dependent meaning:
	// - For "add" operations: the phase to add the task to
	// - For "add-phase" operations: the name of the phase to create
	Phase string `json:"phase,omitempty"`

	// Fields for dependencies and streams
	Stream    *int     `json:"stream,omitempty"`
	BlockedBy []string `json:"blocked_by,omitempty"` // Hierarchical IDs
	Owner     *string  `json:"owner,omitempty"`
	Release   bool     `json:"release,omitempty"` // Clear owner
}

// BatchRequest represents a request for multiple operations
type BatchRequest struct {
	File             string      `json:"file"`
	Operations       []Operation `json:"operations"`
	DryRun           bool        `json:"dry_run"`
	RequirementsFile string      `json:"requirements_file,omitempty"`
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

	// Sort operations for correct execution order (position insertions and removes in reverse)
	// This ensures references use original pre-batch state
	sortedOps := sortOperationsForExecution(ops)

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
		if len(op.Requirements) > 0 {
			for _, reqID := range op.Requirements {
				if !validateTaskIDFormat(reqID) {
					return fmt.Errorf("invalid requirement ID format: %s", reqID)
				}
			}
		}
		// Validate new fields for add operations
		if err := validateExtendedFields(tl, op); err != nil {
			return err
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
		if len(op.Requirements) > 0 {
			for _, reqID := range op.Requirements {
				if !validateTaskIDFormat(reqID) {
					return fmt.Errorf("invalid requirement ID format: %s", reqID)
				}
			}
		}
		// Validate new fields for update operations
		if err := validateExtendedFields(tl, op); err != nil {
			return err
		}
		// For updates with blocked-by, check for cycles
		if len(op.BlockedBy) > 0 {
			task := tl.FindTask(op.ID)
			if task != nil && task.StableID != "" {
				if err := validateNoCycle(tl, task.StableID, op.BlockedBy); err != nil {
					return err
				}
			}
		}
	case addPhaseOperation:
		// Trim whitespace to match CLI behavior (cmd/add_phase.go:59)
		phaseName := strings.TrimSpace(op.Phase)
		if err := ValidatePhaseName(phaseName); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
	return nil
}

// validateExtendedFields validates stream, blocked-by, and owner fields
func validateExtendedFields(tl *TaskList, op Operation) error {
	// Validate stream
	if op.Stream != nil && *op.Stream < 0 {
		return ErrInvalidStream
	}

	// Validate blocked-by references, auto-assigning stable IDs as needed
	if len(op.BlockedBy) > 0 {
		for _, hid := range op.BlockedBy {
			task := tl.FindTask(hid)
			if task == nil {
				return fmt.Errorf("blocked-by task %s not found", hid)
			}
			if task.StableID == "" {
				existingIDs := tl.collectStableIDs()
				idGen := NewStableIDGenerator(existingIDs)
				newID, err := idGen.Generate()
				if err != nil {
					return fmt.Errorf("generating stable ID for task %s: %w", hid, err)
				}
				task.StableID = newID
			}
		}
	}

	// Validate owner
	if op.Owner != nil {
		if err := validateOwner(*op.Owner); err != nil {
			return err
		}
	}

	return nil
}

// validateNoCycle checks that adding dependencies would not create a cycle
func validateNoCycle(tl *TaskList, fromStableID string, blockedByIDs []string) error {
	// Build dependency index
	index := BuildDependencyIndex(tl.Tasks)

	// Resolve hierarchical IDs to stable IDs
	for _, hid := range blockedByIDs {
		task := tl.FindTask(hid)
		if task == nil || task.StableID == "" {
			continue // Already validated in validateExtendedFields
		}

		// Check if adding this dependency would create a cycle
		if hasCycle, path := index.DetectCycle(fromStableID, task.StableID); hasCycle {
			return &CircularDependencyError{Path: path}
		}
	}

	return nil
}

// applyOperation executes a single operation
func applyOperation(tl *TaskList, op Operation) error {
	switch strings.ToLower(op.Type) {
	case addOperation:
		return applyAddOperation(tl, op)
	case removeOperation:
		return tl.RemoveTask(op.ID)
	case updateOperation:
		return applyUpdateOperation(tl, op)
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
}

// applyAddOperation handles add operations with extended fields
func applyAddOperation(tl *TaskList, op Operation) error {
	// Check if any extended fields are used
	hasExtendedFields := op.Stream != nil || len(op.BlockedBy) > 0 || op.Owner != nil

	if hasExtendedFields {
		// Use AddTaskWithOptions for extended fields
		opts := AddOptions{
			Position: op.Position,
		}
		if op.Stream != nil {
			opts.Stream = *op.Stream
		}
		if len(op.BlockedBy) > 0 {
			opts.BlockedBy = op.BlockedBy
		}
		if op.Owner != nil {
			opts.Owner = *op.Owner
		}

		newTaskID, err := tl.AddTaskWithOptions(op.Parent, op.Title, opts)
		if err != nil {
			return err
		}

		// If details, references, or requirements are provided, update the newly added task
		if len(op.Details) > 0 || len(op.References) > 0 || len(op.Requirements) > 0 {
			if newTaskID != "" {
				return tl.UpdateTask(newTaskID, "", op.Details, op.References, op.Requirements)
			}
		}
		return nil
	}

	// Standard add operation without extended fields
	newTaskID, err := tl.AddTask(op.Parent, op.Title, op.Position)
	if err != nil {
		return err
	}
	// If details, references, or requirements are provided, update the newly added task
	if len(op.Details) > 0 || len(op.References) > 0 || len(op.Requirements) > 0 {
		if newTaskID != "" {
			return tl.UpdateTask(newTaskID, "", op.Details, op.References, op.Requirements)
		}
	}
	return nil
}

// applyUpdateOperation handles update operations with extended fields
func applyUpdateOperation(tl *TaskList, op Operation) error {
	// Check if any extended fields are used
	hasExtendedFields := op.Stream != nil || op.BlockedBy != nil || op.Owner != nil || op.Release

	if hasExtendedFields {
		// Use UpdateTaskWithOptions for extended fields
		opts := UpdateOptions{
			Release: op.Release,
		}
		if op.Title != "" {
			opts.Title = &op.Title
		}
		if op.Details != nil {
			opts.Details = op.Details
		}
		if op.References != nil {
			opts.References = op.References
		}
		if op.Requirements != nil {
			opts.Requirements = op.Requirements
		}
		if op.Stream != nil {
			opts.Stream = op.Stream
		}
		if op.BlockedBy != nil {
			opts.BlockedBy = op.BlockedBy
		}
		if op.Owner != nil {
			opts.Owner = op.Owner
		}

		// Handle status update separately if provided
		if hasStatusField(op) {
			if err := tl.UpdateStatus(op.ID, *op.Status); err != nil {
				return err
			}
		}

		return tl.UpdateTaskWithOptions(op.ID, opts)
	}

	// Standard update operation without extended fields
	if hasStatusField(op) {
		if err := tl.UpdateStatus(op.ID, *op.Status); err != nil {
			return err
		}
	}
	return tl.UpdateTask(op.ID, op.Title, op.Details, op.References, op.Requirements)
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

	// Check if any operations use phases and validate phase names
	hasPhaseOps := false
	for _, op := range ops {
		if op.Phase != "" {
			if err := ValidatePhaseName(op.Phase); err != nil {
				return nil, err
			}
			hasPhaseOps = true
		}
	}

	// Use phase-aware execution if:
	// 1. The file has phase markers (need to preserve phase boundaries on removes), OR
	// 2. Any operation specifies a phase (need to add tasks to phases)
	// Otherwise, use regular batch execution (no phases involved)
	if len(phaseMarkers) == 0 && !hasPhaseOps {
		return tl.ExecuteBatch(ops, dryRun)
	}

	// Sort operations for correct execution order (position insertions and removes in reverse)
	sortedOps := sortOperationsForExecution(ops)

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

// updateTaskDetailsAndReferences updates a task with details, references, and requirements if provided
func updateTaskDetailsAndReferences(tl *TaskList, taskID string, details []string, references []string, requirements []string) error {
	if len(details) > 0 || len(references) > 0 || len(requirements) > 0 {
		if taskID != "" {
			return tl.UpdateTask(taskID, "", details, references, requirements)
		}
	}
	return nil
}

// applyOperationWithPhases executes a single operation with phase support and tracks auto-completed tasks
func applyOperationWithPhases(tl *TaskList, op Operation, autoCompleted map[string]bool, phaseMarkers *[]PhaseMarker) error {
	switch strings.ToLower(op.Type) {
	case addPhaseOperation:
		// Create a new phase at the end of the document
		// Trim whitespace to match CLI behavior (cmd/add_phase.go:59)
		phaseName := strings.TrimSpace(op.Phase)
		// Determine the AfterTaskID - if there are tasks, use the last one's ID
		afterTaskID := ""
		if len(tl.Tasks) > 0 {
			afterTaskID = tl.Tasks[len(tl.Tasks)-1].ID
		}
		// Add the phase marker
		*phaseMarkers = append(*phaseMarkers, PhaseMarker{
			Name:        phaseName,
			AfterTaskID: afterTaskID,
		})
		return nil
	case addOperation:
		if op.Phase != "" {
			// Phase-aware add operation
			return addTaskWithPhaseMarkers(tl, op, phaseMarkers)
		}
		// Check for extended fields
		hasExtendedFields := op.Stream != nil || len(op.BlockedBy) > 0 || op.Owner != nil
		if hasExtendedFields {
			opts := AddOptions{
				Position: op.Position,
			}
			if op.Stream != nil {
				opts.Stream = *op.Stream
			}
			if len(op.BlockedBy) > 0 {
				opts.BlockedBy = op.BlockedBy
			}
			if op.Owner != nil {
				opts.Owner = *op.Owner
			}
			newTaskID, err := tl.AddTaskWithOptions(op.Parent, op.Title, opts)
			if err != nil {
				return err
			}
			return updateTaskDetailsAndReferences(tl, newTaskID, op.Details, op.References, op.Requirements)
		}
		// Regular add operation
		newTaskID, err := tl.AddTask(op.Parent, op.Title, op.Position)
		if err != nil {
			return err
		}
		return updateTaskDetailsAndReferences(tl, newTaskID, op.Details, op.References, op.Requirements)
	case removeOperation:
		adjustPhaseMarkersForRemoval(op.ID, phaseMarkers)
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
		// Check for extended fields
		hasExtendedFields := op.Stream != nil || op.BlockedBy != nil || op.Owner != nil || op.Release
		if hasExtendedFields {
			opts := UpdateOptions{
				Release: op.Release,
			}
			if op.Title != "" {
				opts.Title = &op.Title
			}
			if op.Details != nil {
				opts.Details = op.Details
			}
			if op.References != nil {
				opts.References = op.References
			}
			if op.Requirements != nil {
				opts.Requirements = op.Requirements
			}
			if op.Stream != nil {
				opts.Stream = op.Stream
			}
			if op.BlockedBy != nil {
				opts.BlockedBy = op.BlockedBy
			}
			if op.Owner != nil {
				opts.Owner = op.Owner
			}
			return tl.UpdateTaskWithOptions(op.ID, opts)
		}
		// Handle other field updates (title, details, references, requirements) if provided
		return tl.UpdateTask(op.ID, op.Title, op.Details, op.References, op.Requirements)
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
		// For subtasks, use existing AddTask logic (with extended fields if present)
		hasExtendedFields := op.Stream != nil || len(op.BlockedBy) > 0 || op.Owner != nil
		if hasExtendedFields {
			opts := AddOptions{}
			if op.Stream != nil {
				opts.Stream = *op.Stream
			}
			if len(op.BlockedBy) > 0 {
				opts.BlockedBy = op.BlockedBy
			}
			if op.Owner != nil {
				opts.Owner = *op.Owner
			}
			newTaskID, err := tl.AddTaskWithOptions(op.Parent, op.Title, opts)
			if err != nil {
				return err
			}
			return updateTaskDetailsAndReferences(tl, newTaskID, op.Details, op.References, op.Requirements)
		}
		newTaskID, err := tl.AddTask(op.Parent, op.Title, "")
		if err != nil {
			return err
		}
		return updateTaskDetailsAndReferences(tl, newTaskID, op.Details, op.References, op.Requirements)
	}

	// Resolve blocked-by references if present
	var blockedByStableIDs []string
	if len(op.BlockedBy) > 0 {
		var err error
		blockedByStableIDs, err = tl.resolveToStableIDs(op.BlockedBy)
		if err != nil {
			return err
		}
	}

	// Generate stable ID if extended fields are used
	var stableID string
	hasExtendedFields := op.Stream != nil || len(op.BlockedBy) > 0 || op.Owner != nil
	if hasExtendedFields {
		existingIDs := tl.collectStableIDs()
		idGen := NewStableIDGenerator(existingIDs)
		var err error
		stableID, err = idGen.Generate()
		if err != nil {
			return fmt.Errorf("generating stable ID: %w", err)
		}
	}

	// Insert task at the calculated position
	newTask := Task{
		ID:        "temp", // Will be renumbered
		Title:     op.Title,
		Status:    Pending,
		StableID:  stableID,
		BlockedBy: blockedByStableIDs,
	}
	if op.Stream != nil {
		newTask.Stream = *op.Stream
	}
	if op.Owner != nil {
		newTask.Owner = *op.Owner
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

	// Update phase markers to account for the insertion
	// IMPORTANT: Since we ALWAYS insert at the END of the phase (insertPosition = phaseEndPos),
	// the newly inserted task becomes the last task in the current phase. Therefore, the next
	// phase marker must be updated to point to this newly inserted task's ID.
	// This maintains the invariant that phase markers always point to the last task in the
	// preceding phase.
	if phaseFound {
		// Find the next phase marker after our target phase
		for i, marker := range *phaseMarkers {
			if marker.Name == op.Phase {
				// Look for the next phase marker
				if i+1 < len(*phaseMarkers) {
					nextMarker := &(*phaseMarkers)[i+1]
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

	// If details, references, or requirements are provided, update the newly added task
	if len(op.Details) > 0 || len(op.References) > 0 || len(op.Requirements) > 0 {
		if insertPosition < len(tl.Tasks) {
			newTaskID := tl.Tasks[insertPosition].ID
			return tl.UpdateTask(newTaskID, "", op.Details, op.References, op.Requirements)
		}
	}

	return nil
}
