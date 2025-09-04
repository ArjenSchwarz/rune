# Design Document: Batch Operations Simplification and Enhancement

## Overview

This design document outlines the implementation approach for simplifying batch operations by unifying the `update` and `update_status` operation types, while adding position-based task insertion capabilities. The design focuses on maintaining backward compatibility during the transition while providing a cleaner, more intuitive API surface.

## Architecture

The feature impacts two primary areas of the codebase:

### Core Components Modified
- `internal/task/batch.go` - Batch operation execution and validation logic
- `internal/task/operations.go` - Task manipulation functions (AddTask, UpdateTask, etc.)
- `cmd/add.go` - CLI add command with position support

### Integration Points
- JSON API through batch operations endpoint
- CLI commands for direct user interaction
- Task ID renumbering system for maintaining consistency

## Components and Interfaces

### 1. Enhanced Operation Structure

The existing `Operation` struct in `batch.go` will be extended to support the unified update approach and position insertion:

```go
type Operation struct {
    Type       string   `json:"type"`
    ID         string   `json:"id,omitempty"`
    Parent     string   `json:"parent,omitempty"`
    Title      string   `json:"title,omitempty"`
    Status     Status   `json:"status,omitempty"`        // Used by unified update
    Details    []string `json:"details,omitempty"`
    References []string `json:"references,omitempty"`
    Position   string   `json:"position,omitempty"`      // NEW: For position insertion
}
```

### 2. Unified Update Operation

The `update` operation will replace both `update` and `update_status` operations:

**Current State:**
- `update` operation: Can modify title, details, references
- `update_status` operation: Can only modify status, triggers auto-completion

**New Design:**
- Single `update` operation handles all fields optionally
- Auto-completion triggers when status field is set to completed, regardless of other fields
- Empty updates (no fields provided) treated as no-ops

### 3. Position-Based Task Insertion

**CLI Enhancement:**
```bash
# Insert before current task 3
rune add "New urgent task" --position 3

# Insert subtask before current task 2.1
rune add "New subtask" --position 2.1 --parent 2
```

**Batch Operation:**
```json
{
  "type": "add",
  "title": "New urgent task",
  "position": "3"
}
```

### 4. Task ID Management Strategy

**Current Renumbering Logic:**
- `removeTaskRecursive()` removes tasks from hierarchy
- `renumberTasks()` reassigns sequential IDs after removal
- IDs follow pattern: `1`, `1.1`, `1.2.1`, etc.

**Enhanced for Position Insertion:**
- New `insertTaskAtPosition()` function for position-aware insertion
- Process multiple position insertions in reverse order to maintain reference consistency
- Renumber affected tasks and their subtrees atomically

## Data Models

### Enhanced AddTask Function Signature
```go
func (tl *TaskList) AddTask(parentID, title, position string) error
```

### New Position Insertion Logic
```go
func (tl *TaskList) insertTaskAtPosition(parentID, title, position string) error {
    // 1. Parse and validate position ID format
    // 2. Find insertion point in task hierarchy
    // 3. Insert new task before specified position
    // 4. Renumber affected tasks and subtrees
    // 5. Update parent-child relationships
}
```

### Batch Processing Changes
```go
func (tl *TaskList) ExecuteBatch(ops []Operation, dryRun bool) (*BatchResponse, error) {
    // 1. Validate all operations against original state
    // 2. Sort position insertions in reverse order
    // 3. Apply operations atomically
    // 4. Track auto-completed tasks from unified updates
    // 5. Return comprehensive response
}
```

## Error Handling

### Validation Rules

**Unified Update Operation:**
- ID field required, must exist in task list
- Status validation only when status field provided (0-2 range)
- Title length validation only when title field provided (≤500 chars)
- Empty operations (no fields) allowed as no-ops

**Position Insertion:**
- Position field must follow task ID format: `^\d+(\.\d+)*$`
- Position exceeding list size results in append behavior
- Invalid position format returns validation error
- Parent task must exist if specified

### Atomic Operation Guarantees
- All operations in a batch succeed or all fail
- Position references always use original pre-batch state
- Task renumbering maintains hierarchical consistency
- Auto-completion tracking accurate across all operations

### Error Response Structure
```go
type BatchResponse struct {
    Success       bool     `json:"success"`
    Applied       int      `json:"applied"`
    Errors        []string `json:"errors,omitempty"`    // Operation-specific errors
    Preview       string   `json:"preview,omitempty"`   // Dry-run output
    AutoCompleted []string `json:"auto_completed,omitempty"` // Auto-completed task IDs
}
```

## Testing Strategy

### Unit Tests

**Batch Operations (`batch_test.go`):**
- Test unified update with various field combinations
- Test empty update operations (no-op behavior)
- Test position insertion at various hierarchical levels
- Test multiple position insertions in single batch
- Test auto-completion triggering from unified updates

**Operations (`operations_test.go`):**
- Test enhanced AddTask with position parameter
- Test task renumbering after position insertion
- Test hierarchical consistency maintenance
- Test edge cases (position beyond list size, invalid formats)

**CLI Commands (`add_test.go`):**
- Test --position flag functionality
- Test position validation and error handling
- Test interaction with existing --parent flag
- Test dry-run mode with position insertion

### Integration Tests

**Batch Processing Workflows:**
- Complex multi-operation batches with mixed operation types
- Position insertion followed by updates on renumbered tasks
- Auto-completion cascading effects
- Error handling and rollback scenarios

**CLI Integration:**
- End-to-end file operations with position insertion
- Git discovery integration with new command flags
- Output format consistency across operation types

### Edge Case Testing

**Position Insertion:**
- Insert at position 1 (beginning of list)
- Insert at last position vs. append behavior
- Insert between hierarchical levels (e.g., position 2 when 2.1 exists)
- Multiple insertions at same position
- Position format validation (valid: "1", "2.3", invalid: "0", "2.0", "2.3.0")

**Unified Updates:**
- Status-only updates (should trigger auto-completion)
- Title + status updates (should trigger auto-completion)
- Detail + reference updates (should not trigger auto-completion)
- Updates with status != completed (should not trigger auto-completion)

## Simplified Design Approach

Based on user clarification and code-simplifier review, the design has been simplified to treat these as two separate, straightforward features:

### Feature 1: Unified Update Operation (Simple)

**Approach**: Extend existing validation and application logic to handle all fields optionally.

**Key Changes**:
```go
// Extend existing validateOperation for unified updates
case "update":
    if op.ID == "" {
        return fmt.Errorf("update operation requires id")
    }
    if tl.FindTask(op.ID) == nil {
        return fmt.Errorf("task %s not found", op.ID)
    }
    // Validate only provided fields
    if op.Title != "" && len(op.Title) > 500 {
        return fmt.Errorf("title exceeds 500 characters")
    }
    if hasStatusField(op) && (op.Status < Pending || op.Status > Completed) {
        return fmt.Errorf("invalid status value: %d", op.Status)
    }

// Extend existing applyOperation for unified updates  
case "update":
    task := tl.FindTask(op.ID)
    if op.Title != "" {
        task.Title = op.Title
    }
    if hasStatusField(op) {
        task.Status = op.Status
    }
    if op.Details != nil {
        task.Details = op.Details
    }
    if op.References != nil {
        task.References = op.References
    }
    tl.Modified = time.Now()
```

**Auto-completion Trigger**: Extend existing logic to handle unified updates:
```go
// Auto-complete when any update operation sets status to completed
if strings.ToLower(op.Type) == "update" && op.Status == Completed && hasStatusField(op) {
    completed, err := tl.AutoCompleteParents(op.ID)
    // Track completed tasks...
}
```

### Feature 2: Position Insertion (Simple)

**Approach**: Add position parameter to existing `AddTask` function and use simple array insertion with existing renumbering.

**Position Semantics**: Position "2.1" means the new task becomes task 2.1, pushing current 2.1 to 2.2.

**Key Changes**:
```go
// Extend existing AddTask signature
func (tl *TaskList) AddTask(parentID, title, position string) error {
    if err := validateTaskInput(title); err != nil {
        return err
    }
    
    if position != "" {
        return tl.addTaskAtPosition(parentID, title, position)
    }
    
    // Existing append logic (unchanged)
    return tl.addTaskAppend(parentID, title)
}

func (tl *TaskList) addTaskAtPosition(parentID, title, position string) error {
    // Parse position: extract final number (e.g., "3" -> 2, "2.1" -> 0)
    parts := strings.Split(position, ".")
    lastPart := parts[len(parts)-1]
    targetIndex, err := strconv.Atoi(lastPart)
    if err != nil || targetIndex < 1 {
        return fmt.Errorf("invalid position format: %s", position)
    }
    targetIndex-- // Convert to 0-based index
    
    if parentID != "" {
        parent := tl.FindTask(parentID)
        if parent == nil {
            return fmt.Errorf("parent task %s not found", parentID)
        }
        
        // Bounds check
        if targetIndex > len(parent.Children) {
            targetIndex = len(parent.Children)
        }
        
        // Simple array insertion
        newTask := Task{
            ID:       "temp", // Will be renumbered
            Title:    title,
            Status:   Pending,
            ParentID: parentID,
        }
        parent.Children = append(parent.Children[:targetIndex], 
                               append([]Task{newTask}, parent.Children[targetIndex:]...)...)
    } else {
        // Root level insertion
        if targetIndex > len(tl.Tasks) {
            targetIndex = len(tl.Tasks)
        }
        
        newTask := Task{
            ID:     "temp",
            Title:  title,
            Status: Pending,
        }
        tl.Tasks = append(tl.Tasks[:targetIndex], 
                         append([]Task{newTask}, tl.Tasks[targetIndex:]...)...)
    }
    
    // Use existing renumberTasks() - no changes needed
    tl.renumberTasks()
    tl.Modified = time.Now()
    return nil
}
```

**Batch Processing**: No special handling needed - each operation sees current state, renumbering happens after each insertion.

**CLI Integration**: Add `--position` flag to existing add command:
```go
var addPosition string

func init() {
    addCmd.Flags().StringVar(&addPosition, "position", "", "position to insert task (optional)")
}

func runAdd(cmd *cobra.Command, args []string) error {
    // ... existing logic ...
    
    // Pass position to AddTask
    if err := tl.AddTask(addParent, addTitle, addPosition); err != nil {
        return fmt.Errorf("failed to add task: %w", err)
    }
    
    // ... rest unchanged ...
}
```

## Implementation Phases (Simplified)

### Phase 1: Unified Update Operation
1. Remove `updateStatusOperation` constant and validation case
2. Extend existing `update` case in `validateOperation` to handle status field
3. Extend existing `update` case in `applyOperation` to handle all optional fields
4. Add `hasStatusField()` helper function to detect when status is provided
5. Update `applyOperationWithAutoComplete` to trigger on unified updates

**Estimated Effort**: 1-2 hours of development + testing

### Phase 2: Position Insertion
1. Add `position` parameter to existing `AddTask` function signature
2. Implement `addTaskAtPosition` helper function using simple array insertion
3. Add Position field to Operation struct for batch operations
4. Update `applyOperation` add case to pass position parameter
5. Add `--position` flag to CLI add command

**Estimated Effort**: 2-3 hours of development + testing

### Phase 3: Testing and Documentation
1. Update existing tests for unified update behavior
2. Add tests for position insertion scenarios
3. Update command help text and examples
4. Update API documentation

**Estimated Effort**: 1-2 hours

**Total Implementation**: ~6 hours instead of weeks of complex development

## Migration Strategy (Simplified)

### Breaking Changes
- Remove `update_status` operation type completely (per requirements)
- All `update_status` operations must be converted to `update` operations
- This is a clean break as requested - no backward compatibility needed

### Code Impact
- Very minimal - most existing functionality unchanged
- New functionality is additive (optional position parameter, optional update fields)
- Existing tests mostly unchanged except for `update_status` removal

### User Impact
- Scripts using `update_status` need simple find/replace to `update`
- CLI users see new `--position` option but all existing commands work unchanged
- Batch API gains new optional field but existing operations unchanged

## Security Considerations

### Input Validation
- Position field validated against task ID regex pattern
- Existing file path validation maintained for CLI operations
- Title length and content validation preserved
- Resource limits (MaxTaskCount, MaxHierarchyDepth) enforced

### Atomic Operations
- Batch operations remain all-or-nothing to prevent partial corruption
- Position insertions maintain referential integrity
- Task ID renumbering preserves hierarchical relationships
- File operations use atomic write patterns (temp file + rename)

## Performance Considerations

### Simplified Performance Profile
- **Unified Updates**: No performance impact - same validation/application logic
- **Position Insertion**: O(n) array insertion + existing O(n) renumbering = O(n) total
- **Batch Processing**: No additional overhead - sequential processing with existing patterns
- **Memory Usage**: Minimal - reuses existing structures and algorithms

The simplified design avoids complex algorithms and maintains the existing performance characteristics.

---

This design maintains the existing architectural patterns while providing the enhanced functionality specified in the requirements. The implementation preserves atomic operation guarantees and maintains the security and validation constraints that are critical to the application's reliability.