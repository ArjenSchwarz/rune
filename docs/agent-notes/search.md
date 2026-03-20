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

## How Filter Works

`TaskList.Filter(filter)` performs a recursive depth-first walk, evaluating each task against the `QueryFilter` criteria (status, max depth, parent ID, title pattern). Results are returned as a flat slice of task copies.

**Important**: Result tasks have their `Children` field set to nil. The recursive walk evaluates children independently, so the `Children` field on result tasks must not carry the original children — otherwise non-matching descendants leak into the results. This was a bug fixed in T-515.

The `cmd/list.go` layer has its own recursive filtering (`filterTasksRecursive` for JSON, `flattenTasksWithFilters` for table) that operates on the original tree with full children. Those functions handle child filtering themselves and don't use `TaskList.Filter`.

## FindTask vs Find

`FindTask(taskID)` does an exact ID lookup (returns `*Task` pointer into the tree). `Find(pattern, opts)` does a substring search (returns copies). They serve different purposes and both traverse the tree recursively.
