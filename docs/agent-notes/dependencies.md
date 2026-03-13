# Dependencies

## BuildDependencyIndex

`BuildDependencyIndex` in `internal/task/dependencies.go` creates a `DependencyIndex` with three maps:

- `byStableID` — maps StableID to `*Task`
- `byHierarchical` — maps hierarchical ID to `*Task`
- `dependents` — maps a blocker's StableID to the list of IDs that depend on it

The dependents map registers tasks even if they lack a StableID (using hierarchical ID as fallback). This was fixed in T-422 — previously only tasks with a StableID were registered as dependents, causing `GetDependents` to miss tasks parsed from markdown without explicit stable ID assignment.

## RemoveTaskWithDependents

`RemoveTaskWithDependents` in `internal/task/operations.go` removes a task and cleans up `BlockedBy` references in all other tasks. Key behavior:

- Always calls `removeFromBlockedByLists` when the removed task has a StableID, regardless of what `GetDependents` reports. This is intentional — the index may not capture all dependents, and the tree walk is cheap. (Fixed in T-422.)
- Returns warnings listing how many tasks had references cleaned up.
- After cleanup, delegates to `removeTaskRecursive` + `RenumberTasks`.

## StableID Assignment

Tasks get StableIDs in two ways:
1. `AddTaskWithOptions` — generates one automatically via `StableIDGenerator`
2. `resolveToStableIDs` — auto-assigns when a task is referenced as a blocker but lacks a StableID

Tasks parsed from markdown may have `BlockedBy` references (pointing to other tasks' StableIDs) without having their own StableID. This is a valid state — a task only needs a StableID if other tasks need to reference it, not just because it references others.
