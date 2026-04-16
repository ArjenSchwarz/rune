# Bugfix Report: Remove Leaves Stale Blocker

**Date:** 2026-04-16
**Status:** Fixed
**Ticket:** T-806

## Description of the Issue

When removing a task that has a `StableID` referenced by other tasks' `BlockedBy` lists, the stale reference was not cleaned up. The dependent task continued to render `Blocked-by: <deleted-stable-id>`, pointing at a non-existent blocker.

**Reproduction steps:**
1. Create task 1 with stable ID `abc1234`
2. Create task 2 with `Blocked-by: abc1234`
3. Run `rune remove <file> 1`
4. Observe: task 2 (renumbered to 1) still renders `Blocked-by: abc1234`

**Impact:** Medium — tasks could be permanently blocked by deleted tasks, with no way to unblock them short of manually editing the markdown.

## Investigation Summary

- **Symptoms examined:** Stale `BlockedBy` references after task removal
- **Code inspected:** `cmd/remove.go`, `internal/task/operations.go`, `internal/task/batch.go`
- **Hypotheses tested:** The correct cleanup helper `RemoveTaskWithDependents` exists but is bypassed

## Discovered Root Cause

Three remove code paths called `RemoveTask()` instead of `RemoveTaskWithDependents()`:

1. `RemoveTaskWithPhases()` in `operations.go` — used by `cmd/remove.go`
2. `applyOperation()` in `batch.go` — used by non-phase batch removes
3. `applyOperationWithPhases()` in `batch.go` — used by phase-aware batch removes

`RemoveTask()` only removes the task and renumbers. `RemoveTaskWithDependents()` additionally walks the entire task tree and removes the deleted task's `StableID` from all `BlockedBy` lists before removing.

**Defect type:** Missing function call — correct helper bypassed

**Why it occurred:** `RemoveTaskWithDependents` was added after the original remove paths were written, but those paths were never updated to use it.

## Resolution for the Issue

**Changes made:**
- `internal/task/operations.go:720-728` — `RemoveTaskWithPhases` now calls `RemoveTaskWithDependents` instead of `RemoveTask`
- `internal/task/batch.go:410` — `applyOperation` remove case now calls `RemoveTaskWithDependents`
- `internal/task/batch.go:754` — `applyOperationWithPhases` remove case now calls `RemoveTaskWithDependents`

**Approach rationale:** The simplest correct fix — route all remove paths through the existing helper that already handles dependency cleanup. Warnings from `RemoveTaskWithDependents` are discarded since the callers have no warning channel, and the cleanup is silent by design.

## Regression Test

**Test file:** `internal/task/operations_extended_test.go`
**Test names:** `TestRemoveTaskWithPhases_CleansBlockedBy`, `TestBatchRemove_CleansBlockedBy`

**What they verify:** After removing a blocker task, the remaining dependent task's `BlockedBy` list is empty.

**Run command:** `go test -run "TestRemoveTaskWithPhases_CleansBlockedBy|TestBatchRemove_CleansBlockedBy" ./internal/task/ -v`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/operations.go` | `RemoveTaskWithPhases` calls `RemoveTaskWithDependents` |
| `internal/task/batch.go` | Both batch remove paths call `RemoveTaskWithDependents` |
| `internal/task/operations_extended_test.go` | Added two regression tests |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes (`go test ./...`)

## Prevention

**Recommendations to avoid similar bugs:**
- `RemoveTask()` should be unexported or deprecated — all callers should use `RemoveTaskWithDependents()` to ensure dependency cleanup is never bypassed.
- Consider adding a linter rule or code review checklist item: "Does this remove path go through `RemoveTaskWithDependents`?"
