# Bugfix Report: Comma-Only Blocked-By Input Silently Clears Dependencies

**Date:** 2026-09-07
**Status:** Fixed
**Ticket:** T-1906

## Description of the Issue

`rune update --blocked-by` accepted a comma-only (or comma-plus-whitespace-only)
value as a successful dependency update and silently cleared the task's existing
`Blocked-by` list, with no error and exit code `0`.

**Reproduction steps:**

1. `rune create tasks.md --title "Tasks"`
2. `rune add tasks.md --title "Blocker"` (task 1)
3. `rune add tasks.md --title "Task" --blocked-by 1` (task 2, blocked by task 1)
4. `rune update tasks.md 2 --blocked-by ',' --format json`
5. The command exits `0`, reports `"fields_updated": ["blocked-by"]`, and the
   `Blocked-by:` line for task 2 is removed from the file entirely.

**Impact:** High -- a single stray comma (a plausible typo, e.g. from a shell
variable expanding empty, or copy-pasting a trailing separator) silently destroys
real dependency data with no warning. The same shape (`" , "`, `",,,"`, etc.)
reproduces identically.

## Investigation Summary

- **Symptoms examined:** `update --blocked-by ','` reports success and the
  target task's `Blocked-by:` markdown line disappears from the file.
- **Code inspected:** `cmd/update.go` (`runUpdate`), `cmd/add.go`
  (`parseRequirementIDs`, shared by `add --blocked-by`, `update --blocked-by`,
  and `update --requirements`), `internal/task/operations.go`
  (`UpdateTaskWithOptions`, `UpdateOptions.BlockedBy`).
- **Hypotheses tested:**
  - Whether `internal/task.UpdateTaskWithOptions`'s empty-slice-means-clear
    sentinel is itself the defect -- ruled out as the thing to change. The
    sentinel is legitimate and, per the field's own doc comment
    (`BlockedBy []string // nil = no change, empty = clear`), it is the only
    mechanism `update` has for expressing "clear this task's dependencies"
    (see T-1493, which tracks the CLI never actually reaching that branch
    intentionally). The defect is upstream: nothing stops a malformed,
    non-empty raw flag value from *accidentally* producing the same non-nil
    empty slice that means "clear".
  - Whether `--blocked-by ""` (a literal empty string) can already reach the
    clear sentinel today, which would make comma-only input merely a second,
    redundant path to the same feature -- ruled out. `runUpdate` gates the
    entire blocked-by branch behind `if updateBlockedBy != ""`, so an
    explicitly empty flag value is indistinguishable from the flag not being
    passed at all and never reaches `opts.BlockedBy`. That gap is T-1493's
    concern, not this one. This confirms the clear-sentinel branch in
    `UpdateTaskWithOptions` is, in the current CLI, reachable *only* through
    malformed comma-only input -- i.e., through this bug -- which is why fixing
    it at the parsing layer cannot regress a working "explicit clear" feature
    that doesn't yet exist.

## Discovered Root Cause

`parseRequirementIDs` (`cmd/add.go`) splits its input on `,`, trims each part,
and drops empty tokens, returning `ids := make([]string, 0)` -- a **non-nil**
empty slice -- when every token is empty. `runUpdate` passes this directly to
`opts.BlockedBy`. `internal/task.UpdateTaskWithOptions` distinguishes "no
change" (`nil`) from "clear" (non-nil, `len == 0`) purely by nil-ness, so a
comma-only string reaches the exact same branch as an intentional clear
request and wipes `task.BlockedBy`.

**Defect type:** Missing input validation at a CLI parsing choke point;
ambiguous sentinel reached by accident.

**Why it occurred:** `parseRequirementIDs` was written as a permissive
best-effort splitter for the `--requirements`/`--blocked-by` flags -- dropping
empty tokens is reasonable for `"1,,2"` (stray double comma around real IDs)
but the same leniency silently degrades `","` (no real IDs at all) into the
same "empty list" representation used for the clear sentinel one layer down.
No caller of `parseRequirementIDs` checked whether the *input* was non-empty
but the *output* was empty, which is the only signal that distinguishes
"malformed" from "user is not touching this field."

**Contributing factors:** The clear sentinel in `UpdateTaskWithOptions` and
the lenient parser were written independently (a CLI-layer helper vs. an
internal-package invariant) without a shared contract for what an
empty-but-non-nil slice from user input is supposed to mean.

## Resolution for the Issue

**Changes made:**

- `cmd/update.go` -- added a check, evaluated before the dry-run branch and
  before any file write, that runs whenever `--blocked-by` is given a
  non-empty raw value: if `parseRequirementIDs` reduces it to zero task IDs,
  `runUpdate` now returns `fmt.Errorf("invalid --blocked-by value %q: no task
  IDs found", updateBlockedBy)` instead of proceeding. The parsed result is
  cached in a local (`newBlockedBy`) and reused later when building
  `opts.BlockedBy`, so the string is parsed once.

**Approach rationale:** Chose to make comma-only input an **error**, not a
silent no-op ("leave dependencies unchanged"), because that is what the
ticket's own expected behaviour specifies and it is the more defensible
default: the input is genuinely malformed (the user supplied *something*
that parsed to nothing), and an error surfaces the typo immediately rather
than having the command silently do nothing while still leaving ambiguity
around `fields_updated`/exit-code semantics for a no-op case. An error is
also cheap to keep compatible with T-1493: it does not attempt to add or
repurpose the clear-sentinel plumbing, it just prevents malformed input from
reaching it by accident. When T-1493 later adds a real "clear" mechanism, it
can do so however it likes; this fix does not consume or block that flag/UX
design (comma-only will keep failing under this fix regardless of how
"explicit clear" is eventually spelled).

The fix is scoped to `cmd/update.go` only. `parseRequirementIDs` itself was
left unchanged because it is shared with `update --requirements` and
`add --blocked-by`:

- `update --requirements ","` has the identical comma-only shape but is
  tracked separately as T-2023 and is intentionally not touched here.
- `add --blocked-by ","` produces an empty-but-non-nil `BlockedBy` on a
  *new* task, which has no prior dependency data to destroy -- it is a milder
  "silently ignored typo" issue, not the "silently destroys existing data"
  defect this ticket reports, and is out of scope for this fix.

**Alternatives considered:**

- **Treat comma-only input as "leave dependencies unchanged"** -- rejected.
  This would require distinguishing three states (no flag / malformed flag /
  intentional clear) using only a `string != ""` check plus parse output,
  which is exactly the ambiguity that caused the bug. It would also silently
  swallow a likely typo instead of reporting it, and the ticket's expected
  behaviour explicitly calls for rejection.
- **Fix `parseRequirementIDs` itself to return `nil` instead of an empty
  slice when every token is dropped** -- rejected. This would also silently
  fix (or change) `update --requirements` and `add --blocked-by` behaviour as
  a side effect of a shared helper, contradicting the instruction to leave
  T-2023's requirements-flag bug for its own ticket, and would fold "no
  change" and "malformed" into the same `nil` result for requirements too
  (since `newRequirements` from `parseRequirementIDs` also feeds
  `opts.Requirements`, which has its own nil-vs-empty contract).
- **Add the check inside `internal/task.UpdateTaskWithOptions`** -- rejected.
  By the time `opts.BlockedBy` reaches that function it is already a plain
  `[]string`; the function has no way to tell "caller parsed `\",\"`" apart
  from "caller explicitly wants to clear" without changing the `UpdateOptions`
  contract (e.g. a separate bool or a sentinel type), which is a larger,
  cross-cutting change better suited to T-1493's "add a real clear mechanism"
  work than to this bug fix.

## Regression Test

**Test file:** `cmd/update_test.go`
**Test name:** `TestRunUpdateBlockedByCommaOnlyRejected`

**What it verifies:** Table-driven over three malformed inputs -- `","`,
`" , "`, and `",,,,"` -- against a task file where task 2 already has task 1 as
a real dependency. For each case it asserts: `runUpdate` returns a non-nil
error mentioning `blocked-by`; the file on disk is byte-for-byte identical
before and after the call (`bytes.Equal`); and re-parsing the file afterwards
confirms task 2 still has exactly one `BlockedBy` entry, i.e. the dependency
was not touched. Both the byte-for-byte and semantic checks are included
because a test that only asserts "an error was returned" would still pass if
the fix accidentally moved the error to *after* a partial write.

Confirmed to fail before the fix: reverting the `cmd/update.go` change and
rerunning shows all three subtests failing with "Expected error for
comma-only --blocked-by, got none" (the command instead reports
`Updated task 2 (blocked-by): ...` and clears the dependency, reproducing the
ticket exactly). Restoring the fix makes all three pass.

**Run command:** `go test -run TestRunUpdateBlockedByCommaOnlyRejected ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/update.go` | `runUpdate` now rejects a `--blocked-by` value that parses to zero task IDs, before any file write |
| `cmd/update_test.go` | Added `TestRunUpdateBlockedByCommaOnlyRejected` |
| `CHANGELOG.md` | Documented the fix under Unreleased |

## Verification

**Automated:**

- [x] Regression test passes
- [x] Full test suite passes (`make check`)
- [x] Integration tests pass (`INTEGRATION=1 go test -run TestIntegration ./cmd`)
- [x] Lint passes (`make lint`, part of `make check`)

**Manual verification:**

- Reran the exact ticket repro (`update tasks.md 2 --blocked-by ',' --format
  json`) against a build with the fix: the command now exits non-zero with
  `invalid --blocked-by value ",": no task IDs found` and the file -- including
  task 2's `Blocked-by:` line -- is left unchanged.
- Confirmed `update --blocked-by "1"` (valid single ID) and
  `update --blocked-by "1,2"` (valid multiple IDs) still succeed, matching
  existing `TestRunUpdateWithBlockedBy` coverage.

## Prevention

**Recommendations to avoid similar bugs:**

- When a lenient parser (drops empty/whitespace tokens) feeds a value into a
  downstream nil-vs-empty sentinel, check the *input*, not just the *parsed
  output*, before treating an empty result as intentional. The nil/empty
  distinction alone cannot tell "user asked for empty" apart from "user's
  non-empty input parsed to nothing."
- `parseRequirementIDs` is shared by three call sites (`add --blocked-by`,
  `update --blocked-by`, `update --requirements`) with three different
  "empty result" contracts (new task with no blockers / clear existing
  blockers / clear existing requirements). T-2023 should consider whether
  the requirements call sites need the same non-empty-input-to-empty-output
  guard added here, since the two bugs share a root cause.

## Related

- T-1493 -- `update` has no way to explicitly and intentionally clear
  `blocked-by`; the CLI-level gate (`updateBlockedBy != ""`) means the clear
  sentinel in `UpdateTaskWithOptions` is currently unreachable through valid
  input. This fix does not add or block that capability -- it only prevents
  malformed input from reaching the sentinel by accident.
- T-2023 -- the identical comma-only-input shape for the `--requirements`
  flag. Intentionally not fixed here; `parseRequirementIDs` was left
  unchanged so as not to alter requirements behaviour as a side effect.
- T-1565 -- concurrent fix to `cmd/next.go` and the owner-validation path;
  unrelated to and non-overlapping with this change (`cmd/update.go`).
