# Bugfix Report: Create Command Accepts Newline Titles and Writes Unparseable Files

**Date:** 2026-09-07
**Status:** Fixed
**Ticket:** T-1500

## Description of the Issue

`rune create` accepted any string as `--title`, including one containing an embedded
newline. The title is rendered directly into the markdown H1 heading, so a title with
a `\n` split the heading across two lines. The resulting file could not be parsed back
by rune — the second line was no longer part of the heading.

**Reproduction steps:**

1. Run `rune create tasks.md --title $'Bad\nTitle'`
2. The command succeeds and writes `tasks.md`
3. Run `rune list tasks.md` — the file does not round-trip; the heading is broken

**Impact:** Medium — the command silently produces a file rune itself cannot read back,
and the 500-character title limit documented for tasks was not enforced on the task-list
title at all.

## Investigation Summary

- **Symptoms examined:** A created file whose H1 heading spans multiple lines and fails
  to parse.
- **Code inspected:** `cmd/create.go` (`runCreate`), `internal/task/task.go`
  (`NewTaskList`), `internal/task/operations.go` (`validateTaskInput`,
  `containsNullByte`), `internal/task/render.go` (H1 rendering).
- **Hypotheses tested:** Whether the parser should tolerate multi-line headings —
  rejected, because the correct fix is to never write an unparseable file. Whether
  `NewTaskList` should validate — rejected, see Alternatives.

## Discovered Root Cause

`NewTaskList` sets `tl.Title` unconditionally with no validation. Every other path that
accepts user-supplied text (`AddTask`, `UpdateTask`, batch operations) routes through
`validateTaskInput`, which rejects null bytes and control characters (including `\n`
and `\r`) and enforces `MaxTitleLength`. The task-list title was the one input that
never passed through any such check.

**Defect type:** Missing input validation

**Why it occurred:** The control-character and length checks were added to the task
mutation paths (T-781, and the 500-character enforcement work) but the task-list title
set at creation time was not part of that surface.

## Resolution for the Issue

**Changes made:**

- `internal/task/operations.go` — added exported `ValidateTaskListTitle`, which rejects
  null bytes / control characters and titles exceeding `MaxTitleLength`.
- `cmd/create.go` — `runCreate` calls `ValidateTaskListTitle(createTitle)` before the
  file-exists check and before dry-run output, so nothing is printed and no file is
  written when the title is invalid.

**Approach rationale:** Validating at the CLI layer closes the reachable gap without
touching the 200+ `NewTaskList` call sites (most of which are tests constructing known-
good titles). Running the check first means no partial state leaks on rejection.

**Alternatives considered:**

- **Validate inside `NewTaskList`** — rejected: `NewTaskList` returns no error, so this
  would require changing its signature and every call site, a much larger blast radius
  for a bug only reachable through `create`.
- **Sanitize the title (strip or escape newlines)** — rejected: silently rewriting user
  input is worse than a clear error, and the project's convention elsewhere
  (`validateTaskInput`) is to reject, not sanitize.
- **Reuse `validateTaskInput` directly** — deferred: the two functions are near-identical,
  but `validateTaskInput` is being modified in parallel by T-1561/T-1603/T-1906. Keeping
  a separate function avoids a collision now; consolidating them is a worthwhile follow-up
  once that work lands.

## Regression Test

**Test files:** `cmd/create_test.go`, `internal/task/operations_test.go`
**Test names:** `TestCreateCommandRejectsInvalidTitles`, `TestValidateTaskListTitle`

**What they verify:** `TestCreateCommandRejectsInvalidTitles` checks that `runCreate`
returns an error and writes no file for titles containing `\n`, `\r`, CRLF, a null byte,
or more than `MaxTitleLength` characters. `TestValidateTaskListTitle` covers the validator
directly, including the accepted cases (plain title, empty title, tab, title at exactly
`MaxTitleLength`).

**Run command:**
`go test -run TestCreateCommandRejectsInvalidTitles ./cmd/ && go test -run TestValidateTaskListTitle ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `cmd/create.go` | Validate `--title` before any file check, dry-run output, or write |
| `internal/task/operations.go` | Added `ValidateTaskListTitle` |
| `cmd/create_test.go` | Added `TestCreateCommandRejectsInvalidTitles` |
| `internal/task/operations_test.go` | Added `TestValidateTaskListTitle` |
| `CHANGELOG.md` | Documented the fix under Unreleased |

## Verification

**Automated:**

- [x] Regression tests pass
- [x] Full test suite passes (`make test`)
- [x] Lint passes (`make lint`)
- [x] Build succeeds (`go build ./...`)

## Prevention

**Recommendations to avoid similar bugs:**

- Any struct field that ends up rendered into markdown structure (headings, list markers,
  front matter keys) needs the same control-character check as task titles — treat the
  markdown grammar as the validation contract.
- When adding a validation rule to one input path, grep for the other constructors that
  set the same field. `NewTaskList` was missed because it is a constructor, not a mutation.
- Consolidate `ValidateTaskListTitle` and `validateTaskInput` once T-1561/T-1603/T-1906
  land, so the two checks cannot drift apart.
