# Bugfix Report: Batch Reporting Accuracy

**Date:** 2026-11-04
**Status:** Fixed
**Tickets:** T-1776, T-1780, T-1860, T-1888, T-1933

## Description of the Issues

Five low-priority defects where a batch or remove succeeded but reported or ordered something wrongly.

| Ticket | Symptom |
|--------|---------|
| T-1776 | `auto_completed` listed a parent that a later operation in the same batch reopened, removed, or gave an incomplete child. After a later remove renumbered tasks, it listed the parent's old ID. |
| T-1780 | `operation N` in errors was the position after sorting, so a reordered run of positioned adds or removes blamed the wrong input operation. |
| T-1860 | Removing a blocker cleaned the dependents' `Blocked-by` references but never reported it (requirement 2.7). |
| T-1888 | Dry-run `preview` omitted YAML front matter that the live write keeps. |
| T-1933 | Positioned adds that targeted the same position were saved in reverse request order. |

**Reproduction:** each case is a test in `internal/task/batch_reporting_regression_test.go` or `cmd/blocker_removal_warning_test.go`. Every case failed before the fix, apart from the guard noted under T-1780.

## Investigation Summary

- **T-1776:** `applyOperationWithAutoComplete` and `applyOperationWithPhases` added IDs to a map whenever `AutoCompleteParents` ran, and nothing ever removed them.
- **T-1780:** `runBatch` ranged over `sortedOps` and formatted `i+1`. The ticket's exact example (a missing update before a positioned add) already reports `operation 1` on main, because T-1874 stopped positioned adds being hoisted across other kinds. The bug remained inside reordered runs (`[add pos 1 (invalid), add pos 2]`, `[remove 1, remove 9]`). The ticket's example is kept as a guard case.
- **T-1860:** `RemoveTaskWithDependents` returned the warning, but `RemoveTaskWithPhases` and both batch remove paths discarded it.
- **T-1888:** both preview branches rendered only the task markdown.
- **T-1933:** the sort used `sort.SliceStable` with a position-only key, so equal positions ran in request order. Each insert went before the previous one.

## Resolution

- `sortOperationsForExecution` returns `indexedOperation` (the operation plus its request index). Errors use that index. Positioned adds with equal positions sort by descending request index.
- Auto-completion moved from the two apply functions into `runBatch` (`markAutoCompletedParents`). It flags auto-completed tasks with an unexported `Task.autoCompleted` field, which moves with the task when it is renumbered. `collectAutoCompleted` reports flagged tasks that are still `Completed` and whose descendants are all complete, using their final IDs, and then clears every flag. An explicit status update clears the target's own flag, so a parent that was reopened and then explicitly completed is not reported as auto-completed.
- Removes return `[]Warning` (`dependents_removed`, with the removed task's ID), using the existing `Warning` type and the shape in `docs/json-api.md`. `BatchResponse` and `RemoveResponse` gained `warnings`. Table and Markdown output print `Warning: <message>` to stderr. `RemoveTaskWithPhases` now returns the warnings.
- Both dry-run previews pass through `SerializeWithFrontMatter`, so the preview equals the file a live run writes.

**Alternatives considered:**
- *T-1776:* filtering the event map against the final state is simpler, but it reports the wrong task, or drops a correct one, when a later operation renumbers. Remapping recorded IDs on every add and remove needs hierarchical arithmetic in several places. A flag on the task is the smallest change that stays correct.
- *T-1860:* plain-string warnings matched one design snippet, but `docs/json-api.md` already documents warning objects with the `dependents_removed` code.

## Regression Tests

- `internal/task/batch_reporting_regression_test.go`: auto-complete reversal, remove, added child, explicit re-completion and renumbering cases, each live and dry-run on both the plain and phased paths; operation indices; equal-position adds at the root and under one parent; batch and `RemoveTaskWithPhases` warnings; the front-matter preview compared with the written file.
- `cmd/blocker_removal_warning_test.go`: `remove` JSON, table and Markdown warnings; `batch` JSON and table warnings.

**Run command:** `go test -run 'TestBatch(AutoCompleted|Errors|EqualPosition|RemoveBlocker|DryRunPreview)|TestRemove(TaskWithPhases|Blocker)' ./internal/task ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/batch.go` | Indexed sort and tie-break, auto-complete tracking, warnings, front-matter preview |
| `internal/task/task.go` | Unexported `autoCompleted` flag |
| `internal/task/operations.go` | `RemoveTaskWithPhases` returns warnings; `removeTaskWithWarnings` helper |
| `cmd/remove.go`, `cmd/batch.go`, `cmd/format.go` | `warnings` JSON field and stderr output |
| `docs/json-api.md`, `CHANGELOG.md`, `docs/agent-notes/batch-command.md` | Documentation |
