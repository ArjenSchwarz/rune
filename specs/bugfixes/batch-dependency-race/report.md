# Bugfix Report: Batch Dependency Race

**Date:** 2026-02-16
**Status:** Fixed

## Description of the Issue

When updating a task to add `blocked_by` dependencies (via batch or the `update` command), the task being updated was not auto-assigned a stable ID. This left the task with `BlockedBy` references but no `StableID`, making it invisible to the dependency index.

**Reproduction steps:**
1. Create a task file with tasks that have no stable IDs (the default when created without dependencies)
2. Run a batch operation that updates a task to add `blocked_by` dependencies
3. Observe that the updated task has `BlockedBy` set but no `StableID`

**Impact:** Tasks with dependencies were partially invisible to the dependency tracking system. The dependency index (`BuildDependencyIndex`) only indexes tasks with stable IDs, so a task with `BlockedBy` but no `StableID` would not appear in `GetDependents` results, and cycle detection during validation was skipped for such tasks.

## Investigation Summary

- **Symptoms examined:** Agent task creation sessions with inter-task dependencies could produce tasks with incomplete dependency metadata
- **Code inspected:** `internal/task/operations.go` (`UpdateTaskWithOptions`, `resolveToStableIDs`), `internal/task/batch.go` (`validateOperation`, `validateExtendedFields`), `internal/task/dependencies.go` (`BuildDependencyIndex`)
- **Hypotheses tested:** Multiple scenarios including chain dependencies, cross-batch references, cycle detection, and phase-aware operations. The bug was isolated to `UpdateTaskWithOptions` missing stable ID auto-assignment for the task being updated.

## Discovered Root Cause

**Defect type:** Missing auto-assignment of stable ID

**Why it occurred:** `UpdateTaskWithOptions` called `resolveToStableIDs` to auto-assign stable IDs to dependency *targets* but did not assign a stable ID to the task *being updated*. Similarly, `validateOperation` in batch.go skipped cycle detection entirely when the task being validated had no stable ID.

**Contributing factors:** The `AddTaskWithOptions` path (used for new tasks with dependencies) correctly generated a stable ID for the new task. The asymmetry between add and update paths made the bug non-obvious. T-59 fixed the target side (auto-assigning to referenced tasks) but missed the source side (the task being given dependencies).

## Resolution for the Issue

**Changes made:**
- `internal/task/operations.go:1028-1037` - Auto-assign stable ID to the task being updated in `UpdateTaskWithOptions` before cycle detection, when `BlockedBy` is being set
- `internal/task/batch.go:290-302` - Auto-assign stable ID to the task being validated in `validateOperation` so cycle detection always runs during batch validation

**Approach rationale:** A task with `BlockedBy` must have a `StableID` to be properly indexed in the dependency graph. Auto-assigning ensures consistency regardless of whether the task was originally created with or without extended fields.

**Alternatives considered:**
- Requiring stable IDs at task creation time - Rejected because it would break existing workflows and add overhead to tasks without dependencies

## Regression Test

**Test file:** `internal/task/batch_race_test.go`
**Test name:** `TestBatchRace_UpdateWithDeps/update_task_without_stable_ID_to_add_blocked_by`

**What it verifies:** When a task without a stable ID is updated to add `blocked_by` dependencies, both the dependency target AND the task being updated receive stable IDs.

**Run command:** `go test -run TestBatchRace -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/operations.go` | Auto-assign stable ID in `UpdateTaskWithOptions` when setting `BlockedBy` |
| `internal/task/batch.go` | Auto-assign stable ID in `validateOperation` for cycle detection |
| `internal/task/batch_race_test.go` | Regression tests for batch dependency scenarios |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Integration tests pass
- [x] Linters/validators pass

**Manual verification:**
- Verified stable IDs are assigned to both dependency targets and the task being updated
- Verified cycle detection works for tasks that start without stable IDs

## Prevention

**Recommendations to avoid similar bugs:**
- When a task gains dependency metadata (`BlockedBy`), always ensure it also has a `StableID` for proper indexing
- Consider adding an invariant check: any task with non-empty `BlockedBy` must have a non-empty `StableID`

## Related

- T-23: Race condition for batch with tasks and dependencies
- T-59: Auto-assign stable IDs to dependency targets (partial fix - targets only)
