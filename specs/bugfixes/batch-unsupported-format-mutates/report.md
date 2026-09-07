# Bugfix Report: Batch Unsupported Format Mutates File

**Date:** 2026-09-07
**Status:** Fixed

## Description of the Issue

`rune batch` validated the `--format` value only when it came time to render the response, which is after the batch operations had already been applied to the task list and written to disk. Passing an unsupported format such as `--format yaml` therefore mutated the target file and *then* returned `unsupported output format: yaml` and exited non-zero.

For a command whose headline guarantee is "all operations succeed or none are applied", this is a surprising failure mode: the caller sees an error and a non-zero exit code, reasonably concludes nothing happened, and yet the file on disk has changed. An agent or script retrying the batch after correcting the format would apply the same operations a second time.

**Reproduction steps:**
1. Create a task file, e.g. `test_tasks.md` containing `- [ ] 1. First task`
2. Run a batch that adds a task with an unsupported output format:
   `rune batch --input '{"file":"test_tasks.md","operations":[{"type":"add","title":"Applied despite error"}]}' --format yaml`
3. Observe the command prints `unsupported output format: yaml` and exits 1
4. Inspect `test_tasks.md` — the new task has been added anyway

**Impact:** Silent unintended mutation on a command that reported failure. Worst case is duplicated operations when a caller retries after fixing the format flag.

## Investigation Summary

- **Symptoms examined:** Non-zero exit from `rune batch` with an unsupported `--format`, yet the target file contains the applied operations
- **Code inspected:** `cmd/batch.go` (`runBatch`, `outputBatchJSON`, `outputBatchText`), `internal/task/batch.go` (`ExecuteBatch`, `ExecuteBatchWithPhases`), `cmd/find.go` (format constants)
- **Hypotheses tested:** Confirmed the only `--format` validation in `runBatch` lived in the trailing output-dispatch `switch`, and that both `ExecuteBatchWithPhases` (which saves internally) and the `ExecuteBatch` path (which calls `taskList.WriteFile`) run to completion before that switch is reached

## Discovered Root Cause

`runBatch` performed format validation as a side effect of output dispatch. The function's ordering was:

1. Read and parse the JSON request
2. Parse the task file
3. Execute the operations and write the updated file to disk
4. `switch strings.ToLower(format)` — with a `default` branch returning `unsupported output format`

Step 4 was the *only* place an unsupported format was detected, so by the time the error was produced the write in step 3 had already happened.

**Defect type:** Incorrect validation ordering — input validation performed late, as a by-product of dispatch, rather than up front.

**Why it occurred:** The format check was never written as a validation step; it was the fallthrough case of the rendering switch. That reads as adequate when the switch is the only consumer of `format`, but it silently couples "is this format valid?" to "we have already done the work and now need to print it".

**Contributing factors:** The batch command's atomicity guarantee is implemented inside `internal/task` at the operation level, so it offers no protection against the CLI layer failing *after* a successful save.

## Resolution for the Issue

**Changes made:**
- `cmd/batch.go` — Added an explicit format whitelist check at the very top of `runBatch`, before any input is read, JSON is parsed, or the task file is touched. Unsupported values return `unsupported output format: %s` immediately
- `cmd/batch.go` — Bound the lowercased value once as `outputFormat` and reused it for the output dispatch, so `strings.ToLower(format)` is not recomputed and the supported set is enumerated in exactly one place
- `cmd/batch.go` — Reduced the output dispatch to `case formatJSON` / `default`, since the previous `default` error branch became unreachable dead code once validation moved to the top
- `cmd/find.go` — Added a shared `formatTable = "table"` constant alongside the existing `formatJSON` and `formatMarkdown`
- `cmd/renumber.go` — Switched a bare `"table"` literal to the new `formatTable` constant

**Approach rationale:** Validating cheap, self-contained input (a flag value) before doing any work is the smallest correct fix and needs no rollback machinery. Because the check now precedes even reading the batch input, the format error is reported deterministically regardless of whether the request JSON or the target file is also invalid — the failure mode is a pure no-op.

**Alternatives considered:**
- **Validate just before the write instead of at the top of the function** — Rejected: it still lets the command do avoidable work, and leaves the ordering fragile against future code that saves earlier
- **Roll back the file if rendering fails** — Rejected: significantly more machinery (snapshot and restore, plus its own failure modes) to undo work that never needed to start
- **Register `--format` as a cobra enum-validated flag** — Rejected for this fix: `format` is a root-level persistent flag shared by every command, and the supported set differs per command, so a global constraint would be wrong. Worth revisiting separately

**Linter note:** Adding a third `"table"` literal in `cmd/` tripped `goconst`, which is why the shared `formatTable` constant was introduced in the same change.

## Regression Test

**Test file:** `cmd/batch_test.go`
**Test names:**
- `TestBatchCommand_UnsupportedFormatDoesNotMutateFile` — runs a batch add against a real task file with `--format yaml`, asserts the command errors with `unsupported output format`, and asserts the file contents are byte-for-byte unchanged
- `TestBatchCommand_UnsupportedFormatRejectedBeforeFileAccess` — points the batch request at a nonexistent file with `--format yaml` and asserts the format error still wins over any file error, and that the missing file is not created

**What it verifies:** The first test locks in the actual bug (no mutation). The second locks in the stronger ordering property the fix relies on — validation strictly precedes file access, not merely the write — so a future refactor that merely moves the check earlier than the save, but later than the read, would still be caught.

**Run command:** `go test -run "TestBatchCommand_UnsupportedFormat" -v ./cmd/`

## Affected Files

| File | Change |
|------|--------|
| `cmd/batch.go` | Added the up-front format whitelist in `runBatch`; bound `outputFormat` once; reduced the output dispatch to `formatJSON` plus `default` after the old error branch became unreachable |
| `cmd/find.go` | Added the shared `formatTable` constant |
| `cmd/renumber.go` | Replaced a `"table"` literal with `formatTable` |
| `cmd/batch_test.go` | Added two regression tests covering non-mutation and validation ordering, with output capture via `rootCmd.SetOut`/`SetErr` |
| `CHANGELOG.md` | Added an `[Unreleased]` / `Fixed` entry for the batch format validation fix |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed `TestBatchCommand_UnsupportedFormatDoesNotMutateFile` fails before the fix (file mutated despite the error) and passes after
- Confirmed the format error is returned for a nonexistent target file, proving the check runs before file access
- Ran `make check` (format, lint, test) with no issues

## Prevention

**Recommendations to avoid similar bugs:**
- Validate flag and argument values at the top of a command's `RunE`, before any I/O or state mutation. Never let validation be a side effect of a dispatch `switch` at the end of a function
- When a `default:` branch of a dispatch switch is the only thing rejecting bad input, treat that as a smell: the rejection happens after all the work
- Where a command both mutates state and renders output, make sure every error path that can fire *after* the save is genuinely impossible, not merely unlikely
- Consider a shared helper for per-command output-format whitelists, so other commands that mutate before rendering get the same up-front check

## Related

- Transit ticket: T-1787
- Pull request: #93
