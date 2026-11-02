# Bugfix Report: Update Dry Run Omits and Skips Validation of Extended Fields

**Date:** 2026-11-02
**Status:** Fixed
**Tickets:** T-1381, T-1750

## Description of the Issue

`rune update --dry-run` returned before building `UpdateOptions`, so it neither
previewed nor validated `--stream`, `--blocked-by`, `--owner` or `--release`.

**Reproduction steps:**
1. Create `- [ ] 1. Blocker` / `- [ ] 2. Target`.
2. `rune update tasks.md 2 --dry-run --stream 3 --blocked-by 1 --owner agent-a` printed only the current title/details/references/requirements.
3. `rune update tasks.md 1 --blocked-by 999 --dry-run` exited 0, while the same command without `--dry-run` failed with `task 999 not found`.

**Impact:** Automation using dry-run as a validation gate got false success, and the preview hid mutations a real run would apply.

The `add` half of both tickets (`add --position 1 --dry-run` reporting ID 3, `add --blocked-by 999 --dry-run` succeeding) was already fixed on main by #104, which applies the add in memory before the dry-run branch. This change only adds a regression test for the add validation case.

## Investigation Summary

- **Code inspected:** `cmd/update.go` (dry-run branch), `cmd/add.go`, `internal/task/operations.go` (`UpdateTaskWithOptions`).
- **Finding:** The update dry-run branch sat before the mutation, so validation that lives in `UpdateTaskWithOptions`/`UpdateTask` (stable-ID resolution of blockers, stream range, title/detail checks) never ran.

## Discovered Root Cause

**Defect type:** Missing validation / divergent code path.

**Why it occurred:** The dry-run preview predates the extended fields and was never updated when they were added; it also short-circuited before the shared mutation code.

## Resolution for the Issue

**Changes made:**
- `cmd/update.go` - The update is now applied to the in-memory task list first (same code path as a real run); the dry-run branch runs after it and returns before writing. The preview captures current values beforehand and now prints `New stream`, `New blocked-by`, `New owner` / `(released)`.

**Approach rationale:** Mirrors the fix #104 applied to `add`: one code path for validation, the dry run just skips the write. The task list is parsed fresh per invocation, so mutating it in memory has no side effects.

**Alternatives considered:**
- Duplicating validation in the dry-run branch - would drift again.

## Regression Test

**Test file:** `cmd/update_add_flag_regressions_test.go`
**Test names:** `TestUpdateDryRunPreviewsExtendedFields`, `TestUpdateDryRunValidatesDependencies`, `TestAddDryRunValidatesDependencies`

**Run command:** `go test ./cmd -run 'DryRun(Previews|Validates)'`

## Affected Files

| File | Change |
|------|--------|
| `cmd/update.go` | Dry-run moved after in-memory update; previews extended fields |
| `cmd/update_add_flag_regressions_test.go` | Regression tests |

## Verification

- [x] Regression tests pass
- [x] Full test suite passes (except two pre-existing root-only permission tests, see PR)
- [x] `make lint` clean

## Prevention

- Dry-run branches should run the real mutation on the in-memory copy and only skip the write.
