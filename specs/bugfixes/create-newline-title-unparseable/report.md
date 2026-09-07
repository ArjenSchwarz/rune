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

The same failure has a second trigger: an empty or whitespace-only title. `RenderMarkdown`
writes a bare `# ` heading for an empty title (`render.go:110-114`), and `parse.go:130`
only recognises a heading whose trimmed form starts with `"# "`, so `#` alone is not
consumed as a title and is reported as `line 1: unexpected content at this indentation
level`. `--title ""` passes cobra's `MarkFlagRequired`, which only requires the flag to be
present.

**Impact:** Medium — the command silently produces a file rune itself cannot read back,
and the 500-character title limit documented for tasks was not enforced on the task-list
title at all.

## Investigation Summary

- **Symptoms examined:** A created file whose H1 heading spans multiple lines and fails
  to parse.
- **Code inspected:** `cmd/create.go` (`runCreate`), `internal/task/operations.go`
  (`NewTaskList`, `validateTaskInput`, `containsNullByte`),
  `internal/task/render.go` (H1 rendering), `internal/task/parse.go` (title
  detection).
- **Hypotheses tested:** Whether the parser should tolerate multi-line headings —
  rejected, because the correct fix is to never write an unparseable file. Whether
  `NewTaskList` should validate — rejected, see Alternatives.

## Discovered Root Cause

`NewTaskList` sets `tl.Title` unconditionally with no validation. Every other path that
accepts user-supplied text (`AddTask`, `UpdateTask`, batch operations) routes through
`validateTaskInput`, which rejects null bytes and control characters (including `\n`
and `\r`) and enforces `MaxTitleLength`. The task-list title was the one input that
never passed through any such check.

The parser side of the contract is documented in `docs/agent-notes/parsing.md` ("Title
Detection"): only the first non-empty line is considered, and it must start with `# `
followed by text. Anything the renderer emits that violates that is unreadable.

**Defect type:** Missing input validation

**Why it occurred:** The control-character and length checks were added to the task
mutation paths (T-781, and the 500-character enforcement work) but the task-list title
set at creation time was not part of that surface.

## Resolution for the Issue

**Changes made:**

- `internal/task/operations.go` — added exported `ValidateTaskListTitle`, which rejects
  empty and whitespace-only titles and then delegates to the existing
  `validateTaskInput` for the null byte / control character and `MaxTitleLength` checks.
- `cmd/create.go` — `runCreate` calls `ValidateTaskListTitle(createTitle)` before the
  file-exists check and before dry-run output, so nothing is printed and no file is
  written when the title is invalid.

**Approach rationale:** Validating at the CLI layer closes the whole reachable surface:
of the 220 `NewTaskList` call sites, 219 are tests and the single production caller is
`cmd/create.go:87`. Running the check first means no partial state leaks on rejection.

Note that only the control-character and empty-title checks fix an unparseable file. The
`MaxTitleLength` check is a consistency change — a 501-character title parsed back fine
before — so it removes previously working behaviour, deliberately, to match the limit
already enforced on task titles.

**Alternatives considered:**

- **Validate inside `NewTaskList`** — rejected: `NewTaskList` returns no error, so this
  would require changing its signature and every call site, a much larger blast radius
  for a bug only reachable through `create`.
- **Sanitize the title (strip or escape newlines)** — rejected: silently rewriting user
  input is worse than a clear error, and the project's convention elsewhere
  (`validateTaskInput`) is to reject, not sanitize.
- **Duplicate the `validateTaskInput` body** — rejected: an earlier revision of this fix
  copied the two checks into `ValidateTaskListTitle` to avoid colliding with parallel
  work on `validateTaskInput`. That collision does not exist — T-1603 only adds a call
  site in `AddTaskToPhase` and does not touch the function body, and no branch exists for
  T-1561 or T-1906 — so `ValidateTaskListTitle` now delegates and the two checks cannot
  drift apart.
- **Move the check into `WriteFile`/`WriteFileWithPhases`** — considered: it would also
  cover callers that set `TaskList.Title` directly. Not done here because there are none
  outside tests; worth revisiting if `internal/task` gains another consumer.

## Regression Test

**Test files:** `cmd/create_test.go`, `internal/task/operations_test.go`
**Test names:** `TestCreateCommandRejectsInvalidTitles`,
`TestCreateCommandWritesParseableFile`, `TestValidateTaskListTitle`

**What they verify:** `TestCreateCommandRejectsInvalidTitles` checks that `runCreate`
returns an error and writes no file for titles containing `\n`, `\r`, CRLF or a null byte,
for empty and whitespace-only titles, for a title over `MaxTitleLength`, and for the
dry-run path. `TestCreateCommandWritesParseableFile` is the positive half: an accepted
title must produce a file that `task.ParseFile` reads back with the same title — the
assertion that actually locks in the round trip, and the guard against an over-tightened
validator. `TestValidateTaskListTitle` covers the validator directly, including the
accepted cases (plain title, tab, title at exactly `MaxTitleLength`).

**Run command:**
`go test -run 'TestCreateCommand(RejectsInvalidTitles|WritesParseableFile)' ./cmd/ && go test -run TestValidateTaskListTitle ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `cmd/create.go` | Validate `--title` before any file check, dry-run output, or write; document the title constraints in the command help |
| `internal/task/operations.go` | Added `ValidateTaskListTitle`, delegating to `validateTaskInput` |
| `cmd/create_test.go` | Added `TestCreateCommandRejectsInvalidTitles` and `TestCreateCommandWritesParseableFile` |
| `internal/task/operations_test.go` | Added `TestValidateTaskListTitle` |
| `CHANGELOG.md` | Documented the fix under Unreleased |
| `CLAUDE.md` | Recorded the title constraints alongside the existing limits |

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
- A validator is only half the contract; pair every "rejects bad input" test with a
  round-trip test that parses the written file back. The empty-title case was missed
  precisely because the first revision only had negative tests.

**Known remaining gap (separate ticket):** `RenderMarkdown` still emits `# ` for any
`TaskList` with an empty title (`render.go:113`, `render.go:233`), while `parse.go:130`
rejects it — and `internal/task/parse_basic_test.go:108-112` asserts that rejection. A
task file with no H1 heading therefore parses fine but becomes unparseable after any
command that rewrites it (`add`, `complete`, `update`, `batch`). That is a wider defect
than T-1500 and is not fixed here; `create` can no longer produce such a file, but
rewriting a titleless file still can.
