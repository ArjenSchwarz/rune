# Bugfix Report: Output Context

**Date:** 2026-11-04
**Status:** Fixed

Six low-priority output-completeness bugs, fixed together because each is a
renderer or output path dropping context the data already holds:
T-2070, T-2100, T-2127, T-2182, T-2242, T-2274.

## Description of the Issues

| Ticket | Symptom |
|--------|---------|
| T-2070 | `find --include-parent` with `--max-depth`/`--status`/`--parent` could return only the injected parent context, with the real match filtered out (exit 0, count 1, no task matching the pattern). With `--status completed` the pending parent of a completed match was dropped. |
| T-2100 | `list` on an empty file, or with filters that match nothing, dropped the title and front matter from JSON and Markdown output. |
| T-2127 | `--verbose --format json` printed `Git discovery failed: ...` to stdout when automatic discovery failed. |
| T-2182 | `list --format markdown` showed front-matter references only with `--all`. |
| T-2242 | `find` table and Markdown output omitted stream, owner, Blocked-by and requirements; JSON had them. |
| T-2274 | `next --stream N` table and Markdown output omitted stream and Blocked-by IDs; JSON had them. |

**Reproduction:** Each ticket's reproduction was run against a binary built
from `73e31eb` in fresh temporary directories, and each matched the ticket.
The regression tests in `cmd/output_context_regression_test.go` encode these
reproductions.

**Impact:** Low. The data was never lost; it was missing from some output
formats or paths, and JSON or extra flags gave a workaround in every case.

## Investigation Summary

- **Code inspected:** `cmd/find.go`, `internal/task/search.go`, `cmd/list.go`,
  `cmd/next.go`, `cmd/filename.go`, `cmd/format.go`.
- **Pattern:** Each bug is a single output path that missed context the other
  paths already carried. JSON or other branches had been fixed earlier
  (e.g. T-2176 for `next --phase`, T-1206/T-1995 for find JSON); the human
  renderers and early-return paths here were not updated at the same time.

## Discovered Root Causes

- **T-2070:** `TaskList.Find` inserted parent context into the result slice,
  and `applyAdditionalFilters` then filtered matches and context together
  without knowing which was which.
- **T-2100:** `runList` returned through `outputListEmpty(message)` before
  the parsed `TaskList` reached any renderer, so title and front matter
  were unavailable.
- **T-2127:** `resolveFilename` used `fmt.Printf` for the discovery
  diagnostic instead of `verboseStderr`.
- **T-2182:** `outputMarkdownWithFilters` gated the Document References
  section on `showAll`.
- **T-2242:** `outputSearchResultsTable` used a fixed column list and
  `outputSearchResultsMarkdown` rendered only identity, parent, details and
  references.
- **T-2274:** `runNextWithStream` sent only JSON to the stream-aware
  renderer; table and Markdown used the generic `next` renderers, which had
  no stream or dependency data.

**Defect type:** Incomplete output paths (missing context), plus one
ordering error (T-2070).

## Resolution

**Changes made:**
- `internal/task/search.go` — export `insertParents` as `InsertParents` so
  callers can attach context after filtering. `Find` with `IncludeParent`
  behaves as before.
- `cmd/find.go` — search without parent context, apply filters, then call
  `InsertParents` when `--include-parent` is set (T-2070). Before
  inserting, each result's ParentID is reset to its source value: the
  filters re-point a ParentID past a filtered-out parent (T-549), which
  would otherwise stop a parent that matched the pattern but failed the
  filter from being restored as context. The table adds
  Stream, BlockedBy, Owner and Requirements columns when any result has that
  data (the same conditional pattern `list` uses); Markdown adds
  `Blocked-by`, `Stream`, `Owner` and `Requirements` lines in task-file
  metadata order, with Blocked-by translated against the whole file (T-2242).
- `cmd/list.go` — `outputListEmpty` takes the `TaskList`. Empty JSON keeps
  the existing `success`/`message`/`count`/`data` envelope and adds `Title`
  and `FrontMatter` (same keys as non-empty list JSON, omitted when empty).
  Empty Markdown writes the Document References section and the title
  before the message (T-2100). The `showAll` gate is removed from
  `outputMarkdownWithFilters` (T-2182).
- `cmd/filename.go` — discovery failure goes through `verboseStderr` (T-2127).
- `cmd/next.go` — `outputNextTaskTable`, `outputNextTaskMarkdown` and their
  child helpers take an optional `*task.DependencyIndex`. `next --stream`
  passes its index and gets Stream and BlockedBy for every listed task, in
  the same layout as T-2176's phase output; plain `next` passes nil and its
  output is unchanged (T-2274).

**Approach rationale:** Each fix is the smallest change the ticket allows,
and it reuses the helpers and output layouts that existing paths already use
(`formatBlockedByForDisplay`, `requirementLinks`, `detectNonDefaultStreams`,
the T-2176 layout).

**Alternatives considered:**
- T-2070: track context provenance in the result slice and drop orphaned
  context after filtering. Rejected: this adds state, while filtering before
  inserting context fixes the cause directly.
- T-2274: always show Stream/BlockedBy in plain `next` output. Rejected:
  broader than the ticket, which is scoped to `--stream`.
- T-2100: add table output to the empty-result fix. Not done: the ticket
  names JSON and Markdown only.

**Behaviour note (T-2070):** Context is now attached after `--parent`
filtering too, so `find --parent 1.1 --include-parent` also returns task 1.1
as context. Previously it was dropped, because its own ParentID did not
match the filter.

**Not changed:** `outputMarkdown` and `outputMarkdownWithPhases` in
`cmd/list.go` still carry the `showAll` gate. They have no callers and were
left alone to keep the diff narrow.

## Regression Tests

**Test file:** `cmd/output_context_regression_test.go`

| Test | Ticket |
|------|--------|
| `TestFind_IncludeParentContextOnlyForSurvivingMatches` (max-depth, status, surviving match, filtered-out matching parent) | T-2070 |
| `TestList_EmptyResultKeepsDocumentContext` (json, markdown) | T-2100 |
| `TestList_MarkdownIncludesDocumentReferencesWithoutAll` | T-2182 |
| `TestResolveFilename_DiscoveryFailureVerboseGoesToStderr` | T-2127 |
| `TestFind_HumanOutputShowsTaskMetadata` (table, markdown) | T-2242 |
| `TestNextStream_HumanOutputShowsStreamAndBlockedBy` (table, markdown) | T-2274 |

All of these failed before the fix and pass after it.

**Run command:** `go test ./cmd -run 'TestFind_IncludeParentContextOnly|TestList_EmptyResultKeeps|TestList_MarkdownIncludesDocument|TestResolveFilename_DiscoveryFailure|TestFind_HumanOutputShows|TestNextStream_HumanOutput'`

## Verification

- [x] Regression tests pass
- [x] `make check`: fmt and lint clean (0 issues); all tests pass except
      `TestCreateBackup/error_handling_for_read_failures`
- [x] `make test-integration`: all pass except
      `TestIntegrationRenumber/renumber_write_failure`

Both remaining failures also fail on unmodified `origin/main` (`73e31eb`) in
this environment. They rely on chmod making a file unreadable or unwritable,
and the cloud container runs tests as root, which ignores those permissions.

Manual: the ticket reproductions were re-run against the fixed binary and now
show the expected output.

## Prevention

- When adding metadata to one output format, update the other formats for the
  same command in the same change.
- Early-return "empty" paths should receive the parsed document, not just a
  message string.
