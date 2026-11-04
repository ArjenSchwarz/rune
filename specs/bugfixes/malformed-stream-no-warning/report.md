# Bugfix Report: Malformed Stream Metadata Emits No Warning

**Date:** 2026-11-04
**Status:** Fixed
**Ticket:** T-2197

## Description of the Issue

A `Stream:` line with a malformed value (`0`, negative, non-integer, overflowing) was silently kept as a plain detail. The task looked assigned to a stream but was scheduled on the default stream 1, with no warning.

**Reproduction steps:**
1. Create `# Tasks`, `- [ ] 1. Assigned-looking task`, `  - Stream: 0`.
2. Run `rune list <file> --format json`.
3. Observe exit 0, no warning, `"Stream: 0"` in `Details` and `"stream": 1`.

**Impact:** Low. A typo in a hand-edited stream assignment went unnoticed.

## Investigation Summary

- **Symptoms examined:** No warning on stderr; the line was kept in `Details`.
- **Code inspected:** `applyDetailLine` and `parseStreamLine` in `internal/task/parse.go`; `WarnInvalidStreamValue` in `internal/task/errors.go`; `reportParseWarnings` in `cmd/format.go`.
- **Hypotheses tested:** `parseStreamLine` returned 0 both for "not a Stream line" and "invalid value", so the caller could not tell them apart. `WarnInvalidStreamValue` was defined but never emitted.

## Discovered Root Cause

`parseStreamLine` collapsed "not a Stream line" and "Stream line with an invalid value" into the same return value, so the malformed case fell through to a plain detail without a warning. The design (`specs/task-dependencies-and-streams/design.md`) requires a warning and treat-as-unset.

**Defect type:** Missing validation feedback.

## Resolution for the Issue

**Changes made:**
- `internal/task/parse.go` (`parseStreamLine`) - Return whether the line starts with `Stream:` (case-insensitive) in addition to the value. Values are parsed with `strconv.Atoi` after the existing digits-only pattern, so overflow is still rejected.
- `internal/task/parse.go` (`applyDetailLine`) - For a `Stream:` line with an invalid value, append a `WarnInvalidStreamValue` warning to `TaskList.Warnings` and keep the line as a plain detail (stream stays unset). Commands already print parse warnings to stderr.

**Approach rationale:** Mirrors the existing Blocked-by warning (T-1998): lenient parsing, text preserved, warning surfaced through the existing path.

**Alternatives considered:**
- Reject the file with an error - contradicts the design's lenient behaviour.

## Regression Test

**Test file:** `internal/task/parser_diagnostics_regression_test.go`
**Test names:** `TestMalformedStreamWarns`, `TestValidStreamHasNoWarnings`

**What it verifies:** Zero, negative, non-integer, decimal, overflowing, empty and trailing-text values each produce one `invalid_stream_value` warning, leave `Stream` unset and keep the detail text; valid values produce no warning.

**Run command:** `go test ./internal/task -run 'TestMalformedStreamWarns|TestValidStreamHasNoWarnings'`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/parse.go` | Distinguish malformed Stream lines and warn |
| `internal/task/parser_diagnostics_regression_test.go` | Regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (except two pre-existing permission tests that fail only when run as root)
- [x] Linters/validators pass

**Manual verification:**
- `rune list <file> --format json` prints `Warning: task 1: ignoring invalid stream value in "Stream: 0": a stream must be a positive integer` to stderr, exits 0, and stdout JSON is unchanged.

## Related

- T-1998 (malformed Blocked-by warning)
