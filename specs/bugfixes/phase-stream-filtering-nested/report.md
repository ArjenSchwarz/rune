# Bugfix Report: Phase Stream Filtering Ignores Nested Tasks

**Date:** 2026-03-04
**Status:** Fixed

## Description of the Issue

When filtering tasks by stream within a phase (`rune next --phase --stream N`), nested subtasks that belong to the target stream were not included in the results. Only top-level tasks with the matching stream tag were returned, ignoring children that have their own stream assignments.

**Reproduction steps:**
1. Create a task file with phases containing hierarchical tasks where subtasks have different stream assignments than their parents
2. Run `rune next --phase --stream N` where N is the stream of a nested subtask
3. Observe that the nested subtask is not returned, even though it belongs to the requested stream

**Impact:** Users relying on `--phase --stream` to find work in a specific stream would miss valid ready tasks nested under parents in different streams. This affected both task discovery and task claiming workflows.

## Investigation Summary

- **Symptoms examined:** `FindNextPhaseTasksForStream` returned nil or incomplete results when target-stream tasks were nested under parents in other streams
- **Code inspected:** `FilterByStream` in `streams.go`, `hasReadyTaskInStream` and `FindNextPhaseTasksForStream` in `next.go`, callers in `cmd/next.go`
- **Hypotheses tested:** Confirmed that `FilterByStream` only checked the top-level slice without recursing into `Children`, and that `hasReadyTaskInStream` inherited the same limitation

## Discovered Root Cause

`FilterByStream` iterated only over the top-level tasks in the input slice without recursing into `task.Children`. Since `extractPhasesWithTaskRanges` builds phase task lists with only top-level tasks (which have their children attached), any subtask with a different stream than its parent was invisible to the stream filter.

**Defect type:** Logic error -- missing recursion

**Why it occurred:** `FilterByStream` was written as a flat filter, matching the pattern of the input it originally received (flat lists from `getReadyTasks`). When it was later used in the phase context where tasks are hierarchical, the lack of recursion became a bug.

**Contributing factors:** `AnalyzeStreams` already handled recursion correctly (it has its own `processTasks` recursive closure), so the inconsistency between it and `FilterByStream` was not immediately obvious.

## Resolution for the Issue

**Changes made:**
- `internal/task/streams.go:102` - Made `FilterByStream` recursive: it now uses an inner `collect` closure that traverses children at all levels, collecting any task whose effective stream matches the filter

**Approach rationale:** Making `FilterByStream` itself recursive is the simplest fix. All existing callers either pass flat lists (where recursion is a no-op) or hierarchical lists (where recursion is needed). No caller semantics change negatively.

**Alternatives considered:**
- Create a separate `FilterByStreamRecursive` function for the phase context - Rejected because it would duplicate logic and the existing callers are safe with recursion
- Fix only `hasReadyTaskInStream` and `FindNextPhaseTasksForStream` to manually recurse - Rejected because the root cause is in `FilterByStream` and fixing it there prevents the same bug from recurring in future callers

## Regression Test

**Test files:** `internal/task/streams_test.go`, `internal/task/next_test.go`
**Test names:** `TestFilterByStream_NestedTasks`, `TestHasReadyTaskInStream_NestedTasks`, `TestFindNextPhaseTasksForStream_NestedTasks`

**What they verify:**
- `FilterByStream` finds tasks at all nesting levels matching the target stream
- `hasReadyTaskInStream` detects ready tasks nested inside parents with different streams
- `FindNextPhaseTasksForStream` returns nested stream tasks from the correct phase, skips phases without matching nested tasks, handles deeply nested tasks, and excludes completed nested tasks

**Run command:** `go test -run "TestFilterByStream_NestedTasks|TestHasReadyTaskInStream_NestedTasks|TestFindNextPhaseTasksForStream_NestedTasks" ./internal/task/ -v`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/streams.go` | Made `FilterByStream` recursive to traverse children |
| `internal/task/streams_test.go` | Added `TestFilterByStream_NestedTasks` regression test |
| `internal/task/next_test.go` | Added `TestHasReadyTaskInStream_NestedTasks` and `TestFindNextPhaseTasksForStream_NestedTasks` regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When writing filter/search functions over hierarchical data, always consider whether recursion is needed and document the decision
- `AnalyzeStreams` already handled recursion correctly; using it as a reference pattern would have caught this earlier
- Consider adding a code review checklist item: "Does this function need to handle nested children?"

## Related

- Transit ticket: T-170
