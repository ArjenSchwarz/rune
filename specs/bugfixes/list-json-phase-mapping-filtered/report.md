# Bugfix Report: List JSON Phase Mapping When Filters Remove Phase Boundaries

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

When using `rune list --json` with filters (`--filter`, `--stream`, or `--owner`), tasks in the JSON output could be assigned to the wrong phase or have an empty phase. This happened when the filter removed tasks that served as phase boundaries (the tasks referenced by `PhaseMarker.AfterTaskID`).

Table output was unaffected because it uses the original (unfiltered) TaskList for phase lookup via `GetTaskPhase`.

**Reproduction steps:**
1. Create a task file with phases, e.g. Planning (tasks 1-2), Implementation (tasks 3-4), Testing (tasks 5-6)
2. Mark tasks 1 and 2 as completed (these are the phase boundary tasks)
3. Run `rune list tasks.md --filter pending --format json`
4. Observe: tasks 3-6 have wrong/empty Phase values in JSON; table output shows correct phases

**Impact:** JSON consumers (typically AI agents) received incorrect phase information, causing wrong task categorisation and potentially incorrect workflow decisions.

## Investigation Summary

- **Symptoms examined:** JSON output showed empty or "Planning" phase for tasks that should be in "Implementation" or "Testing"
- **Code inspected:** `cmd/list.go` (`outputJSONWithFilters`, `filterTasksRecursive`), `internal/task/render.go` (`RenderJSONWithPhases`, `GetTaskPhase`)
- **Hypotheses tested:** The issue is in phase resolution, not in filtering itself

## Discovered Root Cause

`outputJSONWithFilters` creates a filtered `TaskList` (containing only tasks that match the filter) and passes it to `RenderJSONWithPhases`. Inside `RenderJSONWithPhases`, `GetTaskPhase` searches the passed TaskList's `.Tasks` slice for phase boundary task IDs. When those boundary tasks have been removed by filtering, `GetTaskPhase` cannot find them and returns the wrong phase (or empty string).

The table path (`flattenTasksWithFilters`) does not have this bug because it calls `GetTaskPhase(taskList, ...)` with the original, unfiltered TaskList.

**Defect type:** Logic error -- wrong data source passed to phase lookup

**Why it occurred:** `outputJSONWithFilters` was added after the table path and reused `RenderJSONWithPhases` by passing it a filtered copy of the task list. The function was originally designed for unfiltered lists and its phase lookup assumed all tasks (including boundary tasks) would be present.

**Contributing factors:** `GetTaskPhase` has an implicit contract that the TaskList it receives contains all tasks, including boundary tasks. This contract was not documented and was easy to violate.

## Resolution for the Issue

**Changes made:**
- `internal/task/render.go`: Added `RenderJSONWithPhasesFromSource(tl, phaseMarkers, phaseLookup)` which accepts a separate `phaseLookup` TaskList for phase resolution. Refactored `RenderJSONWithPhases` to delegate to a shared `renderJSONWithPhases` internal function.
- `cmd/list.go`: Changed `outputJSONWithFilters` to call `RenderJSONWithPhasesFromSource` with the original (unfiltered) TaskList as the phase lookup source.

**Approach rationale:** Introducing a separate function preserves backward compatibility for existing callers of `RenderJSONWithPhases` while allowing the filtered path to specify the correct phase source.

**Alternatives considered:**
- **Modify `GetTaskPhase` to accept a separate lookup list** -- Rejected because it would change a widely-used function signature and push the concern down to a lower level
- **Compute phases in `outputJSONWithFilters` before filtering** -- Rejected because it would duplicate the phase resolution logic from `RenderJSONWithPhases`
- **Store phase on each Task struct during parsing** -- Rejected because phases are a rendering concern, not a data model concern, and would add overhead for non-phase use cases

## Regression Test

**Test file:** `internal/task/render_phase_test.go`
**Test name:** `TestRenderJSONWithPhasesFromSource_FilteredBoundary`

**What it verifies:** When a filtered TaskList is missing phase boundary tasks, `RenderJSONWithPhasesFromSource` (with the original list) produces correct phases, while the old `RenderJSONWithPhases` on the filtered list produces incorrect phases.

**Additional test file:** `cmd/list_test.go`
**Test name:** `TestOutputJSONWithFiltersPreservesPhaseMapping`

**What it verifies:** The full JSON output path (status, stream, owner filters) produces correct phase assignments even when filters remove boundary tasks. Also verifies consistency with the table output path.

**Run command:** `go test -run "TestRenderJSONWithPhasesFromSource_FilteredBoundary|TestOutputJSONWithFiltersPreservesPhaseMapping" ./internal/task/ ./cmd/ -v`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/render.go` | Added `RenderJSONWithPhasesFromSource`, refactored to shared `renderJSONWithPhases` |
| `cmd/list.go` | Changed `outputJSONWithFilters` to use `RenderJSONWithPhasesFromSource` with original TaskList |
| `internal/task/render_phase_test.go` | Added regression test for filtered boundary phase lookup |
| `cmd/list_test.go` | Added regression test for JSON output phase mapping with filters |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When functions depend on a complete data set (like `GetTaskPhase` needing all tasks), document this contract explicitly in the function's docstring
- When adding filtered output paths, test with filters that remove structurally important elements (boundaries, parents, dependencies)
- Consider adding a consistency test that compares JSON and table output phase assignments for filtered queries

## Related

- T-473: Fix list --json phase mapping when filters remove phase boundaries
- T-436: List --json filter to match table output (prior fix for filter consistency)
- T-374: RenderJSONWithPhases pointer reuse fix
