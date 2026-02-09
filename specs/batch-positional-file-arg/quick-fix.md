# Batch Command: Positional Target File Argument

## Problem

AI agents consistently misuse the batch command by passing the target task file as a positional argument alongside `--input`:

```sh
rune batch tasks.md --input '{"operations":[{"type":"add","title":"New task"}]}'
```

The batch command treated the positional argument exclusively as the JSON operations source file. When `--input` was also provided, the positional argument was silently ignored, and the command failed with `file field is required` because the JSON blob omitted the `file` field.

This is a natural mistake. Every other rune command (`add`, `list`, `complete`, etc.) accepts the target task file as a positional argument. Agents reasonably expected the same from `batch`.

## Solution

The positional argument now has context-dependent meaning:

| Input source | Positional arg meaning |
|---|---|
| Neither `--input` nor stdin | JSON operations source file (unchanged) |
| `--input` flag | Target task file |
| stdin | Not applicable (max 1 arg) |

When the positional arg is used as the target file, it is reconciled with the JSON blob's `file` field:

- **JSON has no `file` field**: positional arg fills it in
- **JSON `file` matches positional arg**: accepted without conflict
- **JSON `file` differs from positional arg**: error with a clear message

## Changes

### `cmd/batch.go`

- When `batchInput` is non-empty and a positional arg is provided, store it as `positionalFile`
- After JSON parsing, merge `positionalFile` into `req.File` with conflict detection
- Updated long description and examples to document the new behaviour

### `cmd/batch_test.go`

- Added `resetBatchFlags()` helper to reset Cobra flag state between tests (flag values and `Changed` bits persist across `Execute()` calls in the same process)
- Added `TestBatchCommand_PositionalFileArg` with three cases:
  - Positional arg fills missing `file` field
  - Positional arg matches existing `file` field
  - Positional arg conflicts with `file` field (expects error)
- Applied `resetBatchFlags()` to `TestBatchCommand_FileInput` and `TestBatchCommand_StdinInput` to fix pre-existing test isolation issue

### `docs/agent-notes/batch-command.md`

- Documented the dual-purpose positional arg and the Cobra flag state testing gotcha
