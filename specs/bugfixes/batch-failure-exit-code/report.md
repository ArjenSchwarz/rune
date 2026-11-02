# Bugfix Report: Failed Batch Operations Return Exit Code Zero

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1789

## Description of the Issue

A batch rejected by validation printed `success: false` but the process exited 0, so scripts and CI treated it as successful.

**Reproduction steps:**
1. Create `tasks.md` with one task.
2. Run `rune batch tasks.md --input '{"operations":[{"type":"remove","id":"999"}]}' --format json`.
3. Output shows `success: false`, `applied: 0`, `operation 1: task 999 not found`; `echo $?` prints `0`.

**Impact:** High. Any automation relying on the exit status silently continued after a failed batch.

## Investigation Summary

- **Symptoms examined:** JSON and table output both reported the failure; exit status was 0 in both.
- **Code inspected:** `internal/task/batch.go` (`runBatch`, `ExecuteBatch`, `ExecuteBatchWithPhases`), `cmd/batch.go` (`runBatch`, `outputBatchJSON`, `outputBatchText`), `cmd/root.go` (`Execute`).
- **Hypotheses tested:** The task layer reports operation failures through `BatchResponse.Success` rather than an error, which is intended (the response must still be rendered). The command layer rendered the response and returned `nil` regardless of `Success`.

## Discovered Root Cause

`cmd/batch.go` returned `nil` after rendering, and `outputBatchText` also returned `nil` on the failure branch, so `rootCmd.Execute()` never saw an error and `os.Exit(1)` was never reached.

**Defect type:** Missing error propagation

**Why it occurred:** Operation failures are data in `BatchResponse`, not Go errors, and nothing translated `Success == false` into a command error.

## Resolution for the Issue

**Changes made:**
- `cmd/batch.go` - After rendering the response, if `response.Success` is false, set `SilenceErrors`/`SilenceUsage` on the command and return `errBatchFailed`. The structured output is unchanged; cobra does not print the error or usage a second time, and `Execute` exits 1.

**Approach rationale:** Keeps the rendered response as the single report of what failed while giving the process a non-zero status.

**Alternatives considered:**
- Returning an error from `ExecuteBatch` - rejected: callers need the response to render it, and the task layer is used by tests that inspect `Success`.

## Regression Test

**Test file:** `cmd/batch_test.go`
**Test name:** `TestBatchCommand_FailedBatchReturnsNonZero` (json and table); `TestBatchCommand_ValidationFailures/invalid_operation` now expects an error.

**What it verifies:** A failed batch returns an error, still renders the response, writes nothing to stderr and leaves the file unchanged.

**Run command:** `go test -run 'TestBatchCommand_FailedBatchReturnsNonZero|TestBatchCommand_ValidationFailures' ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/batch.go` | Return a silenced error after rendering a failed batch |
| `cmd/batch_test.go` | Regression test; updated expectation for invalid operation |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (except pre-existing root-permission failures `TestCreateBackup` and `TestIntegrationRenumber/renumber_write_failure`, which also fail on main)
- [x] Linters/validators pass

**Manual verification:**
- Built binary: the reproduction now prints the same JSON and exits 1; the table-format failure also exits 1.

## Prevention

- When a command reports failure in structured output, also return a (silenced) error so the exit status matches.
