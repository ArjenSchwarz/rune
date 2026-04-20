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

`sortOperationsForExecution` in `internal/task/batch.go` reorders operations before execution:

1. **Position insertions** (add ops with a `Position` field) are extracted and sorted in reverse position order, then placed first in the result.
2. **Remove operations** within the remaining ops are sorted in reverse ID order (highest first) so that removing higher IDs first preserves the validity of lower IDs. This sorting is restricted to **contiguous blocks** of removes -- non-remove operations act as boundaries that removes cannot cross (fixed in T-200).

The block-based approach matters because users may intentionally interleave removes with adds/updates to achieve specific sequencing (e.g., remove a task, add a replacement, remove another task).

## Validation and Atomicity

`validateOperation` must validate ALL field content upfront (details, references, title, requirements, extended fields) so that `applyOperation` never fails on content validation. This is important because:

1. `ExecuteBatch` uses a test-copy-first approach that protects the original list, but individual apply functions should still be correct
2. `applyUpdateOperation` and `applyOperationWithPhases` must apply status AFTER other field updates to avoid partial mutation if validation fails
3. When adding new validatable fields to `Operation`, update `validateOperation` to include content validation for both add and update cases

The `validateDetailsAndReferences` helper in batch.go centralises detail/reference content validation for use in `validateOperation`.

## Phase Marker Adjustment

Phase-aware batch adds use `addTaskWithPhaseMarkers` in `internal/task/batch.go`. When inserting a top-level task into an earlier phase, the immediate next phase marker must move to the new task and every later marker must be shifted to account for renumbered top-level tasks. T-787 tracks a bug where the batch path only updates the immediate next marker, which can render later phase headers before the wrong task in files with three or more phases.

## Testing Gotcha: Cobra Flag State

Cobra flag values and `Changed` bits persist across `Execute()` calls in the same process. This matters in tests where multiple tests share `rootCmd`. The `resetBatchFlags()` helper in `batch_test.go` resets `batchInput` and the flag's `Changed` bit. Call it at the start of any batch test that does NOT use `--input` to avoid false positives from stale state.

## Known Gap: Phase Detection for Plain Operations

`cmd/batch.go` currently routes to `ExecuteBatchWithPhases` only when an operation has a `phase` field or type `add-phase`. If the target file already has phase markers but the batch contains only plain operations such as `remove`, it uses `ExecuteBatch` and then `WriteFile`, which reuses original phase markers without adjusting them for removed top-level tasks. T-820 tracks this; the command should detect existing phase markers before choosing the execution path.
