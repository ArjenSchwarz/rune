# Bugfix Report: Batch Remove Shifts Phase Boundaries

**Date:** 2025-07-21
**Status:** Fixed
**Ticket:** T-820

## Description of the Issue

When running batch remove operations on a file that contains phases (H2 headers), the phase boundaries were destroyed. All phase headers were stripped from the output, and tasks that were in later phases would shift to incorrect positions.

**Reproduction steps:**
1. Create a task file with multiple phases (e.g., Planning, Implementation, Deployment)
2. Run a batch operation with only `remove` operations (no `phase` field on any op)
3. Observe that all `## Phase` headers are missing from the resulting file

**Impact:** High — any batch remove on phased files silently destroys the phase structure, losing organizational information.

## Investigation Summary

- **Symptoms examined:** Phase headers disappeared after batch remove operations
- **Code inspected:** `cmd/batch.go` (CLI entry point), `internal/task/batch.go` (ExecuteBatchWithPhases), `internal/task/operations.go` (adjustPhaseMarkersForRemoval)
- **Hypotheses tested:** Initially investigated whether `adjustPhaseMarkersForRemoval` had incorrect math when processing multiple removes in reverse order — the math was correct. The real issue was upstream in the CLI dispatch logic.

## Discovered Root Cause

In `cmd/batch.go`, the `executeBatchCommand` function decided whether to use phase-aware execution based solely on whether any **operation** referenced a phase (`op.Phase != ""` or `op.Type == "add-phase"`). It did **not** check whether the **file itself** contained phases.

When batch removes were submitted without a `phase` field (which is the normal case for removes), `hasPhaseOps` was `false`, so the code took the non-phase-aware path: `ParseFile` (no phase extraction) → `ExecuteBatch` (no phase preservation) → `WriteFile` (no phase headers).

**Defect type:** Logic error — incomplete condition in dispatch logic

**Why it occurred:** The phase-aware path was added to support adding tasks to specific phases. The condition only checked for operations that explicitly name a phase, missing the case where the file already has phases that need preserving.

**Contributing factors:** The internal `ExecuteBatchWithPhases` function already had the correct guard (`len(phaseMarkers) > 0 || hasPhaseOps`), but the CLI layer short-circuited before reaching it.

## Resolution for the Issue

**Changes made:**
- `cmd/batch.go` — Always parse the file with `ParseFileWithPhases` so phase markers are detected. Use `ExecuteBatchWithPhases` when the file has phases or operations reference phases; fall back to regular `ExecuteBatch` only when neither condition is true.

**Approach rationale:** This aligns the CLI dispatch logic with the guard already present inside `ExecuteBatchWithPhases`, which correctly handles both phase sources. `ParseFileWithPhases` returns an empty markers slice for non-phased files, so the fallback path is still exercised when appropriate.

**Alternatives considered:**
- Adding a separate `hasPhases` file check before the dispatch — rejected as unnecessarily complex when we can simply always parse with phases and let the existing internal guard decide.

## Regression Test

**Test file:** `cmd/batch_test.go`
**Test name:** `TestBatchCommand_RemoveOnPhasedFilePreservesPhases`

**What it verifies:** That batch remove operations on a phased file (without any `phase` field on the operations) preserve all phase headers and keep tasks in their correct phases.

**Run command:** `go test -run TestBatchCommand_RemoveOnPhasedFilePreservesPhases -v ./cmd/`

## Affected Files

| File | Change |
|------|--------|
| `cmd/batch.go` | Always use `ParseFileWithPhases`; dispatch to phase-aware path when file has phases |
| `cmd/batch_test.go` | Added regression test `TestBatchCommand_RemoveOnPhasedFilePreservesPhases` |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`make test`)
- [x] Build succeeds (`go build`)

## Prevention

**Recommendations to avoid similar bugs:**
- When adding phase-aware code paths, always check both the file's phase state AND the operation's phase references
- The internal library already had the correct condition — the CLI layer should delegate this decision rather than duplicating it
