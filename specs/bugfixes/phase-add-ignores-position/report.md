# Bugfix Report: Phase-Aware Add Ignores Position

**Date:** 2026-11-02
**Status:** In progress
**Tickets:** T-919, T-973

## Description of the Issue

A top-level add that sets both a phase and a position silently ignored the
position. In a batch it could also corrupt the phase structure with a
duplicate phase header.

**Reproduction steps:**
1. CLI: `rune add-phase tasks.md Build`, then
   `rune add tasks.md --title Inserted --phase Build --position 2`. The task is
   appended to the end of `Build`; `--position` has no effect and no error is
   returned.
2. Batch (T-919): on a file with phases `Alpha` and `Beta`, run
   `[{"type":"add-phase","phase":"Planning"},{"type":"add","title":"Inserted","phase":"Planning","position":"1"}]`.
   The output contains `## Planning` twice, the second one empty.
3. Batch (T-973): `add Task A`, `add Task B`, `add-phase Build`,
   `add Inserted phase=Build position=2` produced a `## Build` header before
   Task A and a second, empty `## Build` header at the end.

**Impact:** Medium. Silent data loss of the caller's intent (position), and
in batches a malformed file with duplicate phase headers.

## Investigation Summary

- **Symptoms examined:** CLI and batch output for the repros above, all
  reproduced on main at 2ab5286.
- **Code inspected:** `cmd/add.go`, `internal/task/batch.go`
  (`sortOperationsForExecution`, `runBatch`, `applyOperationWithPhases`,
  `AddTaskWithPhases`, `addTaskWithPhaseMarkers`).
- **Hypotheses tested:** The ticket for T-973 says `cmd/add.go` calls
  `AddTaskToPhase` without the position. That is stale: main now builds an
  `Operation` and calls `AddTaskWithPhases` for every add, so the CLI and
  batch share one code path. The position is dropped inside
  `addTaskWithPhaseMarkers`, which never reads `op.Position`.

## Discovered Root Cause

`AddTaskWithPhases` routes every top-level add with a phase to
`addTaskWithPhaseMarkers`, which always appends to the end of the phase (or
creates the phase at the end of the document) and ignores `op.Position`.

In batches, `sortOperationsForExecution` additionally hoists every positioned
add to the front of the batch. A positioned phase add therefore ran before the
`add-phase` (and any setup adds) it depended on, created the phase itself, and
the later `add-phase` appended a second header.

**Defect type:** Missing validation / unsupported field combination.

**Why it occurred:** Position and phase were each designed independently. A
position is a global hierarchical ID, while a phase add means "end of this
phase", and nothing defined what both together should mean.

**Contributing factors:** The batch sort treats every positioned add as
order-independent.

## Resolution for the Issue

_To be filled in after the fix._

## Regression Test

**Test files:** `internal/task/batch_operations_test.go`, `cmd/add_test.go`
**Test names:** `TestExecuteBatchWithPhases_PhaseWithPositionRejected`,
`TestRunAddPhaseWithPositionRejected`

**What it verifies:** Both T-919 and T-973 batch repros, a positioned add into
an existing phase, and the CLI with and without `--dry-run` fail with a
phase/position conflict error and leave the file unchanged.

**Run command:** `go test ./internal/task -run PhaseWithPositionRejected && go test ./cmd -run TestRunAddPhaseWithPositionRejected`

## Related

- T-919, T-973
- PR #110 (T-1874) separately stops `sortOperationsForExecution` from moving
  positioned adds across other operations.
