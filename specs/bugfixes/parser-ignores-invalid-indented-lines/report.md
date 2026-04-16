# Bugfix Report: parser-ignores-invalid-indented-lines

**Date:** 2026-04-16
**Status:** Fixed
**Transit:** T-674

## Description of the Issue

The parser's `parseTasksAtLevel` function silently ignored non-task lines that had indentation deeper than the current parsing level. Instead of returning a parse error, these lines were skipped via a `continue` in the `default` switch branch, causing malformed content to be silently dropped.

**Reproduction steps:**
1. Create a markdown file with indented non-task text before or instead of tasks:
   ```markdown
   # Tasks
     not-a-task line
   - [ ] 1. Real task
   ```
2. Parse the file with `rune list`
3. Observe: parse succeeds, the indented non-task line is silently ignored

**Impact:** Malformed task files would parse without error, silently dropping content. Users wouldn't know their file had formatting issues.

## Investigation Summary

- **Symptoms examined:** Non-task lines with deeper indentation were silently skipped
- **Code inspected:** `internal/task/parse.go`, specifically `parseTasksAtLevel`
- **Hypotheses tested:** The `default` branch of the switch in `parseTasksAtLevel` (line ~249) used `continue` instead of returning an error

## Discovered Root Cause

In `parseTasksAtLevel`, after checking if a line is a task and checking for same-level non-task content, the `default` switch branch caught all remaining cases — non-task lines at deeper indentation (exactly `expectedIndent+2`). This branch simply called `continue`, silently skipping the line.

**Defect type:** Missing validation

**Why it occurred:** The default branch was written as a catch-all with the assumption that invalid indentation would be caught elsewhere, but lines at exactly `expectedIndent+2` that weren't tasks slipped through.

**Contributing factors:** The indentation validation above the switch correctly rejected indentation that wasn't a multiple of 2, but lines at the next valid indent level that weren't tasks were not caught.

## Resolution for the Issue

**Changes made:**
- `internal/task/parse.go:250` — Changed `default` branch from `continue` to return an error: `"unexpected content at this indentation level"`

**Approach rationale:** The simplest correct fix — any non-task content at a deeper indentation level than expected in `parseTasksAtLevel` is invalid and should produce a parse error, consistent with how the `indent == expectedIndent` case already handles unexpected content.

**Alternatives considered:**
- Treating the content as a detail line — rejected because detail lines belong under tasks, not at the root parsing level

## Regression Test

**Test file:** `internal/task/parse_invalid_indent_test.go`
**Test name:** `TestParseRejectsIndentedNonTaskLines`

**What it verifies:** The parser returns an error for non-task lines with indentation deeper than the current parsing level, including plain text before tasks, standalone indented text, deeply indented text, and indented non-task content at child levels.

**Run command:** `go test -run TestParseRejectsIndentedNonTaskLines -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/parse.go` | Changed `default` branch from `continue` to error return |
| `internal/task/parse_invalid_indent_test.go` | New regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`make test`)
- [x] Build succeeds

## Prevention

**Recommendations to avoid similar bugs:**
- Avoid bare `continue` in parser switch defaults — always explicitly handle or error on unexpected input
- Parser switch statements should have exhaustive case handling with explicit error returns for invalid states
