# Bugfix Report: Batch Update Partial Apply

**Date:** 2026-03-04
**Status:** Fixed

## Description of the Issue

Batch update operations could partially apply a status change before encountering and failing on invalid details or references. This violated the atomic "all succeed or all fail" guarantee of batch operations.

The `validateOperation` function did not validate detail or reference content (length limits, control characters) for update or add operations. These checks only happened later inside `UpdateTask`/`UpdateTaskWithOptions`. Meanwhile, `applyUpdateOperation` and `applyOperationWithPhases` applied the status change via `UpdateStatus` before calling those functions, so a failure on invalid details/references left the status already applied.

**Reproduction steps:**
1. Create a task list with at least one task
2. Execute a batch update operation that sets both a status change and an overlong detail (>1000 chars) or overlong reference (>500 chars) on the same task
3. Observe that `applyUpdateOperation` applies the status change, then fails on the detail/reference validation, leaving the task with a partially applied update

**Impact:** Violated batch atomicity guarantee. While `ExecuteBatch` protected the original task list via its test-copy-first approach, the partial apply was still a latent defect that could surface if the execution model changed.

## Investigation Summary

- **Symptoms examined:** `applyUpdateOperation` applies status via `UpdateStatus()` before calling `UpdateTask()`/`UpdateTaskWithOptions()`, which validate and may reject details/references
- **Code inspected:** `internal/task/batch.go` (validateOperation, applyUpdateOperation, applyOperationWithPhases), `internal/task/operations.go` (UpdateTask, UpdateTaskWithOptions, validateDetails, validateReferences)
- **Hypotheses tested:** Confirmed that `validateOperation` does not call `validateDetails`/`validateReferences` for either update or add operations, and that `applyUpdateOperation` applies status before the call that validates content

## Discovered Root Cause

Two related defects:

1. `validateOperation` did not validate detail/reference content for update or add operations, so invalid content passed validation and was only caught during apply
2. `applyUpdateOperation` and the update case in `applyOperationWithPhases` applied status before other fields, causing partial mutation when subsequent field validation failed

**Defect type:** Missing validation and incorrect operation ordering

**Why it occurred:** The validation function focused on structural checks (ID exists, type valid, title length, status range) but omitted content validation for details and references. The apply functions treated status as an independent step applied first.

**Contributing factors:** The test-copy-first approach in `ExecuteBatch` masked the partial-apply issue at the end-to-end level, since the original list was protected by the copy failing first.

## Resolution for the Issue

**Changes made:**
- `internal/task/batch.go` - Added `validateDetailsAndReferences` helper that calls `validateDetails` and `validateReferences` from operations.go
- `internal/task/batch.go` - Added calls to `validateDetailsAndReferences` in `validateOperation` for both the `add` and `update` cases
- `internal/task/batch.go` - Reordered `applyUpdateOperation` to apply status AFTER `UpdateTask`/`UpdateTaskWithOptions` succeed
- `internal/task/batch.go` - Reordered the update case in `applyOperationWithPhases` to apply status AFTER field updates succeed

**Approach rationale:** Two-layer fix: (1) catch invalid content early in validation so `applyOperation` never encounters it, and (2) reorder the apply function as defense-in-depth so that even if validation is bypassed, status is not partially applied.

**Alternatives considered:**
- Moving status into `UpdateTaskWithOptions` as a parameter - Rejected because it would change the API of a widely-used function for a batch-specific concern
- Only fixing validation without reordering apply - Rejected because defense-in-depth is important for atomicity guarantees

## Regression Test

**Test file:** `internal/task/batch_validation_test.go`
**Test names:**
- `TestValidateOperation_RejectsInvalidDetailsAndReferences` - validates that `validateOperation` catches invalid details/references for both update and add operations
- `TestApplyUpdateOperation_NoPartialStatusApply` - validates that `applyUpdateOperation` does not partially apply status when details/references are invalid
- `TestExecuteBatch_UpdateStatusNotAppliedBeforeInvalidDetails` - end-to-end test that the original task list is not modified when a batch update with status + invalid details/references fails

**What it verifies:** Invalid detail/reference content is rejected during validation, and status is never partially applied even if validation is bypassed.

**Run command:** `go test -run "TestValidateOperation_RejectsInvalidDetailsAndReferences|TestApplyUpdateOperation_NoPartialStatusApply|TestExecuteBatch_UpdateStatusNotAppliedBeforeInvalidDetails" -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/batch.go` | Added `validateDetailsAndReferences` helper; added validation calls in `validateOperation` for add and update; reordered status application in `applyUpdateOperation` and `applyOperationWithPhases` |
| `internal/task/batch_validation_test.go` | Added three regression test functions covering validation, apply ordering, and end-to-end atomicity |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed regression tests fail before fix (6 validateOperation failures, 4 partial-apply failures)
- Confirmed all tests pass after fix
- Ran `make check` (format, lint, test) with no issues

## Prevention

**Recommendations to avoid similar bugs:**
- When adding new validatable fields to operations, ensure `validateOperation` includes content validation for the field, not just structural checks
- Apply operations that can fail (content validation) before operations that mutate state (status changes)
- The test-copy-first pattern in `ExecuteBatch` is a good safety net but should not be relied on as the sole atomicity mechanism

## Related

- Transit ticket: T-323
