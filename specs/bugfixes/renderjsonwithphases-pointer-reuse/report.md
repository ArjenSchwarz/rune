# Bugfix Report: RenderJSONWithPhases Pointer Reuse

**Date:** 2026-03-09
**Status:** Fixed

## Description of the Issue

`RenderJSONWithPhases` used `&task` (address of range loop variable) when building the `TaskWithPhase` slice. In Go versions prior to 1.22, the range loop reuses the same variable across iterations, so all `TaskWithPhase` entries would point to the same memory location — containing the last task's data. This results in all tasks in the JSON output being identical copies of the final task.

**Reproduction steps:**
1. Create a TaskList with multiple tasks and phase markers
2. Call `RenderJSONWithPhases` to generate JSON output
3. In Go < 1.22, all tasks in the output would have the same ID and Title (the last task's values)

**Impact:** JSON output from `RenderJSONWithPhases` would contain incorrect task data when built with Go < 1.22. With Go 1.22+, the language spec changed to scope range variables per iteration, making this a latent bug. However, the code pattern is still considered incorrect practice and fragile.

## Investigation Summary

- **Symptoms examined:** The `for _, task := range tl.Tasks` loop takes the address of the loop variable copy (`&task`) rather than the address of the actual slice element.
- **Code inspected:** `internal/task/render.go`, lines 314-322, the `RenderJSONWithPhases` function.
- **Hypotheses tested:** Confirmed that Go 1.22+ per-iteration scoping prevents the bug from manifesting at runtime, but the code pattern is still incorrect and would break if the module's Go version were lowered.

## Discovered Root Cause

The loop `for _, task := range tl.Tasks` creates a copy of each task. Taking `&task` captures a pointer to this copy rather than to the element in `tl.Tasks`. In Go < 1.22, the copy variable is reused across iterations, so all pointers alias the same memory. In Go 1.22+, each iteration gets a new variable, masking the bug.

**Defect type:** Pointer aliasing / loop variable capture

**Why it occurred:** Common Go anti-pattern of taking the address of a range loop variable.

**Contributing factors:** Go 1.22's change to per-iteration loop variable scoping hid the bug from runtime detection.

## Resolution for the Issue

**Changes made:**
- `internal/task/render.go:316-321` — Changed `for _, task := range tl.Tasks` to `for i := range tl.Tasks` and replaced `&task` with `&tl.Tasks[i]`. This ensures each `TaskWithPhase` points directly to the element in the `tl.Tasks` slice.

**Approach rationale:** Using index-based access with `&tl.Tasks[i]` is the idiomatic Go pattern for obtaining pointers to slice elements. It avoids unnecessary copies, is safe regardless of Go version, and makes the intent clear.

**Tests added:**
- `TestRenderJSONWithPhases_PointerReuse` in `internal/task/render_phase_test.go` — verifies that each task in the JSON output retains its distinct ID, Title, and Phase values.
