# Search Module Notes

## File Structure

- `internal/task/search.go` - Core search logic: `Find`, `findInTasks`, `Filter`, `filterTasks`, `insertParents`
- `internal/task/search_test.go` - Unit tests for `Find`, `Filter`, `FindTask`, `getTaskDepth`
- `cmd/find.go` - CLI command: `find` with flags `--pattern`, `--search-details`, `--search-refs`, `--case-sensitive`, `--include-parent`, `--status`, `--max-depth`, `--parent`
- `cmd/find_test.go` - Command-level tests

## How Find Works

`TaskList.Find(pattern, opts)` performs a recursive depth-first search through the task tree. It checks titles (always), details (if `SearchDetails`), and references (if `SearchRefs`). Case sensitivity is controlled by `CaseSensitive`.

When `IncludeParent` is true, a post-processing step (`insertParents`) runs after the recursive search. It walks the results front-to-back, and for each task with a `ParentID` whose parent is not already in the results, it looks up the parent via `FindTask` and splices it in immediately before the child. A `present` map prevents duplicates.

## Post-Search Filtering

The `cmd/find.go` layer applies additional filters (`applyAdditionalFilters`) after `Find` returns. These filters (status, max-depth, parent ID) are separate from `QueryOptions` and operate on the flat results slice. This means `--include-parent` parents can be filtered out by `--status` or `--max-depth` if they don't match — this is by design since the parent is included for context, not as a forced result.

## FindTask vs Find

`FindTask(taskID)` does an exact ID lookup (returns `*Task` pointer into the tree). `Find(pattern, opts)` does a substring search (returns copies). They serve different purposes and both traverse the tree recursively.
