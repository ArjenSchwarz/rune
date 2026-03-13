# Bugfix Report: Find --include-parent Flag Ignored

**Date:** 2026-03-13
**Status:** Fixed

## Description of the Issue

The `find` command's `--include-parent` flag is wired into the CLI layer and passed through `QueryOptions.IncludeParent` to `TaskList.Find`, but the `Find` method never reads the flag. The result is that `--include-parent` produces identical output to a search without the flag.

**Reproduction steps:**
1. Create a task file with parent/child tasks (e.g., task "1" with child "1.1")
2. Run `rune find <file> --pattern <term-matching-child> --include-parent`
3. Observe that the parent task is absent from the results

**Impact:** Low severity. Users cannot get parent context in search results, reducing the usefulness of the find command for navigating hierarchical task lists.

## Investigation Summary

- **Symptoms examined:** `--include-parent` flag produces the same output as without the flag.
- **Code inspected:** `cmd/find.go` (CLI wiring), `internal/task/search.go` (Find/findInTasks), `internal/task/search_test.go` (existing tests).
- **Hypotheses tested:** The flag is correctly parsed and passed to `QueryOptions.IncludeParent`. The defect is entirely in `findInTasks`, which never checks `opts.IncludeParent`.

## Discovered Root Cause

The `findInTasks` method in `internal/task/search.go` performs a recursive search and appends matching tasks to the results slice. It never checks `opts.IncludeParent`. When a child task matches, its parent is not added to the results.

**Defect type:** Missing implementation — the flag was plumbed through the CLI and data structures but the core logic was never written.

**Why it occurred:** The `IncludeParent` field was added to `QueryOptions` during initial design but the search implementation only handled the other three options (CaseSensitive, SearchDetails, SearchRefs).

**Contributing factors:** The existing test `find_with_parent_context` was incorrect — it set `IncludeParent: true` but only expected the child task, effectively asserting the broken behaviour.

## Resolution for the Issue

**Changes made:**
- `internal/task/search.go` — Rewrote `Find` to perform a two-pass approach when `IncludeParent` is true: first collect matching tasks, then prepend their parents (deduplicated, in tree order).

**Approach rationale:** A post-processing pass keeps `findInTasks` simple and avoids complicating the recursive search with parent-tracking logic. Parents are inserted in tree order (before their children) to produce natural output.

**Alternatives considered:**
- Tracking parent context during recursion by passing the parent task into `findInTasks` — rejected because it adds complexity to every recursive call even when `IncludeParent` is false.

## Regression Test

**Test file:** `internal/task/search_test.go`
**Test names:** `find_with_parent_context`, `find_include_parent_multiple_children_same_parent`, `find_include_parent_details_match`, `find_include_parent_top_level_match_no_extra`, `find_include_parent_disabled`

**Test file:** `cmd/find_test.go`
**Test names:** `include_parent_for_child_match`, `include_parent_top_level_no_extra`

**What it verifies:** That when `IncludeParent` is true, parent tasks of matching children appear in results; that parents are not duplicated; that top-level matches don't produce spurious entries; that the flag has no effect when false.

**Run command:** `go test -run "TestTaskList_Find|TestFindCommand" ./internal/task/ ./cmd/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/search.go` | Implement `IncludeParent` logic in `Find` |
| `internal/task/search_test.go` | Fix existing test, add regression tests |
| `cmd/find_test.go` | Add cmd-level regression tests for `--include-parent` |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When adding a field to an options struct, ensure the consuming function actually reads it. A test that asserts distinct behaviour for both `true` and `false` values catches this.
- Review tests for correctness — the original test asserted the broken behaviour.

## Related

- Transit ticket: T-413
