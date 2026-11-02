# Bugfix Report: Scheduling and Output Context

**Date:** 2026-11-02
**Status:** Fixed
**Tickets:** T-1206, T-1626, T-1809, T-2010, T-2176, T-2184

## Description of the Issue

| Ticket | Command | Observed | Expected |
|--------|---------|----------|----------|
| T-1206 | `list --filter pending --format json`, `find --format json` | Task whose blocker is outside the result has no `blockedBy` | `blockedBy` translated from the full file |
| T-1626 | `streams` with an owned, not-started pending task | "No streams with pending tasks" / empty JSON | Stream listed with its claimed work, unavailable |
| T-1809 | `list` with a Blocked-by ID that matches no task | No blocker shown, yet task is blocked in `streams` and `next` | Unresolved ID reported |
| T-2176 | `next --phase` table and markdown | No stream; table never shows blocked-by | Stream and blocked-by for every task, as in JSON |
| T-2184 | `next --claim` (all formats) | Only id/title/status/stream/owner | Details, task references, front matter references |
| T-2010 | `next --claim ''` | Exit 0, read-only `next_task`, file unchanged | Rejected |

**Reproduction steps:** each ticket's repro, run against the PR122 head (6bc56f7). All six reproduced.

**Impact:** Medium. Scheduling state and dependency output disagreed, and claim output dropped the context agents need.

## Investigation Summary

- **Code inspected:** `internal/task/render.go` (`toJSONTask`, `RenderJSONWithPhases`), `internal/task/dependencies.go` (`IsReady`, `TranslateToHierarchical`), `internal/task/streams.go` (`AnalyzeStreams`), `cmd/list.go`, `cmd/find.go`, `cmd/streams.go`, `cmd/next.go`.
- **Open PRs checked:** #120 (T-1998) warns about *syntactically invalid* Blocked-by IDs and ignores them, per requirement 2.11. T-1809 is about *valid but missing* IDs, which stay blocking, so the two do not overlap.

## Discovered Root Cause

- **T-1206:** JSON rendering built its `DependencyIndex` from the rendered (filtered) tasks, so blockers outside the result could not be translated and were dropped.
- **T-1809:** `IsReady` keeps a missing blocker as blocking, but `TranslateToHierarchical` drops IDs it cannot resolve, so output showed no blocker.
- **T-1626:** `AnalyzeStreams` put owned pending tasks in no category, and `filterEmptyStreams` then dropped the stream as empty.
- **T-2176:** The phase table built rows from ID/title/status/level/phase only. The markdown renderer wrote blockers only as a "(blocked by: …)" note while still blocked, and never wrote the stream.
- **T-2184:** The claim renderers only wrote identity, status, stream, owner (and, since PR122, requirements). The front matter argument went unused.
- **T-2010:** `runNext` routed to claim mode only when `claimFlag != ""`, so an explicit empty value took the read-only path.

## Resolution for the Issue

**Changes made:**
- `internal/task/render.go` - `RenderJSONWithPhases` builds the dependency index from `phaseSource` (the unfiltered list) when given. `toJSONTask` adds `unresolvedBlockedBy`.
- `cmd/find.go` - Search JSON renders through `RenderJSONWithPhases(results, nil, source)`, which produces the same structure as `RenderJSON` while resolving blockers against the source file.
- `internal/task/dependencies.go` - `UnresolvedBlockers(stableIDs)`.
- `cmd/helpers.go` - `formatBlockedByForDisplay` lists hierarchical IDs plus `<id> (unresolved)`. Used by the list table and the new phase columns.
- `internal/task/streams.go`, `cmd/streams.go` - New `claimed` category (JSON array, table column). `filterEmptyStreams` keeps streams that have claimed tasks; availability is still based on ready tasks only.
- `cmd/next.go`
  - Phase table: Stream and BlockedBy columns for every row.
  - Phase markdown: `Stream:` and, when the task has blockers, `Blocked-by:` lines.
  - Claim JSON: `details`, `references` and `front_matter_references`.
  - Claim markdown and table: the same context.
  - `--claim` given with an empty or whitespace-only value returns an error.
- `docs/json-api.md`, `README.md` - Document `unresolvedBlockedBy`, `claimed`, and the claim context fields.

**Approach rationale:**
- **T-1809:** unresolved IDs get their own JSON field instead of being mixed into `blockedBy`. The documented `blockedBy` schema allows only hierarchical IDs, and a 7-character stable ID could be all digits.
- **T-1626:** a new category is additive. The three categories required by task-dependencies-and-streams 5.2/5.6 keep their meaning.
- **T-2176:** metadata is added only to tasks the phase output already shows. Claimed and blocked subtasks that PR113 hides from plain `next` stay hidden.

**Alternatives considered:**
- T-1809: keep unresolved IDs inside `blockedBy`. Rejected for the schema reason above.
- T-1626: keep claimed-only streams but leave all counts at zero. Rejected because the output would not explain why the stream is listed.
- T-2010: route empty claims into claim mode and print "No ready tasks to claim". Rejected because an empty agent ID is a caller error, and the ticket prefers rejection.

**Test-harness note (T-2010):** `cmd.Flags().Changed("claim")` stays true across `Execute` calls in one process, and the command tests call `rootCmd.Execute` many times. `runNext` clears the bit after reading it, the same way the tests already reset the flag's value. The CLI runs one command per process, so this has no effect there.

## Regression Test

**Test file:** `cmd/scheduling_output_regressions_test.go`

| Test | Ticket |
|------|--------|
| `TestFilteredJSONKeepsExternalBlockedBy` (list filter, find, find + status) | T-1206 |
| `TestStreamsReportsClaimedPendingTasks` (JSON and table) | T-1626 |
| `TestListReportsUnresolvedBlockers` (JSON and table) | T-1809 |
| `TestUnresolvedBlockerStillBlocks` (guard: passes before and after) | T-1809 |
| `TestNextPhaseTableShowsStreamAndBlockedBy`, `TestNextPhaseMarkdownShowsStreamAndBlockedBy` (task and subtask) | T-2176 |
| `TestNextClaimIncludesTaskContext` (JSON, markdown, table, dry run) | T-2184 |
| `TestNextEmptyClaimIsRejected` (empty, whitespace; file unchanged; later plain `next` still works) | T-2010 |

All tests except the guard failed on 6bc56f7 and pass after the fix.

## Verification

- [x] Regression tests fail before and pass after the fix
- [x] `make check`: lint 0 issues; only `TestCreateBackup/error_handling_for_read_failures` fails (pre-existing; root ignores file permissions)
- [x] `INTEGRATION=1 go test ./...`: only that test and `TestIntegrationRenumber/renumber_write_failure` fail, both pre-existing for the same reason
- [x] Each ticket's repro re-run with the fixed build

## Related

- Stacked on PR #122 (→ #117 → #113). Merge order: #113, #117, #122, then this PR.
- PR #120 (stacked on #112) changes reference parsing (escaped commas) and adds parse warnings. Claim and phase output print `Task.References` as parsed, so after both merge they show #120's parsed references without code changes. Expect textual conflicts only around `reportParseWarnings` call sites in `cmd/next.go`/`cmd/find.go`, which #113 also adds.
