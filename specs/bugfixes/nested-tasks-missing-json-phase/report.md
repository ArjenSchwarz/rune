# Bugfix Report: nested-tasks-missing-json-phase

**Date:** 2026-11-01
**Status:** Fixed
**Ticket:** T-2342

## Description of the Issue

`rune list --format json` on a phased task file included `Phase` only on
top-level tasks. Nested children had no `Phase` field, so programmatic
consumers had to infer a child's phase from its ancestry. Table output already
reports the inherited phase for every task.

**Reproduction steps:**
1. Create a file with `## Planning` followed by `- [ ] 1. Parent` and a nested
   `- [ ] 1.1. Child`.
2. Run `rune list tasks.md --format json`.
3. Task 1 has `"Phase": "Planning"`; task 1.1 has no `Phase` field.

**Impact:** Medium. Violates specs/task-phases/requirements.md acceptance
criteria 5.2, 5.3 and 5.5, and JSON output disagrees with table output.

## Investigation Summary

- **Code inspected:** `internal/task/render.go` (`RenderJSONWithPhases`,
  `toJSONTask`, `GetTaskPhase`), `cmd/list.go` (`outputJSONWithPhases`,
  `outputJSONWithFilters`, `filterTasksRecursive`).
- **Finding:** `GetTaskPhase` already resolves by the root segment of a task
  ID, so it returns the right phase for any task, including children promoted
  to the top level by filtering. The defect was only in the JSON shape.

## Discovered Root Cause

`RenderJSONWithPhases` wrapped only root entries in `jsonTaskWithPhase`.
Children were converted by `toJSONTasks`, which returned plain `jsonTask`
values with no phase field.

**Defect type:** Incomplete (non-recursive) data conversion.

## Resolution for the Issue

**Changes made:**
- `internal/task/render.go`: added `Phase` (`json:"Phase,omitempty"`) to
  `jsonTask` and removed the root-only `jsonTaskWithPhase` wrapper.
  `RenderJSONWithPhases` resolves each root's phase and assigns it to the root
  and all its descendants via `setJSONTaskPhase`.

**Why this works:** Children always belong to their root task's phase. Phases
are only set when phase markers exist, and `omitempty` keeps the field out of
output for files without phases and for tasks before the first phase header,
so existing output for those cases is unchanged.

## Regression Test

- `internal/task/render_phase_test.go`:
  `TestRenderJSONWithPhases_NestedChildrenInheritPhase` covers nested
  children, promoted children (with a phase source), tasks before the first
  phase, and files without phases. It fails without the fix.
- `cmd/list_test.go`: the phase-boundary filter table test now walks nested
  children and adds "nested children inherit phase" and "promoted children
  keep original phase" cases through `filterTasksRecursive`.

## Related

- T-473 / T-537: phase mapping when filtering removes top-level boundary tasks.
