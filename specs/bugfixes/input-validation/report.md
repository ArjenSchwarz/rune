# Bugfix Report: Input Validation

**Date:** 2026-11-04
**Status:** Fixed
**Tickets:** T-1243, T-1754, T-1782, T-1793, T-2145, T-2325

## Description of the Issues

Six low-priority cases where input was accepted, then silently dropped, reinterpreted or misreported.

| Ticket | Symptom |
|--------|---------|
| T-1243 | Commands other than `batch` accepted any `--format` value and fell back to table output. |
| T-1754 | `add --parent X --phase P --format json` reported `phase: P`, although the subtask stays under its parent, in whatever phase that is. |
| T-1782 | An owner of `"   "` was written as `- Owner:    ` and read back as a detail, leaving the task unowned. Surrounding whitespace was trimmed on read. |
| T-1793 | Title, detail and reference limits counted UTF-8 bytes, so 300 `é` characters exceeded the 500-character title limit. |
| T-2145 | `add --requirements-file` without `--requirements` reported success, but the path is stored only in requirement links, so it was lost. |
| T-2325 | `add --blocked-by ","` was silently ignored. |

**Reproduction:** each case is a test in `cmd/input_validation_regression_test.go` or `internal/task/input_validation_regression_test.go`. Every case failed before the fix, except `batch --format yaml`, which was already rejected.

## Investigation Summary

- **T-1243:** each command switched on `format` and used `default` for table output. Only `batch` validated the value.
- **T-1754:** `cmd/add.go` always put the normalized `--phase` in `AddResponse`. `AddTaskWithPhases` ignores the phase for subtasks.
- **T-1782:** `validateOwner` rejected only control characters. `parseOwnerLine` trims, and an empty owner is reclassified as a detail.
- **T-1793:** `len(string)` was used in `validateTaskInput`, `validateDetails`, `validateReferences`, `Task.Validate` and both batch title checks.
- **T-2145:** `RenderMarkdown` writes `RequirementsFile` only inside requirement links, and the parser rebuilds it from them.
- **T-2325:** `parseRequirementIDs` drops empty tokens. `update` got a guard for this in T-1906, but `add` did not.

## Resolution

- **T-1243:** a root `PersistentPreRunE` (`validateFormatFlag`) rejects anything other than `table`, `markdown` or `json` with `unsupported output format: <value>`, the same message `batch` uses. Values are compared case-insensitively and lowercased, as `batch` already did, so `--format JSON` now produces JSON on every command instead of table output.
- **T-1754:** the response omits the phase for subtasks (in JSON and the dry-run table). This is the smallest option the ticket permits. Top-level adds still report their phase.
- **T-1782:** `validateOwner` rejects an owner that differs from its trimmed form, wrapping `ErrInvalidOwner`. Add, update, batch and claim all go through this validator. Inner spaces (`agent one`) are still allowed and round-trip.
- **T-1793:** all six checks use `utf8.RuneCountInString`.
- **T-2145:** `add --requirements-file` without `--requirements` is rejected before any change, dry-run included. This is the first option the ticket permits; there is no file-level representation to persist the path otherwise.
- **T-2325:** `add` rejects a `--blocked-by` value with no IDs, using the same message as `update`.

**Behaviour changes for review:**
- Previously, `add --requirements-file X` with no requirements, on a file whose tasks already had requirement links, rewrote those links to `X`. That now has to be done together with `--requirements`.
- Uppercase format values now select that format on every command, instead of table output.

## Regression Tests

- `internal/task/input_validation_regression_test.go`: owner rejection on add, update, batch (plain and phased) and claim; inner-space owner round trip; character-based limits for every length check.
- `cmd/input_validation_regression_test.go`: unsupported format on `list`, `add`, `next` and `batch`, with no output and no file change; case-insensitive format; subtask phase omitted (live and dry-run); top-level phase still reported; requirements-file and blocked-by rejections (live and dry-run, file unchanged); requirements-file kept when requirements are given.

**Run command:** `go test -run 'TestOwner|TestLengthLimits' ./internal/task && go test -run 'TestUnsupportedFormat|TestFormatIsCase|TestAdd(SubtaskDoesNot|TopLevelStill|RejectsSilently|RequirementsFileWith)' ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/root.go` | Global `--format` validation |
| `cmd/add.go` | Phase reporting, requirements-file and blocked-by checks |
| `internal/task/operations.go` | Owner whitespace check; character-based limits |
| `internal/task/task.go`, `internal/task/batch.go` | Character-based title limits |
| `README.md`, `CHANGELOG.md` | Documentation |
