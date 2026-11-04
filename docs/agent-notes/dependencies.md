# Dependencies

## BuildDependencyIndex

`BuildDependencyIndex` in `internal/task/dependencies.go` creates a `DependencyIndex` with three maps:

- `byStableID` — maps StableID to `*Task`
- `byHierarchical` — maps hierarchical ID to `*Task`
- `dependents` — maps a blocker's StableID to the list of IDs that depend on it

The dependents map registers tasks even if they lack a StableID (using hierarchical ID as fallback). This was fixed in T-422 — previously only tasks with a StableID were registered as dependents, causing `GetDependents` to miss tasks parsed from markdown without explicit stable ID assignment.

## RemoveTaskWithDependents

`RemoveTaskWithDependents` in `internal/task/operations.go` removes a task and cleans up `BlockedBy` references in all surviving tasks. Key behavior:

- Collects the StableIDs of the removed task *and all its descendants* (T-1772) and strips all of them from surviving `BlockedBy` lists via a tree walk, regardless of what `GetDependents` reports (T-422).
- Returns warnings counting how many surviving tasks had references cleaned up. The `remove` command does not print these warnings.
- The `remove` command (via `RemoveTaskWithPhases`) and batch `remove` both go through this function.

## Readiness and duplicate IDs

- `IsReady` treats a blocker as done only when `hasIncompleteWork(blocker)` is false, i.e. the blocker and its whole subtree are completed (T-2161).
- `IsReady` treats a blocker stable ID that matches no task as not done (T-337), but `TranslateToHierarchical` drops it. Use `UnresolvedBlockers` to report those IDs; JSON puts them in `unresolvedBlockedBy` (never in `blockedBy`, whose schema is hierarchical IDs) and `formatBlockedByForDisplay` in `cmd/helpers.go` shows them as `<id> (unresolved)` (T-1809).
- A `DependencyIndex` used to render a filtered or search result must be built from the full task list, or blockers outside the result disappear. `RenderJSONWithPhases` builds it from `phaseSource` when given (T-1206).
- `AutoCompleteParents` will not complete a parent that `IsBlocked` (T-2126).
- Plain `next` uses `FindNextReadyTask`, which skips tasks that are blocked or have an owner (T-1777). The stream/claim paths use `isTaskReady` in `cmd/next.go`, which additionally requires `Pending` status.
- The parser drops a duplicated StableID from every occurrence after the first and records a `WarnDuplicateStableID` in `TaskList.Warnings`; commands print these to stderr via `reportParseWarnings` (T-1762). `BuildDependencyIndex` is also first-wins. Since parsing dedupes, the rest of the code can assume unique StableIDs.

## StableID Assignment

Tasks get StableIDs in two ways:
1. `AddTaskWithOptions` — generates one automatically via `StableIDGenerator`
2. `resolveToStableIDs` — auto-assigns when a task is referenced as a blocker but lacks a StableID

Tasks parsed from markdown may have `BlockedBy` references (pointing to other tasks' StableIDs) without having their own StableID. This is a valid state — a task only needs a StableID if other tasks need to reference it, not just because it references others.
