# Batch Command

## Positional File Argument

The batch command accepts a positional file argument that serves dual purposes depending on context:

1. **Without `--input`**: positional arg is the JSON operations source file (e.g., `operations.json`)
2. **With `--input`**: positional arg is the target task file (e.g., `tasks.md`)

When used as target file, it's reconciled with the JSON blob's `file` field:
- If JSON has no `file` field: positional arg fills it in
- If JSON `file` matches positional arg: no conflict
- If they differ: error

## Stdin Support

The `--input` flag accepts `-` as a stdin marker. When `--input -` is passed, the batch command reads JSON from stdin instead of treating the value as literal JSON. This allows combining piped input with a positional file argument: `echo '...' | rune batch tasks.md --input -`.

## Operation Sorting (sortOperationsForExecution)

`sortOperationsForExecution` in `internal/task/batch.go` only reorders operations within contiguous runs of the same kind:

1. **Positioned adds** (add ops with a `Position`) are sorted in reverse position order within a run. Equal positions run in reverse request order so they land in request order (T-1933).
2. **Removes** are sorted in reverse ID order within a run, so removing higher IDs first keeps lower IDs valid.

Non-matching operations are boundaries that nothing crosses. Positioned adds used to be hoisted ahead of everything (T-1874), which retargeted IDs in earlier updates/removes; removes used to be sorted globally (T-200). Don't reintroduce cross-kind reordering.

It returns `indexedOperation`s carrying each op's request index; `operation N` in error messages uses that index, not the execution position (T-1780).

## Auto-Completed Reporting

`runBatch` (not the apply functions) auto-completes parents after each completing update and flags them with the unexported `Task.autoCompleted` field, so the flag follows the task through renumbering. `collectAutoCompleted` reports flagged tasks that are still complete with complete descendants and clears every flag before the result is adopted. An explicit status update clears the target's own flag (T-1776).

## Validation and Atomicity

`validateOperation` must validate ALL field content upfront (details, references, title, requirements, extended fields) so that `applyOperation` never fails on content validation. This is important because:

1. `ExecuteBatch` and `ExecuteBatchWithPhases` share `runBatch`, which validates and applies every operation once to a deep copy (`deepCopy`/`cloneTasks`, a plain struct copy — not a render/parse round trip, which was lossy and slow). On success the copy and its phase markers replace the original (`*tl = *result`); on failure the original is untouched. Operations are never replayed against the original, so individual apply functions should still be correct but cannot partially mutate the caller's list
2. `applyUpdateOperation` and `applyOperationWithPhases` must apply status AFTER other field updates to avoid partial mutation if validation fails
3. When adding new validatable fields to `Operation`, update `validateOperation` to include content validation for both add and update cases

The `validateDetailsAndReferences` helper in batch.go centralises detail/reference content validation for use in `validateOperation`.

Cycle detection for `blocked_by` updates happens only when the update is applied (`UpdateTaskWithOptions`), not again in `validateOperation`; the error is reported the same way.

## Phase Marker Adjustment

All batch adds (with or without a phase) go through `TaskList.AddTaskWithPhases` in `internal/task/batch.go`, shared with the `add` command; see `streams-and-phases.md`. Phase adds update the immediate next marker and shift every later one, and positional root inserts now shift the markers too (they previously left them pointing at the pre-insertion numbering).

## Testing Gotcha: Cobra Flag State

Cobra flag values and `Changed` bits persist across `Execute()` calls in the same process. This matters in tests where multiple tests share `rootCmd`. The `resetBatchFlags()` helper in `batch_test.go` resets `batchInput` and the flag's `Changed` bit. Call it at the start of any batch test that does NOT use `--input` to avoid false positives from stale state.

## Known Gap: Phase Detection for Plain Operations

`cmd/batch.go` currently routes to `ExecuteBatchWithPhases` only when an operation has a `phase` field or type `add-phase`. If the target file already has phase markers but the batch contains only plain operations such as `remove`, it uses `ExecuteBatch` and then `WriteFile`, which reuses original phase markers without adjusting them for removed top-level tasks. T-820 tracks this; the command should detect existing phase markers before choosing the execution path.
