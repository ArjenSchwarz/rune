# Bugfix Report: Batch Remove Reordering Crosses Non-Remove Ops

**Date:** 2026-03-04
**Status:** Fixed

## Description of the Issue

When processing batch operations, the `sortOperationsForExecution` function sorted all remove operations globally by reverse ID order, then reinserted them at their original indices. This caused removes to be reordered across non-remove operations (add, update) when they were interleaved, changing the execution semantics.

**Reproduction steps:**
1. Submit a batch with operations: `[remove ID=1, add "foo", remove ID=2]`
2. The sort function collects removes globally and sorts them reverse: `[remove(2), remove(1)]`
3. Reinserting at original indices produces: `[remove ID=2, add "foo", remove ID=1]`
4. Remove ID=2 now executes before the add, and remove ID=1 executes after -- the opposite of what the user specified

**Impact:** Any batch that interleaved remove operations with non-remove operations could produce incorrect results. The removes would execute in a different order relative to adds/updates than the user intended.

## Investigation Summary

- **Symptoms examined:** The `sortOperationsForExecution` function comment claimed it preserved relative position of removes to non-removes, but the implementation did not uphold this invariant.
- **Code inspected:** `internal/task/batch.go`, specifically the remove-sorting logic in `sortOperationsForExecution`.
- **Hypotheses tested:** The global collect-sort-reinsert approach was identified as the root cause. Sorting within contiguous blocks was confirmed as the correct approach.

## Discovered Root Cause

The sorting algorithm collected all remove operations regardless of their position in the operation list, sorted them as a single group, and then placed the sorted removes back at the original indices. When removes were separated by non-remove operations, this caused removes from later positions to appear at earlier indices (and vice versa), crossing the non-remove boundaries.

**Defect type:** Logic error

**Why it occurred:** The algorithm treated all removes as a single sortable group rather than respecting the boundaries created by intervening non-remove operations.

**Contributing factors:** The function comment described the intended behavior (preserving relative position to non-removes) but the implementation did not match.

## Resolution for the Issue

**Changes made:**
- `internal/task/batch.go:57-77` - Replaced global remove collection and reinsertion with contiguous block sorting. The new algorithm walks through operations, identifies contiguous runs of remove operations, and sorts each block independently in reverse ID order.

**Approach rationale:** Sorting within contiguous blocks is the minimal change that fixes the bug while preserving the existing behavior for the common case (all removes grouped together). It correctly respects non-remove operations as boundaries that removes should not cross.

**Alternatives considered:**
- Removing the sort entirely -- rejected because reverse-order sorting within a block of removes is necessary for correct ID-based removal (removing highest ID first preserves lower IDs).
- Tracking original positions more carefully -- rejected as it adds complexity without benefit over the simpler block-based approach.

## Regression Test

**Test file:** `internal/task/batch_operations_test.go`
**Test name:** `TestSortOperationsForExecution_RemoveBlockBoundaries` and `TestExecuteBatch_RemoveAddRemovePreservesSemantics`

**What it verifies:** The unit test covers six scenarios including removes that should not cross add operations, removes that should not cross update operations, contiguous blocks sorted correctly, and edge cases. The integration-style test verifies end-to-end batch execution with interleaved remove and add operations.

**Run command:** `go test -run "TestSortOperationsForExecution_RemoveBlockBoundaries|TestExecuteBatch_RemoveAddRemovePreservesSemantics" -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/batch.go` | Fixed remove sorting to use contiguous blocks instead of global sorting |
| `internal/task/batch_operations_test.go` | Added regression tests for block-bounded remove sorting |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When sorting a subset of items within a larger sequence, verify that the sort does not reorder items across boundary elements
- Ensure function comments and implementations stay in sync; the comment described the correct behavior but the code did not implement it

## Related

- Transit ticket: T-200
