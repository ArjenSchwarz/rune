# Bugfix Report: list-markdown-ignoring-filters

**Date:** 2026-03-27
**Status:** Fixed
**Ticket:** T-579

## Description of the Issue

When using `rune list --format markdown` with filter flags (`--status`, `--stream`, `--owner`), the markdown output ignored the filters and rendered all tasks. Table and JSON formats correctly respected the filters.

**Reproduction steps:**
1. Create a task file with mixed statuses (pending + completed)
2. Run: `rune list file.md --filter pending --format markdown`
3. Observe: both pending and completed tasks appear in the output
4. Expected: only pending tasks should appear (matching table/JSON behaviour)

**Impact:** Medium — markdown output was unusable for filtered views, forcing users to use table or JSON format when filtering was needed.

## Investigation Summary

- **Symptoms examined:** Markdown output always showed all tasks regardless of filter flags
- **Code inspected:** `cmd/list.go` (runList switch statement, outputMarkdownWithPhases, outputJSONWithFilters, filterTasksRecursive), `internal/task/render.go` (RenderMarkdownWithPhases)
- **Hypotheses tested:** The `outputMarkdownWithPhases` function did not receive or apply `filterOpts`

## Discovered Root Cause

In `cmd/list.go:119`, the `runList()` function's format switch statement called `outputMarkdownWithPhases(taskList, phaseMarkers)` without passing `filterOpts`. The JSON path correctly passed filters via `outputJSONWithFilters(taskList, phaseMarkers, depIndex, filterOpts)`, and the table path used pre-filtered `taskData`. The markdown path was simply never wired to the filtering logic.

**Defect type:** Missing filter application (code path omission)

**Why it occurred:** When the filter system was added, the JSON and table code paths were updated but the markdown path was overlooked.

**Contributing factors:** No test coverage existed for markdown output with filters, so the gap was not caught.

## Resolution for the Issue

**Changes made:**
- `cmd/list.go:119` — Changed switch case to call `outputMarkdownWithFilteredPhases(taskList, phaseMarkers, filterOpts)` instead of `outputMarkdownWithPhases(taskList, phaseMarkers)`
- `cmd/list.go` — Added `outputMarkdownWithFilteredPhases()` that applies `filterTasksRecursive()` before rendering, matching the JSON output pattern
- `cmd/list.go` — Added testable `outputMarkdownWithFilters()` helper that returns the markdown string

**Approach rationale:** Followed the same pattern as `outputJSONWithFilters()` — filter the task list first, then render. This keeps all three format paths consistent.

**Alternatives considered:**
- Modifying `RenderMarkdownWithPhases()` in `internal/task/render.go` to accept filter criteria — rejected because filtering is a `cmd`-layer concern, not a rendering concern

## Regression Test

**Test file:** `cmd/list_test.go`
**Test names:** `TestMarkdownOutputRespectsFilters`, `TestMarkdownFilterMatchesJSONFilter`

**What it verifies:**
- Status, stream, and owner filters exclude non-matching tasks from markdown output
- Combined filters work correctly
- No filters still shows all tasks
- Phase headers are preserved with filters
- Markdown and JSON filter output contain the same tasks

**Run command:** `go test -run "TestMarkdownOutputRespectsFilters|TestMarkdownFilterMatchesJSONFilter" -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/list.go` | Added filtered markdown output path; new `outputMarkdownWithFilteredPhases()` and `outputMarkdownWithFilters()` functions |
| `cmd/list_test.go` | Added 8 test cases across 2 new test functions |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Code formatted (`make fmt`)

**Manual verification:**
- Built binary and confirmed `rune list --format markdown --filter pending` only shows pending tasks

## Prevention

**Recommendations to avoid similar bugs:**
- When adding filter support to a new output format, ensure all format branches in the switch statement receive the filter options
- Add cross-format consistency tests (like `TestMarkdownFilterMatchesJSONFilter`) whenever a new format is added
- Consider refactoring the three format paths to share a single pre-filtered task list, eliminating the possibility of format-specific filter omissions

## Related

- T-579: Fix `rune list --format markdown` ignoring filter flags
- T-473: Phase boundary resolution for filtered JSON output (similar pattern)
- T-537: Filtered JSON output preserves phase boundaries
