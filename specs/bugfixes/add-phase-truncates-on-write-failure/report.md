# Bugfix Report: add-phase-truncates-on-write-failure

**Date:** 2026-09-07
**Status:** Fixed
**Ticket:** T-1854

## Description of the Issue

`rune add-phase <file> <name>` wrote the appended content back to the target file with a plain `os.WriteFile`, which opens the file with `O_TRUNC` and truncates it before writing any bytes. If the write failed partway through — disk full, quota exceeded, a file-size limit — the target file was left truncated to whatever fit before the failure, and the original content was permanently lost. Every other mutating command in the project (`add`, `remove`, `update`, `complete`, `renumber`, batch operations, etc.) persists through `TaskList.WriteFile` or `WriteFileWithPhases`, both of which write to a temporary file and rename it into place atomically; `add-phase` never had that protection, since it manipulates the file's raw bytes directly instead of going through a `TaskList`.

**Reproduction steps:**
1. Copy `examples/complex.md` (4,169 bytes) to a working file, e.g. `tasks.md`.
2. Run `( ulimit -f 1; rune add-phase tasks.md "Test Phase" )` — the `ulimit -f 1` file-size limit causes the write to fail partway through with `EFBIG` once it exceeds one block.
3. Observe: the command reports `Error: failed to write file: write tasks.md: file too large` and exits non-zero, but `tasks.md` is now only ~512–1024 bytes (a truncated fragment) instead of its original 4,169 bytes.

**Impact:** Any interruption of the write — a full disk, a hit quota, a file-size limit, or any other error that occurs after the OS has opened-and-truncated the file but before the new content is fully written — destroys the task file's prior content with no way to recover it. This is a silent data-loss bug: the command does report an error, but nothing indicates the original file is gone.

## Investigation Summary

- **Symptoms examined:** `runAddPhase` in `cmd/add_phase.go` read the existing file, computed the new content (`contentStr` with the appended `## {phase}` header), and wrote it back with `os.WriteFile(filename, []byte(contentStr), perm)`.
- **Code inspected:** `cmd/add_phase.go` (`runAddPhase`), and the atomic-write pattern already used by `TaskList.WriteFile` and `WriteFileWithPhases` in `internal/task/operations.go` (write to `filePath + ".tmp"`, then `os.Rename` into place, with cleanup of the temp file on rename failure).
- **Hypotheses tested:** Confirmed via manual reproduction (`ulimit -f 1` around a built `rune` binary) that the pre-fix binary truncates `tasks.md` from 4,169 bytes down to whatever fit under the file-size limit before failing — matching the ticket's triage evidence exactly.
- **Note on adjacent scope:** T-1791 (add-phase modifying malformed task files) and the open tickets about atomic writes following symlinks/overwriting hard links are separate concerns and were left untouched.

## Discovered Root Cause

`runAddPhase` persisted its result with a direct `os.WriteFile` call on the target path instead of the project's established write-to-temp-then-rename pattern. `os.WriteFile` opens its target with `O_WRONLY|O_CREATE|O_TRUNC`, so the file is truncated to zero length as part of a *successful* open, before any content is written; if the subsequent `Write` fails partway (e.g. `EFBIG` from a file-size limit, or `ENOSPC` from a full disk/quota), the truncation has already happened and cannot be undone, leaving a partially-written, corrupted file.

**Defect type:** Missing atomicity / non-transactional file write.

**Why it occurred:** `add-phase` predates (or was implemented independently of) the shared `TaskList`-based write path. Because it only appends a raw text header rather than mutating a parsed `TaskList`, it never had a reason to call `TaskList.WriteFile`, and the direct `os.WriteFile` call was never routed through — or refactored to share — the atomic write helper the rest of the codebase already relies on.

**Contributing factors:** Two other `add-phase` bugs were fixed independently on this same function shortly before this one (T-1473 added `task.ValidateFilePath`, T-1603 replaced ad hoc phase-name validation with `task.NormalizePhaseName`), but neither touched the write step itself, so the truncation-on-failure defect remained.

## Resolution for the Issue

**Changes made:**
- `internal/task/operations.go` — Extracted the temp-file-and-rename logic that `TaskList.WriteFile` and `WriteFileWithPhases` each duplicated into a new exported `WriteFileAtomic(filePath string, content []byte) error`. It resolves the target's existing permissions (or `0644` for a new file), writes the full content to `filePath + ".tmp"`, and renames it into place; the temp file is removed both when the initial write fails and when the rename fails, so a failed call never leaves a stray `.tmp` file or a partially written original behind. `WriteFile` and `WriteFileWithPhases` were refactored to call this helper instead of inlining the same steps.
- `cmd/add_phase.go` (`runAddPhase`) — Replaced the direct `os.WriteFile(filename, []byte(contentStr), perm)` call with `task.WriteFileAtomic(filename, []byte(contentStr))`, so a write failure now leaves `filename` completely untouched instead of truncated. The now-redundant `os.Stat` call used only to capture `perm` was removed, since `WriteFileAtomic` resolves permissions from the target file itself.

**Approach rationale:** The ticket's own triage evidence pointed directly at this: "unlike `WriteFile`/`WriteFileWithPhases`, this command never had atomic-write protection to begin with, so the fix is routing it through the existing pattern used elsewhere." `add-phase` doesn't parse the file into a `TaskList` (it manipulates raw bytes to preserve exact formatting and support malformed/legacy files), so it can't call `TaskList.WriteFile` directly — but the two existing atomic-write call sites already had the exact same 15-line temp-file-and-rename block duplicated between them. Extracting that block into `WriteFileAtomic` gave `add-phase` a route onto the same atomic path without introducing a second, bespoke implementation, and also removed the pre-existing duplication between `WriteFile` and `WriteFileWithPhases`.

**Alternatives considered:**
- Writing the temp file with `os.WriteFile(tmpFile, content, perm)` alone — rejected after review measurement showed it silently drops permission bits. `open(2)` filters the mode through the process umask when it creates a file, so a `0664` target came back `0644` under the common umask `022`. The in-place write this fix replaces never hit that, because `open(2)` ignores the mode argument for a file that already exists. `WriteFileAtomic` therefore calls `os.Chmod` on the temp file before the rename, which also corrects the same latent loss on the two pre-existing callers.
- Inlining a third copy of the temp-file-and-rename block directly in `cmd/add_phase.go` — rejected because it would create a third independent implementation of the same logic (the exact "second bespoke atomic writer" the ticket warned against), instead of a single shared one all three call sites share.
- Parsing the file into a `TaskList` in `add-phase` and writing it back through `TaskList.WriteFile` — rejected because `add-phase` intentionally works on raw bytes so it can add a phase header to files that wouldn't parse cleanly as a `TaskList` (this is the malformed-file behaviour tracked separately by T-1791); routing through `TaskList.WriteFile` would require parsing first and change that behaviour, which is out of scope for this bug.

## Regression Test

**Test file:** `cmd/add_phase_test.go`
**Test name:** `TestAddPhaseCommandWriteFailurePreservesFile`

**Test file:** `cmd/integration_add_phase_test.go`
**Test name:** `TestIntegrationAddPhase` (subtest `add_phase_write_failure_preserves_file`)

**What they verify:**
- `TestAddPhaseCommandWriteFailurePreservesFile` is a fast, deterministic unit test (part of `make test`) that forces the write step to fail by pre-creating `tasks.md.tmp` as a directory, so `WriteFileAtomic`'s `os.WriteFile` to that path fails with "is a directory" before ever touching `tasks.md`. It asserts `runAddPhase` returns an error and that `tasks.md` is byte-for-byte unchanged afterward. Against the pre-fix code this setup has no effect (the old code never creates a `.tmp` file at all), so the write silently succeeds and the test fails with "expected add-phase to fail ... got nil".
- `TestIntegrationAddPhase`/`add_phase_write_failure_preserves_file` (requires `INTEGRATION=1`) reproduces the ticket's exact scenario: it copies `examples/complex.md` (4,169 bytes) into a working file and runs the built `rune` binary as a subprocess under `sh -c 'ulimit -f 1 && ...'`, so the write fails partway through with a real `EFBIG`/"file too large" error. It asserts the command fails and that the file on disk is still byte-for-byte identical to the original. Against the pre-fix code, this test fails because the file is truncated (e.g. to 1,024 bytes instead of 4,169).

**Run commands:**
- `go test -run TestAddPhaseCommandWriteFailurePreservesFile -v ./cmd`
- `INTEGRATION=1 go test -run TestIntegrationAddPhase -v ./cmd`

Both tests were confirmed to fail against the pre-fix code (by temporarily reverting `cmd/add_phase.go` and `internal/task/operations.go` via `git stash`) and to pass after the fix was restored.

## Affected Files

| File | Change |
|------|--------|
| `internal/task/operations.go` | Added exported `WriteFileAtomic(filePath string, content []byte) error`; refactored `TaskList.WriteFile` and `WriteFileWithPhases` to call it instead of duplicating the temp-file-and-rename logic; the temp file is now also cleaned up when the initial write (not just the rename) fails |
| `cmd/add_phase.go` | `runAddPhase` now writes via `task.WriteFileAtomic` instead of a direct `os.WriteFile`; removed the now-redundant `os.Stat` call used only to capture file permissions; updated a stale comment referencing the old `os.WriteFile` write path |
| `internal/task/operations_test.go` | Added `TestWriteFileAtomic`, a table test covering new-file creation, replacement with permission preservation, write-to-temp failure leaving the original byte-for-byte intact, rename failure, and preservation of a group-writable mode the umask would otherwise strip |
| `cmd/add_phase_test.go` | Added `TestAddPhaseCommandWriteFailurePreservesFile` regression test |
| `cmd/integration_add_phase_test.go` | New file; added `TestIntegrationAddPhase` with the `add_phase_write_failure_preserves_file` subtest reproducing the ticket's `ulimit -f 1` scenario |
| `CHANGELOG.md` | Added `[Unreleased] / Fixed` entry for the atomic-write fix |

## Verification

**Automated:**
- [x] Regression tests pass (`TestAddPhaseCommandWriteFailurePreservesFile`, `TestIntegrationAddPhase`)
- [x] Full test suite passes (`make check`)
- [x] Full integration suite passes (`INTEGRATION=1 go test -run TestIntegration ./cmd`)
- [x] Linters/validators pass (`make lint` via `make check`, 0 issues)

**Manual verification:**
- Ran the exact repro from the ticket against the pre-fix binary: copied `examples/complex.md` to `tasks.md`, ran `( ulimit -f 1; rune add-phase tasks.md "Test Phase" )` — confirmed the file truncated from 4,169 bytes to 512/1,024 bytes.
- Re-ran the same repro against the fixed binary: the command still fails with `write tasks.md.tmp: file too large`, but `tasks.md` is verified byte-for-byte unchanged from the original, and no `tasks.md.tmp` file is left behind.

## Prevention

**Recommendations to avoid similar bugs:**
- Any command that persists a task file directly (rather than through `TaskList.WriteFile`) should route through `task.WriteFileAtomic` rather than calling `os.WriteFile` on the target path.
- When adding a new file-mutating command, prefer extending/reusing the existing atomic-write helper over writing a new one; if two implementations of the same "write to temp, then rename" logic exist, that's a signal the shared helper should be extracted (as was done here for `WriteFile`/`WriteFileWithPhases`).

## Related

- Ticket: T-1854
- Related prior fixes to `add-phase` on the same function: T-1473 (path containment via `task.ValidateFilePath`), T-1603 (phase-name validation via `task.NormalizePhaseName`)
- Explicitly out of scope: T-1791 (add-phase modifying malformed task files), and open tickets about atomic writes following symlinks / overwriting hard links
