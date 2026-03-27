# Bugfix Report: phase-nonsequential-ids

**Date:** 2026-03-27
**Status:** Fixed

## Description of the Issue

`rune next --phase` could drop phase tasks or report no pending tasks when markdown task IDs were non-sequential (e.g., `10`, `20`, `30` instead of `1`, `2`, `3`), which can occur due to manual editing or deletions.

**Reproduction steps:**
1. Create a markdown file with non-sequential task IDs in phases:
   ```
   ## Phase A
   - [ ] 10. First task
   ## Phase B
   - [ ] 20. Second task
   ```
2. Run `rune next --phase <file>`
3. Observe: tasks are missing or no phase result returned

**Impact:** Phase-based task retrieval silently loses tasks, causing agents and users to miss work items.

## Investigation Summary

The bug exists in three functions that map raw markdown task IDs to parsed task objects for phase association.

- **Symptoms examined:** `FindNextPhaseTasks` returns nil when phases contain pending tasks with non-sequential IDs
- **Code inspected:** `next.go` (`extractPhasesWithTaskRanges`), `phase.go` (`getTaskPhase`, `buildTaskPhaseMap`), `parse.go` (`parseTasksAtLevel`)
- **Hypotheses tested:** Confirmed that the parser discards raw markdown IDs and assigns sequential positional IDs

## Discovered Root Cause

**Defect type:** ID mapping mismatch between raw markdown and parsed representation

Three functions extracted raw numeric IDs from markdown lines (e.g., `10`, `20`) and used them as keys to look up parsed tasks in a map. However, `parseTasksAtLevel` assigns sequential IDs (`1`, `2`, `3`) based on position, discarding the raw markdown IDs entirely. The lookups silently failed, causing tasks to be dropped from phase results.

**Why it occurred:** The phase extraction code was written assuming markdown IDs would always match parsed IDs, which is only true for well-formed sequential files.

**Contributing factors:** No tests existed with non-sequential markdown IDs; all test fixtures used sequential numbering.

## Resolution for the Issue

**Changes made:**
- `internal/task/next.go:extractPhasesWithTaskRanges` - Use positional counter instead of raw markdown ID for taskMap lookup
- `internal/task/phase.go:getTaskPhase` - Use positional counter instead of raw markdown ID for comparison
- `internal/task/phase.go:buildTaskPhaseMap` - Use positional counter as map key instead of raw markdown ID

**Approach rationale:** Positional counting mirrors how `parseTasksAtLevel` assigns IDs: the nth top-level task line gets ID `n`. This ensures phase association always matches regardless of what IDs appear in the markdown.

**Alternatives considered:**
- Storing raw markdown IDs in parsed Task structs — too invasive, changes the data model
- Re-parsing from raw lines instead of using parsed tasks — duplicates parsing logic, error-prone

## Regression Test

**Test file:** `internal/task/next_test.go`, `internal/task/phase_test.go`
**Test names:**
- `TestExtractPhasesWithTaskRanges_NonSequentialIDs`
- `TestFindNextPhaseTasks_NonSequentialIDs`
- `TestFindNextPhaseTasksForStream_NonSequentialIDs`
- `TestGetTaskPhase_NonSequentialIDs`
- `TestBuildTaskPhaseMap_NonSequentialIDs`
- `TestGetNextPhaseTasks_NonSequentialIDs`

**What it verifies:** Phase association works correctly when markdown files contain non-sequential task IDs (gaps, large numbers, etc.)

**Run command:** `go test ./internal/task/ -run "NonSequentialIDs" -v`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/next.go` | Fixed `extractPhasesWithTaskRanges` to use positional counter |
| `internal/task/phase.go` | Fixed `getTaskPhase` and `buildTaskPhaseMap` to use positional counter |
| `internal/task/next_test.go` | Added 9 regression test cases |
| `internal/task/phase_test.go` | Added 8 regression test cases |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Code formatted with `make fmt`

**Manual verification:**
- Confirmed regression tests fail before fix and pass after
- Confirmed all existing tests continue to pass (no regressions)

## Prevention

**Recommendations to avoid similar bugs:**
- When matching parsed task data with raw markdown, always use positional mapping rather than raw IDs
- Include non-sequential ID test cases in phase-related test fixtures
- Consider adding a lint check that flags raw regex group extraction used as map keys for parsed task lookup

## Related

- Transit ticket: T-604
