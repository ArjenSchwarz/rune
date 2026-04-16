# Bugfix Report: ValidateFilePath Symlink Escape

**Date:** 2026-04-16
**Status:** Fixed

## Description of the Issue

`ValidateFilePath` only performed lexical path resolution (`filepath.Abs` + `filepath.Clean`) and checked containment via `strings.HasPrefix`. A symlink inside the working directory pointing to a file outside it would pass validation, allowing commands like `renumber` to modify files outside the project root.

**Reproduction steps:**
1. Create a file outside the working directory
2. Create a symlink inside the working directory pointing to that file
3. Run any rune command (e.g., `rune renumber symlink.md`) targeting the symlink
4. Observe: the command succeeds and modifies the external file

**Impact:** Security — any command using `ValidateFilePath` (all write commands) could be tricked into modifying arbitrary files via symlinks.

## Investigation Summary

- **Symptoms examined:** `ValidateFilePath` uses only lexical path checks
- **Code inspected:** `internal/task/operations.go:ValidateFilePath`, `cmd/integration_renumber_test.go` (known-issue comments at lines 532-537)
- **Hypotheses tested:** Confirmed that `filepath.Abs(filepath.Clean(path))` does not resolve symlinks

## Discovered Root Cause

`ValidateFilePath` resolved paths lexically but never called `filepath.EvalSymlinks`. A symlink like `workdir/escape.md -> /outside/secret.md` has a lexical absolute path inside the working directory, so the `HasPrefix` check passes.

**Defect type:** Missing validation (symlink resolution)

**Why it occurred:** The original implementation only considered lexical path traversal attacks (`../`), not filesystem-level indirection via symlinks.

## Resolution for the Issue

**Changes made:**
- `internal/task/operations.go:ValidateFilePath` — After the lexical containment check, resolve symlinks on both the working directory and the target path using `filepath.EvalSymlinks`, then re-check containment.
- `internal/task/operations.go:resolveExistingPrefix` — New helper that resolves symlinks for the longest existing ancestor path, handling the case where the target file doesn't exist yet.

**Approach rationale:** Two-phase validation (lexical then physical) preserves the fast-path rejection of obvious traversal attempts while catching symlink escapes.

**Alternatives considered:**
- Rejecting all symlinks outright — too restrictive; symlinks within the working directory to other locations within it are legitimate.
- Using `os.Lstat` to detect symlinks before opening — doesn't catch symlinked parent directories.

## Regression Test

**Test file:** `internal/task/fileops_test.go`
**Test name:** `TestValidateFilePath_SymlinkEscape`

**What it verifies:** Symlinks to files and directories outside the working directory are rejected; normal files still pass.

**Run command:** `go test -run TestValidateFilePath_SymlinkEscape -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/operations.go` | Added symlink resolution to `ValidateFilePath`; added `resolveExistingPrefix` helper |
| `internal/task/fileops_test.go` | Added `TestValidateFilePath_SymlinkEscape` regression test |
| `cmd/integration_renumber_test.go` | Changed known-issue warning to a proper assertion |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)

## Prevention

**Recommendations to avoid similar bugs:**
- Any path-based security check must resolve symlinks before containment validation
- Consider using `os.OpenFile` with `O_NOFOLLOW` where symlink following is explicitly undesired

## Related

- Transit ticket T-685
