# Bugfix Report: next --claim Accepts Invalid Owner Strings

**Date:** 2026-09-07
**Status:** Fixed

## Description of the Issue

`rune next --claim` wrote the `--claim` value directly into a task's `Owner`
metadata with no validation. `add --owner` and `update --owner` both
validate owner strings (rejecting newlines and other control characters)
before writing them, but `next --claim` bypassed that check entirely,
mutating the in-memory task and writing the file straight away.

**Reproduction steps:**
1. `printf '# Claim Owner Test\n\n- [ ] 1. Ready\n' > /tmp/claim-owner-validation.md`
2. `go run . next /tmp/claim-owner-validation.md --claim $'agent\nbad'`
3. Observe the command exits 0 and the file gains a corrupted `Owner:` line:
   ```md
   - [-] 1. Ready
     - Owner: agent
   bad
   ```
4. `go run . list /tmp/claim-owner-validation.md --format json` then fails
   with `line 4: unexpected content at this indentation level`.

**Impact:** Structural corruption of the task markdown file -- a claim value
could inject a bare, unparseable line into the file, breaking every
subsequent read of that file (`list`, `find`, `next`, further `update`
calls) until it was hand-edited back into shape. Reachable from any
`next --claim AGENT_ID` invocation, including `--stream`, `--phase`, `--one`
and their combinations, since all of them funnel through the single
`runNextWithClaim` mutation loop.

## Investigation Summary

- **Symptoms examined:** `next --claim $'agent\nbad'` exits 0, writes an
  unparseable file, and a subsequent `list --format json` fails to parse.
- **Code inspected:** `cmd/next.go` (`runNextWithClaim`), the owner
  validation already used by `add`/`update`
  (`internal/task/operations.go`: `validateOwner`, its call sites in
  `AddTaskWithOptions` and `UpdateTaskWithOptions`), and the recently added
  `internal/task/validation.go` (`NormalizePhaseName`), which solved the
  analogous newline-injection problem for phase names (T-1603).
- **Hypotheses tested:** confirmed `validateOwner` already exists and is
  exercised by `add --owner`/`update --owner` via `AddTaskWithOptions`
  and `UpdateTaskWithOptions`, so the defect isn't a missing rule -- it's
  that `cmd/next.go` assigns `taskPtr.Owner = claimFlag` directly (line 271
  before this fix) without going through either of those option structs or
  calling any validator. Since `validateOwner` is unexported, `cmd/next.go`
  had no way to reuse it without an exported entry point.

## Discovered Root Cause

**Defect type:** Missing input validation on a code path that bypasses the
existing validated API.

**Why it occurred:** `next --claim` was implemented as a direct field
mutation (`taskPtr.Status = task.InProgress; taskPtr.Owner = claimFlag`)
rather than being routed through `UpdateTaskWithOptions`, which is the
choke point that already validates owner strings for `update --owner`.
`AddTaskWithOptions`/`UpdateTaskWithOptions` validate internally via the
unexported `validateOwner`, but nothing exported that same rule for a
caller, like `next --claim`, that needed to validate without going through
the full option-struct update path (claiming also sets `Status`, which
`UpdateTaskWithOptions` does not handle).

**Contributing factors:** owner assignment has three independent write
sites in the codebase (`AddTaskWithOptions`, `UpdateTaskWithOptions`, and
this direct assignment in `next --claim`), and only the first two were
built after `validateOwner` existed and wired through it. The claim path
was never revisited against that convention.

## Resolution for the Issue

**Changes made:**
- `internal/task/operations.go` -- added `ValidateOwner`, an exported
  one-line wrapper around the existing unexported `validateOwner`. This
  follows the same delegation pattern already used by
  `ValidateTaskListTitle` (which wraps `validateTaskInput`), so a `cmd`
  package entry point that needs the exact validation rules `add`/`update`
  already apply can call it without duplicating the character-class logic
  or reimplementing it with different semantics.
- `cmd/next.go` -- `runNextWithClaim` now calls `task.ValidateOwner(claimFlag)`
  as its first step, before parsing the task file or claiming anything, and
  returns a wrapped error (`invalid claim value: %w`) if it fails. Because
  this runs before any mutation and before the file is even parsed, the
  task file is left completely untouched on invalid input, and the check
  applies uniformly whether or not `--dry-run` is set.

**Approach rationale:** reuse the exact validation rules already applied to
`add --owner`/`update --owner`, rather than inventing a new rule set for
claim values. The ticket's expected behaviour explicitly calls for "the
same owner rules as other owner-setting paths" -- introducing a second,
slightly different set of rules (e.g. one that also trims whitespace, the
way `NormalizePhaseName` does for phase names) would create exactly the
kind of drift between entry points that a previous bug (T-1603, phase name
newline injection) was caused by and fixed by *unifying*. Exporting the
existing function is the smallest change that closes the gap without
touching `AddTaskWithOptions`, `UpdateTaskWithOptions`, or `batch.go`
(all of which already validate correctly and are out of scope for this fix).

**Alternatives considered:**
- **Route `next --claim` through `UpdateTaskWithOptions`** -- rejected:
  `UpdateOptions` has no `Status` field (status changes go through
  `UpdateStatus`), so claiming would need either a new field on
  `UpdateOptions` or two separate calls per task. Either adds surface area
  to a widely-used shared type/method for a single caller's convenience,
  when the actual gap is just "validate before writing," not "route
  through the full options struct."
- **Add a `NormalizeOwner(string) (string, error)` that also trims
  surrounding whitespace, mirroring `NormalizePhaseName`** -- rejected:
  `validateOwner` deliberately allows internal spaces (e.g. `"My Agent"`)
  and neither `add --owner` nor `update --owner` trim the value today.
  Trimming only on the claim path would make the same owner string behave
  differently depending on which command set it, reintroducing the
  divergence the phase-name fix was written to eliminate.
- **Duplicate the character-class check directly in `cmd/next.go`** --
  rejected: two copies of "which control characters are allowed in an
  owner" is exactly the kind of drift-prone duplication the codebase has
  already paid for once (phase names had this problem before
  `NormalizePhaseName`).

## Regression Test

**Test file:** `cmd/next_test.go`
**Test name:** `TestNextCommandClaimRejectsInvalidOwner`

**What it verifies:** a map-based table of claim values --
newline, carriage return, tab, and NUL byte (all `wantErr: true`), plus a
sanity case with internal spaces (`wantErr: false`) to confirm the fix
doesn't over-restrict. For each invalid case it asserts: `next --claim`
returns a non-nil error, the task file's bytes are byte-for-byte identical
before and after the (rejected) claim attempt, and the file is still
parseable via `task.ParseFile` afterwards -- the last check is the direct
regression for the ticket's observed failure (`list --format json` erroring
on the corrupted file).

Verified failing before the fix: with `cmd/next.go` and
`internal/task/operations.go` reverted to their pre-fix state (via
`git stash`, keeping only the new test), all four invalid-owner subtests
failed with "expected an error ... got nil", each showing the corrupted
owner value already written into the JSON output -- the same failure mode
as `main`. Restoring the fix turned all five subtests green with no
changes to the test itself.

**Run command:**
```sh
go test -run TestNextCommandClaimRejectsInvalidOwner -v ./cmd
```

## Affected Files

| File | Change |
|------|--------|
| `internal/task/operations.go` | Added exported `ValidateOwner`, delegating to the existing unexported `validateOwner`, so entry points outside the option-struct APIs can reuse the same rules. |
| `cmd/next.go` | `runNextWithClaim` validates `claimFlag` via `task.ValidateOwner` before parsing the file or mutating anything, returning an error and leaving the file untouched on invalid input. |
| `cmd/next_test.go` | New `TestNextCommandClaimRejectsInvalidOwner` regression test. |
| `CHANGELOG.md` | `[Unreleased]` entry describing the fix. |

## Verification

**Automated:**
- [x] Regression test passes (`TestNextCommandClaimRejectsInvalidOwner`)
- [x] Confirmed to fail before the fix (reverted `cmd/next.go` and
      `internal/task/operations.go` via `git stash`, re-ran the test, all
      four invalid-owner subtests failed; restored the fix and re-ran, all
      five subtests passed)
- [x] Full test suite passes (`make check`: fmt, `golangci-lint` -- 0
      issues, `go test ./...`)
- [x] Integration suite passes (`INTEGRATION=1 go test -run TestIntegration ./cmd`)

**Manual verification:** built and ran against the exact ticket repro.
`next /tmp/claim-owner-validation.md --claim $'agent\nbad'` now exits 1
with `Error: invalid claim value: owner contains invalid characters`; the
task file is unchanged (`- [ ] 1. Ready`, no `Owner:` line, no stray `bad`
line); a subsequent `list --format json` against the same file succeeds
and returns the single pending task.

## Prevention

**Recommendations to avoid similar bugs:**
- Any command that mutates a field also validated on another command's
  path (owner, stream, blocked-by, title, phase name, ...) should call the
  same validator, not reimplement or skip it. When a validator is
  unexported because its only callers were previously all in-package,
  export it (or add a thin exported wrapper) the moment a `cmd`-level
  caller needs the same rule, rather than duplicating or omitting the
  check.
- When adding a new command or flag that writes into existing task
  metadata, check for a `Validate*`/`validate*` function for that field
  before writing the assignment, the way `add --owner` and
  `update --owner` already do.

## Related

- Transit ticket: T-1565
- Related pattern (not the same fix, but the source of the "normalize vs.
  validate" reasoning above): T-1603, phase name newline injection, which
  introduced `NormalizePhaseName` in `internal/task/validation.go`.
