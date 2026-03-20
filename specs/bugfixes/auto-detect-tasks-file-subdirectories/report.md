# Bugfix Report: Auto-detect tasks file fails from subdirectories

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

When `rune` is run from a subdirectory of a git repository, auto-detection of the tasks file fails even though the file exists at the repo root. The same issue affects `.rune.yml` config file loading.

**Reproduction steps:**
1. Create a git repo with `specs/my-feature/tasks.md` and optionally `.rune.yml` at the root
2. `cd` into a subdirectory (e.g., `src/pkg/`)
3. Run any `rune` command that relies on auto-detection (no explicit filename argument)
4. Observe: "task file not found" error despite the file existing at repo root

**Impact:** Any user or agent running `rune` from a subdirectory of a repo cannot use auto-detection, which is the primary intended workflow.

## Investigation Summary

- **Symptoms examined:** `DiscoverFileFromBranch` returns "task file not found" when CWD is not repo root
- **Code inspected:** `internal/config/discovery.go` (fileExists, DiscoverFileFromBranch), `internal/config/config.go` (loadConfigUncached)
- **Hypotheses tested:** Confirmed that `os.Stat` on relative paths resolves against CWD, not repo root

## Discovered Root Cause

Two functions use relative paths that resolve against the current working directory instead of the git repository root:

1. `fileExists` in `discovery.go` calls `os.Stat(path)` where `path` is a relative path like `specs/my-feature/tasks.md`. When CWD is a subdirectory, this resolves to `<subdir>/specs/my-feature/tasks.md` instead of `<repo-root>/specs/my-feature/tasks.md`.

2. `loadConfigUncached` in `config.go` checks `./.rune.yml` as a relative path, which similarly fails from subdirectories.

**Defect type:** Path resolution error — relative paths assumed CWD equals repo root.

**Why it occurred:** The original implementation assumed `rune` would always be run from the repo root directory.

**Contributing factors:** `git rev-parse --abbrev-ref HEAD` works from any subdirectory (git traverses up), masking the fact that the subsequent file checks do not.

## Resolution for the Issue

**Changes made:**
- `internal/config/discovery.go` - Added `getRepoRoot` / `getRepoRootImpl` function that calls `git rev-parse --show-toplevel` to determine the repo root. Made it a package-level function variable (like `getCurrentBranch`) for testability. Modified `DiscoverFileFromBranch` to resolve candidate paths against the repo root using `filepath.Join(repoRoot, path)`.
- `internal/config/config.go` - Modified `loadConfigUncached` to prepend the repo-root-relative `.rune.yml` path to the search list, so it is checked before the CWD-relative path. The CWD-relative path is kept as a fallback for non-git usage.
- `internal/config/discovery_test.go` - Updated existing `TestDiscoverFileFromBranch` tests to mock `getRepoRoot` (returning the temp dir), so they continue to work in non-git temp directories. Added new `TestDiscoverFileFromBranchSubdirectory` regression test.
- `internal/config/config_test.go` - Added `TestLoadConfigFromSubdirectory` regression test.

**Approach rationale:** Using `git rev-parse --show-toplevel` is the standard way to find the repo root from any subdirectory. The function variable pattern is already established in the codebase (`getCurrentBranch`) and provides clean testability.

**Alternatives considered:**
- Walking up the directory tree looking for `.git` — more complex, duplicates what git already does, and would not handle git worktrees correctly.
- Changing CWD to the repo root at startup — too invasive, would affect all path handling globally.

## Regression Test

**Test file:** `internal/config/discovery_test.go`
**Test name:** `TestDiscoverFileFromBranchSubdirectory`

**What it verifies:** That `DiscoverFileFromBranch` finds the task file when CWD is a subdirectory of the git repo root.

**Test file:** `internal/config/config_test.go`
**Test name:** `TestLoadConfigFromSubdirectory`

**What it verifies:** That `loadConfigUncached` finds `.rune.yml` at the repo root when CWD is a subdirectory.

**Run command:** `go test -run "TestDiscoverFileFromBranchSubdirectory|TestLoadConfigFromSubdirectory" -v ./internal/config/`

## Affected Files

| File | Change |
|------|--------|
| `internal/config/discovery.go` | Add `getRepoRoot` function; resolve candidate paths against repo root |
| `internal/config/config.go` | Prepend repo-root `.rune.yml` path to search list |
| `internal/config/discovery_test.go` | Add subdirectory regression test; mock `getRepoRoot` in existing tests |
| `internal/config/config_test.go` | Add subdirectory config loading regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When resolving relative paths in a git-aware tool, always resolve against the repo root, not CWD
- Consider adding a `getRepoRoot()` helper function to centralize repo root resolution

## Related

- T-482: Auto-detect tasks file fails when running from subdirectories
