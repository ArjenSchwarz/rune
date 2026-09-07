# Bugfix Report: Phase Names Allow Newline Injection

**Date:** 2026-09-07
**Status:** Fixed

## Description of the Issue

`add-phase`, `add --phase`, and batch phase operations only rejected phase
names that were empty (or whitespace-only) after trimming. A phase name
containing a newline was accepted, and since phase names are written
verbatim into a `## {name}` markdown header, the newline let the caller
inject arbitrary extra lines -- including fake task lines -- into the task
file.

**Reproduction steps:**
1. `cp examples/simple.md /tmp/phase-name.md`
2. `go run . add-phase /tmp/phase-name.md $'Bad\n- [ ] 999. Injected' --format json`
3. Observe the command exits 0 and the file gains two new lines:
   ```md
   ## Bad
   - [ ] 999. Injected
   ```

**Impact:** Structural corruption of the task markdown file -- a phase name
value could inject fabricated task lines (or other markdown) that did not
go through normal task-creation validation. Reachable from every path that
accepts a user-supplied phase name: `add-phase`, `add --phase`, the
exported `AddTaskToPhase`, and the two phase-aware batch execution paths.

## Investigation Summary

- **Symptoms examined:** running `add-phase` with a newline-containing name
  writes the extra content verbatim instead of erroring.
- **Code inspected:** `internal/task/validation.go` (the shared phase-name
  validator), its call sites (`cmd/add.go`, `cmd/add_phase.go`,
  `internal/task/operations.go`, `internal/task/batch.go` twice), and
  `internal/task/render.go` (`RenderMarkdownWithPhases`, which writes
  `"## %s\n\n"` with no escaping).
- **Hypotheses tested:** the first hypothesis -- that every entry point
  funnelled through the single shared validator, so a fix there would cover
  them all -- turned out to be only half true. The validator was shared, but
  the *trimming* was not: `cmd/add_phase.go` and the two batch sites trimmed
  before validating, while `cmd/add.go` and `AddTaskToPhase` validated the
  raw argument and then used it. Adding a control-character check to the
  validator alone therefore produced inconsistent behaviour rather than a
  uniform fix, and regressed a batch input (`{"phase": "Planning\n"}`) that
  had previously worked. The fix had to make normalisation and validation a
  single, shared operation.

## Discovered Root Cause

**Defect type:** Missing input validation, compounded by a split contract.

**Why it occurred:** the phase-name validator only checked
`strings.TrimSpace(name) == ""`. It never checked for newlines or other
control characters, even though the name is later interpolated directly
into a markdown heading line during rendering (`fmt.Fprintf(&buf, "## %s\n\n",
name)` in `RenderMarkdownWithPhases`, and an equivalent `fmt.Sprintf("## %s",
phaseName)` in `cmd/add_phase.go`). No layer between validation and
rendering ever escaped or rejected control characters.

**Contributing factors:** phase names are treated as plain strings
end-to-end with no dedicated "safe for a markdown heading" contract; the
empty-check was written to solve the empty-name case and was never revisited
when phase names started being written straight into headers. Because
trimming was left to each caller rather than being part of that contract,
the callers had already drifted apart before this bug was filed -- which is
why a validator-only fix could not close the hole consistently.

## Resolution for the Issue

**Changes made:**
- `internal/task/validation.go` - `ValidatePhaseName` replaced by
  `NormalizePhaseName(string) (string, error)`, which trims surrounding
  whitespace and validates the trimmed result in one step, returning the
  canonical name callers must use from that point on. A name is rejected if
  it is empty after trimming, or if the trimmed name still contains a
  control character. Making normalisation and validation a single operation
  is what stops the two from drifting apart again.
- `cmd/add.go` - normalises `--phase` above the `--dry-run` early return, so
  the dry-run preview errors the same way a real run does instead of
  printing the raw multi-line name, and uses the normalised name for the
  preview, for `AddTaskToPhase`, and in the JSON response.
- `cmd/add_phase.go` - its manual `strings.TrimSpace` plus validate pair
  replaced by the single `NormalizePhaseName` call.
- `internal/task/operations.go` - `AddTaskToPhase` normalises `phaseName`
  itself. It is an exported API that writes the name verbatim into a
  `## {name}` header, so a direct caller that skipped validation would
  reopen the hole; normalising here also means its phase-marker lookup
  compares the trimmed name and matches an existing header.
- `internal/task/batch.go` - all three phase sites (`validateOperation`'s
  `add-phase` case, the `ExecuteBatchWithPhases` pre-check loop, and
  `applyOperationWithPhases`) go through `NormalizePhaseName`, and
  `addTaskWithPhaseMarkers` normalises `op.Phase` before matching or
  creating a phase marker.

**Approach rationale:** the hole is at the boundary between "a caller-supplied
string" and "a markdown heading", so the fix belongs in one function that
owns that whole boundary. Returning the canonical name rather than only an
error is the load-bearing part: a validator that merely trimmed internally
would have let `NormalizePhaseName("\nBad")` pass while a caller went on to
use the untrimmed value, which is exactly the class of divergence that
caused the original inconsistency. Rejecting outright (rather than stripping
or escaping) keeps behaviour predictable and matches how the sibling
empty-name case is already handled -- an explicit error, not silent
correction -- which is the parser's existing philosophy (report errors,
don't auto-correct).

**Tab is deliberately allowed.** `unicode.IsControl` classifies tab as a
control character, but `containsNullByte` (used for task titles, details and
references) permits it, and the parser reads a `## Design<TAB>Phase` header
back without complaint. Rejecting tab would have left such a phase in a
broken half-state: it would still parse and list, but no path could add a
task to it, and no migration exists to rename it. Tab is also not an
injection vector -- only line terminators can start a new markdown line.
The rule is therefore "every control character except tab", which keeps
phase names consistent with the rest of the package and with the parser, at
the cost of one deliberate exception to `unicode.IsControl`.

**Alternatives considered:**
- **Fold `TrimSpace` into the existing `ValidatePhaseName`** - rejected: it
  would make `ValidatePhaseName("\nBad")` return nil while `cmd/add.go`
  still forwarded the untrimmed value to `AddTaskToPhase`, trading one
  divergence for a quieter one.
- **Escape/strip control characters instead of rejecting** - rejected
  because silently mangling user input is more surprising than an explicit
  error, and it would still require deciding what a "safe" replacement
  looks like.
- **Validate only for `\n`/`\r`** - rejected in favour of rejecting all
  control characters except tab, since the ticket calls out "newlines and
  other control characters," and other control bytes (e.g. NUL) have no
  meaning in a markdown heading.
- **Reject tab as well, for a strict `unicode.IsControl` rule** - rejected;
  see "Tab is deliberately allowed" above.
- **Normalise at each call site instead of in the shared function** -
  rejected because that is precisely how the original divergence arose.

## Regression Test

**Test files and names:**
- `internal/task/validation_test.go` - `TestNormalizePhaseName` (direct unit
  tests for the shared normaliser: trimming, tab and trailing-newline
  acceptance, and newline/CR/CRLF/NUL/DEL rejection)
- `internal/task/batch_operations_test.go` -
  `TestExecuteBatchWithPhases_AddOperationPhaseNameGuard` (the pre-check loop
  in `ExecuteBatchWithPhases` is the *only* guard for an `add` operation
  carrying a phase, since `validateOperation`'s `add` case never inspects
  `op.Phase`; verified to fail when that call is deleted) plus new cases in
  `TestValidateOperation_AddPhase` for the batch `add-phase` path
- `cmd/add_test.go` - `"phase name with newline is rejected"` and
  `"phase name with trailing newline is trimmed"` in `TestRunAddWithPhase`,
  and `TestRunAddWithPhaseDryRunRejectsNewline`
- `cmd/add_phase_test.go` - `TestRunAddPhaseRejectsNewline` (reproduces the
  exact ticket scenario end-to-end) and
  `TestRunAddPhaseTrimsTrailingNewline`
- `internal/task/phase_operations_test.go` -
  `TestAddTaskToPhaseNormalizesPhaseName` (the exported `AddTaskToPhase`
  rejects control-character names on its own and writes the normalised name)

**What it verifies:** a phase name with an embedded control character is
rejected with an error on every path, the file is left unmodified, and no
injected content appears in the output; and a name whose only problem is
surrounding whitespace is trimmed and accepted on every path, matching the
existing phase header rather than creating a duplicate.

**Run command:**
```sh
go test ./internal/task/... -run 'TestNormalizePhaseName|TestValidateOperation_AddPhase|TestExecuteBatchWithPhases_AddOperationPhaseNameGuard|TestAddTaskToPhaseNormalizesPhaseName'
go test ./cmd/... -run 'TestRunAddPhaseRejectsNewline|TestRunAddPhaseTrimsTrailingNewline|TestRunAddWithPhase'
```

## Affected Files

| File | Change |
|------|--------|
| `internal/task/validation.go` | `ValidatePhaseName` replaced by `NormalizePhaseName`, which trims and validates in one step and returns the canonical name; rejects control characters (tab excepted) as well as empty/whitespace names. |
| `internal/task/validation_test.go` | Direct unit tests for `NormalizePhaseName`, covering trimming, tab, and each rejected control character. |
| `internal/task/operations.go` | `AddTaskToPhase` normalises `phaseName` itself, so the exported API is safe and its marker lookup compares the canonical name. |
| `internal/task/batch.go` | All three phase sites and `addTaskWithPhaseMarkers` normalise through `NormalizePhaseName`. |
| `internal/task/batch_operations_test.go` | New `TestExecuteBatchWithPhases_AddOperationPhaseNameGuard` covering the sole guard on the batch `add`-with-phase path; new trailing-newline and tab cases in `TestValidateOperation_AddPhase`. |
| `internal/task/phase_operations_test.go` | `TestAddTaskToPhaseNormalizesPhaseName` asserts both rejection and the normalised header that gets written. |
| `cmd/add.go` | `--phase` normalised ahead of the `--dry-run` early return; the normalised name is used for the preview, `AddTaskToPhase`, and the JSON response. |
| `cmd/add_test.go` | Regression cases for `add --phase` (rejection and trimming) and a dry-run case; also fixed a test-isolation gap where an `expectError` case didn't reset shared command flags. |
| `cmd/add_phase_test.go` | End-to-end regression tests for `add-phase`: rejection of an embedded newline and trimming of a trailing one. |
| `CHANGELOG.md` | `[Unreleased]` entry for the user-visible behaviour change. |

## Verification

**Automated:**
- [x] Regression tests pass on every phase entry point
- [x] `TestExecuteBatchWithPhases_AddOperationPhaseNameGuard` confirmed to
      fail when the `ExecuteBatchWithPhases` pre-check is removed
- [x] Full test suite passes (`make check`)
- [x] Integration suite passes (`make test-integration`)
- [x] Linters/validators pass (`golangci-lint`: 0 issues)

**Manual verification:** against binaries built from `main` and from this
branch.
- The ticket repro, `add-phase <file> $'Bad\n- [ ] 999. Injected'`, exits
  non-zero with `Error: phase name cannot contain control characters (e.g.
  newlines)` and leaves the file unchanged. The same holds for
  `add --phase` (with and without `--dry-run`) and for both batch shapes,
  `{"type":"add-phase","phase":...}` and `{"type":"add","phase":...}`.
- A phase name of `"Planning\n"` is trimmed and accepted on all five paths,
  as it was on `main`. On `main` the batch `add` form of it silently created
  a *second* `## Planning` header because the marker lookup compared the raw
  name; it now lands in the existing phase.
- A pre-existing `## Design<TAB>Phase` phase still parses, lists, and
  accepts new tasks.

## Prevention

**Recommendations to avoid similar bugs:**
- Any value that gets interpolated directly into markdown structure (not
  just task titles) should be validated for control characters at the
  point of input, not assumed safe because it passed an "is it empty"
  check.
- When a value needs both normalisation and validation, make them one
  function that returns the normalised value. A separate "trim it yourself,
  then call the validator" contract will drift; this bug is what that drift
  looks like once it reaches a security boundary.
- When adding a new field that flows into `render.go`, check whether it
  needs the same control-character guard already applied to titles/phase
  names.

## Related

- Transit ticket: T-1603
- Related tickets (different scope, fixed in parallel): T-1500 (newline
  titles in the `create` command), T-1561 (empty `add` titles)
