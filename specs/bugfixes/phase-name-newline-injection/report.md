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
go through normal task-creation validation. Reachable from any of the four
call sites that accept a user-supplied phase name (`add-phase`, `add
--phase`, and the two phase-aware batch execution paths).

## Investigation Summary

- **Symptoms examined:** running `add-phase` with a newline-containing name
  writes the extra content verbatim instead of erroring.
- **Code inspected:** `internal/task/validation.go` (`ValidatePhaseName`),
  its four call sites (`cmd/add.go`, `cmd/add_phase.go`,
  `internal/task/batch.go` twice), and `internal/task/render.go`
  (`RenderMarkdownWithPhases`, which writes `"## %s\n\n"` with no escaping).
- **Hypotheses tested:** confirmed all four call sites route through the
  single shared `ValidatePhaseName` function, so a fix there covers every
  entry point without touching call-site code.

## Discovered Root Cause

**Defect type:** Missing input validation.

**Why it occurred:** `ValidatePhaseName` only checked
`strings.TrimSpace(name) == ""`. It never checked for newlines or other
control characters, even though the name is later interpolated directly
into a markdown heading line during rendering (`fmt.Fprintf(&buf, "## %s\n\n",
name)` in `RenderMarkdownWithPhases`, and an equivalent `fmt.Sprintf("## %s",
phaseName)` in `cmd/add_phase.go`). No layer between validation and
rendering ever escaped or rejected control characters.

**Contributing factors:** phase names are treated as plain strings
end-to-end with no dedicated "safe for a markdown heading" contract; the
empty-check was written to solve the empty-name case and was never revisited
when phase names started being written straight into headers.

## Resolution for the Issue

**Changes made:**
- `internal/task/validation.go` - `ValidatePhaseName` now also rejects any
  name containing a control character (via `unicode.IsControl`), which
  covers `\n`, `\r`, `\t`, and other C0/C1 control codes, in addition to the
  existing empty/whitespace-only check.

**Approach rationale:** All four call sites already funnel through this one
function, so fixing it there closes the hole everywhere at once with a
minimal, localized change. Rejecting outright (rather than stripping or
escaping) keeps behaviour predictable and matches how the sibling
empty-name case is already handled -- an explicit error, not silent
correction -- which is the parser's existing philosophy (report errors,
don't auto-correct).

**Alternatives considered:**
- **Escape/strip control characters instead of rejecting** - rejected
  because silently mangling user input is more surprising than an explicit
  error, and it would still require deciding what a "safe" replacement
  looks like.
- **Validate only for `\n`/`\r`** - rejected in favor of rejecting all
  control characters, since the ticket calls out "newlines and other
  control characters," and other control bytes (e.g. NUL) are equally
  invalid in a markdown heading.
- **Add validation at each call site instead of the shared function** -
  rejected because it would duplicate logic across four places for no
  benefit; the shared function is the correct choke point.

## Regression Test

**Test files and names:**
- `internal/task/validation_test.go` - `TestValidatePhaseName` (direct unit
  tests for the shared validator, including newline/CR/CRLF/tab/NUL cases)
- `internal/task/batch_operations_test.go` - new case
  `"phase name with newline injection"` in `TestValidateOperation_AddPhase`
  (batch operation path)
- `cmd/add_test.go` - new case `"phase name with newline is rejected"` in
  `TestRunAddWithPhase` (`add --phase` CLI path)
- `cmd/add_phase_test.go` - `TestRunAddPhaseRejectsNewline` (`add-phase` CLI
  path, reproduces the exact ticket scenario end-to-end)

**What it verifies:** a phase name containing a newline (or other control
character) is rejected with an error, the file is left unmodified, and no
injected content appears in the output.

**Run command:**
```sh
go test ./internal/task/... -run 'TestValidatePhaseName|TestValidateOperation_AddPhase'
go test ./cmd/... -run 'TestRunAddPhaseRejectsNewline|TestRunAddWithPhase'
```

## Affected Files

| File | Change |
|------|--------|
| `internal/task/validation.go` | `ValidatePhaseName` now rejects control characters (newlines included), not just empty/whitespace names. |
| `internal/task/validation_test.go` | New direct unit tests for `ValidatePhaseName`. |
| `internal/task/batch_operations_test.go` | New regression case for batch `add-phase` operation validation. |
| `cmd/add_test.go` | New regression case for `add --phase`; also fixed a test-isolation gap where an `expectError` case didn't reset shared command flags, which the new case exposed as pollution into other tests in the same package. |
| `cmd/add_phase_test.go` | New end-to-end regression test for the `add-phase` command. |

## Verification

**Automated:**
- [x] Regression tests pass (all four, covering every call site)
- [x] Full test suite passes (`make check`)
- [x] Linters/validators pass (`golangci-lint`: 0 issues)

**Manual verification:**
- Reproduced the exact ticket repro command against the fixed binary:
  `go run . add-phase <file> $'Bad\n- [ ] 999. Injected' --format json` now
  exits non-zero with `Error: phase name cannot contain control characters
  (e.g. newlines)`, and the target file is unchanged.

## Prevention

**Recommendations to avoid similar bugs:**
- Any value that gets interpolated directly into markdown structure (not
  just task titles) should be validated for control characters at the
  point of input, not assumed safe because it passed an "is it empty"
  check.
- When adding a new field that flows into `render.go`, check whether it
  needs the same control-character guard already applied to titles/phase
  names.

## Related

- Transit ticket: T-1603
- Related tickets (different scope, fixed in parallel): T-1500 (newline
  titles in the `create` command), T-1561 (empty `add` titles)
