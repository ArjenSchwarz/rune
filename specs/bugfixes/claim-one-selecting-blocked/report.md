# Bugfix Report: Claim One Selecting Blocked Tasks

**Date:** 2026-03-04
**Status:** Fixed

## Description of the Issue

The `next --claim <agent> --one` command ignores dependency blocking checks when selecting a task to claim. It traverses the task hierarchy using `FindNextIncompleteTask` and `FilterToFirstIncompletePath` to find the deepest incomplete leaf task, then claims it without verifying the task is actually ready (pending, unblocked, no owner).

**Reproduction steps:**
1. Create a task file where the deepest leaf task on the `--one` path has a `Blocked-by:` dependency on an incomplete task
2. Run `rune next tasks.md --one --claim agent-1 --format json`
3. Observe that the blocked task is claimed anyway, with `blockedBy` populated in the output

**Impact:** Agents using `--claim --one` can claim tasks that are blocked by incomplete dependencies, leading to premature work on tasks whose prerequisites are not met.

## Investigation Summary

- **Symptoms examined:** `--one --claim` claims blocked tasks
- **Code inspected:** `cmd/next.go` — the `runNextWithClaim` function, specifically the `default` case (no --stream or --phase) when `oneFlag` is true
- **Hypotheses tested:** Single hypothesis — the `--one` branch bypasses the ready-task check by using `FindNextIncompleteTask` instead of consulting the dependency index

## Discovered Root Cause

In `runNextWithClaim` (lines 217-237 of `cmd/next.go`), the `default` case first computes `readyTasks` (pending, unblocked, unclaimed), but when `oneFlag` is true, it ignores `readyTasks` entirely. Instead, it calls `FindNextIncompleteTask` (which only checks completion status, not blocking), applies `FilterToFirstIncompletePath`, and claims whatever leaf task it finds.

**Defect type:** Missing validation

**Why it occurred:** The `--one` flag was added after the claim logic, and the `--one` path reused `FindNextIncompleteTask` which was designed for display purposes (showing what comes next) rather than claim eligibility.

**Contributing factors:** The non-`--one` branch correctly uses `readyTasks[0]`, but the `--one` branch diverges to a completely different code path that lacks the readiness check.

## Resolution for the Issue

**Changes made:**
- `cmd/next.go` — Added `isTaskReady` helper function that checks pending status, not blocked, and no owner (same criteria as `getReadyTasks`)
- `cmd/next.go` — Added an `isTaskReady` guard in the `--one` branch before adding the deepest task to `taskIDsToClaim`

**Approach rationale:** The fix is minimal: a single condition check on the already-computed `deepestTask` using the already-available `index`. If the deepest task is blocked, no task is added to `taskIDsToClaim`, and the existing empty-result handling returns "No ready tasks to claim".

**Alternatives considered:**
- Walking up the `--one` path to find the deepest ready ancestor — rejected because claiming a parent task when the specific leaf is blocked would be confusing; better to report nothing available
- Filtering `readyTasks` to find one matching the `--one` path — more complex without benefit; the simple guard achieves the same result

## Regression Test

**Test file:** `cmd/next_test.go`
**Test names:**
- `TestNextCommandOneWithClaimSkipsBlockedTasks` — verifies that a blocked leaf task on the `--one` path is NOT claimed
- `TestNextCommandOneWithClaimSelectsReadyLeaf` — verifies that a leaf task whose blocker is completed IS claimed normally

**Run command:** `go test -run "TestNextCommandOneWithClaimSkipsBlockedTasks|TestNextCommandOneWithClaimSelectsReadyLeaf" -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/next.go` | Added `isTaskReady` helper; added readiness guard in `--one` claim path |
| `cmd/next_test.go` | Added two regression tests for blocked and ready leaf scenarios |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass (`make check`)

**Manual verification:**
- Confirmed the regression test fails before the fix and passes after

## Prevention

**Recommendations to avoid similar bugs:**
- When adding new flag combinations that interact with claim logic, ensure the readiness check (pending, unblocked, no owner) is applied regardless of the code path
- The `getReadyTasks` function encapsulates readiness criteria; new claim paths should either use it or explicitly check the same conditions

## Related

- Transit ticket: T-288
