# Bugfix Report: Filtered Markdown Phase Headers

**Date:** 2026-04-16
**Status:** Fixed
**Transit:** T-698

## Description of the Issue

`rune list --format markdown` with filters (e.g., `--status pending`) could omit phase headers when the task referenced by `PhaseMarker.AfterTaskID` was filtered out.

**Reproduction steps:**
1. Create a task list with at least two phases, where Phase B has `AfterTaskID: "1"`
2. Apply a filter that excludes task `1` but includes tasks from Phase B
3. Observe that markdown output shows Phase B tasks without the `## Phase B` header

**Impact:** Filtered markdown output lost structural context. Users relying on `--format markdown` with filters would see tasks without their phase groupings.

## Investigation Summary

- **Symptoms examined:** Phase headers missing from filtered markdown output when boundary tasks were removed
- **Code inspected:** `cmd/list.go` (`outputMarkdownWithFilters`), `internal/task/render.go` (`RenderMarkdownWithPhases`), prior fix for JSON in T-537
- **Hypotheses tested:** Confirmed the root cause matches the pattern from T-537 — the renderer compared `AfterTaskID` against adjacent tasks in the filtered list rather than the original list

## Discovered Root Cause

`RenderMarkdownWithPhases` used positional comparison between `PhaseMarker.AfterTaskID` and the previous task's ID in the rendered task list. After filtering, boundary task IDs were absent from the list, so phase markers never matched and headers were silently skipped.

**Defect type:** Missing parameter — rendering function lacked access to the original unfiltered task ordering.

**Why it occurred:** `RenderMarkdownWithPhases` was designed for unfiltered lists. When filtering was added in `cmd/list.go`, the function was called with the filtered list but had no way to resolve boundaries against the original ordering. The analogous JSON function (`RenderJSONWithPhases`) was fixed in T-537 by adding a `phaseSource` parameter, but the markdown function was not updated.

## Resolution for the Issue

**Changes made:**
- `internal/task/render.go` — Added `phaseSource *TaskList` parameter to `RenderMarkdownWithPhases`. Rewrote phase header emission to use positional indexing against the resolution list (original or filtered), emitting headers when their start position falls between consecutive rendered tasks.
- `cmd/list.go` — Pass original `taskList` as `phaseSource` in `outputMarkdownWithFilters`; pass `nil` in unfiltered path.
- All other callers (`batch.go`, `operations.go`, tests) — Pass `nil` as `phaseSource` (no filtering, backward-compatible).

**Approach rationale:** Mirrors the proven `phaseSource` pattern from the JSON fix (T-537). Using positional indexing correctly handles empty phases, duplicate phase names, and boundary tasks that have been filtered out.

**Alternatives considered:**
- Remapping markers before rendering — Rejected because it would require duplicating the resolution logic and wouldn't handle all edge cases (empty phases, duplicates)
- Using `GetTaskPhase` per task — Attempted first but failed for empty phases (no task to query) and duplicate phase names (name-based comparison loses boundary info)

## Regression Test

**Test file:** `internal/task/render_phase_test.go`

**Test names:**
- `TestRenderMarkdownWithPhases_FilteredBoundaryTask` — Two phases, boundary task filtered out
- `TestRenderMarkdownWithPhases_FilteredBoundaryThreePhases` — Three phases, multiple boundary tasks filtered out
- `TestRenderMarkdownWithPhases_NilPhaseSource` — Backward compatibility with nil phaseSource

**What it verifies:** Phase headers appear correctly in markdown output even when the tasks referenced by `PhaseMarker.AfterTaskID` are absent from the filtered task list.

**Run command:** `go test -run "TestRenderMarkdownWithPhases_Filtered|TestRenderMarkdownWithPhases_Nil" ./internal/task/ -v`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/render.go` | Added `phaseSource` parameter, rewrote phase emission logic |
| `cmd/list.go` | Pass `taskList` as phaseSource in filtered path, `nil` in unfiltered |
| `internal/task/batch.go` | Pass `nil` phaseSource (2 call sites) |
| `internal/task/operations.go` | Pass `nil` phaseSource (2 call sites) |
| `internal/task/render_phase_test.go` | Added 3 regression tests, updated existing call sites |
| `internal/task/phase_operations_test.go` | Updated 2 call sites to pass `nil` |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full unit test suite passes
- [x] Integration tests pass
- [x] Code formatted with `make fmt`

## Prevention

**Recommendations to avoid similar bugs:**
- When adding filtering to an output path, always check whether the renderer depends on task ordering or boundary references that filtering might remove
- The `phaseSource` pattern should be applied consistently across all phase-aware renderers
- Document in `docs/agent-notes/streams-and-phases.md` that `RenderMarkdownWithPhases` now also accepts a `phaseSource` parameter

## Related

- T-537: Same bug in `RenderJSONWithPhases` (fixed by adding `phaseSource` parameter)
- T-374: `RenderJSONWithPhases` pointer reuse fix
- `specs/bugfixes/filtered-json-phase-boundaries/report.md`: Prior fix report for JSON path
