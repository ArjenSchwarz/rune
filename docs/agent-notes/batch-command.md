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

## Testing Gotcha: Cobra Flag State

Cobra flag values and `Changed` bits persist across `Execute()` calls in the same process. This matters in tests where multiple tests share `rootCmd`. The `resetBatchFlags()` helper in `batch_test.go` resets `batchInput` and the flag's `Changed` bit. Call it at the start of any batch test that does NOT use `--input` to avoid false positives from stale state.
