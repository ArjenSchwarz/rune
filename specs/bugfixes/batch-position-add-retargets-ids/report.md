# Bugfix Report: Batch Position Adds Retarget Earlier ID Operations

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1874

## Description of the Issue

`sortOperationsForExecution` moved every `add` with a `position` ahead of all other operations. An update/remove listed before the add then ran after it, against renumbered IDs.

**Reproduction steps:**
1. `tasks.md` contains `1. Alpha` and `2. Beta`.
2. Run a batch with `[{"type":"update","id":"1","title":"Alpha updated"},{"type":"add","title":"Inserted","position":"1"}]`.
3. Result: `1. Alpha updated`, `2. Alpha`, `3. Beta` — the inserted task was renamed and the original Alpha untouched.

**Impact:** High. Silent data corruption for any batch mixing positioned adds with earlier ID-based operations (update, remove, status, dependencies), in both phased and non-phased execution.

## Investigation Summary

- **Code inspected:** `internal/task/batch.go` (`sortOperationsForExecution`, `runBatch`), existing tests in `internal/task/batch_add_test.go`, `internal/task/batch_operations_test.go`, `cmd/integration_batch_test.go`.
- **Hypotheses tested:** The hoisting exists so several positioned adds can all reference pre-batch positions (inserted highest first). That only needs reordering among the positioned adds themselves. Every existing test lists its positioned adds first and contiguously, so none depend on cross-kind hoisting.

## Discovered Root Cause

Positioned adds were extracted from the whole list and placed first, crossing operations of other kinds. T-200 fixed the same class of bug for removes by restricting their sort to contiguous blocks; positioned adds were not given the same restriction.

**Defect type:** Logic error (operation reordering)

## Resolution for the Issue

**Changes made:**
- `internal/task/batch.go` - `sortOperationsForExecution` now walks the list and sorts only contiguous runs of the same reorderable kind: positioned adds (reverse position) or removes (reverse ID). Nothing crosses an operation of a different kind. Uses a stable sort so equal keys keep input order. Shared by `ExecuteBatch` and `ExecuteBatchWithPhases` via `runBatch`.

**Approach rationale:** Preserves input sequence across kinds (the ticket's expected behaviour) while keeping the documented "multiple positioned adds reference original positions" behaviour for adjacent adds.

**Alternatives considered:**
- Binding IDs to task identities before execution - rejected: far larger change; input-order execution already gives the expected result.
- Removing the sort for positioned adds entirely - rejected: breaks the existing behaviour that adjacent positioned adds reference pre-batch positions.

## Regression Test

**Test file:** `internal/task/batch_operations_test.go`
**Test names:** `TestExecuteBatch_PositionAddDoesNotRetargetEarlierOps` (non-phased and phased), `TestSortOperationsForExecution_PositionAddsStayInSequence`

**What it verifies:** The ticket's reproduction yields `Inserted, Alpha updated, Beta` on both execution paths; positioned adds are sorted within a run and never cross an update or remove.

**Run command:** `go test -run 'PositionAdd' ./internal/task`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/batch.go` | Block-bounded sorting for positioned adds |
| `internal/task/batch_operations_test.go` | Regression tests |
| `docs/json-api.md`, `docs/agent-notes/batch-command.md` | Documented execution order |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes (pre-existing root-permission failures excepted, see T-1789 report)
- [x] Linters/validators pass

**Manual verification:**
- Built binary on the ticket's reproduction file: output is `1. Inserted`, `2. Alpha updated`, `3. Beta`.

## Known limitations

- `operation N` in error messages is still the index in execution order, which differs from input order inside a reordered run (pre-existing, unchanged).

## Related

- T-200 / `specs/bugfixes/batch-remove-reordering/report.md`
