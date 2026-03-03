# Config Discovery (internal/config/discovery.go)

## Overview

Branch-based file discovery for rune. Determines which task file to use based on the current git branch name and a template pattern (e.g., `specs/{branch}/tasks.md`).

## Key Architecture

- `getCurrentBranch` is a package-level function variable pointing to `getCurrentBranchImpl`, allowing tests to mock it
- `gitCommandTimeout` is a package-level variable (default 5s) controlling the git command timeout, also for testability
- `DiscoverFileFromBranch` strips the branch prefix (before first `/`) and tries both stripped and full branch names

## Git Command Execution

Uses `exec.CommandContext` with a timeout context to prevent hangs. Key details:

- `cmd.WaitDelay` (500ms) is set to ensure pipe cleanup after process kill -- without this, child processes inheriting pipes can keep `cmd.Wait()` blocking indefinitely even after SIGKILL
- The effective maximum wait time is `gitCommandTimeout + WaitDelay`
- The `ctx.Err()` check after `cmd.Run()` distinguishes timeout errors from other git failures

## Testing

- Tests in `discovery_test.go` use mock git scripts via PATH manipulation
- `setupMockGitCommand` creates shell scripts in temp dirs and prepends to PATH
- Global state mutation (PATH, gitCommandTimeout) means tests MUST NOT use `t.Parallel()`
- The timeout test (`TestGetCurrentBranchTimeout`) uses a 200ms timeout with a mock git that sleeps 10s, verifying the function returns within the computed bound

## Gotchas

- `exec.Command` vs `exec.CommandContext`: The former ignores context entirely. Always use `CommandContext` when a timeout is needed.
- On Unix, killing a shell script does not kill its child processes. `WaitDelay` is essential to prevent `cmd.Run()` from blocking on inherited pipes.
