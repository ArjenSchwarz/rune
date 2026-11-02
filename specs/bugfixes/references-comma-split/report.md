# Bugfix Report: Task References Split on Commas

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1808

## Description of the Issue

A task reference containing a comma was read back as several references.

**Reproduction steps:**
1. `rune batch tasks.md --input '{"operations":[{"type":"add","title":"With URL","references":["https://example.test/search?q=a,b"]}]}'`
2. `rune list tasks.md --format json`
3. `References` is `["https://example.test/search?q=a", "b"]`.

**Impact:** Medium. Valid URLs and paths with commas were corrupted on the first write.

## Discovered Root Cause

**Defect type:** Ambiguous serialisation format.

`renderTask` (`internal/task/render.go`) joined references with `", "` and `parseReferences` (`internal/task/parse.go`) split on every comma, so a comma inside a reference could not be told apart from the separator.

## Resolution for the Issue

**Changes made:**
- `internal/task/render.go` - new `formatReferenceList` writes a comma inside a reference as `\,`. A backslash is written as `\\` only where it would otherwise be ambiguous: before a comma or a backslash, or at the end of a reference.
- `internal/task/parse.go` - `parseReferences` splits only on unescaped commas and turns `\,` and `\\` back into `,` and `\`. Any other backslash is kept as is.

**Approach rationale:** References without commas, including Windows paths like `docs\a.md`, are written exactly as before, so existing files and output do not change.

**Alternatives considered:**
- One `References:` line per reference - rejected: it changes the output of every file with more than one reference, and the parser currently keeps only the last `References:` line.
- Percent-encoding commas - rejected: it changes the stored string for paths.

**Compatibility note:** An existing file containing `\\`, or a backslash right before a comma, inside a `References:` line is now read with those escapes applied. A UNC-style `\\server\share` reference written by an older version would read back as `\server\share`.

## Regression Test

**Test files:** `internal/task/references_comma_regression_test.go`, `cmd/batch_references_regression_test.go`
**Test names:** `TestReferencesWithCommasRoundTrip`, `TestReferencesLegacyFormat`, `TestBatchReferencesWithCommaRoundTrip`, `TestBatchCommand_ReferenceWithCommaRoundTrip`

**Run command:** `go test ./internal/task ./cmd -run 'ReferencesWithComma|ReferencesLegacy|ReferenceWithComma'`
