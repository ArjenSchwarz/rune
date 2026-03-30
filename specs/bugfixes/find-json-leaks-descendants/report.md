# Bugfix Report: find-json-leaks-descendants

**Date:** 2026-03-30
**Status:** Fixed

## Description of the Issue

`find --format json` leaked non-matching descendant tasks into JSON output and could duplicate tasks that matched independently as both nested children and top-level results.

**Reproduction steps:**
1. Create a task file where task 1 title matches a pattern but task 1.1 does not
2. Run `rune find <file> --pattern <parent-term> --format json`
3. Observe that task 1.1 appears in JSON output under task 1's children despite not matching

**Impact:** JSON consumers received incorrect search results — non-matching tasks appeared in output and matching children were duplicated.

## Investigation Summary

- **Symptoms examined:** JSON output contained full nested hierarchies for matched tasks instead of flat matching-only results
- **Code inspected:** `internal/task/search.go` (`findInTasks`, `insertParents`), `cmd/find.go` (`outputSearchResultsJSON`), `internal/task/render.go` (`RenderJSON`, `toJSONTask`)
- **Hypotheses tested:** Compared `Find` behavior with `Filter` — Filter correctly strips Children, Find did not

## Discovered Root Cause

**Defect type:** Logic error — missing data sanitization before output

**Why it occurred:** `findInTasks` appended matched tasks with their full `Children` slices intact (line 114). When `RenderJSON` processed these results via `toJSONTask`, it recursively serialized all nested children regardless of whether they matched the search pattern. Similarly, `insertParents` spliced in parent tasks from `FindTask` with full children.

**Contributing factors:** The `Filter` function already had the correct pattern (copying tasks without Children), but `Find` was written independently and didn't follow the same approach. No tests verified that Find results had empty Children slices.

## Resolution for the Issue

**Changes made:**
- `internal/task/search.go:113-120` - Copy matched task and nil out Children before appending to results
- `internal/task/search.go:58-67` - Copy parent task and nil out Children in `insertParents`

**Approach rationale:** Matches the established pattern from `filterTasks` — the recursive walk evaluates each child independently, so carrying the original Children slice is unnecessary and harmful.

**Alternatives considered:**
- Stripping children in `outputSearchResultsJSON` (cmd layer) — rejected because the bug is in the data layer and would leave other callers affected
- Creating a dedicated search-result JSON type — over-engineered for this fix

## Regression Test

**Test file:** `internal/task/search_test.go`
**Test names:**
- `TestTaskList_Find_ExcludesNonMatchingChildren`
- `TestTaskList_Find_NoDuplicateWhenChildAlsoMatches`
- `TestTaskList_Find_IncludeParent_ExcludesChildren`
- `TestTaskList_Find_PreservesAllFields`

**What they verify:** Matched tasks have empty Children slices, no duplication when parent and child both match, inserted parents also have empty Children, and all non-Children fields are preserved.

**Run command:** `go test -run "TestTaskList_Find_Excludes|TestTaskList_Find_NoDuplicate|TestTaskList_Find_IncludeParent_Excludes|TestTaskList_Find_PreservesAllFields" -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/search.go` | Strip Children from matched tasks in `findInTasks` and inserted parents in `insertParents` |
| `internal/task/search_test.go` | Add 4 regression tests verifying Children are excluded from Find results |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes
- [x] Code formatted with `make fmt`

## Prevention

**Recommendations to avoid similar bugs:**
- When returning task subsets, always strip Children to produce flat result lists — the recursive walk handles child evaluation independently
- Follow the established pattern in `filterTasks` for any new search/query functions

## Related

- Transit ticket: T-629
- Similar fix pattern: `filterTasks` in `internal/task/search.go` (lines 165-178)
