# Bugfix Report: phase-lookup-nonsequential-ids

**Date:** 2026-04-16
**Status:** Fixed

## Description of the Issue

Phase labeling in `rune list` output (table, JSON, markdown) was incorrect when task files contained non-sequential top-level markdown IDs (e.g., `10`, `20`, `30`). Phase boundaries were never matched because `ExtractPhaseMarkers` stored raw markdown IDs while parsed tasks used sequential IDs.

**Reproduction steps:**
1. Create a task file with phases and non-sequential top-level task IDs (e.g., `10`, `20`, `30`)
2. Run `rune list` with phase-aware output
3. Observe phase assignment does not switch at expected boundaries

**Impact:** Incorrect or missing phase values in all output formats when task files contain non-sequential IDs.

## Investigation Summary

- **Symptoms examined:** Phase labels missing or incorrect in list outputs
- **Code inspected:** `internal/task/parse.go` (`ExtractPhaseMarkers`), `internal/task/render.go` (`GetTaskPhase`, `RenderMarkdownWithPhases`), `internal/task/phase.go` (`getTaskPhase`, `buildTaskPhaseMap`)
- **Hypotheses tested:** Confirmed that `AfterTaskID` stored raw markdown IDs while consumers compared against sequential parsed IDs

## Discovered Root Cause

`ExtractPhaseMarkers` stored the raw markdown task ID (e.g., `"10"`) in `PhaseMarker.AfterTaskID`, but `ParseMarkdown` renumbers all tasks sequentially (`"1"`, `"2"`, ...). Downstream functions (`GetTaskPhase`, `RenderMarkdownWithPhases`) compared `AfterTaskID` against `tl.Tasks[i].ID` (sequential), so boundaries with non-sequential raw IDs never matched.

**Defect type:** ID domain mismatch — raw IDs vs sequential parsed IDs

**Why it occurred:** `ExtractPhaseMarkers` was written to capture the literal task ID from markdown text, without accounting for the renumbering that `ParseMarkdown` performs.

**Contributing factors:** The functions in `phase.go` (`getTaskPhase`, `buildTaskPhaseMap`) independently implemented positional counting and worked correctly, masking the fact that `ExtractPhaseMarkers` itself was producing incompatible IDs.

## Resolution for the Issue

**Changes made:**
- `internal/task/parse.go:526-558` - Changed `ExtractPhaseMarkers` to track a positional counter for top-level tasks and store sequential IDs in `AfterTaskID` instead of raw markdown IDs

**Approach rationale:** Fixing `ExtractPhaseMarkers` at the source ensures all downstream consumers (`GetTaskPhase`, `RenderMarkdownWithPhases`, `RenderJSONWithPhases`) automatically work correctly without individual fixes.

**Alternatives considered:**
- Fix each consumer individually to do positional mapping — rejected because it's more code, more error-prone, and doesn't fix the root cause

## Regression Test

**Test file:** `internal/task/phase_test.go`
**Test names:**
- `TestExtractPhaseMarkers_NonSequentialIDs` — verifies `AfterTaskID` is sequential
- `TestGetTaskPhase_NonSequentialMarkdownIDs` — verifies `GetTaskPhase` returns correct phases
- `TestRenderMarkdownWithPhases_NonSequentialIDs` — verifies markdown rendering places phase headers correctly

**Run command:** `go test ./internal/task/ -run "NonSequential" -v`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/parse.go` | `ExtractPhaseMarkers` now stores sequential positional IDs |
| `internal/task/phase_test.go` | Added 3 regression test functions for T-742 |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass (fmt)

## Prevention

**Recommendations to avoid similar bugs:**
- When functions produce IDs, document whether they are raw markdown IDs or sequential parsed IDs
- Prefer a single ID domain throughout the pipeline to avoid mismatches

## Related

- T-604: Previous fix for `getTaskPhase`/`buildTaskPhaseMap` (positional counting in `phase.go`)
- T-742: This bug (the same fix was needed in `ExtractPhaseMarkers` for render.go consumers)
