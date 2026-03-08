# Bugfix Report: AddTaskToPhase Insertion Leaves Later Phase Markers Stale

**Date:** 2026-03-09
**Status:** Fixed

## Description of the Issue

When adding a task to a phase in a file with three or more phases, only the immediately next phase marker was updated after insertion. Phase markers beyond the next one retained their original `AfterTaskID` values, which became stale after task renumbering. This caused `RenderMarkdownWithPhases` to place later phase headers at incorrect positions or not at all.

**Reproduction steps:**
1. Create a task file with three or more phases (e.g., Planning, Implementation, Testing)
2. Add a task to the first phase using `rune add --phase Planning "New task" file.md`
3. Observe that the Testing phase header appears at the wrong position in the output

**Impact:** Phase structure corruption when adding tasks to any phase except the last one in files with 3+ phases. The rendered output would have misplaced phase headers.

## Investigation Summary

- **Symptoms examined:** Phase headers appearing at wrong positions after adding tasks to a phase
- **Code inspected:** `AddTaskToPhase` in `operations.go`, `adjustPhaseMarkersForRemoval` (analogous function for removal), `RenderMarkdownWithPhases` in `render.go`
- **Hypotheses tested:** Confirmed that only `phaseMarkers[i+1]` was updated, leaving `phaseMarkers[i+2]`, `phaseMarkers[i+3]`, etc. with stale AfterTaskID values

## Discovered Root Cause

In `AddTaskToPhase` (operations.go), the marker update loop after insertion only updated the immediately next phase marker (`phaseMarkers[i+1]`). It did not adjust markers beyond that. When tasks are renumbered after insertion (all tasks at the insertion position and later shift up by one), phase markers referencing those shifted tasks become stale.

**Defect type:** Logic error -- incomplete marker adjustment

**Why it occurred:** The original implementation only considered the immediate next phase boundary. The analogous removal function (`adjustPhaseMarkersForRemoval`) already handled all markers correctly, but this pattern was not replicated for insertion.

## Resolution for the Issue

**Changes made:**
- `internal/task/operations.go` -- Replaced the single-marker update with a two-step approach:
  1. Update the immediate next phase marker to point to the newly inserted task (preserving the invariant that it marks the end of the current phase)
  2. Increment `AfterTaskID` for all markers beyond the next one whose referenced task number is at or after the insertion position
- `internal/task/operations.go` -- Added `adjustPhaseMarkersForInsertion` helper function for the generic marker shift logic (analogous to `adjustPhaseMarkersForRemoval`)

**Approach rationale:** The fix mirrors the existing `adjustPhaseMarkersForRemoval` pattern. The immediate next marker needs special handling (it points to the new task, not just a shifted existing task), while all subsequent markers follow the generic increment rule.

**Alternatives considered:**
- Recomputing all markers from scratch after insertion (parse the rendered output again) -- rejected because it would be inefficient and fragile
- Using `adjustPhaseMarkersForInsertion` for all markers including the next one -- rejected because the next marker has different semantics (it should point to the new task, not just be incremented)

## Regression Tests Added

- `TestAddTaskToPhaseUpdatesAllLaterMarkers` -- End-to-end test with 3 and 4 phases, verifying correct file output after adding tasks to different phases
- `TestAdjustPhaseMarkersForInsertion` -- Unit tests for the new helper function covering edge cases (start, middle, end insertion; empty AfterTaskIDs)
