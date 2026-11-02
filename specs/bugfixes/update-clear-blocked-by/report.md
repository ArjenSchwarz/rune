# Bugfix Report: Update Cannot Clear Blocked-by Dependencies

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1493

## Description of the Issue

`rune update tasks.md 2 --blocked-by ""` failed with "at least one update flag
must be provided". Combined with `--title`, the title changed but the
`Blocked-by:` line stayed and JSON `fields_updated` only listed `title`.

## Discovered Root Cause

**Defect type:** Flag presence checked by value instead of `Changed`.

`cmd/update.go` used `updateBlockedBy != ""` to decide whether the flag was
given, so an explicitly empty value was indistinguishable from an absent flag.
`UpdateTaskWithOptions` already treats a non-nil empty `BlockedBy` as "clear".

## Resolution for the Issue

**Changes made:**
- `cmd/update.go` - `blockedBySet` uses `cmd.Flags().Changed("blocked-by")` (as `--requirements` does for T-1974). An empty value passes an empty slice, which clears dependencies, and `blocked-by` is reported in `fields_updated`. A non-empty value with no IDs (`","`) is still rejected (T-1906 behaviour preserved). Flag help, `Long` text and README document the empty form.

**Alternatives considered:**
- A separate `--clear-blocked-by` flag - the ticket allowed either; the empty-value form matches the existing `--requirements ""` contract and needs no new flag.

## Regression Test

**Test file:** `cmd/update_add_flag_regressions_test.go`
**Test name:** `TestUpdateEmptyBlockedByClears` (empty flag alone, and with `--title`)

Existing `TestRunUpdateBlockedByCommaOnlyRejected` still passes.

## Verification

- [x] Regression test passes
- [x] Full test suite passes (except two pre-existing root-only permission tests)
- [x] `make lint` clean
