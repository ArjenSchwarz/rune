# Bugfix Report: RemoveTaskWithDependents Skips Dependents Without Stable IDs

**Date:** 2026-03-13
**Status:** Fixed

## Description of the Issue

When removing a task that has dependents (other tasks that reference it in their `BlockedBy` list), `RemoveTaskWithDependents` fails to clean up the `BlockedBy` references if the dependent task does not have its own `StableID`. The dependent task remains blocked by a now-deleted task.

**Reproduction steps:**
1. Create task A with a stable ID (e.g., via `AddTaskWithOptions`)
2. Add task B without a stable ID but with `BlockedBy` referencing A's stable ID (e.g., parsed from markdown)
3. Call `RemoveTaskWithDependents(A)` — B's `BlockedBy` still contains A's stable ID

**Impact:** Tasks remain blocked by deleted tasks, causing them to appear perpetually blocked in `next` command output and stream navigation. This affects any workflow where tasks are parsed from markdown with `blocked_by` metadata but without explicit stable IDs assigned to the dependent.

## Investigation Summary

- **Symptoms examined:** After removing a blocker task, dependent tasks without their own StableID retain stale `BlockedBy` references
- **Code inspected:** `BuildDependencyIndex` in `dependencies.go`, `RemoveTaskWithDependents` and `removeFromBlockedByLists` in `operations.go`
- **Hypotheses tested:** The dependency index registration path and the conditional cleanup path were both examined

## Discovered Root Cause

Two cooperating defects:

**Defect 1 — `BuildDependencyIndex` (dependencies.go:33-39):** The dependents map is only populated inside the `if task.StableID != ""` block. Tasks without a StableID but with BlockedBy references are never registered as dependents. This means `GetDependents()` returns an empty list for the blocker, even though a dependent exists.

**Defect 2 — `RemoveTaskWithDependents` (operations.go:1100-1109):** The method only calls `removeFromBlockedByLists` when `GetDependents` returns a non-empty list. Since Defect 1 causes `GetDependents` to miss dependents without StableIDs, the cleanup is skipped entirely.

**Defect type:** Logic error — conditional cleanup path with incomplete index

**Why it occurred:** The dependency index was designed around the assumption that all tasks participating in dependency relationships would have StableIDs. However, tasks parsed from markdown may have `BlockedBy` references without having their own StableID assigned.

**Contributing factors:** The `BlockedBy` field stores stable IDs of blockers, but the dependent task itself doesn't need a StableID to reference blockers. The index building code conflated "having a StableID" with "participating in dependencies."

## Resolution for the Issue

**Changes made:**
- `internal/task/dependencies.go:22-49` — Extended `BuildDependencyIndex` to register dependents even when the dependent task lacks a StableID, using the hierarchical ID as fallback identifier
- `internal/task/operations.go:1099-1109` — Changed `RemoveTaskWithDependents` to always call `removeFromBlockedByLists` when removing a task with a StableID, regardless of what the dependency index reports

**Approach rationale:** The two-pronged fix addresses both the index gap and the fragile conditional. Making `removeFromBlockedByLists` unconditional is cheap (single tree walk) and eliminates the class of bugs where the index might miss dependents for any reason.

**Alternatives considered:**
- Only fix `BuildDependencyIndex` — would fix this case but leaves `RemoveTaskWithDependents` fragile if other index gaps are found later
- Only make `removeFromBlockedByLists` unconditional — would fix the cleanup but leaves the index returning incorrect data for other callers (e.g., `GetDependents` used elsewhere)

## Regression Test

**Test file:** `internal/task/operations_extended_test.go`
**Test name:** `TestRemoveTaskWithDependents/RemoveTask_cleans_up_dependents_without_stable_IDs`

**What it verifies:** When a task without a StableID has BlockedBy references to a task being removed, the BlockedBy references are cleaned up after removal.

**Additional test file:** `internal/task/dependencies_test.go`
**Test name:** `TestBuildDependencyIndex/dependent_without_stable_id`

**What it verifies:** `BuildDependencyIndex` registers dependents that lack StableIDs in the dependents map using their hierarchical ID.

**Run command:** `go test -run "TestRemoveTaskWithDependents/RemoveTask_cleans_up_dependents_without_stable_IDs|TestBuildDependencyIndex/dependent_without_stable_id" ./internal/task/ -v`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/dependencies.go` | Register dependents without StableID in index |
| `internal/task/operations.go` | Always call removeFromBlockedByLists for tasks with StableID |
| `internal/task/dependencies_test.go` | Add regression test for index building |
| `internal/task/operations_extended_test.go` | Add regression test for removal cleanup |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When building indexes, consider that not all fields used for lookup may be populated on all participating objects
- Prefer unconditional cleanup operations over conditional ones gated on index lookups — the cost is minimal and eliminates a class of bugs

## Related

- Transit ticket T-422
