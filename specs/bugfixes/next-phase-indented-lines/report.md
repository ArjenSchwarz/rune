# Bugfix Report: next-phase-indented-lines

**Date:** 2025-07-14
**Status:** Fixed

## Description of the Issue

When using `rune next --phase`, indented lines were incorrectly treated as phase headers or task references. An indented `## ...` line would be misclassified as a phase header, and indented task-like detail lines could be matched as top-level tasks.

**Reproduction steps:**
1. Create a task file with phases and indented continuation or note lines containing `## ` or task-like patterns
2. Run `rune next --phase`
3. Observe wrong phase boundaries or incorrect task assignments

**Impact:** `next --phase` could return tasks from wrong phases or create spurious phases from indented content.

## Investigation Summary

- **Symptoms examined:** `extractPhasesWithTaskRanges` produced extra phases and misassigned tasks
- **Code inspected:** `internal/task/next.go` (extractPhasesWithTaskRanges), `internal/task/parse.go` (ExtractPhaseMarkers, regex patterns)
- **Hypotheses tested:** `strings.TrimSpace` on line 223 strips indentation that the regexes rely on for correct matching

## Discovered Root Cause

`extractPhasesWithTaskRanges` called `strings.TrimSpace(line)` on each line before matching against `phaseHeaderPattern` (`^## (.+)$`) and `taskLinePattern` (`^(\s*)- ...`). This stripped leading whitespace, making indented lines appear as if they started at column 0.

**Defect type:** Logic error — incorrect string preprocessing

**Why it occurred:** The function was written with `TrimSpace` for convenience, not accounting for the fact that the regex anchors (`^`) depend on preserved indentation.

**Contributing factors:** The sibling function `ExtractPhaseMarkers` in `parse.go` correctly uses `strings.TrimRight(line, "\r")` (CRLF-only trim), but this pattern wasn't followed in `next.go`.

## Resolution for the Issue

**Changes made:**
- `internal/task/next.go:223` - Changed `strings.TrimSpace(line)` to `strings.TrimRight(line, "\r")`

**Approach rationale:** Aligns with the proven-correct behavior in `ExtractPhaseMarkers`. Only strips trailing `\r` for CRLF compatibility while preserving leading indentation needed by the regex patterns.

**Alternatives considered:**
- Adding explicit indentation checks after trim — more complex, same result
- Changing the regex patterns to require non-space start — would break the capture groups

## Regression Test

**Test file:** `internal/task/next_test.go`
**Test name:** `TestExtractPhasesWithTaskRangesIndentedLines`

**What it verifies:** Indented `## ` lines are not treated as phases, and subtask/detail lines are not treated as top-level task references.

**Run command:** `go test -run TestExtractPhasesWithTaskRangesIndentedLines -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/next.go` | Changed `TrimSpace` to `TrimRight("\r")` on line 223 |
| `internal/task/next_test.go` | Added regression test with 4 sub-cases |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Code formatted (`make fmt`)

## Prevention

**Recommendations to avoid similar bugs:**
- When parsing indentation-sensitive markdown, never use `TrimSpace` — use `TrimRight("\r")` for CRLF normalization only
- Reference `ExtractPhaseMarkers` as the canonical pattern for line preprocessing in phase/task parsing

## Related

- Transit T-594
- T-488: Normalize CRLF line endings in phase marker parsing (related prior fix)
