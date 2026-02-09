# Batch Command

## Positional File Argument

The batch command accepts a positional file argument that serves dual purposes depending on context:

1. **Without `--input`**: positional arg is the JSON operations source file (e.g., `operations.json`)
2. **With `--input`**: positional arg is the target task file (e.g., `tasks.md`)

When used as target file, it's reconciled with the JSON blob's `file` field:
- If JSON has no `file` field: positional arg fills it in
- If JSON `file` matches positional arg: no conflict
- If they differ: error

## Testing Gotcha: Cobra Flag State

Cobra flag values and `Changed` bits persist across `Execute()` calls in the same process. This matters in tests where multiple tests share `rootCmd`. The `resetBatchFlags()` helper in `batch_test.go` resets `batchInput` and the flag's `Changed` bit. Call it at the start of any batch test that does NOT use `--input` to avoid false positives from stale state.
