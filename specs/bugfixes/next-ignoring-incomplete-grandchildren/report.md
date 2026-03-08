# Bugfix Report: Next Command Ignoring Incomplete Grandchildren

**Date:** 2026-03-09
**Status:** Fixed

## Description of the Issue

The `next` command does not properly account for incomplete grandchildren when determining which tasks to display or claim. When a child task is marked as completed but has pending descendants, the child is excluded from results because the filtering only checks direct status, not the full hierarchy.

**Reproduction steps:**
1. Create a task list with a parent task containing a completed child that has a pending grandchild
2. Run `rune next` — the incomplete grandchild branch is shown (FindNextIncompleteTask works correctly)
3. Run `rune next --stream N` — the completed child with pending grandchildren is excluded from IncompleteChildren
4. Run `rune next --one --claim agent` — findDeepestTask stops at the parent because filterIncompleteChildren returns no children

**Impact:** Tasks with incomplete grandchildren under completed parents are invisible to stream filtering, the `--one` deepest-task selector, and markdown/table rendering of subtasks.

## Investigation Summary

The `internal/task/next.go` package has a correct `filterIncompleteChildren` function that uses `hasIncompleteWork` to check the full hierarchy. However, `cmd/next.go` defines its own `filterIncompleteChildren` that only checks `child.Status != task.Completed`.

- **Symptoms examined:** Stream output and `--one` selection skip branches where a parent is completed but has pending descendants
- **Code inspected:** `cmd/next.go` (filterIncompleteChildren, addIncompleteChildrenToData, renderTaskMarkdown), `internal/task/next.go` (filterIncompleteChildren, hasIncompleteWork)
- **Hypotheses tested:** The duplicate function in cmd/next.go was identified as doing a shallow status check vs the internal package's full hierarchy check

## Discovered Root Cause

`cmd/next.go` defines its own `filterIncompleteChildren` (line 308) that checks `child.Status != task.Completed` instead of traversing the hierarchy. The same shallow check exists in `addIncompleteChildrenToData` (line 530) and `renderTaskMarkdown` (line 565).

**Defect type:** Logic error — shallow status check instead of recursive hierarchy check

**Why it occurred:** The `internal/task` package's `filterIncompleteChildren` is unexported, so `cmd/next.go` created its own copy with simplified (incorrect) logic.

**Contributing factors:** The two functions have the same name but different semantics, making the discrepancy non-obvious.

## Resolution for the Issue

**Changes made:**
- `internal/task/next.go` — Export `HasIncompleteWork` as a public function
- `cmd/next.go:308-316` — Fix `filterIncompleteChildren` to use `task.HasIncompleteWork` instead of shallow status check
- `cmd/next.go:530` — Fix `addIncompleteChildrenToData` to use `task.HasIncompleteWork`
- `cmd/next.go:565` — Fix `renderTaskMarkdown` to use `task.HasIncompleteWork`

**Approach rationale:** Export the existing correct function from the task package rather than duplicating the logic. This ensures a single source of truth for "has incomplete work" checks.

**Alternatives considered:**
- Duplicate the full recursive logic in cmd/next.go — rejected because it creates maintenance burden and risks future divergence

## Regression Test

**Test file:** `cmd/next_test.go`
**Test names:** `TestFilterIncompleteChildrenWithGrandchildren`, `TestFindDeepestTaskWithCompletedIntermediate`, `TestAddIncompleteChildrenToDataWithGrandchildren`, `TestRenderTaskMarkdownWithGrandchildren`

**What it verifies:** That completed children with pending grandchildren are included in filtering, deepest-task traversal, table data generation, and markdown rendering.

**Run command:** `go test -run "TestFilterIncompleteChildrenWithGrandchildren|TestFindDeepestTaskWithCompletedIntermediate|TestAddIncompleteChildrenToDataWithGrandchildren|TestRenderTaskMarkdownWithGrandchildren" -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/next.go` | Export `HasIncompleteWork` |
| `cmd/next.go` | Fix `filterIncompleteChildren`, `addIncompleteChildrenToData`, `renderTaskMarkdown` to check full hierarchy |
| `cmd/next_test.go` | Add 4 regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When a package has utility functions needed externally, export them rather than creating duplicates with simplified logic
- Consider linting for duplicate function names across packages with different semantics

## Related

- Transit ticket: T-358
