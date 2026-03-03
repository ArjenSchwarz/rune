# Bugfix Report: Git Branch Discovery Timeout Not Enforced

**Date:** 2026-03-04
**Status:** Fixed

## Description of the Issue

The `getCurrentBranchImpl` function in `internal/config/discovery.go` created a `context.WithTimeout` but never attached it to the git command execution. It used `exec.Command` instead of `exec.CommandContext`, so a hung or slow git process could block indefinitely despite the intended 5-second timeout.

**Reproduction steps:**
1. Configure rune to use branch-based file discovery
2. Run rune in an environment where `git rev-parse` hangs (e.g., network filesystem stall, corrupted `.git` directory)
3. Observe that rune blocks indefinitely instead of timing out after 5 seconds

**Impact:** Any rune command using branch discovery could hang indefinitely if git became unresponsive, requiring manual process termination.

## Investigation Summary

The bug was identified directly from the code structure.

- **Symptoms examined:** The context was created but never connected to the command
- **Code inspected:** `internal/config/discovery.go`, specifically `getCurrentBranchImpl()`
- **Hypotheses tested:** Confirmed that `exec.Command` ignores context entirely and that `exec.CommandContext` is required

## Discovered Root Cause

The function created a timeout context on line 71 but used `exec.Command` (line 65) instead of `exec.CommandContext`. The context was only checked after `cmd.Run()` returned (line 76), but `cmd.Run()` would never return early because the context was not attached to the command.

**Defect type:** API misuse -- `exec.Command` vs `exec.CommandContext`

**Why it occurred:** The context and command were created in the wrong order, and `exec.Command` was used instead of `exec.CommandContext`. The post-run `ctx.Err()` check gave a false sense of timeout handling.

**Contributing factors:** The `exec.Command` and `exec.CommandContext` APIs are easy to confuse, and the existing tests mocked `getCurrentBranch` at the function-pointer level rather than testing the actual implementation's timeout behaviour.

## Resolution for the Issue

**Changes made:**
- `internal/config/discovery.go:68-78` - Moved context creation before command creation, replaced `exec.Command` with `exec.CommandContext`, and added `cmd.WaitDelay` for robust pipe cleanup
- `internal/config/discovery.go:63-65` - Extracted timeout duration to a package-level `gitCommandTimeout` variable for testability

**Approach rationale:** `exec.CommandContext` is the standard Go approach for enforcing timeouts on external commands. `WaitDelay` ensures that if the killed process has child processes holding pipes open, Go will close the pipes after a short grace period rather than blocking indefinitely.

**Alternatives considered:**
- `cmd.Start()` + manual `select` on context - More complex with no benefit over `CommandContext`
- `CommandContext` without `WaitDelay` - Would still block if killed process has children holding pipes (confirmed during testing with shell script mock)

## Regression Test

**Test file:** `internal/config/discovery_test.go`
**Test name:** `TestGetCurrentBranchTimeout`

**What it verifies:** That `getCurrentBranchImpl` returns a timeout error within a reasonable time when the git command hangs, confirming the timeout is enforced at the process level.

**Run command:** `go test -run TestGetCurrentBranchTimeout -v ./internal/config/`

## Affected Files

| File | Change |
|------|--------|
| `internal/config/discovery.go` | Fixed timeout enforcement with `exec.CommandContext` and `WaitDelay` |
| `internal/config/discovery_test.go` | Added `TestGetCurrentBranchTimeout` regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass (`make check`)

**Manual verification:**
- Confirmed that `TestGetCurrentBranchTimeout` completes in under 1 second with a 200ms timeout, proving the timeout is enforced

## Prevention

**Recommendations to avoid similar bugs:**
- When using `context.WithTimeout` with external commands, always use `exec.CommandContext` -- never `exec.Command` with a separate context check
- Set `cmd.WaitDelay` when using `CommandContext` to handle pipe cleanup robustly
- Test timeout behaviour directly rather than only mocking at the function-pointer level

## Related

- Transit ticket: T-249
