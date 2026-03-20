# Bugfix Report: CRLF Phase Marker Parsing

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

Phase-related functions in `internal/task/` split file content on `\n` without trimming trailing `\r` from CRLF line endings. While the current regex patterns (`phaseHeaderPattern`, `taskLinePattern`) happen to match because `.+` consumes `\r`, this is fragile: captured group values contain `\r` (e.g., phase names, task titles), and any regex using a character class at the end position (like `\d+$` in `streamPattern`) would fail outright.

**Reproduction steps:**
1. Create a markdown task file with CRLF line endings containing phase headers
2. Call `ParseFileWithPhases`, `ExtractPhaseMarkers`, or `FindNextPhaseTasks` on it
3. Observe that captured values may contain trailing `\r` characters

**Impact:** On Windows-origin files (or any CRLF-encoded files), phase markers and task data could contain trailing `\r` in captured regex groups. While the current code masks this with `TrimSpace` on phase names and uses `.+` patterns that absorb `\r`, the approach is inconsistent with `parseContent` (which properly trims `\r`) and creates a latent defect if regex patterns are refined.

## Investigation Summary

- **Symptoms examined:** Regex matching behavior with `\r` in lines; comparison of captured values across code paths
- **Code inspected:** `parse.go`, `phase.go`, `next.go`, `operations.go`, `has_phases.go`
- **Hypotheses tested:** Confirmed that `^## (.+)$` matches `"## Planning\r"` (`.+` consumes `\r`), but `^stream:\s*(\d+)$` fails on `"Stream: 2\r"` because `\d+` does not match `\r`. This demonstrates the fragility.

## Discovered Root Cause

Multiple functions split content on `\n` without stripping `\r`, unlike `parseContent` which properly normalizes at lines 123-125. These functions bypass the established CRLF handling.

**Defect type:** Missing input normalization

**Why it occurred:** Phase parsing was added as a separate code path from `parseContent`. Each new function independently split content on `\n` without copying the `\r`-trimming pattern from `parseContent`.

**Contributing factors:** No shared `splitLines` helper existed; each function duplicated the `strings.Split(content, "\n")` call independently.

## Resolution for the Issue

**Changes made:**
- `internal/task/parse.go` - Added `splitLines` helper that splits on `\n` and trims `\r`; updated `ParseFileWithPhases` and `parseContent` to use it; added `\r` trimming inside `ExtractPhaseMarkers` for external callers
- `internal/task/phase.go` - Updated `getTaskPhase`, `getNextPhaseTasks`, `findPhasePosition` to use `splitLines`
- `internal/task/next.go` - Updated `FindNextPhaseTasks`, `FindNextPhaseTasksForStream` to use `splitLines`
- `internal/task/operations.go` - Updated `WriteFile`, `RemoveTaskWithPhases`, `UpdateTaskWithPhases`, `UpdateStatusWithPhases` to use `splitLines`

**Approach rationale:** A centralized `splitLines` helper eliminates the duplication and ensures all content-splitting paths handle CRLF consistently. `ExtractPhaseMarkers` also trims `\r` internally as a defensive measure for callers that pass pre-split lines without trimming (like `cmd/has_phases.go`).

**Alternatives considered:**
- Modifying regex patterns to tolerate `\r` (e.g., `\r?$`) - Rejected because it treats symptoms rather than the cause and would need to be applied to every regex
- Only normalizing at the file-reading boundary - Rejected because some functions receive `[]byte` content (not file paths) and the normalization needs to happen at the split point

## Regression Test

**Test file:** `internal/task/phase_test.go`, `internal/task/next_test.go`
**Test names:** `TestExtractPhaseMarkersCRLF`, `TestGetTaskPhaseCRLF`, `TestBuildTaskPhaseMapCRLF`, `TestGetNextPhaseTasksCRLF`, `TestHasPhasesCRLF`, `TestFindNextPhaseTasksCRLF`, `TestExtractPhasesWithTaskRangesCRLF`

**What it verifies:** Phase markers, task-phase maps, and phase task results are produced correctly from CRLF-encoded content, with no `\r` in output values.

**Run command:** `go test -run "CRLF" -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/parse.go` | Added `splitLines` helper; updated `ParseFileWithPhases` and `parseContent`; added `\r` trim in `ExtractPhaseMarkers` |
| `internal/task/phase.go` | Updated `getTaskPhase`, `getNextPhaseTasks`, `findPhasePosition` to use `splitLines` |
| `internal/task/next.go` | Updated `FindNextPhaseTasks`, `FindNextPhaseTasksForStream` to use `splitLines` |
| `internal/task/operations.go` | Updated `WriteFile`, `RemoveTaskWithPhases`, `UpdateTaskWithPhases`, `UpdateStatusWithPhases` to use `splitLines` |
| `internal/task/phase_test.go` | Added 5 CRLF regression tests |
| `internal/task/next_test.go` | Added 2 CRLF regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- Use `splitLines` instead of `strings.Split(content, "\n")` when processing markdown content in the `task` package
- The existing CRLF handling note in `docs/agent-notes/parsing.md` should be updated to reference `splitLines`

## Related

- T-488: Handle CRLF in phase marker parsing
- Existing CRLF normalization in `ParseFrontMatter` and `parseContent`
