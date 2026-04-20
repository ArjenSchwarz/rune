# Testing

## Shared Command State

Command tests often call `runX` helpers directly and share package-level globals such as `format` and `dryRun`. Tests that expect default table output must set and restore `format` explicitly, otherwise earlier JSON/markdown tests can leak state. T-857 tracks the current `TestRunCompleteDryRun` failure.

## Current Known Test Failures

- T-856: `internal/task/phase_test.go` has a stale two-argument call to `RenderMarkdownWithPhases`; the production function now requires a `phaseSource *TaskList`.
- T-859: `cmd.TestRenumberPreservesAllPhaseMarkers` shows `runRenumber` misplacing phase markers for files with gapped/non-sequential top-level IDs. `ExtractPhaseMarkers` already returns sequential IDs, but `cmd/renumber.go` still maps markers through raw file task IDs.
