# Fix Phase Marker Corruption on Task Removal

## Overview

When removing a task from a markdown file with phases (H2 headers), tasks are incorrectly redistributed across phases. The root cause is that phase markers reference task IDs by number, but after `RemoveTask` renumbers remaining tasks, the phase markers become stale. This causes tasks to shift into wrong phases when the file is written.

Additionally, batch operations currently process removes sequentially, requiring users to specify post-renumbering IDs. This is counterintuitive - users expect to specify original task IDs.

## Requirements

- The system MUST adjust phase marker `AfterTaskID` values when removing top-level tasks so that phases continue to reference the correct task boundaries after renumbering
- The system MUST process batch remove operations in reverse order (highest ID first) so users can specify original task IDs
- The system MUST handle batch operations containing remove operations by adjusting phase markers after each individual remove
- The system MUST handle removal of the first task in a phase correctly (phase marker's `AfterTaskID` becomes empty or points to previous task)
- The system MUST preserve empty phases when all tasks in a phase are removed
- The system MUST NOT adjust phase markers when removing subtasks (e.g., `1.1`), since subtask removal does not affect top-level task numbering
- The system MUST work correctly for files without phases (no regression)
- The system SHOULD use the existing `RemoveTaskWithPhases` function for CLI operations

## Phase Marker Adjustment Algorithm

When a top-level task with number N is removed:

1. Extract the top-level task number from the task ID (e.g., `1` from `1` or `1.2`)
2. For each phase marker:
   - If `AfterTaskID` is empty: no change (phase is at document start)
   - If `AfterTaskID` equals N: set to N-1 (or empty if N=1)
   - If `AfterTaskID` > N: decrement by 1
   - If `AfterTaskID` < N: no change

For subtask removal (ID contains `.`): Skip adjustment entirely since top-level task numbers don't change.

## Implementation Approach

**Files to modify:**

1. `cmd/remove.go` - function `runRemove`
   - Read original file content with `os.ReadFile` before parsing
   - Use `task.ParseMarkdown(content)` instead of `task.ParseFile`
   - Set `tl.FilePath = filename` after parsing
   - Replace `tl.RemoveTask` + `tl.WriteFile` with `tl.RemoveTaskWithPhases(taskID, content)`
   - `RemoveTaskWithPhases` handles both phase and non-phase files internally

2. `internal/task/batch.go` - function `sortOperationsForPositionInsertions`
   - Rename to `sortOperationsForExecution` (more accurate name)
   - Also sort remove operations in reverse order (highest ID first), like position insertions
   - This ensures users can specify original task IDs and they'll be removed correctly

3. `internal/task/batch.go` - function `applyOperationWithPhases`, case `removeOperation`
   - After calling `tl.RemoveTask(op.ID)`, add phase marker adjustment
   - Only adjust if the removed task is a top-level task (no `.` in ID)
   - Use the algorithm above to adjust `*phaseMarkers` in place

**Why sort removes in reverse order:**
- User specifies `[{"type": "remove", "id": "1"}, {"type": "remove", "id": "3"}]`
- Without sorting: remove 1 first, task 3 becomes task 2, then remove task 2 (wrong!)
- With reverse sort: remove 3 first (still task 3), then remove 1 (still task 1) - correct!

**Why inline adjustment in batch.go instead of using RemoveTaskWithPhases:**
- `RemoveTaskWithPhases` handles file I/O internally (reads content, writes file)
- Batch operations manage file I/O at the batch level (write once at end)
- Batch operations pass phase markers as a mutable slice across all operations
- Inlining the adjustment allows batch to maintain its atomic transaction model

**Existing patterns to follow:**
- `RemoveTaskWithPhases` in `internal/task/operations.go` - contains the correct adjustment logic
- `getTaskNumber` helper in `internal/task/operations.go` - extracts top-level task number
- `sortOperationsForPositionInsertions` - pattern for sorting position insertions in reverse

**Dependencies:**
- `RemoveTaskWithPhases` function (already exists and tested)
- `getTaskNumber` helper function (already exists, same package)

**Out of Scope:**
- Consolidating phase-aware and non-phase-aware functions (tracked in issue #19)
- Changes to other commands (update, add, etc.)

## Test Cases

**CLI Remove - Phase preservation:**
```
Input file:
# Tasks

## Planning
- [ ] 1. Define requirements
- [ ] 2. Create design

## Implementation
- [ ] 3. Write code
- [ ] 4. Write tests

Operation: rune remove tasks.md 1

Expected output:
# Tasks

## Planning
- [ ] 1. Create design

## Implementation
- [ ] 2. Write code
- [ ] 3. Write tests
```

**Batch Remove - Multiple removes with original IDs:**
```
Input: Tasks 1, 2, 3, 4
Operations: [{"type": "remove", "id": "1"}, {"type": "remove", "id": "3"}]

With reverse sort (new behavior):
1. Remove task 3 first → tasks are now 1, 2, 4 → renumber to 1, 2, 3
2. Remove task 1 → tasks are now 2, 3 → renumber to 1, 2

Result: Original tasks 1 and 3 removed, tasks 2 and 4 remain as 1 and 2
```

**Batch Remove with Phases:**
```
Same as above, but verify phase markers point to correct tasks after all removes.
```

## Risks and Assumptions

- **Risk:** Sorting removes in reverse is a breaking change for users relying on current behavior | **Mitigation:** This is unlikely; the current behavior is counterintuitive and poorly documented
- **Risk:** Multiple batch removes with mixed adds may have complex interactions | **Mitigation:** Removes are sorted separately, adds remain in order; position insertions still processed first
- **Assumption:** `getTaskNumber` correctly returns -1 for invalid IDs | **Validation:** Existing tests cover this
- **Assumption:** Subtask removal does not affect phase markers | **Validation:** Subtasks don't change parent numbering; add test to verify
- **Prerequisite:** `getTaskNumber` is accessible from batch.go (same package - satisfied)
