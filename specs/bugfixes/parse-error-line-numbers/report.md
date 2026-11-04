# Bugfix Report: Parse Error Line Numbers

**Date:** 2026-11-04
**Status:** Fixed
**Ticket:** T-1879

## Description of the Issue

Parse errors reported line numbers that did not match the source file. For a file with three lines of front matter, a closing delimiter and an H1 title, an invalid checkbox on line 8 was reported as `line 3: invalid status: [q]`. Files without front matter were still off by one after a title.

**Reproduction steps:**
1. Create a file with front matter (lines 1-4), `# Error Lines` (line 5), a blank line, `- [ ] 1. Valid` and `- [q] 2. Invalid checkbox` (line 8).
2. Run `rune list <file>`.
3. Observe `line 3: invalid status: [q]`.

**Impact:** Low. Diagnostics pointed users at the wrong line of hand-edited files.

## Investigation Summary

- **Symptoms examined:** Reported line numbers were short by the number of front matter lines plus one when a title was present.
- **Code inspected:** `parseContent`, `parseTasksAtLevel`, `parseDetailsAndChildren` in `internal/task/parse.go`; `ParseFrontMatter` in `internal/task/references.go`.
- **Hypotheses tested:** Errors are all formatted as `line %d` from `i+1`, where `i` indexes the line slice passed to the parser, so any line removed from that slice before parsing shifts every later number.

## Discovered Root Cause

`parseContent` parsed the content left after `ParseFrontMatter` removed the front matter, and then removed the title line from the slice. Line indices were relative to that transformed slice, not the source.

**Defect type:** Logic error (index into transformed data reported as a source position).

**Why it occurred:** The line slice was used both as parser input and as the source of line numbers, and the preprocessing steps removed elements from it.

## Resolution for the Issue

**Changes made:**
- `internal/task/parse.go` (`parseContent`) - Prepend one blank line per front matter line (counted on CRLF-normalised content) and blank the title line instead of removing it. The parser already skips blank lines, so parsing is unchanged and indices match source lines.

**Approach rationale:** Smallest change that keeps every existing `line %d` format string correct, including those in nested parsing.

**Alternatives considered:**
- Pass a line offset through the recursive parse functions - touches every error site and still breaks on the removed title line.

## Regression Test

**Test file:** `internal/task/parser_diagnostics_regression_test.go`
**Test name:** `TestParseErrorReportsSourceLineNumber`

**What it verifies:** Error line numbers match the source for front matter plus title, title only, title after blank lines, no title, CRLF front matter, and an error inside nested subtask details.

**Run command:** `go test ./internal/task -run TestParseErrorReportsSourceLineNumber`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/parse.go` | Preserve source line positions while skipping front matter and title |
| `internal/task/parser_diagnostics_regression_test.go` | Regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (except two pre-existing permission tests that fail only when run as root)
- [x] Linters/validators pass

**Manual verification:**
- `rune list` on the reproduction file now reports `line 8: invalid status: [q]`.

## Prevention

- Keep parser input aligned with source lines; blank skipped content instead of removing it.
