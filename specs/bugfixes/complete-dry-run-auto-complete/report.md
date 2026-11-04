# Bugfix Report: complete-dry-run-auto-complete

**Date:** 2026-11-02
**Status:** Fixed
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

**Changes made:**
- `cmd/complete.go` (`runComplete`) — Moved `UpdateStatus` and `AutoCompleteParents` before the dry-run branch so they run in both modes; only `WriteFile` is skipped in dry-run. The current status is captured before mutation because `targetTask` points into the list. Dry-run JSON now sets `AutoCompleted`; markdown and table output add a line listing the parents that would be auto-completed.
- `CHANGELOG.md` — Added a Fixed entry.

**Approach rationale:** `tl` is parsed fresh from disk, so it is already an in-memory copy that is discarded unless written. Running the same mutation calls as the live path makes the preview identical to the real result by construction, with no separate simulation logic to drift out of sync.

**Alternatives considered:**
- Deep-copy the list (as batch dry-run does with `deepCopy`) before mutating — not needed; batch copies because it may keep the original on failure, while `complete` never writes `tl` in dry-run.
- Compute would-be auto-completions without mutating — duplicates `AutoCompleteParents` logic and risks divergence.

## Regression Test

**Test file:** `cmd/complete_test.go`
**Test name:** `TestRunCompleteDryRunReportsAutoCompletedParents`

**What it verifies:** For a parent whose last pending child is completed in dry-run, JSON output lists the parent in `auto_completed`, markdown and table output preview the auto-completion, and the file on disk is unchanged.

**Run command:** `go test -run TestRunCompleteDryRunReportsAutoCompletedParents -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/complete.go` | Simulate status update and parent auto-completion in dry-run; report them in all formats |
| `cmd/complete_test.go` | Regression test across JSON, markdown and table formats |
| `CHANGELOG.md` | Fixed entry |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes, except `TestCreateBackup/error_handling_for_read_failures` and `TestIntegrationRenumber/renumber_write_failure`, which rely on file permissions and fail identically on `main` when run as root
- [x] Linters/validators pass (`make lint`: 0 issues)

**Manual verification:**
- Ran the ticket's reproduction against `examples/simple.md`: dry-run JSON reports `"auto_completed": ["2"]`, matching the subsequent live run, and the file is byte-for-byte unchanged after dry-run. Table and markdown output list the auto-completion.

## Prevention

**Recommendations to avoid similar bugs:**
- Implement dry-run as "run the operation, skip the write" rather than a separate descriptive branch, so new side effects of the live path show up in previews automatically.
- `uncomplete` and `progress` have no cascading effects today; if one is added, follow the same pattern.

## Related

- T-1934
- `specs/bugfixes/complete-dry-run-json/` (T-725) — earlier fix adding format-aware dry-run output to `complete`
