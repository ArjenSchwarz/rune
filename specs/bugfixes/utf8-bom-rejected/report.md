# Bugfix Report: UTF-8 BOM Task Files Rejected

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-2063

## Description of the Issue

A task file starting with a UTF-8 byte-order mark (`EF BB BF`), as written by some Windows editors, failed to parse with `line 1: unexpected content at this indentation level`. Every command that reads the file failed.

**Reproduction steps:**
1. `printf '\xef\xbb\xbf# BOM Project\n\n- [ ] 1. A\n' > bom.md`
2. `rune list bom.md --format json`
3. The command exits 1. The same file without the BOM parses, and so does the equivalent file with front matter.

**Impact:** Medium. Otherwise valid files were unusable.

## Investigation Summary

- **Code inspected:** `ParseMarkdown` and `phaseMarkersFromContent` (`internal/task/parse.go`), `ParseFrontMatter` (`internal/task/references.go`).
- **Finding:** The bytes were converted to a string as is, so line 1 started with U+FEFF. Neither the `# ` title check nor the `---` front matter check matched, and the line then failed task parsing.

## Discovered Root Cause

**Defect type:** Missing input normalisation.

**Why it occurred:** CRLF line endings are normalised in the parser, but the BOM was never considered.

## Resolution for the Issue

**Changes made:**
- `internal/task/parse.go` - `ParseMarkdown` strips exactly one leading BOM after the 10 MiB size check. `phaseMarkersFromContent` (used by `ParseFileWithPhases`) does the same, so front matter is still skipped before phase headers are read.

**Approach rationale:** One central strip covers H1, front matter and CRLF variants for every reader that goes through `ParseMarkdown`. Only one BOM is removed; a second one is content.

**Behaviour note:** Files are written back without the BOM.

**Review follow-up (PR #120):** Three production paths re-read raw file bytes to find phase headers: `FindNextPhaseTasks` and `FindNextPhaseTasksForStream` (`internal/task/next.go`, used by `next --phase` with and without `--stream`) and `has-phases` (`cmd/has_phases.go`). With a BOM directly before `## First`, they missed that header, so `next --phase` skipped to the second phase. A BOM before front matter stopped them skipping it, so a `## ` YAML comment counted as a phase. These paths now call the exported `task.StripBOM` before splitting lines. In `next.go` the strip happens after `ParseMarkdown`, which strips its own copy, so only one BOM is ever removed. The `phase.go` helpers `getTaskPhase`, `getNextPhaseTasks` and `findPhasePosition` also read raw bytes, but only tests call them, so they are unchanged.

## Regression Test

**Test file:** `internal/task/bom_regression_test.go`
**Test names:** `TestParseMarkdownStripsUTF8BOM` (H1, CRLF H1, front matter, CRLF front matter, no title, doubled BOM), `TestPhaseMarkersIgnoreUTF8BOM`

**Test file:** `cmd/bom_phase_regression_test.go`
**Test names:** `TestNextPhase_UTF8BOM` (BOM before the first phase, with and without `--stream`; BOM before front matter), `TestHasPhases_UTF8BOM`

**Run command:** `go test ./internal/task ./cmd -run BOM`
