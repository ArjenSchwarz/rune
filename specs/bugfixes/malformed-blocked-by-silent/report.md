# Bugfix Report: Malformed Blocked-by Metadata Silently Ignored

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1998

## Description of the Issue

```markdown
- [x] 1. Finished prerequisite <!-- id:aaaaaaa -->
- [ ] 2. Must wait <!-- id:bbbbbbb -->
  - Blocked-by: abc123 (missing prerequisite)
```

`rune list --format json` showed no warning, and `rune next` selected task 2. Requirement 2.11 of `specs/task-dependencies-and-streams` requires a warning when a blocked-by reference has an invalid stable ID, and the invalid reference to be ignored.

**Impact:** Medium. A typo in a dependency silently made a task ready with no hint why.

## Investigation Summary

- **Code inspected:** `parseBlockedByLine`, `blockedByRefPattern` and `applyDetailLine` in `internal/task/parse.go`; read commands in `cmd/`.
- **Finding:** `blockedByRefPattern` searched the whole value for any 7-character lowercase alphanumeric word. In the reproduction, the word `missing` inside the title hint was taken as a stable ID, so the task got `BlockedBy: ["missing"]`. That ID resolves to nothing, so the task counted as ready. A rewrite would have saved the line as `Blocked-by: missing`. Lines with no such word fell back to a plain detail. In neither case was a warning produced, and the parser had no way to report one.

## Discovered Root Cause

**Defect type:** Unanchored matching plus no warning channel.

## Resolution for the Issue

**Changes made:**
- `internal/task/parse.go` - `parseBlockedByLine` splits the value on commas outside parenthesised title hints (falling back to every comma when the parentheses do not balance). Each reference must be a 7-character stable ID, optionally followed by a title hint. Valid IDs are kept, invalid references are returned separately, and `applyDetailLine` records a `WarnInvalidBlockedBy` warning on `TaskList.Warnings`. A line with no valid reference stays a plain detail, so its text is not lost on a write.
- `internal/task/task.go`, `internal/task/errors.go` - `TaskList.Warnings` field and `WarnInvalidBlockedBy` code.
- `cmd/format.go`, `cmd/list.go`, `cmd/next.go`, `cmd/find.go` - `reportParseWarnings` writes warnings to stderr after parsing, for every output format, so JSON on stdout stays valid.

**Overlap with PR #113:** PR #113 (T-1762 and others) adds the same `TaskList.Warnings` field and `reportParseWarnings` helper for duplicate stable IDs. These are copied byte-for-byte, with the call sites in `list`, `next` and `find` placed exactly where PR #113 puts them, so the two branches merge with no conflict or only a trivial one. PR #113 also reports warnings from mutating commands; this branch only covers the read commands the ticket names.

**Merging with PR #113 (review follow-up):** #113 adds `taskList.Warnings = dropDuplicateStableIDs(taskList.Tasks)` to `parseContent`. That assignment replaces the warnings this fix appends while parsing lines, even when there are no duplicates. Without intervention, git auto-merged it and silently lost every invalid Blocked-by warning. This branch puts a comment stating the append-only rule at that exact spot, so the merge now conflicts in `internal/task/parse.go`. Resolve it as:

```go
	// taskList.Warnings already holds warnings found while parsing lines
	// (invalid Blocked-by references). Checks that run on the parsed tasks
	// must append to it, never assign it, or those warnings are lost.
	taskList.Warnings = append(taskList.Warnings, dropDuplicateStableIDs(taskList.Tasks)...)
```

`TestInvalidBlockedByWarningSurvivesDuplicateStableIDs` (and `TestMalformedBlockedByWarns`) fail if the plain assignment is kept. `CHANGELOG.md` also conflicts under `[Unreleased] / Fixed`: keep both sides' entries. This was checked in a disposable local merge with #113 at `58b7085`. With this resolution the full suite passed (apart from the root-only `TestCreateBackup` failure), and `rune list` printed both the invalid-blocker and duplicate-ID warnings for a file with both problems.

**Behaviour changes:**
- In the reproduction, `next` still selects task 2: requirement 2.11 says the invalid reference is ignored, so it does not block. What changes is that every read command now warns about it.
- Words inside a title hint are no longer taken as IDs.
- A partly valid line (`aaaaaaa (A), abc123`) keeps `aaaaaaa`, warns about `abc123`, and is written back without the invalid reference on the next mutation, which is the "ignore" in requirement 2.11.
- Space-separated IDs without commas (`aaaaaaa bbbbbbb`) are now invalid; the spec format is comma-separated and rune always writes commas.

## Regression Test

**Test files:** `internal/task/blocked_by_warning_regression_test.go`, `cmd/blocked_by_warning_regression_test.go`
**Test names:** `TestMalformedBlockedByWarns`, `TestValidBlockedByHasNoWarnings`, `TestInvalidBlockedByWarningSurvivesDuplicateStableIDs`, `TestReadCommands_WarnOnMalformedBlockedBy` (list JSON, list table, next JSON)

**Run command:** `go test ./internal/task ./cmd -run 'BlockedByWarns|ValidBlockedByHasNoWarnings|WarnOnMalformedBlockedBy'`
