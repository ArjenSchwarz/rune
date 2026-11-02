# Bugfix Report: Comma-only Requirements Input Silently Clears Links

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-2023

## Description of the Issue

`rune update tasks.md 1 --requirements ','` exited 0, reported
`fields_updated: ["requirements"]` and removed the `Requirements:` line.
`rune add --requirements ','` created an unlinked task.

## Discovered Root Cause

**Defect type:** Missing validation.

`parseRequirementIDs` drops empty tokens and returns a non-nil empty slice. The
per-ID validation loop then had nothing to check, and the empty slice reached
`UpdateTask`/`UpdateTaskWithOptions` as the "clear" value. This is the
requirements analogue of T-1906 (`--blocked-by ','`).

## Resolution for the Issue

**Changes made:**
- `cmd/update.go` - Reject a non-empty `--requirements` value that parses to zero IDs before the file is touched. `--requirements ""` still clears (T-1974) and `--clear-requirements` is unchanged.
- `cmd/add.go` - Same rejection for `add --requirements` (covers the phase and extended paths, which share this parsing).

## Regression Test

**Test file:** `cmd/update_add_flag_regressions_test.go`
**Test name:** `TestCommaOnlyRequirementsRejected` (`","`, `" , "`, `",,,"` for both add and update; asserts an error and a byte-for-byte unchanged file)

## Verification

- [x] Regression test passes
- [x] Full test suite passes (except two pre-existing root-only permission tests)
- [x] `make lint` clean
