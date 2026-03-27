# Bugfix Report: next-claim-one-no-fallback

**Date:** 2026-03-27
**Status:** Fixed

## Description of the Issue

When using `rune next --claim <agent> --one`, the command could incorrectly report "No ready tasks to claim" even when there were pending, unblocked, unowned tasks available.

**Reproduction steps:**
1. Create a task file where the first task in DFS order is blocked by an incomplete dependency
2. Ensure other tasks are pending, unblocked, and unowned (ready)
3. Run `rune next --claim agent-1 --one`
4. Observe: "No ready tasks to claim" despite ready tasks existing

**Impact:** Agents using `--claim --one` would stall when the DFS-deepest task happened to be blocked, even though claimable work existed.

## Investigation Summary

The `runNextWithClaim` function in `cmd/next.go` follows a special code path when `--one` is set. It computes a flat list of `readyTasks` but then ignores it in favour of the DFS-deepest task from `FindNextIncompleteTask` + `FilterToFirstIncompletePath`.

- **Symptoms examined:** Empty claim response despite ready tasks in the task list
- **Code inspected:** `cmd/next.go:runNextWithClaim` (lines 217–238), `findDeepestTask`, `isTaskReady`
- **Hypotheses tested:** Confirmed that the `oneFlag` branch never falls back to `readyTasks[0]` when the DFS leaf is blocked

## Discovered Root Cause

**Defect type:** Logic error — missing fallback

In the `oneFlag` branch of the `default` case in `runNextWithClaim`, the code:
1. Finds the DFS-deepest incomplete task via `FindNextIncompleteTask` + `FilterToFirstIncompletePath` + `findDeepestTask`
2. Checks if that deepest task is ready (`isTaskReady`)
3. If not ready: does nothing — `taskIDsToClaim` stays empty

The non-`oneFlag` branch correctly claims `readyTasks[0]`, but the `oneFlag` branch had no equivalent fallback.

**Why it occurred:** The `--one --claim` combination was added to prefer the DFS-deepest leaf, but the blocked-task guard (added in T-288) did not include a fallback path.

**Contributing factors:** The T-288 regression test asserted the wrong expected behavior — it validated the empty response as correct rather than expecting a fallback to a ready task.

## Resolution for the Issue

**Changes made:**
- `cmd/next.go:230–234` — Added fallback: when the DFS leaf is nil or blocked, claim `readyTasks[0]` instead of returning empty

**Approach rationale:** This is the minimal change that makes `--one --claim` consistent with plain `--claim` behavior. The DFS path is still preferred when its leaf is ready; the fallback only activates when it isn't.

**Alternatives considered:**
- Walking the DFS path to find the deepest *ready* ancestor — rejected as over-engineering; the flat `readyTasks` list already provides a deterministic selection

## Regression Test

**Test file:** `cmd/next_test.go`
**Test names:**
- `TestNextCommandOneWithClaimFallsBackToReadyTask` — T-615 specific repro
- `TestNextCommandOneWithClaimSkipsBlockedTasks` — updated from T-288 to expect fallback

**What it verifies:** When the DFS-deepest task is blocked, `--claim --one` still claims a ready task via fallback.

**Run command:** `go test -run 'TestNextCommandOneWithClaimFallsBackToReadyTask|TestNextCommandOneWithClaimSkipsBlockedTasks' -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/next.go` | Added fallback to `readyTasks[0]` in `oneFlag` branch |
| `cmd/next_test.go` | Updated T-288 test expectations, added T-615 regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Code formatted (`make fmt`)

**Manual verification:**
- Confirmed the fix only adds a 3-line fallback; no changes to unrelated logic

## Prevention

**Recommendations to avoid similar bugs:**
- When adding guard conditions (like the T-288 blocked-task check), always consider the fallback path
- Tests for "skip blocked task" scenarios should verify that an alternative task IS claimed, not just that the blocked one isn't

## Related

- T-615: `next --claim --one` can report no claimable task despite ready tasks
- T-288: Original blocked-task guard (whose test asserted the buggy empty-response behavior)
