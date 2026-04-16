# Bugfix Report: Batch Add to Earlier Phase Misplaces Later Phase Markers

**Date:** 2026-04-16
**Status:** Fixed

## Description of the Issue

When `rune batch` adds a top-level task to an earlier phase in a file with three or more phases, `addTaskWithPhaseMarkers` updates only the immediate next phase marker after insertion. Later markers keep their old `AfterTaskID` even though top-level tasks were renumbered, so rendering moves later phase headers before the wrong tasks.

**Reproduction steps:**
1. Create a task file with three phases (A, B, C) each containing one task
2. Run batch JSON adding a new task to phase A
3. Observe that phases B and C are misplaced in the output — phase C header appears before B1 instead of before C1

**Impact:** Any batch add to an earlier phase in a multi-phase file corrupts the phase layout of all phases beyond the immediately next one.

## Investigation Summary

- **Symptoms examined:** Phase headers rendered before incorrect tasks after batch add
- **Code inspected:** `internal/task/batch.go` (`addTaskWithPhaseMarkers`), `internal/task/operations.go` (`AddTaskToPhase`, `adjustPhaseMarkersForInsertion`)
- **Hypotheses tested:** Compared the phase marker update logic in both code paths

## Discovered Root Cause

In `internal/task/batch.go`, the `addTaskWithPhaseMarkers` function only updates the immediate next phase marker (i+1) after inserting a task. It does not adjust the `AfterTaskID` of markers beyond i+1, even though task renumbering shifts all subsequent task IDs.

**Defect type:** Missing logic — incomplete port of phase marker adjustment

**Why it occurred:** The standalone `AddTaskToPhase` in `operations.go` was later enhanced with `adjustPhaseMarkersForInsertion` for markers at i+2 and beyond, but the parallel batch code path in `batch.go` was not updated to match.

**Contributing factors:** Two separate code paths performing the same logical operation (phase-aware task insertion) without shared implementation.

## Resolution for the Issue

**Changes made:**
- `internal/task/batch.go:967-983` — Added `adjustPhaseMarkersForInsertion` call for markers at index i+2 and beyond, matching the logic in `AddTaskToPhase`

**Approach rationale:** The fix mirrors exactly what `AddTaskToPhase` already does correctly, ensuring both code paths behave identically.

**Alternatives considered:**
- Refactor both paths to share a single implementation — better long-term but higher risk for this fix

## Regression Test

**Test file:** `internal/task/batch_operations_test.go`
**Test name:** `TestBatchAddToEarlierPhaseMisplacesLaterMarkers`

**What it verifies:** Adding a task to phase A in a 3-phase file (A, B, C) preserves correct phase header placement for phases B and C.

**Run command:** `go test -run TestBatchAddToEarlierPhaseMisplacesLaterMarkers -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/batch.go` | Added `adjustPhaseMarkersForInsertion` for markers beyond the immediate next |
| `internal/task/batch_operations_test.go` | Added regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Build succeeds

## Prevention

**Recommendations to avoid similar bugs:**
- Consider extracting the shared phase-marker-update logic into a single helper function used by both `AddTaskToPhase` and `addTaskWithPhaseMarkers`
- When adding phase marker adjustment logic, always test with 3+ phases to catch off-by-one marker updates

## Related

- Transit ticket: T-787
