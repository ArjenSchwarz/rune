# Bugfix Report: has-phases-front-matter-headings

**Date:** 2026-03-30
**Status:** Fixed
**Ticket:** T-655

## Description of the Issue

`rune has-phases` incorrectly reported files with YAML front matter as containing phases when the front matter contained lines starting with `## ` (H2 markdown heading syntax).

**Reproduction steps:**
1. Create a task file with YAML front matter containing a `## ` line (e.g., `## not-a-phase` inside a YAML block scalar)
2. Run `rune has-phases <file>`
3. Observe `hasPhases: true` and non-zero phase count despite no real phase headers in the task content

**Impact:** Any file with YAML front matter containing `## ` patterns would be misidentified as having phases, causing incorrect behaviour in scripts and agents relying on `has-phases` for phase detection.

## Investigation Summary

- **Symptoms examined:** `has-phases` returns `hasPhases: true` for files where front matter YAML contains H2-like lines
- **Code inspected:** `cmd/has_phases.go`, `internal/task/parse.go` (both `ParseFileWithPhases` and `ExtractPhaseMarkers`)
- **Hypotheses tested:** Compared `has_phases.go` code path with `ParseFileWithPhases` — confirmed the latter correctly strips front matter while the former does not

## Discovered Root Cause

`cmd/has_phases.go` reads the file, splits it into lines, and passes those lines directly to `task.ExtractPhaseMarkers()` without stripping YAML front matter. Meanwhile, `ParseFileWithPhases()` correctly calls `stripFrontMatterLines()` before extracting phase markers.

**Defect type:** Missing input sanitisation — inconsistent code path

**Why it occurred:** The `has-phases` command was implemented independently from `ParseFileWithPhases` and did not reuse the front-matter stripping logic already present in the parsing pipeline.

**Contributing factors:** `stripFrontMatterLines` was unexported, making it less discoverable for reuse by the command layer.

## Resolution for the Issue

**Changes made:**
- `internal/task/parse.go:85` — Exported `stripFrontMatterLines` → `StripFrontMatterLines` for reuse by the command layer
- `cmd/has_phases.go:66-68` — Added `task.StripFrontMatterLines()` call before `ExtractPhaseMarkers()`

**Approach rationale:** Reuses the existing, tested front-matter stripping function rather than duplicating logic. Exporting the function makes it available to any future consumers.

**Alternatives considered:**
- Calling `ParseFileWithPhases()` instead — rejected because it parses the full TaskList unnecessarily
- Duplicating front-matter stripping inline — rejected to avoid code duplication

## Regression Test

**Test file:** `cmd/has_phases_test.go`
**Test names:**
- `TestHasPhasesDetection/front_matter_with_h2_no_real_phases`
- `TestHasPhasesDetection/front_matter_with_h2_and_real_phases`
- `TestHasPhasesDetection/front_matter_with_multiple_h2_lines`

**What it verifies:** H2-like lines inside YAML front matter are not counted as phases; real phases after front matter are still detected correctly.

**Run command:** `go test -run TestHasPhasesDetection/front_matter -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/parse.go` | Exported `StripFrontMatterLines` |
| `internal/task/parse_phases_frontmatter_test.go` | Updated call to use exported name |
| `cmd/has_phases.go` | Strip front matter before phase extraction |
| `cmd/has_phases_test.go` | Added 3 regression test cases; updated all tests to use `StripFrontMatterLines` |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`make test`)
- [x] Manual validation confirms fix

**Manual verification:**
- Created file with front matter containing `## not-a-phase`, confirmed `has-phases` returns `false`
- Verified files with real phases after front matter still report correctly

## Prevention

**Recommendations to avoid similar bugs:**
- When adding commands that operate on file content, always strip front matter first — reuse `StripFrontMatterLines`
- Keep utility functions exported when they represent reusable file-processing steps

## Related

- T-458: Previous fix for `ParseFileWithPhases` front-matter/horizontal-rule interaction
- `specs/bugfixes/parse-file-with-phases-front-matter-hr/` — Related earlier bugfix
