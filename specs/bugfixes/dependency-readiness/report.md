# Bugfix Report: Dependency Readiness

**Date:** 2026-11-02
**Status:** Fixed
**Tickets:** T-1762, T-1772, T-1777, T-2126, T-2161

## Description of the Issue

Five related defects let tasks be reported as ready (or blocked) for the wrong reason.

| Ticket | Symptom |
|--------|---------|
| T-1762 | Two tasks with the same `<!-- id:... -->`: the later task silently took over the ID, so `Blocked-by` resolved to it. A dependent of a completed task was reported blocked, and its `blockedBy` JSON pointed at the wrong task. No warning. |
| T-1772 | `remove` of a parent cleaned references to the parent's stable ID only. A surviving task blocked by a deleted child stayed blocked forever. |
| T-1777 | Plain `next` (no `--stream`/`--claim`) returned the first incomplete task even when it was blocked. |
| T-2126 | Completing a parent's last child auto-completed the parent even when the parent itself was blocked, unlocking the parent's dependents. |
| T-2161 | A blocker marked `[x]` with a pending subtask satisfied its dependents. |

**Reproduction steps:** as described in each ticket. All five were reproduced on `2ab5286` with a freshly built binary before any change (outputs matched the tickets).

**Impact:** Agents using `next`, `next --stream`, `next --claim` or `streams` could pick up work whose prerequisites were not done, or never see work that was in fact unblocked.

## Investigation Summary

- **Symptoms examined:** `streams --json`, `list --format json`, `next` (plain and `--stream`), `complete --format json` and `remove` output on minimal files per ticket.
- **Code inspected:** `internal/task/parse.go`, `dependencies.go`, `operations.go`, `autocomplete.go`, `next.go`, `cmd/next.go`, `cmd/streams.go`, and all callers of `IsReady`/`IsBlocked`, `RemoveTaskWithDependents` and `AutoCompleteParents`.
- **Hypotheses tested:** That `remove` and batch `remove` bypass `RemoveTaskWithDependents` (an older agent note said so). Ruled out: both paths now call it; the defect was the root-only stable-ID cleanup.

## Discovered Root Cause

| Ticket | Root cause |
|--------|-----------|
| T-1762 | The parser accepted duplicate stable IDs, and `BuildDependencyIndex` assigned `byStableID[id] = task` unconditionally, so the last occurrence won. |
| T-1772 | `RemoveTaskWithDependents` called `removeFromBlockedByLists(task.StableID)` for the selected task only, then deleted the whole subtree. |
| T-1777 | The plain branch of `runNext` called `FindNextIncompleteTask`, which only checks completion state and never consults a `DependencyIndex`. |
| T-2126 | `AutoCompleteParents` only checked `allChildrenComplete(parent)`. |
| T-2161 | `DependencyIndex.IsReady` checked `blocker.Status != Completed`, ignoring the blocker's descendants. |

**Defect type:** Logic errors / missing validation.

**Why it occurred:** Readiness was added for stream and claim workflows, and each of these paths kept the older "checkbox only" notion of completion that predates dependencies.

## Resolution for the Issue

**Changes made:**
- `internal/task/parse.go` — `dropDuplicateStableIDs` keeps a stable ID on its first task (document order), clears it from later tasks, and records a `duplicate_stable_id` `Warning` in the new `TaskList.Warnings` field (T-1762).
- `internal/task/dependencies.go` — `BuildDependencyIndex` is first-wins for stable IDs (T-1762); `IsReady` uses `hasIncompleteWork(blocker)` so a blocker must be completed along with its whole subtree (T-2161).
- `cmd/format.go` + every command that parses a task file — `reportParseWarnings` prints parse warnings to stderr as `Warning: ...`, keeping JSON on stdout clean (T-1762).
- `internal/task/operations.go` — `RemoveTaskWithDependents` collects the stable IDs of the whole subtree, removes it, then strips all of those IDs from surviving `BlockedBy` lists. The returned warning counts surviving tasks only (T-1772).
- `internal/task/next.go` + `cmd/next.go` — new `FindNextReadyTask` and `FilterToFirstReadyPath`; plain `next` skips tasks that are blocked or claimed, `--one` follows the first child with ready work, and an all-blocked file reports "No ready tasks: remaining tasks are blocked or claimed" (T-1777).
- `internal/task/autocomplete.go` — `AutoCompleteParents` skips a parent that is blocked (T-2126). This covers `complete` and batch status updates; `cmd/complete.go` (touched by open PR #108) is not changed beyond the one-line warning call.

**Approach rationale:** Each fix is at the single place the wrong decision is made, so every caller (CLI, batch, streams, claim) picks it up. Dropping the duplicate ID at parse time means downstream code (removal, cycle detection, translation) can keep assuming unique stable IDs.

**Alternatives considered:**
- T-1762: keep both copies of the ID in the file and only make the index first-wins — rejected because removal and other per-ID operations would still act on both tasks.
- T-1762: reject duplicates as a parse error — rejected because the design documents this case as a warning, and an error would make `list` unusable on such files.
- T-1777: descend into a blocked parent's children — rejected; the children belong to the blocked parent's work, and the stream path already treats readiness per task.

## Regression Test

**Test files:** `internal/task/dependency_readiness_regression_test.go`, `cmd/dependency_regressions_test.go`

**Tests:**
- `TestParseDuplicateStableIDKeepsFirstOccurrence`, `TestBuildDependencyIndexDuplicateStableIDFirstWins`, `TestDuplicateStableIDDoesNotRebindDependency` (T-1762)
- `TestRemoveTaskWithDependentsCleansDescendantReferences`, `TestRemoveParentCleansChildDependencyCLI`, `TestRemoveParentCleansChildDependencyBatch` (T-1772)
- `TestFindNextReadyTask`, `TestFilterToFirstReadyPathSkipsBlockedChild`, `TestPlainNextSkipsBlockedTask`, `TestPlainNextNoReadyWork` (T-1777)
- `TestAutoCompleteParentsSkipsBlockedParent`, `TestCompleteDoesNotAutoCompleteBlockedParent` (T-2126)
- `TestIsReadyRequiresBlockerSubtreeComplete`, `TestCompletedBlockerWithPendingChildKeepsDependentBlocked` (T-2161)

All CLI tests were confirmed failing before the fix.

**Run command:** `go test ./internal/task ./cmd -run 'Duplicate|CleansDescendant|CleansChild|NextReady|ReadyPath|PlainNext|AutoComplete.*Blocked|BlockedParent|BlockerSubtree|PendingChild'`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/parse.go` | Drop duplicate stable IDs, record warnings |
| `internal/task/task.go` | `TaskList.Warnings` field (`json:"-"`) |
| `internal/task/dependencies.go` | First-wins index; subtree-aware `IsReady` |
| `internal/task/operations.go` | Subtree stable-ID cleanup on remove |
| `internal/task/next.go` | `FindNextReadyTask`, `FilterToFirstReadyPath` |
| `internal/task/autocomplete.go` | Skip blocked parents |
| `cmd/next.go` | Plain `next` uses ready-aware lookup |
| `cmd/format.go`, `cmd/*.go` | Print parse warnings to stderr |
| `README.md`, `CHANGELOG.md`, `docs/agent-notes/dependencies.md` | Docs |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes, except `TestCreateBackup/error_handling_for_read_failures` and `TestIntegrationRenumber/renumber_write_failure`, which fail identically on unmodified `main` because the cloud container runs tests as root (a mode-0000 file stays readable)
- [x] `make lint`: 0 issues

**Manual verification:**
- Re-ran each ticket's reproduction with the rebuilt binary; all outputs match the expected behaviour.

## Prevention

- Readiness decisions should go through `DependencyIndex` rather than checking `Status` directly.

## Related

- T-1934 / PR #108 (completion dry-run) — separate fix in `cmd/complete.go`, not duplicated here.
- T-288, T-347 (earlier `next` readiness work, stream/claim paths only).
- T-422 (earlier `RemoveTaskWithDependents` fix).
