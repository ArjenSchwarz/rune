# Critical Implementation Concerns

## 1. Position Insertion Logic Missing

The design calls for position-based insertion but no implementation exists. Key challenges:

### Hierarchical Position Handling
```go
// Example: Insert at position "2" when hierarchy is:
// 1. Task A
// 2. Task B
//   2.1. Subtask B1
//   2.2. Subtask B2
// 3. Task C

// Result should be:
// 1. Task A  
// 2. NEW TASK (inserted)
// 3. Task B (was 2, now shifted)
//   3.1. Subtask B1 (was 2.1, now 3.1)
//   3.2. Subtask B2 (was 2.2, now 3.2)
// 4. Task C (was 3, now 4)
```

### Required Functions
- `insertTaskAtPosition(parentID, title, position string) error`
- `shiftTasksFromPosition(position string) error`
- Enhanced renumbering logic for position-aware operations

## 2. Auto-completion Logic Inconsistency

Current code only triggers auto-completion on `update_status` operations:

```go
// Current - only handles update_status
if strings.ToLower(op.Type) == updateStatusOperation && op.Status == Completed {
    completed, err := tl.AutoCompleteParents(op.ID)
    // ...
}
```

Design requires triggering on ANY update with `status=completed`:

```go
// Required - handle unified update with status field
if (strings.ToLower(op.Type) == "update" && op.Status == Completed) ||
   (strings.ToLower(op.Type) == updateStatusOperation && op.Status == Completed) {
    completed, err := tl.AutoCompleteParents(op.ID)
    // ...
}
```

## 3. Validation Logic Gaps

The unified update operation needs comprehensive validation:

```go
case "update":
    if op.ID == "" {
        return fmt.Errorf("update operation requires id")
    }
    if tl.FindTask(op.ID) == nil {
        return fmt.Errorf("task %s not found", op.ID)
    }
    // MISSING: Status validation when status field provided
    if op.Status != 0 && (op.Status < Pending || op.Status > Completed) {
        return fmt.Errorf("invalid status value: %d", op.Status)
    }
    if op.Title != "" && len(op.Title) > 500 {
        return fmt.Errorf("title exceeds 500 characters")
    }
```

## 4. Breaking Change Impact

Removing `update_status` entirely is a hard breaking change. Consider:

### Alternative: Deprecation Path
1. Keep `update_status` but mark deprecated
2. Route `update_status` operations through unified `update` logic
3. Remove in next major version

### Migration Complexity
- All existing scripts using `update_status` will break immediately
- No backward compatibility during transition
- Documentation and examples need complete rewrite

## Recommendations

### 1. Implement Position Logic First
Focus on position insertion as it's the most complex:
- Add `Position` field to `Operation` struct
- Implement position parsing and validation
- Create position-aware insertion logic
- Handle reverse-order processing for multiple insertions

### 2. Gradual Migration for Breaking Changes
Instead of immediate removal:
```go
case updateStatusOperation:
    // Deprecated: route through unified update
    unifiedOp := Operation{
        Type:   "update",
        ID:     op.ID,
        Status: op.Status,
    }
    return applyOperationWithAutoComplete(tl, unifiedOp, autoCompleted)
```

### 3. Enhanced Testing Strategy
- Position insertion edge cases (hierarchical boundaries)
- Multiple position insertions in single batch
- Auto-completion triggering from unified updates
- Backward compatibility during migration

### 4. Performance Considerations
- Position insertion is O(n) where n = tasks after position
- Multiple insertions could be O(n²) without optimization
- Consider batch renumbering for multiple operations

## Risk Assessment

**High Risk:**
- Position insertion with hierarchical tasks (complex edge cases)
- Breaking change impact on existing users
- Performance degradation with large task lists

**Medium Risk:**
- Auto-completion logic changes
- Validation complexity increase
- Testing coverage gaps

**Low Risk:**
- Unified update field handling
- JSON schema changes
- Documentation updates
