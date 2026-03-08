# Bugfix Report: Prevent Blocked Tasks from Being Returned/Claimed with --stream

**Date:** 2026-03-09
**Status:** Fixed

## Description of the Issue

When using `rune next --stream N`, blocked child tasks that match the target stream are returned and can be claimed. The `--stream` flag should only return tasks that are ready (pending, unblocked, unclaimed), but `FilterByStream` recurses into `.Children` of ready tasks and picks up blocked children.

**Reproduction steps:**
1. Create a task file where a ready task in stream 1 has a child in stream 2 that is blocked by an incomplete task
2. Run `rune next tasks.md --stream 2 --format json`
3. Observe that the blocked child task is returned
4. Run `rune next tasks.md --stream 2 --claim agent-1 --format json`
5. Observe that the blocked child task is claimed

**Impact:** Agents using `--stream` or `--stream --claim` can receive or claim tasks whose dependencies are not yet met, leading to premature work.

## Investigation Summary

- **Symptoms examined:** `--stream` returns blocked children; `--stream --claim` claims blocked children
- **Code inspected:** `cmd/next.go` (`runNextWithStream`, `runNextWithClaim`), `internal/task/streams.go` (`FilterByStream`)
- **Root cause identified:** `FilterByStream` recurses into `.Children`, but `getReadyTasks` produces a flat list where each task retains its full `.Children` slice including non-ready children

## Discovered Root Cause

In `cmd/next.go`, both `runNextWithStream` (line 156) and `runNextWithClaim` with `streamFlag > 0` (line 213) call `getReadyTasks()` followed by `FilterByStream()`. `getReadyTasks` correctly produces a flat list of ready tasks (pending, unblocked, unclaimed), but each task in that list retains its original `.Children` slice. `FilterByStream` recurses into `.Children`, which can contain blocked, in-progress, or owned tasks that should not be returned.

**Defect type:** Incorrect recursion scope

**Why it occurred:** `FilterByStream` was made recursive (in T-170) to handle hierarchical phase task lists correctly. However, when applied to the already-flattened output of `getReadyTasks`, that recursion reaches into children that were not themselves validated as ready.

## Resolution

Added `FilterByStreamFlat` to `internal/task/streams.go` — a non-recursive variant of `FilterByStream` that only examines the top-level slice. Updated the two call sites in `cmd/next.go` where `FilterByStream` is called on the output of `getReadyTasks` to use `FilterByStreamFlat` instead.

The existing recursive `FilterByStream` is preserved for callers that need hierarchical traversal (e.g., `FindNextPhaseTasksForStream`).

### Alternatives considered

- **Strip `.Children` from `getReadyTasks` output** — Would prevent the recursion issue but loses child context that is used for display purposes (e.g., `filterIncompleteChildren` in `runNextWithStream`)
- **Add a readiness check inside `FilterByStream`** — Would change the semantics for all callers and break the phase-stream path where blocked tasks are intentionally included in output

### Files changed

| File | Change |
|------|--------|
| `internal/task/streams.go` | Added `FilterByStreamFlat` function |
| `internal/task/streams_test.go` | Added `TestFilterByStreamFlat_DoesNotRecurse` and `TestFilterByStreamFlat_EmptyResult` |
| `cmd/next.go` | Changed two `FilterByStream` calls to `FilterByStreamFlat` |
| `cmd/next_test.go` | Added `TestNextStreamExcludesBlockedChildren` and `TestNextStreamClaimExcludesBlockedChildren` |

## Test Verification

**Test names:** `TestFilterByStreamFlat_DoesNotRecurse`, `TestFilterByStreamFlat_EmptyResult`, `TestNextStreamExcludesBlockedChildren`, `TestNextStreamClaimExcludesBlockedChildren`

**Run command:** `go test -run "TestFilterByStreamFlat|TestNextStreamExcludesBlockedChildren|TestNextStreamClaimExcludesBlockedChildren" ./internal/task/ ./cmd/ -v`
