# Bugfix Report: JSON Output Omits Effective Stream and Uses Stable BlockedBy IDs

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

JSON renderers (`RenderJSON` and `RenderJSONWithPhases`) marshal `Task` structs directly. This causes two problems:

1. Tasks without an explicit `Stream` value (i.e., `Stream: 0`) have the field omitted from JSON output due to `omitempty`. JSON consumers see no stream field instead of the effective default stream 1.
2. `BlockedBy` values remain as internal stable IDs rather than being translated to hierarchical IDs. JSON consumers cannot map these to visible task identifiers.

**Reproduction steps:**
1. Create a task file with two tasks: first has `Stream: 2`, second has no Stream and `Blocked-by: <stable-id-of-first>`
2. Run `rune list tasks.md --json` or `rune find tasks.md --pattern ... --json`
3. Observe: task without Stream omits the `stream` field; `blockedBy` values are stable IDs

**Impact:** JSON consumers misinterpret stream defaults and cannot resolve dependency references. The `next --phase --stream --json` command already handles this correctly via `PhaseTaskJSONWithStreams`, so the inconsistency is limited to `list` and `find` JSON output paths.

## Investigation Summary

- **Symptoms examined:** JSON output missing `stream` field for tasks with no explicit stream; `blockedBy` containing opaque stable IDs instead of hierarchical IDs
- **Code inspected:** `internal/task/render.go` (RenderJSON, RenderJSONWithPhases), `internal/task/task.go` (Task struct, json tags), `cmd/next.go` (outputPhaseTasksJSONWithStreams as a correct reference), `internal/task/dependencies.go` (TranslateToHierarchical)
- **Hypotheses tested:** Confirmed that `omitempty` on `Stream int` causes 0-value omission; confirmed `BlockedBy` is marshaled as-is without translation

## Discovered Root Cause

Two issues in the `Task` struct's JSON serialization:

1. `Stream int` tagged with `json:"stream,omitempty"` — Go's `omitempty` treats zero-value int as empty, so tasks with `Stream: 0` (meaning "not explicitly set, defaults to 1") have the field silently dropped.
2. `RenderJSON` and `RenderJSONWithPhases` pass `Task` structs directly to `json.MarshalIndent` without transforming `BlockedBy` stable IDs to hierarchical IDs.

**Defect type:** Incorrect serialization — missing data transformation before marshaling.

**Why it occurred:** The JSON rendering functions were written before the dependencies/streams feature was mature. The `next` command added its own correct `convertTask` transformation, but `RenderJSON`/`RenderJSONWithPhases` were never updated to match.

**Contributing factors:** The `omitempty` tag on `Stream` is correct for markdown rendering (don't emit `Stream: 0`), but wrong for JSON where consumers need the effective value.

## Resolution for the Issue

**Changes made:**
- `internal/task/render.go` — Added `jsonTask` struct and `toJSONTasks` / `toJSONTask` functions that transform tasks before JSON marshaling: set effective stream via `GetEffectiveStream` and translate `BlockedBy` via `DependencyIndex.TranslateToHierarchical`. Updated `TaskListJSON` and `TaskListJSONWithPhases` to use `jsonTask`. Updated `RenderJSON` and `RenderJSONWithPhases` to build a `DependencyIndex` and transform tasks before marshaling.

**Approach rationale:** A dedicated `jsonTask` struct with `json:"stream"` (no omitempty) ensures the effective stream is always emitted. Building the dependency index at render time and calling `TranslateToHierarchical` follows the same pattern already used by `next.go`.

**Alternatives considered:**
- Custom `MarshalJSON` on `Task` — Rejected because it would affect all JSON marshaling globally, including batch operations where the raw struct may be needed. A render-specific transformation is safer.
- Removing `omitempty` from `Task.Stream` — Rejected because this would emit `stream: 0` which is still wrong (should be 1), and would affect other code paths.

## Regression Test

**Test file:** `internal/task/render_test.go`
**Test names:** `TestRenderJSON_EffectiveStream`, `TestRenderJSON_BlockedByTranslatesToHierarchicalIDs`, `TestRenderJSONWithPhases_EffectiveStreamAndBlockedBy`, `TestRenderJSON_ChildrenEffectiveStreamAndBlockedBy`

**What it verifies:**
- Tasks without explicit stream emit `stream: 1` in JSON
- Tasks with explicit stream emit their set value
- `blockedBy` contains hierarchical IDs, not stable IDs
- Child tasks also get correct effective stream and translated blockedBy
- Both `RenderJSON` and `RenderJSONWithPhases` are covered

**Run command:** `go test -run "TestRenderJSON_EffectiveStream|TestRenderJSON_BlockedByTranslatesToHierarchicalIDs|TestRenderJSONWithPhases_EffectiveStreamAndBlockedBy|TestRenderJSON_ChildrenEffectiveStreamAndBlockedBy" ./internal/task/ -v`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/render.go` | Add `jsonTask` struct, `toJSONTasks`, `toJSONTask`; update `RenderJSON` and `RenderJSONWithPhases` |
| `internal/task/render_test.go` | Add 4 regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Verified `next.go` already handles this correctly and is unaffected by the change

## Prevention

**Recommendations to avoid similar bugs:**
- When adding new fields to `Task` that have meaningful zero values, consider whether `omitempty` is appropriate for JSON rendering vs markdown rendering
- JSON rendering should always go through a transformation layer rather than marshaling domain structs directly
- The existing correct implementation in `next.go` should have been used as a reference when `RenderJSON` was last modified

## Related

- Transit ticket: T-505
