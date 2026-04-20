# Testing

## Shared Command State

Command tests share package-level globals from `cmd/root.go` (`format`, `dryRun`, `verbose`) because Cobra binds persistent flags to those variables. Once a test calls `rootCmd.Execute()` with `--format json`, the global stays as `"json"` for every subsequent test in the process.

`resetBatchFlags()` in `cmd/batch_test.go` resets both `batchInput` and `format` (back to `"table"`) along with the `Changed` bits on the corresponding flags. Any batch test that uses `--format json` should register `t.Cleanup(resetBatchFlags)` so the global doesn't leak. Tests that depend on the default `format == "table"` (e.g. `TestRunCompleteDryRun`) get protected indirectly by that cleanup.

## Config / Git Discovery in Tests

`config.LoadConfig` caches the loaded config under a `sync.Once`, so the first call wins for the lifetime of the test process. `internal/config` exposes `ResetConfigCache()` for tests that need to install a different config or chdir into a different repo.

`config.DiscoverFileFromBranch` strips the first path segment of the branch name (e.g. `T-824/homebrew-install` → `homebrew-install`) before substituting `{branch}` into the template. Tests that expect discovery to fail must isolate themselves: chdir into a fresh git repo with a `.rune.yml` that sets `discovery.enabled: false`. Otherwise the test outcome depends on the developer's current branch name and which `specs/<name>/tasks.md` files happen to exist.

## Renumber and Phase Markers

`task.ExtractPhaseMarkers` (T-742) stores `AfterTaskID` as a 1-based sequential count of preceding top-level tasks, NOT as the literal IDs from the markdown. Anything in `cmd/renumber.go` that maps phase markers to positions must convert that count directly (`position = N - 1`); attempting to look up the value in a map of raw file IDs will silently fail for any non-trivial file (T-859 was caused by that mismatch).
