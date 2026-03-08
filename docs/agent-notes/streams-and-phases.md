# Streams and Phases

## Key Functions

- `FilterByStream(tasks, stream)` in `internal/task/streams.go` -- filters tasks by effective stream. Recurses into children (fixed in T-170).
- `AnalyzeStreams(tasks, index)` in `internal/task/streams.go` -- computes stream status (ready/blocked/active). Already recursive via its own `processTasks` closure.
- `FindNextPhaseTasksForStream(filepath, stream)` in `internal/task/next.go` -- finds stream tasks in first phase with a ready task in that stream. Uses `FilterByStream` and `hasReadyTaskInStream`.
- `hasReadyTaskInStream(tasks, stream, index)` in `internal/task/next.go` -- checks if any task in the stream is ready. Delegates to `FilterByStream`.
- `extractPhasesWithTaskRanges(lines, allTasks)` in `internal/task/next.go` -- builds phase-to-task mapping. Only adds top-level tasks to phases, but those tasks carry their children.

## Data Flow for `--phase --stream`

1. File is parsed into hierarchical `TaskList`
2. `extractPhasesWithTaskRanges` associates top-level tasks (with children) to phases
3. `hasReadyTaskInStream` checks each phase for ready tasks in the target stream
4. `FilterByStream` collects all matching tasks (including nested) from the selected phase
5. Non-completed matching tasks are returned

## Gotchas

- `FilterByStream` returns a flat list of matching tasks from all nesting levels. Callers that expect hierarchical output should be aware.
- `FilterByStream` deduplicates by task ID. This is needed because `getReadyTasks` in `cmd/next.go` flattens the hierarchy but preserves `Children` on each task struct. Without deduplication, a child task could appear twice: once from recursing into its parent's `Children`, and once as a direct entry in the flat list.
- `GetEffectiveStream` returns 1 for tasks with `Stream <= 0`. This means untagged tasks default to stream 1.
- `RenderJSONWithPhases` builds `[]TaskWithPhase` with `*Task` pointers. These must point to `&tl.Tasks[i]` (slice elements), not to range variable copies. Fixed in T-374.
