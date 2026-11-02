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

**Not changed:** A few phase helpers re-read raw file bytes themselves (`internal/task/next.go`, `internal/task/phase.go`). Their phase detection already works on BOM files (tested with `next --phase` and `add --phase`). Only a BOM file whose front matter contains a `## ` YAML comment could still have that comment read as a phase header by those helpers.

## Regression Test

**Test file:** `internal/task/bom_regression_test.go`
**Test names:** `TestParseMarkdownStripsUTF8BOM` (H1, CRLF H1, front matter, CRLF front matter, no title, doubled BOM), `TestPhaseMarkersIgnoreUTF8BOM`

**Run command:** `go test ./internal/task -run BOM`
