# Bugfix Report: Next Stream/Phase Scope

**Date:** 2026-11-02
**Status:** Fixed
**Tickets:** T-905, T-1319, T-1433, T-1705, T-2000

## Description of the Issue

Five related problems in `rune next` when `--phase`, `--stream` and `--claim` are combined. Each one either hid ready work or showed work from outside the requested scope.

| Ticket | Command | Observed | Expected |
|--------|---------|----------|----------|
| T-905 | `next --phase --claim A` with a completed parent and a pending child in the first phase | "No ready tasks to claim" | Claims the child (1.1), as plain `--claim` does |
| T-1319 | `next --phase --stream 1` on a file without `##` headers | Success with empty `tasks` | Stream 1 tasks of the file, as `--phase` already does for such files |
| T-1433 | `next --stream 2` where the stream 2 parent has a stream 1 child | Stream 1 child in `children` (JSON, table, markdown) | Only stream 2 tasks |
| T-1705 | `next --phase --stream 2`, same layout inside a phase | Stream 1 child in `children` (JSON, table, markdown) | Only stream 2 tasks |
| T-2000 | `next --phase --format json` with stream 2 work only in a later phase | `streams_summary` lists stream 2 as available | Summary covers the selected phase only |

**Reproduction steps:** the markdown from each ticket, run with the build from the PR113 branch (58b7085). All five reproduced as described.

**Impact:** Medium. Agents that coordinate work through stream- or phase-scoped `next` output were shown work from other streams or later phases, or were told no work existed when it did.

## Investigation Summary

- **Symptoms examined:** JSON output of each repro on the base build.
- **Code inspected:** `cmd/next.go` (`runNextWithClaim`, `runNextWithStream`, `runNextPhase`, `outputPhaseTasksJSONWithStreams`, the table/markdown renderers), `internal/task/next.go` (`FindNextPhaseTasks`, `FindNextPhaseTasksForStream`), `internal/task/streams.go` (`FilterByStream`, `AnalyzeStreams`), and `specs/stream-phase-navigation`.
- **Hypotheses tested:** The likely causes named in each ticket were confirmed. No other cause was found.

## Discovered Root Cause

**Defect type:** Scope errors: filters applied at one level but not carried through to the next.

1. **T-905:** The `--phase --claim` branch applied `isTaskReady` to the top-level entries of `FindNextPhaseTasks` only. `FindNextPhaseTasks` keeps a completed parent whose child is pending, so the phase was selected, but the completed parent is not ready and its child was never looked at.
2. **T-1319:** `FindNextPhaseTasksForStream` returned `nil` when `!hasPhases(lines)`. That followed requirement 1.7 of `stream-phase-navigation`, but `FindNextPhaseTasks` falls back to the whole file in the same case, so the two variants disagreed.
3. **T-1433:** `runNextWithStream` filtered the flat ready list by stream, then rendered the chosen task with its full `Children`. Every renderer recurses into `Children` without checking the stream.
4. **T-1705:** `FindNextPhaseTasksForStream` returned stream-matching tasks with their full (incomplete) subtrees. The phase renderers again recurse into `Children` without checking the stream.
5. **T-2000:** `outputPhaseTasksJSONWithStreams` called `AnalyzeStreams` on the whole file's tasks instead of the selected phase's.

**Why it occurred:** Stream filtering (`FilterByStream`, `FilterByStreamFlat`) selects tasks but leaves their `Children` intact, and the output code assumes every child belongs in the output.

## Resolution for the Issue

**Changes made:**
- `internal/task/streams.go` - New `PruneToStream(task, stream)` returns a copy whose descendants are limited to one stream. A descendant in another stream is dropped with its subtree.
- `internal/task/next.go` (`FindNextPhaseTasksForStream`) - A file without phase headers is one unnamed phase (T-1319). Returned tasks are passed through `PruneToStream` (T-1705), which fixes the JSON, table and markdown phase renderers together.
- `cmd/next.go` (`runNextWithStream`) - The selected task is passed through `PruneToStream` before it is rendered (T-1433).
- `cmd/next.go` (`readyPhaseTaskIDs`) - `--phase --claim` claims ready phase tasks and looks through completed tasks to their ready subtasks (T-905). Pending, in-progress and claimed parents are not looked through, so a pending parent is still claimed as a whole. This matches `hasReadyWork` from PR113.
- `cmd/next.go` (`outputPhaseTasksJSONWithStreams`) - The streams summary is computed from `phaseResult.Tasks`, with the file-wide dependency index kept so blockers in other phases still resolve (T-2000). With `--stream`, children are stripped first because the result is a flat list that also holds nested matches.
- `specs/stream-phase-navigation/requirements.md` 1.7 amended; Decision 6 added to its decision log.

**Approach rationale:** Each fix is placed where the scope is decided, so all output formats pick it up.

**Alternatives considered:**
- T-1319: reject `--phase --stream` on unphased files with an explicit error. Not chosen because `--phase` alone falls back, and the two variants would still disagree. See Decision 6. The reviewer can still pick this option.
- T-1433/T-1705: promote other-stream children's same-stream descendants instead of dropping the whole subtree. Not chosen because it would render tasks under a parent they are not direct children of. Those descendants are still found as separate entries by `--stream --claim` and `--phase --stream`.

## Regression Test

**Test file:** `cmd/next_stream_phase_regressions_test.go`

| Test | Ticket |
|------|--------|
| `TestNextPhaseClaimReadyChildOfCompletedParent` | T-905 |
| `TestNextPhaseClaimOnlyDescendsThroughCompletedParents` | T-905 (guard: a pending or claimed parent is unchanged) |
| `TestNextPhaseStreamUnphasedFile`, `TestNextPhaseStreamClaimUnphasedFile` | T-1319 |
| `TestNextStreamExcludesOtherStreamChildren` | T-1433 (json, table, markdown) |
| `TestNextPhaseStreamExcludesOtherStreamChildren` | T-1705 (json, table, markdown) |
| `TestNextPhaseStreamsSummaryScopedToPhase`, `TestNextPhaseStreamSummaryScopedToPhase` | T-2000 |

Updated tests: `TestFindNextPhaseTasksForStream/no_phases_in_document` and the integration workflow `no_phases_falls_back_to_file` (previously `no_phases_returns_error`). Both encoded the old T-1319 behavior.

**Run command:** `go test ./cmd -run 'TestNextPhase|TestNextStream'`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/streams.go` | Add `PruneToStream` |
| `internal/task/next.go` | Unphased fallback and stream pruning in `FindNextPhaseTasksForStream` |
| `cmd/next.go` | Stream pruning for `--stream`, phase claim recursion, phase-scoped summary |
| `cmd/next_stream_phase_regressions_test.go` | New regression tests |
| `internal/task/next_test.go`, `cmd/integration_test.go` | Update the no-phases expectation |
| `specs/stream-phase-navigation/*` | Amend requirement 1.7, add Decision 6 |

## Verification

**Automated:**
- [x] Regression tests fail before the fix (7 of 8; the guard test passes on both) and pass after it
- [x] Full unit and integration suites pass, except `TestCreateBackup/error_handling_for_read_failures` and `TestIntegrationRenumber/renumber_write_failure`. Both fail identically on the unmodified base because the session runs as root, which ignores file permission bits.
- [x] `make lint`: 0 issues; `go vet ./...` clean

**Manual verification:**
- Re-ran each ticket's repro with the fixed build; the output matches the expected column above.

## Prevention

- Pass stream-filtered tasks through `PruneToStream` before rendering anything that recurses into `Children`.
- Scope summaries to the same task set as the main result.

## Related

- Stacked on PR #113 (`claude/rune-blocker-readiness-jtmpuo`); merge #113 first.
- T-1191 (abandoned): completed children stay visible as context. These fixes leave completion-based filtering alone; only stream membership is filtered.
