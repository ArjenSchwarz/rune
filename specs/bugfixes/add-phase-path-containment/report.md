# Bugfix Report: add-phase Path Containment Bypass

**Date:** 2026-09-07
**Status:** Fixed

## Description of the Issue

`rune add-phase` read and rewrote the target task file directly with `os.ReadFile`/`os.WriteFile` and never called `task.ValidateFilePath`. Every other mutating command enforces working-directory containment, either by calling the validator directly (`renumber`) or by persisting through `TaskList.WriteFile`, which validates internally. `add-phase` did neither, so it would happily append a phase header to any markdown file the user could reach on disk.

**Reproduction steps:**
1. From any working directory, create a markdown task file somewhere outside it (e.g. `/tmp/outside/tasks.md`)
2. Run `rune add-phase /tmp/outside/tasks.md "Escaped"` (a `../` relative path works equally well)
3. Observe: the command succeeds and the external file gains a `## Escaped` header

**Impact:** Security — the containment guarantee the rest of the CLI provides did not hold for `add-phase`, so it could modify files outside the project root. The write is append-only and limited to a validated phase-name header, so it cannot corrupt arbitrary file content, but it can still modify files the user never targeted.

## Investigation Summary

- **Symptoms examined:** `add-phase` accepted absolute and `../`-relative paths that other commands reject
- **Code inspected:** `cmd/add_phase.go:runAddPhase`, `cmd/renumber.go` (the direct-call pattern), `internal/task/operations.go:ValidateFilePath`, `internal/task/operations.go:WriteFile` / `WriteFileWithPhases`
- **Hypotheses tested:** Grepped every `ValidateFilePath` call site; `add-phase` was absent from the list, confirming the check was missing rather than being bypassed by a faulty condition

## Discovered Root Cause

`runAddPhase` implements its own read/append/write cycle instead of going through the `task` package's file operations. Because the containment check lives inside `TaskList.WriteFile` and `WriteFileWithPhases`, a command that writes with plain `os.WriteFile` silently opts out of it. Nothing in the code or tests flagged the omission.

**Defect type:** Missing validation (omitted call)

**Why it occurred:** The validation is implicit for most commands — it comes free with the write helper. `add-phase` is the only mutating command that does not use that helper, so it was also the only one that had to call the validator explicitly, and that requirement was not obvious from the surrounding code.

## Resolution for the Issue

**Changes made:**
- `cmd/add_phase.go:runAddPhase` — Call `task.ValidateFilePath(filename)` immediately after the filename is resolved (including via git discovery) and before any filesystem access, wrapping failures as `invalid file path: %w`. A comment records why this command needs the explicit call while the others do not.
- `internal/task/batch.go` — Two comments referenced `cmd/add_phase.go:59` for the phase-name trimming behaviour they mirror. The new check shifted that line, so the pointers now name `runAddPhase` instead of a line number.

**Approach rationale:** Validating before the `os.Stat` existence check means a contained-path violation is reported as such rather than leaking whether the out-of-tree file exists, and it guarantees no filesystem access happens on a rejected path. Calling the shared `task.ValidateFilePath` keeps `add-phase` on exactly the same rules — lexical containment plus symlink resolution — as every other command, with no second implementation to drift.

**Alternatives considered:**
- Rewriting `runAddPhase` to parse the file and persist through `WriteFileWithPhases` — would fix this by construction, but `add-phase` deliberately appends raw bytes so it does not reformat or renumber the rest of the file. Switching to the parse/render path is a behaviour change well beyond a containment fix.
- Moving the check into a shared pre-run hook for all commands — broader blast radius than the bug warrants, and read-only commands have different path rules.

## Regression Test

**Test file:** `cmd/add_phase_test.go`
**Test names:** `TestAddPhaseCommandRejectsPathOutsideWorkingDirectory`, `TestAddPhaseCommandAcceptsPathInsideWorkingDirectory`

**What it verifies:** The negative test runs `add-phase` from an isolated temp working directory against a file in a different temp directory, and asserts both that the command returns a path containment error (checking the `invalid file path` wrap prefix *and* the `path traversal` reason, so an unrelated failure cannot satisfy it) and that the outside file is byte-for-byte unchanged. The positive test is its counterpart: a plain relative path inside the working directory is still accepted and the phase header is appended as expected, so the new check cannot regress into rejecting valid input without `make test` catching it.

**Run command:** `go test -run 'TestAddPhaseCommand(Rejects|Accepts)Path' -v ./cmd/`

Verified red/green: without the `ValidateFilePath` call the negative test fails because the outside file is modified.

## Affected Files

| File | Change |
|------|--------|
| `cmd/add_phase.go` | Added `task.ValidateFilePath` call before any filesystem access |
| `cmd/add_phase_test.go` | Added the rejection and acceptance regression tests |
| `internal/task/batch.go` | Updated two stale `cmd/add_phase.go:59` comment pointers |
| `CHANGELOG.md` | Added `[Unreleased]` section documenting the behaviour change |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] `make check` passes (format, lint, full test suite)

## Prevention

**Recommendations to avoid similar bugs:**
- Any command that touches the filesystem with `os.ReadFile`/`os.WriteFile` instead of the `task` package write helpers must call `task.ValidateFilePath` itself — the check is not inherited
- When adding a mutating command, grep the existing `ValidateFilePath` call sites and confirm the new command appears in one of the two accepted patterns
- Pair every containment check with both a rejection and an acceptance test, so a check that is too strict fails the unit suite rather than only the integration suite

## Related

- Transit tickets T-1473, T-1752 (filed as a duplicate)
- PR #94
- Builds on the symlink containment work in `specs/bugfixes/validate-filepath-symlink-escape/report.md`
