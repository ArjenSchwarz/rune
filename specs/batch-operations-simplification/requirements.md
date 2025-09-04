# Batch Operations Simplification and Enhancement

## Introduction

This feature simplifies the batch operations API by unifying the `update` and `update_status` operation types into a single `update` operation, while also introducing the ability to insert tasks at specific positions within a list.

## Requirements

### 1. Unified Update Operation

**User Story:** As a developer using the batch API, I want a single update operation that can modify any task field, so that I can update multiple properties in one operation without needing to understand artificial distinctions.

**Acceptance Criteria:**
1.1. The system SHALL remove the `update_status` operation type entirely
1.2. The `update` operation SHALL accept optional fields for status, title, details, and references
1.3. The system SHALL validate status values (0=pending, 1=in-progress, 2=completed) only when the status field is provided
1.4. The system SHALL trigger auto-completion logic whenever the status field is set to completed, regardless of other fields being updated simultaneously
1.5. The system SHALL apply all field updates atomically within a single operation
1.6. The system SHALL validate title length (max 500 characters) only when the title field is provided
1.7. When no fields are provided in an update operation, the system SHALL treat it as a no-op without error

### 2. Task Position Insertion

**User Story:** As a user organizing tasks, I want to insert new tasks at specific positions in the list, so that I can maintain logical ordering without manually renumbering tasks.

**Acceptance Criteria:**
2.1. The `rune add` command SHALL accept an optional `--position` flag to specify insertion location
2.2. The batch `add` operation SHALL accept an optional `position` field to specify insertion location
2.3. When position is specified, the system SHALL insert the new task BEFORE the task at that position, causing that task and all subsequent tasks to be renumbered
2.4. The position SHALL be specified as a task ID (e.g., "4" to insert before current task 4, making the new task become task 4)
2.5. If the specified position exceeds the current list size, the system SHALL append the task at the end
2.6. The system SHALL validate that the position ID follows the standard task ID format (^\d+(\.\d+)*$)
2.7. When inserting between hierarchical tasks, the system SHALL maintain parent-child relationships correctly
2.8. The system SHALL update all affected task IDs in a single atomic operation
2.9. If no position is specified, the system SHALL maintain the current behavior of appending to the end
2.10. In batch operations with multiple position insertions, the system SHALL process insertions in reverse order to maintain consistent position references to the original state

## Edge Cases and Considerations

### Update Operation
- Empty update operations (no fields provided) will be treated as no-ops without error
- Updates to non-existent tasks will fail the entire batch operation (atomic behavior)
- The `update_status` operation type will be completely removed

### Position Insertion
- Multiple insertions at the same position will be processed in reverse order as per requirement 2.9
- Inserting at a position that would break hierarchical structure (e.g., inserting "2" when "2.1" exists) will shift the entire subtree
- Position insertion is allowed for both root-level tasks and subtasks (e.g., position "2.3" inserts before current task 2.3)

## Technical Constraints

- Maximum file size remains 10MB
- Maximum task title remains 500 characters
- Task ID pattern must follow ^\d+(\.\d+)*$ format
- All batch operations must be atomic (all succeed or all fail)
- File operations must maintain security validation (paths within working directory)

## Success Criteria

- Task renumbering after position insertion maintains consistency across the entire list
- All operations in a batch reference the original pre-batch state
- All existing tests pass with the new implementation
- New tests cover all edge cases for the new features
- The `rune add` command supports position insertion via --position flag
- Documentation is updated to reflect the removal of `update_status` and addition of position insertion

## Examples

### Unified Update Operation
```json
// Before - requires two operations
[
  {"type": "update_status", "id": "1", "status": 2},
  {"type": "update", "id": "1", "title": "Completed task"}
]

// After - single operation (update_status no longer exists)
[
  {"type": "update", "id": "1", "status": 2, "title": "Completed task"}
]
```

### Position Insertion

**CLI Command:**
```bash
# Insert a new task before current task 3
rune add "New urgent task" --position 3

# Insert a subtask before current task 2.1
rune add "New subtask" --position 2.1 --parent 2
```

**Batch Operations:**
```json
// Insert a new task before current task 3
[
  {"type": "add", "title": "New urgent task", "position": "3"}
]
// Result: New task becomes task 3, old task 3 becomes 4, etc.

// Multiple insertions - processed in reverse order
[
  {"type": "add", "title": "Task A", "position": "2"},
  {"type": "add", "title": "Task B", "position": "4"}
]
// Task B is inserted first (before original task 4)
// Then Task A is inserted (before original task 2)
```