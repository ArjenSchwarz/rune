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
- `internal/task/render.go` - new `formatReferenceList` writes a comma inside a reference as `\,`. It is the only escape; backslashes are written as is.
- `internal/task/parse.go` - `parseReferences` splits only on commas not preceded by a backslash and turns `\,` back into `,`. Every other backslash is literal.
- `internal/task/operations.go` - `validateReferences` rejects a reference ending in a backslash when another reference follows it. That is the one value this format cannot store: the backslash would run into the separator comma and read back as `\,`. As the last reference it round-trips and is accepted.

**Approach rationale:** References without commas, including Windows paths like `docs\a.md` and UNC paths like `\\server\share\file.md`, are written exactly as before. Because rendering only inserts a backslash before each comma, and parsing only removes it there, parsing and then rendering any existing `References:` line gives back the same text. A mutation never changes a line it doesn't touch.

**Review follow-up (PR #120):** The first version also treated `\\` as an escape for a backslash. A legacy `References: \\server\share\file.md` line then loaded as `\server\share\file.md`, and any mutation wrote that changed path back. Dropping the `\\` escape fixes this. The only case it was needed for, a backslash at the end of a reference, is now rejected when ambiguous instead.

**Alternatives considered:**
- One `References:` line per reference - rejected: it changes the output of every file with more than one reference, and the parser currently keeps only the last `References:` line.
- Percent-encoding commas - rejected: it changes the stored string for paths.

**Compatibility note:** An older file whose `References:` line has a reference ending in a backslash followed by another reference (`dir\, b.md`) now loads as one reference, `dir, b.md`. The line text is still written back unchanged.

## Regression Test

**Test files:** `internal/task/references_comma_regression_test.go`, `cmd/batch_references_regression_test.go`
**Test names:** `TestReferencesWithCommasRoundTrip` (including UNC paths), `TestReferencesLegacyFormat` (legacy Windows and UNC lines load unchanged and survive a mutation byte for byte), `TestReferencesTrailingBackslash`, `TestBatchReferencesWithCommaRoundTrip`, `TestBatchCommand_ReferenceWithCommaRoundTrip`

**Run command:** `go test ./internal/task ./cmd -run 'References|ReferenceWithComma'`
