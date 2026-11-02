# Streams and Phases

## Key Functions

- `FilterByStream(tasks, stream)` in `internal/task/streams.go` -- filters tasks by effective stream. Recurses into children (fixed in T-170). Use only with hierarchical input (e.g., phase tasks).
- `FilterByStreamFlat(tasks, stream)` in `internal/task/streams.go` -- non-recursive stream filter. Use when input is already flattened (e.g., output of `getReadyTasks`). Added in T-347.
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

## Exported Functions

- `HasIncompleteWork(task)` in `internal/task/next.go` -- exported in T-358 so `cmd/next.go` can check if a task or any descendant has incomplete work. Uses `hasIncompleteWorkWithDepth` internally with a max depth of 100.

## Phase Marker Adjustment

When tasks are inserted or removed, phase markers (`PhaseMarker.AfterTaskID`) must be adjusted to reflect new task numbering:

- **Removal**: `adjustPhaseMarkersForRemoval` decrements all markers referencing tasks after the removed task. If a marker references the removed task itself, it shifts to the previous task.
- **Insertion**: every add (the `add` command, batch `add`, and the file-level `AddTaskToPhase` wrapper) goes through `TaskList.AddTaskWithPhases` in `internal/task/batch.go`, which keeps the markers in step:
  - Into a phase (`op.Phase`, no parent): the task is appended to the end of that phase; the immediate next marker is set to the new task's ID and all markers beyond it are shifted via `adjustPhaseMarkersForInsertion`. Combining `op.Phase` with `op.Position` (no parent) is rejected (T-919, T-973): position is a global ID, phase means "end of this phase", and in batches the positioned add used to be hoisted ahead of the `add-phase` it needed, producing duplicate headers.
  - At a root position (`op.Position`, no phase): markers at or after the insertion point are shifted via `adjustPhaseMarkersForInsertion`, so the new task joins the phase of the task that previously held its position.
  - Subtasks (`op.Parent` set) stay under their parent: `op.Phase` is ignored (but still validated) and never creates a phase.
- Both adjustments only apply to top-level tasks (IDs without dots), since subtask changes don't affect phase marker numbering.
- `TaskList.WriteFile` re-reads the markers from the file on disk. That is only correct when the top-level structure is unchanged since the file was read. Any code that inserts, removes or reorders top-level tasks must track markers itself and write with `WriteFileWithPhases`.

## Scanning Lines for Phases

All raw-line phase scanning (`ExtractPhaseMarkers`, `extractPhasesWithTaskRanges`, `buildTaskPhaseMap`, `getTaskPhase`) goes through `walkPhaseLines` in `internal/task/phase.go`. A task line is top-level only if it is unindented, matching the parser, which nests by indentation and ignores the raw IDs. Counting by "raw ID has no dot" (the old approach) mis-mapped every later task when a subtask was hand-written as `  - [ ] 1. child`. Use `phaseMarkersFromContent` to get markers from raw file content; it strips front matter first so YAML `## comment` lines aren't read as phases.

## Gotchas

- `FilterByStream` returns a flat list of matching tasks from all nesting levels. Callers that expect hierarchical output should be aware.
- `FilterByStream` deduplicates by task ID. This is needed because `getReadyTasks` in `cmd/next.go` flattens the hierarchy but preserves `Children` on each task struct. Without deduplication, a child task could appear twice: once from recursing into its parent's `Children`, and once as a direct entry in the flat list.
- **Do not use `FilterByStream` on `getReadyTasks` output** — the recursion into `.Children` will pick up non-ready (blocked/owned/in-progress) children. Use `FilterByStreamFlat` instead (T-347).
- `GetEffectiveStream` returns 1 for tasks with `Stream <= 0`. This means untagged tasks default to stream 1.
- `RenderJSONWithPhases` builds `[]jsonTaskWithPhase` (previously `[]TaskWithPhase` with `*Task` pointers). These must point to `&tl.Tasks[i]` (slice elements), not to range variable copies. Fixed in T-374.
- `RenderJSONWithPhases` accepts an optional `phaseSource *TaskList` parameter. When the rendered task list has been filtered, pass the original unfiltered list as `phaseSource` so that `GetTaskPhase` can find boundary tasks referenced by `PhaseMarker.AfterTaskID`. Pass `nil` when no filtering is applied. Fixed in T-537.
- `RenderMarkdownWithPhases` accepts an optional `phaseSource *TaskList` parameter (same pattern as `RenderJSONWithPhases`). When the rendered task list has been filtered, pass the original unfiltered list as `phaseSource` so that phase boundary positions are resolved correctly even when boundary tasks are absent. Pass `nil` when no filtering is applied. Fixed in T-698. Filtering can promote a child (ID `2.1`) to the top level, so the renderer looks up positions by the ID's top-level segment (T-1352).
- `RenderJSON` and `RenderJSONWithPhases` use `jsonTask` struct (not raw `Task`) for JSON output. The `toJSONTasks`/`toJSONTask` functions apply `GetEffectiveStream` (so stream is always emitted, never omitted) and `TranslateToHierarchical` (so `blockedBy` contains hierarchical IDs, not stable IDs). Do NOT marshal `Task` structs directly for JSON output — always go through the `jsonTask` transformation. Fixed in T-505.
- `cmd/next.go` has its own `filterIncompleteChildren` which must use `task.HasIncompleteWork` (not a shallow status check). Fixed in T-358 along with `addIncompleteChildrenToData` and `renderTaskMarkdown`.
- `filterTasksRecursive` in `cmd/list.go` (JSON filter path) must NOT include non-matching parents when children match. Non-matching parents are skipped and their matching children are promoted to the parent level. This aligns with `flattenTasksWithFilters` (table filter path). Fixed in T-436.
- `getNextPhaseTasks` in `internal/task/phase.go` must use `hasIncompleteWork(&task)` (not `task.Status != Completed`) when checking whether a phase has pending work. A completed parent with pending children still has incomplete work. The exported `FindNextPhaseTasks` in `next.go` already uses `getAllPendingTasks` which calls `hasIncompleteWork` correctly. Fixed in T-528.
