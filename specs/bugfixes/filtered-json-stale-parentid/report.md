# Bugfix Report: Filtered JSON Output Keeps Stale ParentID After Promotion

**Date:** 2026-03-27
**Status:** Fixed
**Ticket:** T-549

## Description of the Issue

When tasks are filtered (via `list --filter` or `find` with status/depth filters) and some parent tasks are excluded from the result, remaining child tasks still carry their original `ParentID` pointing to the now-absent parent. JSON consumers that rebuild hierarchy trees from `ParentID` encounter dangling references.

**Reproduction steps:**
1. Create a task file with a pending parent and a completed child
2. Run `rune list tasks.md --filter completed --format json`
3. Observe the child appears at the top level but its `ParentID` still references the excluded parent

**Impact:** JSON consumers (AI agents, APIs) receive tasks with dangling parent references, breaking hierarchy reconstruction and count calculations.

## Investigation Summary

- **Symptoms examined:** Promoted child tasks in JSON output had ParentID referencing non-existent tasks
- **Code inspected:** `cmd/list.go` (`filterTasksRecursive`), `cmd/find.go` (`applyAdditionalFilters`), `internal/task/render.go` (`toJSONTask`)
- **Hypotheses tested:** Checked whether the JSON renderer (`toJSONTask`) could validate ParentIDs — but the fix belongs in the filtering layer which has the context about what was excluded

## Discovered Root Cause

**Defect type:** Logic error — missing state update during tree restructuring

**Why it occurred:** `filterTasksRecursive` in `cmd/list.go` correctly promotes children when their parent is excluded by filters, but never updates the promoted children's `ParentID` field. Similarly, `applyAdditionalFilters` in `cmd/find.go` removes tasks from a flat list without fixing remaining tasks' ParentID references.

**Contributing factors:** The `ParentID` field was treated as immutable metadata rather than a relationship that needs maintenance when the referenced task is removed from the output.

## Resolution for the Issue

**Changes made:**
- `cmd/list.go:filterTasksRecursive` — When promoting children past an excluded parent, set each child's `ParentID` to the excluded parent's `ParentID`. This correctly chains through multiple levels of promotion.
- `cmd/find.go:applyAdditionalFilters` — After filtering, walk each task's `ParentID` up the hierarchy until finding a surviving ancestor or an ancestor that was never in the search results (hierarchy context, not stale). Respects `--parent` filter value as intentional context.

**Approach rationale:** Fixing ParentID at the filtering layer (rather than the rendering layer) keeps the data model consistent for all output formats, not just JSON.

**Alternatives considered:**
- Validate/fix in `toJSONTask` — rejected because it would only fix JSON, not other consumers of filtered task lists
- Always clear ParentID on promotion — rejected because it loses information when a grandparent survives

## Regression Test

**Test file:** `cmd/list_test.go`, `cmd/find_test.go`
**Test names:**
- `TestFilterTasksRecursiveClearsStaleParentID` (5 sub-cases)
- `TestFilteredJSONOutputNoStaleParentIDs`
- `TestApplyAdditionalFiltersClearsStaleParentID` (4 sub-cases)

**What it verifies:** Promoted/filtered tasks have ParentID pointing to a surviving ancestor (or empty for root), never to an absent task.

**Run command:** `go test -run "TestFilterTasksRecursiveClearsStaleParentID|TestFilteredJSONOutputNoStaleParentIDs|TestApplyAdditionalFiltersClearsStaleParentID" -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/list.go` | Update `filterTasksRecursive` to set promoted children's ParentID to excluded parent's ParentID |
| `cmd/find.go` | Add stale ParentID cleanup after filtering in `applyAdditionalFilters` |
| `cmd/list_test.go` | Add regression tests for `filterTasksRecursive` ParentID fix |
| `cmd/find_test.go` | Add regression tests for `applyAdditionalFilters` ParentID fix |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Code formatted with `make fmt`

## Prevention

**Recommendations to avoid similar bugs:**
- When restructuring tree data (promoting, removing nodes), always audit relationship fields (ParentID, BlockedBy, etc.) for dangling references
- Consider adding a `ValidateReferences()` method to TaskList that checks all ParentIDs resolve to existing tasks

## Related

- T-549: Filtered JSON output keeps stale ParentID after promotion
