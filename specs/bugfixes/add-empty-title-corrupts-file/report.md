# Bugfix Report: Empty Add Title Corrupts Task File

**Date:** 2026-09-07
**Status:** Fixed
**Ticket:** T-1561

## Description of the Issue

`rune add` accepted an explicitly empty `--title` and wrote it verbatim as a task
bullet with no title text. The resulting line does not match the parser's grammar
for a task, so any subsequent command that reads the file fails.

A whitespace-only title is now rejected as well, but for consistency rather than
necessity: `- [ ] 1.    ` does parse, round-tripping as a task whose title is blank
spaces. Only the fully empty case corrupts the file. (The TaskList equivalent differs
— both `# ` and `#    ` fail to parse — which is why T-1500 treated the two together.)

**Reproduction steps:**

1. `rune create tasks.md --title "Tasks"`
2. `rune add tasks.md --title ""`
3. The command exits 0 and prints `Added task 1:`; the file now contains
   `- [ ] 1. ` (no title text after the numbering)
4. `rune list tasks.md` fails with `failed to read task file: line 2: invalid task
   format`

**Impact:** High — the command silently produces a file rune itself cannot read back.
Every command that parses the file afterwards fails, including read-only ones like
`list` and `find`.

## Investigation Summary

- **Symptoms examined:** A file with a task bullet containing no title text
  (`- [ ] 1. `), which `ParseMarkdown` rejects on the next read with "invalid task
  format".
- **Code inspected:** `cmd/add.go` (`runAdd`), `internal/task/operations.go`
  (`AddTask`, `AddTaskToPhase`, `AddTaskWithOptions`, `UpdateTask`,
  `UpdateTaskWithOptions`, `validateTaskInput`, `ValidateTaskListTitle`),
  `internal/task/batch.go` (`applyAddOperation`, `addTaskWithPhaseMarkers`),
  `internal/task/task.go` (`Task.Validate`), `internal/task/parse.go` (task-line
  grammar).
- **Hypotheses tested:** Whether `cmd/add.go` should validate the flag directly, the
  way `cmd/create.go` validates `--title` for `create` (T-1500) — ruled out, because
  unlike `create`, every `add` variant already funnels through
  `internal/task.validateTaskInput` inside `internal/task`, so the CLI layer is not
  the only, or even the primary, entry point for this input (batch JSON operations and
  any future direct caller of the task package hit the same gap). Whether
  `Task.Validate()` (which already rejects `t.Title == ""`) is invoked anywhere in the
  write path — confirmed it is not; it is currently dead code with no callers outside
  its own tests.

## Discovered Root Cause

`validateTaskInput`, the shared helper called by every operation that sets a task's
title (`AddTask`, `AddTaskToPhase`, `AddTaskWithOptions`, and the title branches of
`UpdateTask` / `UpdateTaskWithOptions`, including their batch equivalents), only checked
for null bytes/control characters and the `MaxTitleLength` limit. It did not check for
emptiness. `cmd/add.go` relies on Cobra's `MarkFlagRequired("title")`, which only
guarantees the flag was supplied on the command line — `--title ""` satisfies it.
Nothing downstream then caught the empty string before it was rendered into
`- [ ] {id}. {title}` and written to disk.

**Defect type:** Missing input validation

**Why it occurred:** T-1500 added the equivalent empty/whitespace check for the
`TaskList` title (`ValidateTaskListTitle`) but scoped it to that one entry point
because the reported bug was about `create`. `validateTaskInput`, the analogous
choke point for task titles, was left with the same gap it always had — the null-byte
and length checks were added for T-781 without ever including emptiness, and no ticket
had previously connected "empty task title" to "unparseable file".

## Resolution for the Issue

**Changes made:**

- `internal/task/operations.go:442` — `validateTaskInput` now rejects a title that is
  empty or only whitespace (`strings.TrimSpace(input) == ""`) before the existing
  null-byte/control-character and length checks, with the message `"task title cannot
  be empty"`.
- `internal/task/operations.go:466` — `ValidateTaskListTitle` no longer duplicates the
  empty check itself; it now delegates entirely to `validateTaskInput`, which performs
  the identical check. Its externally observable behaviour (error text still contains
  "cannot be empty", same check ordering) is unchanged.

**Approach rationale:** The empty check belongs in `validateTaskInput`, not in
`cmd/add.go`, because `validateTaskInput` — despite its generic name — is already used
exclusively for task titles and is the one function every title-setting code path
funnels through: `AddTask`, `AddTaskToPhase`, `AddTaskWithOptions`, the title branch of
`UpdateTask`/`UpdateTaskWithOptions`, and the batch operations `applyAddOperation` and
`addTaskWithPhaseMarkers`. A single change there closes the CLI `add` bug reported in
T-1561 and, as a side effect, the same gap in `add --phase`, `add` with stream/owner/
blocked-by options, `update --title`, and the batch JSON API — all of which shared the
exact same unchecked call before this fix, whether or not they were reported.

This mirrors, rather than duplicates, T-1500's fix: `create` validates at the CLI layer
because `NewTaskList` (its only production entry point besides tests) does not run any
shared validation and does not return an error. `add`'s entry points already run shared
validation and already return errors, so the fix belongs one layer down, at the shared
helper itself. Consolidating the check into `validateTaskInput` (and having
`ValidateTaskListTitle` delegate to it) also means the two title-validation rules
— task title and task-list title — cannot drift apart in the future.

**Alternatives considered:**

- **Validate `addTitle` in `cmd/add.go`, mirroring `runCreate`** — rejected: would only
  fix the direct CLI path and miss `AddTaskToPhase`, `AddTaskWithOptions`, and the batch
  JSON operations, which have their own call sites into `internal/task` and do not
  route through `cmd/add.go` at all (batch operations are invoked from `cmd/batch.go`
  and `internal/task/batch.go` directly).
- **Wire up the existing but unused `Task.Validate()`** — rejected: it only checks
  `t.Title == ""`, not whitespace-only, and would need to be called from every
  construction site (`AddTask`, `addTaskAtPosition`, `AddTaskWithOptions`, ...),
  duplicating exactly the reach `validateTaskInput` already has as a pre-construction
  guard. Calling it post-construction would also mean a half-built `Task` had already
  been appended to the list by the time validation ran, complicating rollback.
- **Duplicate the empty check in `ValidateTaskListTitle` and add a separate one in
  `validateTaskInput`** — rejected: this is what existed before this fix if you swap
  which function owns the "real" check. It leaves two independent copies of "what makes
  a title acceptable" that could diverge; the follow-on change to delegate
  `ValidateTaskListTitle` entirely to `validateTaskInput` avoids that.

## Regression Test

**Test files:** `internal/task/operations_test.go`, `cmd/add_test.go`
**Test names:** `TestAddTaskRejectsEmptyTitle`, `TestRunAddRejectsEmptyTitle`

**What they verify:** `TestAddTaskRejectsEmptyTitle` exercises `validateTaskInput`
through `AddTask`, `AddTaskWithOptions`, and `UpdateTaskWithOptions` directly (in
`internal/task`) with an empty title, a whitespace-only title, a title with a newline
(existing behaviour, unchanged), a plain title, and a title with a tab (both still
accepted) — confirming the new check fires for the write paths and that
`UpdateTaskWithOptions` correctly still treats the empty string as its
"leave unchanged" sentinel rather than an error. `TestRunAddRejectsEmptyTitle` drives
the actual CLI entry point (`runAdd`) with `--title ""` and `--title "   "` against an
existing task file and asserts: the command returns an error containing "cannot be
empty", the file on disk is byte-for-byte unchanged (no partial/corrupt write), and
`task.ParseFile` can still read it back — directly reproducing and closing the ticket's
repro (`add --title ""` followed by a failing `list`).

Both tests were confirmed to fail before the fix: reverting the `internal/task/
operations.go` change and re-running reproduces the exact reported bug in
`TestRunAddRejectsEmptyTitle` (`Added task 2: ` with no error) and the equivalent gap
in `TestAddTaskRejectsEmptyTitle` (`AddTask`/`AddTaskWithOptions` accept the empty and
whitespace-only titles with no error).

**Run command:**
`go test -run TestAddTaskRejectsEmptyTitle ./internal/task/ && go test -run TestRunAddRejectsEmptyTitle ./cmd/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/operations.go` | `validateTaskInput` now rejects an empty/whitespace-only title; `ValidateTaskListTitle` delegates to it instead of duplicating the check |
| `internal/task/operations_test.go` | Added `TestAddTaskRejectsEmptyTitle` |
| `cmd/add_test.go` | Added `TestRunAddRejectsEmptyTitle` |
| `CHANGELOG.md` | Documented the fix under Unreleased |

## Verification

**Automated:**

- [x] Regression tests pass
- [x] Full test suite passes (`make check`)
- [x] Integration tests pass (`INTEGRATION=1 go test -run TestIntegration ./cmd`)
- [x] Lint passes (`make lint`, part of `make check`)
- [x] Build succeeds (`go build ./...`)

**Manual verification:**

- Reran the exact repro from the ticket (`add --title ""` then `list --format json`)
  against a build with the fix: `add` now exits non-zero with `task title cannot be
  empty` and the file is left untouched, so the subsequent `list` succeeds.

## Prevention

**Recommendations to avoid similar bugs:**

- When a shared validation helper like `validateTaskInput` gains a new rule for one
  reported bug (T-1500's `ValidateTaskListTitle`), check whether an analogous shared
  helper for a sibling concept (task titles vs. task-list titles) has the same gap —
  the two were fixed a ticket apart even though the underlying defect (accepting an
  empty string that renders into unparseable markdown) was identical.
- `Task.Validate()` exists and encodes some of the same rules but is never called from
  any write path. Either wire it into task construction/mutation or remove it — an
  unused validator that looks authoritative is a trap for the next person investigating
  a similar bug.
- Pair every "rejects bad input" regression test with a check that the file was not
  partially written and still parses — `TestRunAddRejectsEmptyTitle` asserts both,
  which is what actually proves the "corrupts task file" half of this bug is closed,
  not just that an error is returned.

## Related

- T-1500 — `create` accepts newline/empty titles and writes unparseable files (added
  `ValidateTaskListTitle`, the sibling fix this ticket mirrors for task titles).
- T-781 — original embedded-newline validation that `validateTaskInput`'s
  control-character check descends from.
