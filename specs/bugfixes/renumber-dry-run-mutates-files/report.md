# Bugfix Report: renumber-dry-run-mutates-files

**Date:** 2026-09-07
**Status:** Fixed
**Ticket:** T-1345

## Description of the Issue

`rune renumber <file> --dry-run` created a `.bak` backup file and wrote the renumbered content back to the task file, even though `--dry-run` is documented to preview changes without applying them.

**Reproduction steps:**
1. Create a task file with a gap in numbering: `- [ ] 1. First` / `- [ ] 3. Third`
2. Run: `rune renumber tasks.md --dry-run --format json`
3. Observe the command exits 0, reports `success: true` with a `backup_file`, and the file on disk now has `2. Third` instead of `3. Third`

**Impact:** Anyone (human or agent) relying on `--dry-run` to safely preview a renumber before committing to it got a silent mutation instead — including creation of a stray `.bak` file. This breaks the safety contract the global `--dry-run` flag has in every other command (`add`, `remove`, `update`, `complete`, etc.).

## Investigation Summary

- **Symptoms examined:** File content and mtime change during a `--dry-run` renumber call; `.bak` file appears; JSON output reports success as if a real write happened.
- **Code inspected:** `cmd/renumber.go` (`runRenumber`), and the `dryRun` handling in `cmd/add.go`, `cmd/remove.go`, `cmd/update.go`, `cmd/complete.go` for comparison.
- **Hypotheses tested:** Confirmed via `grep -n "dryRun" cmd/renumber.go` that the global `dryRun` flag (declared in `cmd/root.go`) is never referenced in `renumber.go` at all — every other command that mutates files checks it before writing.

## Discovered Root Cause

`runRenumber` never checked the global `dryRun` flag before calling `createBackup` and writing the renumbered file back to disk. Every other mutating command (`add`, `remove`, `update`, `complete`, `progress`, `batch`, etc.) has an explicit `if dryRun { ... return nil }` short-circuit before its write path; `renumber.go` was missing this entirely, so the backup-and-write phases (Phase 4 and Phase 6 in the function's own comments) always executed regardless of the flag.

**Defect type:** Missing conditional / incomplete flag propagation

**Why it occurred:** The `renumber` command was likely implemented without following the `dryRun` convention already established by the other file-mutating commands, and no test exercised `--dry-run` for this command, so the gap went unnoticed.

**Contributing factors:** No existing test in `cmd/renumber_test.go` invoked `runRenumber` with `dryRun = true`, unlike `add_test.go`, `remove_test.go`, `update_test.go`, and `complete_test.go`, which all have dedicated dry-run coverage.

## Resolution for the Issue

**Changes made:**
- `cmd/renumber.go:77-141` (`runRenumber`) — Added a dry-run short-circuit immediately after resource-limit validation (Phase 3) and before backup creation (Phase 4): `if dryRun { return displayDryRunSummary(taskList, format) }`.
- `cmd/renumber.go` — Added `displayDryRunSummary`, a format-aware (table/markdown/json) preview function that reports the task count and a "Dry run - no changes made" status without a backup file, mirroring the JSON `DryRun`-field pattern already used by `CompleteResponse`.
- `cmd/renumber.go` — Added `DryRun bool` (`json:"dry_run,omitempty"`) to `RenumberResponse` so JSON consumers can distinguish a dry-run preview from a real write.

**Approach rationale:** This follows the exact convention already used by `add.go`, `remove.go`, `update.go`, and `complete.go` — check `dryRun` before any mutation and return early with a preview. `renumber` already had full table/markdown/json output support via `displaySummary`, so the preview was made format-aware (like `complete.go`'s dry-run path) rather than falling back to plain-text-only output (like `add.go`/`remove.go`), since `--format json` is explicitly part of the bug report's repro and is relied on by the JSON API consumers this tool targets.

**Alternatives considered:**
- Reusing `displaySummary` with an added `isDryRun bool` parameter — rejected because it would require updating three existing table/markdown/JSON tests' call sites (`TestDisplaySummaryTable`, `TestDisplaySummaryMarkdown`, `TestDisplaySummaryJSON`) for no functional benefit; a small dedicated `displayDryRunSummary` function keeps the change minimal and isolated.
- Plain-text-only dry-run output (matching `add.go`/`remove.go`) — rejected because `renumber` already fully supports `--format json`/`markdown`, and the ticket's own repro used `--format json`, so silently ignoring `--format` in dry-run mode would reintroduce the same kind of inconsistency previously fixed for `complete`/`uncomplete` (see `specs/bugfixes/complete-dry-run-json/report.md`).

## Regression Test

**Test file:** `cmd/renumber_test.go`
**Test names:** `TestRenumberDryRunDoesNotModifyFile`, `TestRenumberDryRunJSON`, `TestRenumberDryRunMarkdown`

**What they verify:**
- `TestRenumberDryRunDoesNotModifyFile` reproduces the exact ticket scenario (IDs `1`/`3`) with table format, asserts the file content is byte-for-byte unchanged after `--dry-run`, and asserts no `.bak` file is created.
- `TestRenumberDryRunJSON` reproduces the ticket's `--format json` repro, asserts the JSON response has `success: true`, `dry_run: true` and an empty `backup_file`, and asserts the file and absence of a `.bak` file, same as above.
- `TestRenumberDryRunMarkdown` covers the `--format markdown` branch of the dry-run path, asserting the preview reports the dry-run status with no backup file line, and that the task file is untouched.

**Run command:** `go test ./cmd -run 'TestRenumberDryRun' -v`

Both tests were confirmed to fail against the pre-fix code (file mutated, `.bak` created, `backup_file` populated) and pass after the fix.

## Affected Files

| File | Change |
|------|--------|
| `cmd/renumber.go` | Added `dryRun` short-circuit before backup/write; added `displayDryRunSummary`; added `DryRun` field to `RenumberResponse` |
| `cmd/renumber_test.go` | Added `TestRenumberDryRunDoesNotModifyFile` and `TestRenumberDryRunJSON` regression tests |

## Verification

**Automated:**
- [x] Regression tests pass (`TestRenumberDryRunDoesNotModifyFile`, `TestRenumberDryRunJSON`)
- [x] Full test suite passes (`make test` via `make check`)
- [x] Linters/validators pass (`make lint` via `make check`, 0 issues)

**Manual verification:**
- Ran the exact repro from the ticket (`go run . renumber "$file" --dry-run --format json`): exits 0, JSON reports `"success": true, "backup_file": "", "dry_run": true`, file content unchanged (`3. Third` preserved), no `.bak` file created.

## Prevention

**Recommendations to avoid similar bugs:**
- Any new command that mutates a task file should have an explicit dry-run test from the start, mirroring `add_test.go`/`remove_test.go`/`update_test.go`/`complete_test.go`.
- Consider a lightweight shared helper or lint rule that every `RunE` touching `dryRun`-eligible mutations at least references the `dryRun` package variable, to catch this class of omission earlier.

## Related

- Ticket: T-1345
- Related prior fix for the same class of bug in dry-run/format handling: `specs/bugfixes/complete-dry-run-json/report.md` (T-725), `specs/bugfixes/progress-dry-run-json-format/report.md`
