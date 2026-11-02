# Bugfix Report: Uncomplete/Progress Leave Ancestors Completed

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1456

## Description of the Issue

`uncomplete` and `progress` changed only the target task's status, so a parent
that had been auto-completed stayed `[x]` while a child became `[ ]` or `[-]`.

**Reproduction steps:**
1. `- [ ] 1. Parent` with `[x] 1.1` and `[ ] 1.2`.
2. `rune complete tasks.md 1.2` (auto-completes 1).
3. `rune uncomplete tasks.md 1.1` → file shows `[x] 1. Parent` over `[ ] 1.1`. `progress 1.1` gives the same shape with `[-]`.

**Impact:** Breaks the invariant implied by automatic parent completion; `next` and claim paths then see completed parents with incomplete descendants.

## Investigation Summary

- **Code inspected:** `cmd/uncomplete.go`, `cmd/progress.go`, `internal/task/autocomplete.go`, `internal/task/operations.go` (`UpdateStatus`).
- **Finding:** `AutoCompleteParents` handles the upward direction for completion; there was no counterpart for rollbacks.

## Discovered Root Cause

**Defect type:** Missing state propagation.

**Why it occurred:** Upward completion was added for `complete` without a matching reopen step for the commands that move a task out of the completed state.

## Resolution for the Issue

**Changes made:**
- `internal/task/autocomplete.go` - New `TaskList.ReopenCompletedAncestors(taskID)` walks up the ancestor chain and sets every completed ancestor to pending.
- `cmd/uncomplete.go`, `cmd/progress.go` - Call it after `UpdateStatus`.

**Approach rationale:** Reopened ancestors become pending in both commands: the parent itself was not started by the user, and pending is the neutral "not done" state. The whole chain is walked so a completed grandparent is reopened even when the direct parent was already open.

**Alternatives considered:**
- Putting the logic in `UpdateStatus` - would also change batch `update` status semantics and other callers; kept out of scope.

## Regression Test

**Test file:** `cmd/status_rollback_test.go`
**Test name:** `TestStatusRollbackReopensCompletedAncestors`

**What it verifies:** For both commands, rolling back `1.1.1` reopens `1.1` and `1` and leaves siblings and unrelated tasks completed.

## Affected Files

| File | Change |
|------|--------|
| `internal/task/autocomplete.go` | `ReopenCompletedAncestors` |
| `cmd/uncomplete.go`, `cmd/progress.go` | Reopen ancestors after status change |

## Verification

- [x] Regression test passes
- [x] Full test suite passes (except two pre-existing root-only permission tests)
- [x] `make lint` clean

## Remaining limitations

- Batch `update` operations with a `status` field do not reopen ancestors; the ticket scoped the fix to the `uncomplete`/`progress` commands.
- Command output does not list reopened ancestors.
