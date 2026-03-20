# Bugfix Report: Filtered JSON Phase Boundaries

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

When using `rune list --json` with filters (e.g., `--filter`, `--stream`, `--owner`), tasks in the JSON output could be assigned incorrect phase labels. If the task referenced by a `PhaseMarker.AfterTaskID` was filtered out, the phase boundary was never detected, causing later-phase tasks to be mislabeled as belonging to an earlier phase.

**Reproduction steps:**
1. Create a task file with two phases: "Design" (tasks 1-2) and "Build" (tasks 3-4), where the phase boundary is after task 2
2. Run `rune list --json --filter completed` when tasks 1-2 are completed and task 4 is pending
3. Task 4 appears in the output with Phase="Design" instead of Phase="Build"

**Impact:** JSON consumers relying on the `Phase` field for filtering or grouping tasks would receive incorrect phase assignments whenever filters excluded boundary tasks.

## Investigation Summary

- **Symptoms examined:** Phase labels in filtered JSON output did not match the phases those tasks belong to in the unfiltered view
- **Code inspected:** `cmd/list.go` (`outputJSONWithFilters`, `filterTasksRecursive`), `internal/task/render.go` (`RenderJSONWithPhases`, `GetTaskPhase`)
- **Hypotheses tested:** Confirmed that `GetTaskPhase` scans `tl.Tasks` for `marker.AfterTaskID`; when the boundary task is absent from the filtered list, the boundary is never crossed

## Discovered Root Cause

`outputJSONWithFilters` creates a filtered `TaskList` and passes it to `RenderJSONWithPhases`. Inside that function, `GetTaskPhase` resolves phase boundaries by scanning the task list for the boundary task ID. When the boundary task has been filtered out, the scan never finds it, so the phase boundary is never detected and tasks are assigned the wrong phase.

**Defect type:** Incorrect data source — phase resolution used the filtered list instead of the original list

**Why it occurred:** `RenderJSONWithPhases` was designed to use a single `TaskList` for both rendering and phase resolution. Filtering was added later without accounting for the fact that phase boundaries depend on the original task ordering.

**Contributing factors:** The table output path (`flattenTasksWithFilters`) calls `GetTaskPhase` with the original `taskList`, so it was not affected. The inconsistency was only in the JSON path.

## Resolution for the Issue

**Changes made:**
- `internal/task/render.go:294-340` — Added `phaseSource *TaskList` parameter to `RenderJSONWithPhases`. When non-nil, phase resolution uses `phaseSource` instead of the rendered task list.
- `cmd/list.go:659` — Pass original `taskList` as `phaseSource` in `outputJSONWithFilters`
- `cmd/list.go:381` — Pass `nil` as `phaseSource` in `outputJSONWithPhases` (no filtering, no change needed)

**Approach rationale:** Adding the optional `phaseSource` parameter keeps the fix localized. Callers that don't filter can pass `nil` for identical behavior. The filtered JSON path passes the original unfiltered list so boundary tasks are always found.

**Alternatives considered:**
- Duplicating `GetTaskPhase` logic in `outputJSONWithFilters` — rejected because it duplicates code and diverges from the render function
- Always storing the original task list in a global/context — rejected because it introduces coupling

## Regression Test

**Test file:** `internal/task/render_phase_test.go`
**Test name:** `TestRenderJSONWithPhases_FilteredBoundaryTask`

**Test file:** `cmd/list_test.go`
**Test name:** `TestFilteredJSONOutputPreservesPhaseBoundaries`

**What it verifies:** Phase labels are correct when boundary tasks are absent from the filtered task list, covering status, stream, and owner filters.

**Run command:** `go test -run "TestRenderJSONWithPhases_FilteredBoundaryTask|TestFilteredJSONOutputPreservesPhaseBoundaries" ./internal/task/ ./cmd/ -v`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/render.go` | Added `phaseSource` parameter to `RenderJSONWithPhases` |
| `cmd/list.go` | Updated callers to pass original task list for phase resolution |
| `internal/task/render_phase_test.go` | Added `TestRenderJSONWithPhases_FilteredBoundaryTask` regression test |
| `cmd/list_test.go` | Added `TestFilteredJSONOutputPreservesPhaseBoundaries` regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When filtering data for display, always resolve positional metadata (phases, ordering) against the original unfiltered data
- Consider adding a comment on functions that depend on task list completeness (e.g., `GetTaskPhase` assumes all boundary tasks are present)

## Related

- T-537: Transit ticket
- T-374: Previous fix for `RenderJSONWithPhases` pointer reuse bug
- T-436: Previous fix ensuring JSON filter path matches table filter path
