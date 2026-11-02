# Bugfix Report: Mixed Requirements Paths Retargeted

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1973

## Description of the Issue

When a task file had requirement links to more than one file, parsing kept only the first path. Any later write then rendered every link with that path, so links were silently pointed at the wrong file.

**Reproduction steps:**
1. Create `tasks.md`:
   ```markdown
   # Mixed Requirements Repro

   - [ ] 1. First task
     - Requirements: [1.1](requirements-a.md#1.1)
   - [ ] 2. Second task
     - Requirements: [2.1](requirements-b.md#2.1)
   ```
2. Run `rune progress tasks.md 2`.
3. Task 2 is marked in progress, and its link is rewritten to `requirements-a.md#2.1`.

**Impact:** High. Requirement traceability is corrupted without warning, which breaks the round-trip preservation requirement (`specs/task-requirements-linking/requirements.md` 4.5).

## Investigation Summary

- **Symptoms examined:** Links pointing at a second file were rewritten on any mutation, including mutations of unrelated tasks.
- **Code inspected:** `internal/task/parse.go` (`applyDetailLine`, `parseRequirements`), `internal/task/render.go` (`RenderContext.RequirementsFile`, `renderTask`).
- **Hypotheses tested:** Mixing files within a single `Requirements:` line has the same problem: `parseRequirements` kept only the first link's path.

## Discovered Root Cause

`TaskList.RequirementsFile` is the only path the renderer uses (Decision 3 in `specs/task-requirements-linking/decision_log.md`: one requirements file per task list). The parser set it from the first link it saw and ignored every later path, so files that broke that model were accepted and then rewritten.

**Defect type:** Missing validation (lossy round trip)

**Why it occurred:** The model assumes a single file, but the parser never enforced that assumption.

## Resolution for the Issue

**Changes made:**
- `internal/task/parse.go`: `parseRequirements` returns an error when one line links to more than one file. `applyDetailLine` now returns `(string, error)` and errors when a task's links name a file other than the one already recorded for the list. `parseTasksAtLevel` passes that error on, and `detailMetadataField` ignores it, because it only needs the field name.
- `internal/task/parse_frontmatter_test.go`: updated the existing `parseRequirements` call for the new return value.

**Approach rationale:** The ticket accepts failing to parse as the minimum. Rejecting the file keeps Decision 3 intact and means rune never rewrites the file. The error names the task and both files so the user can fix the file by hand.

**Alternatives considered:**
- Store a requirements path per task and render it per task: Not chosen. It contradicts Decision 3 and is a model change, not a bug fix.

**Limitations:**
- Files with mixed paths, which rune previously accepted (while corrupting them on write), now fail on every command, read-only commands included, until they are fixed by hand.
- Out of scope, but related: `add --requirements-file X` or a batch `requirements_file` on a file that already links to `Y` still retargets the existing links. That path sets `RequirementsFile` after parsing and is covered by other tickets (T-1135/T-1856).

## Regression Test

**Test file:** `internal/task/roundtrip_regression_test.go`
**Test name:** `TestParseRejectsMixedRequirementsFiles`

**What it verifies:** Parsing fails, naming both files, when the files differ across tasks, within one line, or between a parent and a subtask. A file whose links all use one path still parses and keeps its links.

**Run command:** `go test ./internal/task -run TestParseRejectsMixedRequirementsFiles -v`

## Verification

- [x] Regression tests failed before the fix (3 cases parsed without error) and pass after it.
- [x] `go test ./internal/...` passes. `make lint` reports 0 issues.
- [x] Manual check: `rune progress tasks.md 2` on the reproduction file exits 1 with `task 2: requirements link to "requirements-b.md", but other tasks link to "requirements-a.md"; ...`, and the file is unchanged.

## Related

- T-1848 (comma-containing paths), T-1135/T-1856 (batch dry-run path propagation)
