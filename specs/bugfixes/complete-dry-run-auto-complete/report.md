# Bugfix Report: complete-dry-run-auto-complete

**Date:** 2026-11-02
**Status:** Investigating
**Ticket:** T-1934

## Description of the Issue

`rune complete --dry-run` previews only the directly targeted task. It omits parent tasks that the live command auto-completes when the target is their last incomplete child.

**Reproduction steps:**
1. `cp examples/simple.md tasks.md`
2. `rune complete tasks.md 2.3` (leaves 2.2 as the last pending child of task 2)
3. `rune complete tasks.md 2.2 --dry-run --format json` — no `auto_completed` field
4. `rune complete tasks.md 2.2 --format json` — reports `"auto_completed": ["2"]`

**Impact:** Dry-run is the safety preview for a mutation, but it under-reports the changes. Users and agents relying on it cannot see that parent tasks will also change status. Affects all output formats.

## Investigation Summary

- **Symptoms examined:** Dry-run output in JSON, markdown and table formats lacks any auto-completion information; live output includes it.
- **Code inspected:** `cmd/complete.go` (`runComplete`), `internal/task/operations.go` (`UpdateStatus`, `AutoCompleteParents`), `internal/task/batch.go` (batch dry-run, which runs operations on a copy and reports `AutoCompleted`).
- **Hypotheses tested:** Whether `AutoCompleteParents` behaves differently in dry-run — ruled out; it is never called in dry-run.

## Discovered Root Cause

The dry-run branch in `runComplete` returns before `UpdateStatus` and `AutoCompleteParents` run, so there is no simulated result to report and `CompleteResponse.AutoCompleted` is never populated.

**Defect type:** Incomplete simulation in a preview code path

**Why it occurred:** Dry-run was implemented as "describe the target task" rather than "perform the operation without persisting". When auto-completion was added to the live path, the early-return dry-run branch was not updated.

**Contributing factors:** The dry-run output only ever echoed the target task, so no test exercised a case where the live command changes more than one task.

## Resolution for the Issue

_To be filled in after the fix._

## Regression Test

**Test file:** `cmd/complete_test.go`
**Test name:** `TestRunCompleteDryRunReportsAutoCompletedParents`

**What it verifies:** For a parent whose last pending child is completed in dry-run, JSON output lists the parent in `auto_completed`, markdown and table output preview the auto-completion, and the file on disk is unchanged.

**Run command:** `go test -run TestRunCompleteDryRunReportsAutoCompletedParents -v ./cmd`
