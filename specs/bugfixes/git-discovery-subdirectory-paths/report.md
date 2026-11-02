# Bugfix Report: Git Discovery Subdirectory Paths

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-959

## Description of the Issue

Branch-based discovery succeeded from a repository subdirectory, but the command then failed to open the discovered file.

**Reproduction steps:**
1. Create a git repo on branch `feature/my-feature` with `specs/my-feature/tasks.md` and a `.rune.yml` enabling discovery; commit.
2. `cd src/pkg` (any subdirectory).
3. Run `rune next --format json --verbose` or `rune complete 1 --verbose`.

Observed:

```
Using task file: specs/my-feature/tasks.md
Error: failed to read task file: reading file: open specs/my-feature/tasks.md: no such file or directory
```

(`complete` reported `file specs/my-feature/tasks.md does not exist`.)

**Impact:** Every command that relies on discovery (next, list, find, add, complete, update, remove, …) was unusable outside the repo root.

## Investigation Summary

- **Symptoms examined:** Discovery reported the right file but the subsequent open failed.
- **Code inspected:** `internal/config/discovery.go:DiscoverFileFromBranch`, `cmd/filename.go:resolveFilename`, `internal/task/operations.go:ValidateFilePath`.
- **Hypotheses tested:** Returning an absolute path (as the ticket suggests) makes reads work, but every write goes through `ValidateFilePath`, which rejects paths outside the current working directory. From `src/pkg`, the repo-root task file is outside the CWD, so `complete` and other mutating commands would still fail, now with `path traversal attempt detected`.

## Discovered Root Cause

`DiscoverFileFromBranch` checks existence with `filepath.Join(repoRoot, path)` but returns the repo-relative `path`. `resolveFilename` passed it straight to the command, which resolved it against the CWD.

**Defect type:** Path base mismatch.

**Why it occurred:** The earlier subdirectory fix (`auto-detect-tasks-file-subdirectories`) made the existence check repo-root aware but left callers interpreting the result relative to the CWD. Its test only exercised `DiscoverFileFromBranch`, not a command.

## Resolution for the Issue

**Changes made:**
- `internal/config/discovery.go` - Added exported `RepoRoot()`; documented that discovered paths are repo-relative.
- `cmd/filename.go` - After a successful discovery from a subdirectory, `resolveFilename` (via `discoveredPath`) changes the working directory to the repo root and returns the absolute path. From the repo root it returns the repo-relative path unchanged.

**Approach rationale:** Running the command from the repo root lets writes pass `ValidateFilePath` without changing it. Returning the absolute path means the filename commands report (`file` in JSON, "Using task file", dry-run text) can be passed back to rune from the caller's directory. A first version returned the repo-relative path after the chdir; review on PR #115 found that the reported path then could not be reused from the subdirectory. Explicit file arguments are untouched and still resolve relative to the invoking directory.

**Alternatives considered:**
- Return an absolute path without changing directory - fixes reads only; writes fail the CWD containment check.
- Return a caller-relative path (`../specs/...`) - unusable internally after the chdir, and would need two paths threaded through every command.
- Relax `ValidateFilePath` to allow anything under the repo root - a security-boundary change for all paths, out of scope.

## Regression Test

**Test files:** `cmd/filename_test.go`, `cmd/integration_discovery_subdir_test.go`
**Test names:** `TestResolveFilenameFromSubdirectory`, `TestCompleteFromSubdirectoryWithDiscovery`, `TestIntegrationDiscoveryFromSubdirectory`

**What it verifies:** From a real git repo subdirectory, the path from `resolveFilename` can be parsed and written, and `runComplete` marks the task complete in the repo-root file. The integration test runs the built binary from `src/pkg`: `add-phase --format json` writes the repo-root file and its reported `file` is accepted by `list` from the same directory; `--dry-run` and `complete --verbose` report the same path; explicit `../../specs/...` reads still work and explicit writes outside the CWD are still rejected; from the repo root the reported path stays repo-relative.

**Run command:** `go test ./cmd -run 'TestResolveFilenameFromSubdirectory|TestCompleteFromSubdirectoryWithDiscovery' && INTEGRATION=1 go test ./cmd -run TestIntegrationDiscoveryFromSubdirectory`

## Affected Files

| File | Change |
|------|--------|
| `cmd/filename.go` | Chdir to repo root after discovery |
| `internal/config/discovery.go` | `RepoRoot()` helper, doc comment |
| `cmd/filename_test.go` | Regression tests |
| `cmd/integration_discovery_subdir_test.go` | CLI round trip from a subdirectory |

## Verification

**Automated:**
- [x] Regression tests pass (failed before the fix)
- [x] Full test suite passes, except two pre-existing permission-based tests that also fail on main when run as root
- [x] `make lint`: 0 issues

**Manual verification:**
- Ticket reproduction: `next --format json --verbose` and `complete 1 --verbose` from `src/pkg` now succeed and update `specs/my-feature/tasks.md`.
- Review scenario: from `src/` on `feature/demo`, `add-phase Planning --format json` reports an absolute `file`, and `list` with the path reported by `add-phase --dry-run` succeeds from `src/`.

## Prevention

- Test path-resolution fixes through a command entry point, not only the helper, and check that any path a command reports can be reused from the caller's directory.

## Related

- `specs/bugfixes/auto-detect-tasks-file-subdirectories/report.md`
