# Bugfix Report: Atomic Write Replaces Symlink

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1796

## Description of the Issue

Mutating a task file through a symlink that resolves inside the workspace destroyed the symlink and wrote a new regular file at the link path. The file the link pointed to was left unchanged.

**Reproduction steps:**
1. `rune create target.md --title T`
2. `ln -s target.md link.md`
3. `rune add link.md --title Added`
4. Observe: `link.md` is now a regular file containing the new task; `target.md` still holds only `# T`.

**Impact:** Medium. Any write command run on an in-workspace symlink (common for shared or aliased task files) silently changes filesystem topology and diverges the two paths. No data outside the workspace is touched (that is guarded by `ValidateFilePath`, T-685).

## Investigation Summary

- **Symptoms examined:** After a write, `link.md` lost its symlink mode; `target.md` content was unchanged.
- **Code inspected:** `WriteFileAtomic`, `TaskList.WriteFile`, `WriteFileWithPhases` (`internal/task/operations.go`), `cmd/add_phase.go` (the other `WriteFileAtomic` caller), `ValidateFilePath`/`resolveExistingPrefix`.
- **Hypotheses tested:** The ticket names `WriteFile` and `WriteFileWithPhases`; on current main both now delegate to the shared `WriteFileAtomic`, so the defect lives in one place.

## Discovered Root Cause

`WriteFileAtomic` writes `filePath + ".tmp"` and then calls `os.Rename(tmp, filePath)`. `rename(2)` replaces the directory entry named `filePath`; when that entry is a symlink, the link itself is replaced rather than the file it points to. `os.Stat` (used for the permission lookup) follows the link, so permissions came from the target while the content went to the link path.

**Defect type:** Logic error (path not resolved before rename).

**Why it occurred:** The atomic write was introduced to avoid truncating the target on failed writes; symlinked targets were not considered.

**Contributing factors:** `ValidateFilePath` already accepts in-workspace symlinks (it checks the resolved path), so the CLI hands symlink paths to the writer.

## Resolution for the Issue

**Changes made:**
- `internal/task/operations.go` (`WriteFileAtomic`) - If `filePath` is itself a symlink (`os.Lstat`), resolve it with `filepath.EvalSymlinks` and perform the temp write, chmod and rename against the resolved target. A dangling link returns `resolving symlink ...` before anything is written.

**Approach rationale:** `WriteFileAtomic` is the single writer used by `TaskList.WriteFile`, `WriteFileWithPhases` and `add-phase`, so one change covers every mutating command. The resolved path is the same one `ValidateFilePath` already checked for workspace containment, so this writes only to a validated target. The temp file now sits next to the target, so the rename stays within one directory and remains atomic. `tl.FilePath` keeps the user-supplied link path.

**Alternatives considered:**
- Reject symlinked paths before mutation - the ticket allows it, but it would break a workflow `ValidateFilePath` deliberately supports (in-workspace links) for no safety gain.
- Write in place through the link (`os.WriteFile`) - follows the link but gives up the no-truncation guarantee the atomic write exists for.

**Limitations:** Only the final path component is resolved. Symlinked parent directories were already handled correctly, since rename replaces only the last entry. Pre-existing `.tmp` symlink traversal (T-1773) and outside-workspace validation (T-685) are out of scope.

## Regression Test

**Test file:** `internal/task/operations_test.go`
**Test names:** `TestWriteFileAtomicThroughSymlink`, `TestWriteFileThroughSymlink`

**What it verifies:** Writing through a sibling symlink and through a relative symlink into a subdirectory updates the target, keeps its permissions, keeps the link a symlink, and leaves no temp files. A dangling symlink is rejected without creating its target. `TestWriteFileThroughSymlink` covers the parse → add → `WriteFile` path used by `rune add`.

**Run command:** `go test ./internal/task -run ThroughSymlink`

## Verification

**Automated:**
- [x] Regression tests pass (`go test ./internal/task -run ThroughSymlink`)
- [x] Full test suite passes, except two pre-existing failures that also fail on unmodified main because the container runs as root (`TestCreateBackup/error_handling_for_read_failures`, `TestIntegrationRenumber/renumber_write_failure`)
- [x] `make lint`: 0 issues

**Manual verification:**
- Rebuilt CLI: `rune add link.md --title Added` leaves `link.md -> target.md` in place and adds the task to `target.md`.

## Prevention

- Atomic-replace helpers must resolve the final path component before renaming; the regression tests guard this.

## Related

- T-685 (outside-workspace symlink validation), T-1773 (`.tmp` symlink traversal)
