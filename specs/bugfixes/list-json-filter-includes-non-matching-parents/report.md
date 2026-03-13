# Bugfix Report: List JSON Filter Includes Non-Matching Parents

**Date:** 2026-03-13
**Status:** Fixed

## Description of the Issue

When using `rune list --json` with `--filter`, `--stream`, or `--owner` flags, non-matching parent tasks are included in JSON output if any of their children match the filter. Table output correctly excludes these non-matching parents, causing a divergence between the two formats.

**Reproduction steps:**
1. Create a task file with a parent task in stream 1 and a child task in stream 2
2. Run `rune list tasks.md --stream 2 --format table` -- parent is excluded
3. Run `rune list tasks.md --stream 2 --format json` -- parent is incorrectly included

**Impact:** Consumers of JSON output (typically AI agents) receive tasks that do not match the requested filters, leading to incorrect task counts and potentially wrong task assignments.

## Investigation Summary

- **Symptoms examined:** JSON output includes parent tasks that do not match status/stream/owner filters when a child matches
- **Code inspected:** `cmd/list.go` (`flattenTasksWithFilters`, `filterTasksRecursive`, `outputJSONWithFilters`)
- **Root cause identified:** `filterTasksRecursive` has an `else if` branch that includes non-matching parents when filtered children exist

## Discovered Root Cause

In `cmd/list.go`, the `filterTasksRecursive` function (used by the JSON output path) contains this logic:

```go
if matchesFilters(&t, opts) {
    // include task with filtered children
} else if len(filteredChildren) > 0 {
    // Include parent if any children match  <-- BUG
    taskCopy := t
    taskCopy.Children = filteredChildren
    result = append(result, taskCopy)
}
```

The `else if` branch includes a parent task that does not match the filters solely because one of its children matches. The table output path (`flattenTasksWithFilters`) has no such logic -- it skips non-matching parents and recursively processes children independently.

**Defect type:** Logic error -- semantic divergence between two filter implementations

**Why it occurred:** The JSON path preserves the hierarchical tree structure and was designed to keep parent context for matching children. The table path flattens the hierarchy and applies filters per-task independently. The two approaches were implemented separately and the "include parent for context" behaviour was never intended to be a filter match.

## Resolution for the Issue

**Changes made:**
- `cmd/list.go`: Removed the `else if len(filteredChildren) > 0` branch from `filterTasksRecursive`. Non-matching parents are no longer included. Matching children are promoted to the parent level in the result list, preserving them in the output while excluding non-matching ancestors.

**Approach rationale:** This aligns JSON output with table output by applying the same per-task filter logic. Children that match are included regardless of whether their parent matches, but the parent is excluded when it does not match.

**Alternatives considered:**
- **Keep JSON behaviour and change table output to match** -- Rejected because including non-matching tasks defeats the purpose of filtering, and table output behaviour is the correct expectation
- **Add a flag to control parent inclusion** -- Rejected as unnecessary complexity; filters should consistently exclude non-matching tasks

## Regression Test

**Test file:** `cmd/list_test.go`
**Test names:** `TestFilterTasksRecursiveExcludesNonMatchingParents`, `TestFilterTasksRecursiveMatchesTableOutput`

**What it verifies:** Non-matching parents are excluded from `filterTasksRecursive` output, and the set of task IDs from the JSON path matches the set from the table path for identical filter options.

**Run command:** `go test -run "TestFilterTasksRecursiveExcludesNonMatchingParents|TestFilterTasksRecursiveMatchesTableOutput" ./cmd/ -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/list.go` | Fixed `filterTasksRecursive` to exclude non-matching parents |
| `cmd/list_test.go` | Added regression tests for filter consistency |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass
