package task

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
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

// sortOperationsForExecution reorders operations only within contiguous runs
// of the same reorderable kind, so that IDs inside a run refer to the state
// before that run:
//  1. Positioned adds (add ops with a Position) are sorted in reverse position
//     order (highest first) so earlier insertions don't shift later positions.
//  2. Remove operations are sorted in reverse ID order (highest first) so
//     removing one task doesn't renumber the others in the run.
//
// Operations are never moved across an operation of a different kind: doing
// so would change which task an ID refers to (T-200, T-1874).
func sortOperationsForExecution(ops []Operation) []Operation {
	result := slices.Clone(ops)

	i := 0
	for i < len(result) {
		kind := reorderKind(result[i])
		if kind == "" {
			i++
			continue
		}
		blockStart := i
		for i < len(result) && reorderKind(result[i]) == kind {
			i++
		}
		block := result[blockStart:i]
		sort.SliceStable(block, func(a, b int) bool {
			return comparePositions(reorderKey(block[a]), reorderKey(block[b])) > 0
		})
	}

	return result
}

// reorderKind returns the run kind an operation belongs to for
// sortOperationsForExecution, or "" if it is never reordered.
func reorderKind(op Operation) string {
	switch opType := strings.ToLower(op.Type); {
	case opType == removeOperation:
		return removeOperation
	case opType == addOperation && op.Position != "":
		return addOperation
	default:
		return ""
	}
}

// reorderKey returns the hierarchical ID a reorderable operation targets.
func reorderKey(op Operation) string {
	if strings.ToLower(op.Type) == removeOperation {
		return op.ID
	}
	return op.Position
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

// batchApplyFunc applies a single validated operation to a task list,
// recording auto-completed parents and updating phase markers as needed.
type batchApplyFunc func(tl *TaskList, op Operation, autoCompleted map[string]bool, phaseMarkers *[]PhaseMarker) error

// runBatch validates and applies ops, in execution order, to a deep copy of
// tl. The original is never touched, so a failing operation leaves it
// exactly as it was. On success the copy and its phase markers are the
// batch's result: the caller adopts them instead of replaying every
// operation against the original, so each operation runs exactly once.
// The returned task list and markers are nil when an operation failed.
func (tl *TaskList) runBatch(ops []Operation, phaseMarkers []PhaseMarker, apply batchApplyFunc) (*BatchResponse, *TaskList, []PhaseMarker) {
	response := &BatchResponse{
		Success:       true,
		Errors:        []string{},
		AutoCompleted: []string{},
	}

	// Sort operations for correct execution order (position insertions and removes in reverse)
	// This ensures references use original pre-batch state
	sortedOps := sortOperationsForExecution(ops)

	work := tl.deepCopy()
	markers := slices.Clone(phaseMarkers)
	autoCompleted := make(map[string]bool)

	for i, op := range sortedOps {
		if err := validateOperation(work, op); err != nil {
			response.Success = false
			response.Errors = append(response.Errors, fmt.Sprintf("operation %d: %v", i+1, err))
			return response, nil, nil
		}
		if err := apply(work, op, autoCompleted, &markers); err != nil {
			response.Success = false
			response.Errors = append(response.Errors, fmt.Sprintf("operation %d: %v", i+1, err))
			return response, nil, nil
		}
	}

	for taskID := range autoCompleted {
		response.AutoCompleted = append(response.AutoCompleted, taskID)
	}
	sort.Strings(response.AutoCompleted)
	response.Applied = len(sortedOps)
	return response, work, markers
}

// ExecuteBatch validates and executes a batch of operations atomically:
// either every operation succeeds and tl is updated, or tl is left unchanged
// and the response lists the first failure. With dryRun, tl is never changed
// and the response carries a preview of the result.
func (tl *TaskList) ExecuteBatch(ops []Operation, dryRun bool) (*BatchResponse, error) {
	response, result, _ := tl.runBatch(ops, nil,
		func(l *TaskList, op Operation, autoCompleted map[string]bool, _ *[]PhaseMarker) error {
			return applyOperationWithAutoComplete(l, op, autoCompleted)
		})
	if result == nil {
		return response, nil
	}

	if dryRun {
		response.Preview = string(RenderMarkdown(result))
		return response, nil
	}

	*tl = *result
	return response, nil
}

// validateOperation checks if an operation is valid without applying it
func validateOperation(tl *TaskList, op Operation) error {
	switch strings.ToLower(op.Type) {
	case addOperation:
		if op.Title == "" {
			return fmt.Errorf("add operation requires title")
		}
		if len(op.Title) > MaxTitleLength {
			return fmt.Errorf("task title exceeds %d characters", MaxTitleLength)
		}
		if op.Parent != "" && tl.FindTask(op.Parent) == nil {
			return fmt.Errorf("parent task %s not found", op.Parent)
		}
		// New tasks are always created pending; reject a status rather than
		// report success while silently ignoring it (T-1996).
		if hasStatusField(op) {
			return fmt.Errorf("add operation does not support status; add the task, then use an update operation to set its status")
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
		// Validate details and references content upfront
		if err := validateDetailsAndReferences(op); err != nil {
			return err
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
		if op.Title != "" && len(op.Title) > MaxTitleLength {
			return fmt.Errorf("task title exceeds %d characters", MaxTitleLength)
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
		// Validate details and references content upfront to prevent
		// partial status application when UpdateTask/UpdateTaskWithOptions
		// later rejects invalid content
		if err := validateDetailsAndReferences(op); err != nil {
			return err
		}
		// Validate new fields for update operations
		if err := validateExtendedFields(tl, op); err != nil {
			return err
		}
		// Cycles are detected when the update is applied
		// (UpdateTaskWithOptions), which reports the same error; checking here
		// too would build the dependency index twice per operation.
	case addPhaseOperation:
		if _, err := NormalizePhaseName(op.Phase); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
	return nil
}

// validateDetailsAndReferences validates detail and reference content upfront
// so that applyOperation never fails on content validation after partially
// applying other fields (e.g., status).
func validateDetailsAndReferences(op Operation) error {
	if op.Details != nil {
		if err := validateDetails(op.Details); err != nil {
			return err
		}
	}
	if op.References != nil {
		if err := validateReferences(op.References); err != nil {
			return err
		}
	}
	return nil
}

// validateExtendedFields validates stream, blocked-by, and owner fields
func validateExtendedFields(tl *TaskList, op Operation) error {
	// Validate stream
	if op.Stream != nil && *op.Stream < 0 {
		return ErrInvalidStream
	}

	// Validate blocked-by references. Stable IDs for the referenced tasks are
	// assigned when the operation is applied (resolveToStableIDs).
	for _, hid := range op.BlockedBy {
		if tl.FindTask(hid) == nil {
			return fmt.Errorf("blocked-by task %s not found", hid)
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

// applyOperation executes a single operation
func applyOperation(tl *TaskList, op Operation) error {
	switch strings.ToLower(op.Type) {
	case addOperation:
		_, err := tl.AddTaskWithPhases(op, &[]PhaseMarker{})
		return err
	case removeOperation:
		_, err := tl.RemoveTaskWithDependents(op.ID)
		return err
	case updateOperation:
		return applyUpdateOperation(tl, op)
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
}

// addTaskFromOperation adds a task for an "add" operation without any phase
// handling, choosing AddTaskWithOptions only when extended fields are set so
// plain adds don't get a stable ID. Returns the new task's hierarchical ID.
func addTaskFromOperation(tl *TaskList, op Operation) (string, error) {
	hasExtendedFields := op.Stream != nil || len(op.BlockedBy) > 0 || op.Owner != nil
	if !hasExtendedFields {
		return tl.AddTask(op.Parent, op.Title, op.Position)
	}

	opts := AddOptions{
		Position:  op.Position,
		BlockedBy: op.BlockedBy,
	}
	if op.Stream != nil {
		opts.Stream = *op.Stream
	}
	if op.Owner != nil {
		opts.Owner = *op.Owner
	}
	return tl.AddTaskWithOptions(op.Parent, op.Title, opts)
}

// AddTaskWithPhases adds the task described by an "add" operation — title,
// parent, position, phase, stream, blocked-by, owner, details, references and
// requirements — and keeps phaseMarkers in step with the resulting top-level
// numbering. It is the single add path shared by the add command and batch
// operations, so every flag is honoured the same way everywhere.
//
//   - With op.Phase set and no parent, the task is appended to the end of that
//     phase, creating the phase at the end of the document if it does not exist.
//     Combining this with op.Position is rejected: the two would place the
//     task in conflicting spots (T-919, T-973).
//   - A subtask (op.Parent set) stays under its parent, so op.Phase has no
//     effect on where it goes and never creates a phase.
//   - A positional insert at the root shifts every later top-level task down
//     by one, and markers after the insertion point are shifted with it. The
//     new task joins the phase of the task that previously held its position.
//
// phaseMarkers must be non-nil; it may point to an empty slice for files
// without phases. Returns the hierarchical ID of the new task.
func (tl *TaskList) AddTaskWithPhases(op Operation, phaseMarkers *[]PhaseMarker) (string, error) {
	if op.Phase != "" && op.Parent == "" {
		if op.Position != "" {
			return "", fmt.Errorf("position cannot be combined with phase for a top-level task")
		}
		return addTaskWithPhaseMarkers(tl, op, phaseMarkers)
	}
	if op.Phase != "" {
		// Validate the name even though a subtask ignores it, so a bad phase
		// name is reported consistently on every path.
		if _, err := NormalizePhaseName(op.Phase); err != nil {
			return "", err
		}
	}

	newTaskID, err := addTaskFromOperation(tl, op)
	if err != nil {
		return "", err
	}

	if op.Parent == "" && op.Position != "" {
		if position, convErr := strconv.Atoi(newTaskID); convErr == nil && position > 0 {
			adjustPhaseMarkersForInsertion(position-1, phaseMarkers)
		}
	}

	return newTaskID, updateTaskDetailsAndReferences(tl, newTaskID, op.Details, op.References, op.Requirements)
}

// applyUpdateOperation handles update operations with extended fields.
// Status is applied AFTER other fields to avoid partial mutation if
// UpdateTask/UpdateTaskWithOptions rejects invalid content.
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

		// Apply other fields first; status is deferred until these succeed
		if err := tl.UpdateTaskWithOptions(op.ID, opts); err != nil {
			return err
		}

		// Apply status after other fields have been validated and applied
		if hasStatusField(op) {
			return tl.UpdateStatus(op.ID, *op.Status)
		}
		return nil
	}

	// Standard update operation without extended fields:
	// Apply title/details/references/requirements first, then status
	if err := tl.UpdateTask(op.ID, op.Title, op.Details, op.References, op.Requirements); err != nil {
		return err
	}
	if hasStatusField(op) {
		return tl.UpdateStatus(op.ID, *op.Status)
	}
	return nil
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

// ExecuteBatchWithPhases validates and executes a batch of operations atomically
// with phase support, keeping phaseMarkers in step with the operations, and
// writes the result to filePath (unless dryRun). Falls back to ExecuteBatch
// when neither the file nor any operation involves phases; in that case the
// caller is responsible for writing the file.
func (tl *TaskList) ExecuteBatchWithPhases(ops []Operation, dryRun bool, phaseMarkers []PhaseMarker, filePath string) (*BatchResponse, error) {
	// Check if any operations use phases and validate phase names
	hasPhaseOps := false
	for _, op := range ops {
		if op.Phase != "" {
			// This is the only phase-name guard for an "add" operation carrying
			// a phase: validateOperation's add case never inspects op.Phase.
			if _, err := NormalizePhaseName(op.Phase); err != nil {
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

	response, result, resultMarkers := tl.runBatch(ops, phaseMarkers, applyOperationWithPhases)
	if result == nil {
		return response, nil
	}

	if dryRun {
		response.Preview = string(RenderMarkdownWithPhases(result, resultMarkers, nil))
		return response, nil
	}

	// Write before adopting the result so a failed write leaves tl unchanged
	if err := WriteFileWithPhases(result, resultMarkers, filePath); err != nil {
		return nil, fmt.Errorf("writing file with phases: %w", err)
	}
	*tl = *result
	return response, nil
}

// deepCopy returns a copy of tl that shares no mutable state with it, so
// operations can be tried on the copy without touching the original.
func (tl *TaskList) deepCopy() *TaskList {
	c := *tl
	c.Tasks = cloneTasks(tl.Tasks)
	if tl.FrontMatter != nil {
		c.FrontMatter = &FrontMatter{
			References: slices.Clone(tl.FrontMatter.References),
			Metadata:   maps.Clone(tl.FrontMatter.Metadata),
			Extra:      slices.Clone(tl.FrontMatter.Extra),
		}
	}
	return &c
}

// cloneTasks deep-copies a task slice, including every nested slice.
func cloneTasks(tasks []Task) []Task {
	if tasks == nil {
		return nil
	}
	out := make([]Task, len(tasks))
	for i, t := range tasks {
		t.Details = slices.Clone(t.Details)
		t.References = slices.Clone(t.References)
		t.Requirements = slices.Clone(t.Requirements)
		t.BlockedBy = slices.Clone(t.BlockedBy)
		t.Children = cloneTasks(t.Children)
		out[i] = t
	}
	return out
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
		phaseName, err := NormalizePhaseName(op.Phase)
		if err != nil {
			return err
		}
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
		_, err := tl.AddTaskWithPhases(op, phaseMarkers)
		return err
	case removeOperation:
		adjustPhaseMarkersForRemoval(op.ID, phaseMarkers)
		_, err := tl.RemoveTaskWithDependents(op.ID)
		return err
	case updateOperation:
		// Apply field updates BEFORE status to avoid partial mutation if
		// UpdateTask/UpdateTaskWithOptions rejects invalid content.
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
			if err := tl.UpdateTaskWithOptions(op.ID, opts); err != nil {
				return err
			}
		} else {
			// Handle other field updates (title, details, references, requirements) if provided
			if err := tl.UpdateTask(op.ID, op.Title, op.Details, op.References, op.Requirements); err != nil {
				return err
			}
		}

		// Apply status after other fields have been validated and applied
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
		return nil
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
}

// addTaskWithPhaseMarkers appends a top-level task to the end of the phase
// named by op.Phase, creating the phase at the end of the document if it does
// not exist, and updates phaseMarkers for the shifted numbering. Subtasks are
// handled by AddTaskWithPhases and never reach here. Returns the new task ID.
func addTaskWithPhaseMarkers(tl *TaskList, op Operation, phaseMarkers *[]PhaseMarker) (string, error) {
	// Validate input
	if err := validateTaskInput(op.Title); err != nil {
		return "", err
	}

	// Normalize the phase name so marker lookups compare (and any new marker
	// stores) the same canonical string the add-phase path writes. Without this,
	// {"phase": "Planning\n"} would fail to match an existing "## Planning"
	// header and silently create a duplicate phase.
	phaseName, err := NormalizePhaseName(op.Phase)
	if err != nil {
		return "", err
	}

	// Check resource limits
	if err := tl.checkResourceLimits(""); err != nil {
		return "", err
	}

	// Validate extended fields before touching the task list or the markers,
	// so a failure leaves both unchanged.
	if op.Stream != nil && *op.Stream < 0 {
		return "", ErrInvalidStream
	}
	if op.Owner != nil {
		if err := validateOwner(*op.Owner); err != nil {
			return "", err
		}
	}
	var blockedByStableIDs []string
	if len(op.BlockedBy) > 0 {
		blockedByStableIDs, err = tl.resolveToStableIDs(op.BlockedBy)
		if err != nil {
			return "", err
		}
	}

	// Find the target phase (first occurrence) and where it ends: the task
	// after which the next phase starts, or the end of the list.
	targetIdx := -1
	for i, marker := range *phaseMarkers {
		if marker.Name == phaseName {
			targetIdx = i
			break
		}
	}

	insertPosition := len(tl.Tasks)
	if targetIdx == -1 {
		// Phase doesn't exist: create it at the end and add the task there
		afterTaskID := ""
		if len(tl.Tasks) > 0 {
			afterTaskID = tl.Tasks[len(tl.Tasks)-1].ID
		}
		*phaseMarkers = append(*phaseMarkers, PhaseMarker{
			Name:        phaseName,
			AfterTaskID: afterTaskID,
		})
	} else if targetIdx+1 < len(*phaseMarkers) {
		next := (*phaseMarkers)[targetIdx+1]
		if next.AfterTaskID == "" {
			insertPosition = 0
		} else if n := getTaskNumber(next.AfterTaskID); n > 0 && n <= len(tl.Tasks) {
			insertPosition = n
		}
	}

	// Generate a stable ID only when extended fields are used, matching
	// AddTask vs AddTaskWithOptions on the non-phase path.
	var stableID string
	if op.Stream != nil || len(op.BlockedBy) > 0 || op.Owner != nil {
		stableID, err = NewStableIDGenerator(tl.collectStableIDs()).Generate()
		if err != nil {
			return "", fmt.Errorf("generating stable ID: %w", err)
		}
	}

	newTask := Task{
		ID:        tempTaskID, // Will be renumbered
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

	tl.Tasks = slices.Insert(tl.Tasks, insertPosition, newTask)
	tl.RenumberTasks()
	tl.Modified = time.Now()
	newTaskID := tl.Tasks[insertPosition].ID

	// The new task is always the last task of its phase, so the next marker
	// must point at it, and every marker after that shifts down by one.
	if targetIdx != -1 {
		if targetIdx+1 < len(*phaseMarkers) {
			(*phaseMarkers)[targetIdx+1].AfterTaskID = newTaskID
		}
		if targetIdx+2 < len(*phaseMarkers) {
			remaining := (*phaseMarkers)[targetIdx+2:]
			adjustPhaseMarkersForInsertion(insertPosition, &remaining)
		}
	}

	return newTaskID, updateTaskDetailsAndReferences(tl, newTaskID, op.Details, op.References, op.Requirements)
}
