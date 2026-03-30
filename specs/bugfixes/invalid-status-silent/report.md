# Bugfix Report: Invalid Status Filter Silent Match-All

**Date:** 2025-07-15
**Status:** Fixed
**Ticket:** T-638

## Description of the Issue

When users passed an invalid value to the `--filter` flag on `list` or the `--status` flag on `find`, the command silently treated it as a match-all filter, returning all tasks. This was confusing because a typo like `--filter complete` (instead of `completed`) would show all tasks with no indication that the filter was ignored.

**Reproduction steps:**
1. Create a task file with pending and completed tasks
2. Run `rune list tasks.md --filter bogus`
3. Observe that ALL tasks are returned instead of an error

**Impact:** Medium — users relying on status filtering could get incorrect results from typos without any warning.

## Investigation Summary

- **Symptoms examined:** Invalid status strings passed to `--filter`/`--status` produced output identical to no filter at all
- **Code inspected:** `cmd/list.go` (matchesStatusFilter, runList), `cmd/find.go` (runFind, applyAdditionalFilters)
- **Hypotheses tested:** Confirmed the `default` case in `matchesStatusFilter()` returned `true`

## Discovered Root Cause

The `matchesStatusFilter()` function in `cmd/list.go:172` used `default: return true` in its switch statement. This meant any unrecognised status string (typos, garbage) would match every task, effectively disabling the filter.

**Defect type:** Missing input validation

**Why it occurred:** The function was designed to handle the "no filter" case (empty string) and valid values, but the default arm was written as a catch-all pass-through rather than a rejection.

**Contributing factors:** No flag-level validation in Cobra, and no early validation in the command runners.

## Resolution for the Issue

**Changes made:**
- `cmd/list.go` — Added `validateStatusFilter()` function that rejects unrecognised values with a clear error listing valid options. Called at the start of `runList()`. Changed `matchesStatusFilter()` default from `return true` to `return false`.
- `cmd/find.go` — Added `validateStatusFilter()` call at the start of `runFind()`.

**Approach rationale:** Early validation with a clear error message is the most user-friendly approach. Changing the default in `matchesStatusFilter` to `false` provides defence-in-depth.

**Alternatives considered:**
- Cobra `ValidArgsFunction` — only helps shell completion, doesn't prevent invalid values
- Case-insensitive matching — would mask the issue but could introduce ambiguity

## Regression Test

**Test file:** `cmd/list_test.go`, `cmd/find_test.go`
**Test names:** `TestValidateStatusFilter`, `TestMatchesStatusFilterRejectsUnknown`, `TestListInvalidFilterReturnsError`, `TestFindInvalidStatusFilterReturnsError`

**What it verifies:** Invalid status values are rejected with errors; valid values (including empty) are accepted; the full command path returns an error for invalid filters.

**Run command:** `go test -run 'TestValidateStatusFilter|TestMatchesStatusFilterRejectsUnknown|TestListInvalidFilterReturnsError|TestFindInvalidStatusFilterReturnsError' -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/list.go` | Added `validateStatusFilter()`, changed `matchesStatusFilter` default to `false`, added validation call in `runList` |
| `cmd/find.go` | Added validation call in `runFind` |
| `cmd/list_test.go` | Added 3 regression tests |
| `cmd/find_test.go` | Added 1 regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Code formatted with `make fmt`

**Manual verification:**
- `rune list file.md --filter bogus` → error with valid options listed
- `rune find file.md -p "x" --status bogus` → error with valid options listed
- `rune list file.md --filter pending` → works correctly
- `rune find file.md -p "x" --status completed` → works correctly

## Prevention

**Recommendations to avoid similar bugs:**
- Validate CLI flag values early in command runners before doing any work
- Use `default: return false` in filter-matching switch statements (fail-closed)
- Consider adding a shared `RegisterStatusFlagValidation` helper for future flags with constrained values
