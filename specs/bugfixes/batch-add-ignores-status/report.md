# Bugfix Report: Batch Add Silently Ignores Requested Status

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1996

## Description of the Issue

A batch `add` with a `status` field reported `success: true, applied: 1` but always created the task as pending.

**Reproduction steps:**
1. `tasks.md` with one task.
2. `rune batch tasks.md --input '{"operations":[{"type":"add","title":"Requested completed","status":2}]}' --format json`
3. Response is `success:true, applied:1`; the new task is `- [ ]` (pending).

**Impact:** Medium. Callers believed a status was applied when it was not.

## Investigation Summary

- **Code inspected:** `internal/task/batch.go` (`Operation`, `validateOperation`, `applyOperation`, `applyOperationWithPhases`, `addTaskFromOperation`), `cmd/add.go`, `docs/json-api.md`.
- **Findings:** `Operation.Status` is shared by all operation types. The add case of `validateOperation` never checks it, and both add paths (`AddTaskWithPhases` → `addTaskFromOperation`) create pending tasks without reading it. The `add` CLI command has no status option, and the JSON schema does not describe status as valid for add.

## Discovered Root Cause

`status` is only meaningful for `update`, but nothing rejected it on `add`, so it was decoded and then dropped.

**Defect type:** Missing validation

## Resolution for the Issue

**Changes made:**
- `internal/task/batch.go` (`validateOperation`, add case) - reject an add operation that carries a status, with an error that points to using an update operation. Validation runs before any operation is applied, so the batch fails atomically, and the command exits 1 (via the T-1789 change in the base branch).

**Approach rationale:** Rejecting is the narrowest fix that satisfies the ticket's "never report success while ignoring a recognised field". It matches the `add` CLI, which has no status option, and avoids new auto-complete semantics (e.g. a completed child added under a pending parent).

**Alternatives considered:**
- Applying the status on add - rejected for now: it needs decisions on parent auto-completion and both add paths (including `AddTaskWithPhases`, which open PR #116 also changes). Add + update in one batch already achieves it.

## Regression Test

**Test files:** `internal/task/batch_operations_test.go`, `cmd/batch_test.go`
**Test names:** `TestExecuteBatch_AddRejectsStatus` (non-phased and phased), `TestBatchCommand_AddWithStatusFails`

**What it verifies:** The batch is rejected with a status error, nothing is applied or written, and the command returns an error.

**Run command:** `go test -run 'AddRejectsStatus|AddWithStatusFails' ./internal/task ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/batch.go` | Reject status on add |
| `internal/task/batch_operations_test.go`, `cmd/batch_test.go` | Regression tests |
| `docs/json-api.md`, `CHANGELOG.md` | Documented |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes (except `TestCreateBackup/error_handling_for_read_failures` and `TestIntegrationRenumber/renumber_write_failure`, which fail identically on main because the container runs as root)
- [x] Linters pass

**Manual verification:**
- Built binary: the reproduction exits 1 with the status error and leaves the file unchanged; add followed by `update` with `status: 2` in one batch succeeds and writes `- [x]`.
