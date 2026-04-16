# Bugfix Report: complete-dry-run-json

**Date:** 2026-04-16
**Status:** Fixed
**Ticket:** T-725

## Description of the Issue

`rune complete --dry-run --format json` emits plain text instead of JSON. The dry-run code path in `runComplete` (and `runUncomplete`) used `fmt.Printf` unconditionally, bypassing the format-aware output switch that the non-dry-run path uses.

**Reproduction steps:**
1. Create a task file: `rune create test.md --title "Test"`
2. Add a task: `rune add test.md --title "Task 1"`
3. Run: `rune complete --dry-run --format json test.md 1`
4. Observe output is plain text lines instead of a JSON object

**Impact:** Breaks automation/scripts that parse JSON output and rely on dry-run safety. The `--format markdown` flag was also ignored in dry-run mode.

## Investigation Summary

- **Symptoms examined:** Dry-run output was always plain text regardless of `--format` flag
- **Code inspected:** `cmd/complete.go`, `cmd/uncomplete.go`, `cmd/progress.go` (which correctly handles this)
- **Hypotheses tested:** Confirmed the dry-run branch simply never checked the `format` variable

## Discovered Root Cause

The dry-run branch in `runComplete` and `runUncomplete` used `fmt.Printf` directly without a `switch format` block, unlike the non-dry-run path and unlike `runProgress` which correctly handles all formats in dry-run mode.

**Defect type:** Missing format dispatch in code path

**Why it occurred:** The dry-run path was likely written before format-aware output was added, and was not updated when the format switch was added to the non-dry-run path.

**Contributing factors:** `progress.go` was implemented correctly, suggesting the pattern was known but not applied consistently.

## Resolution for the Issue

**Changes made:**
- `cmd/complete.go` — Added `switch format` block to dry-run path, matching the pattern used by `progress.go`. Added `DryRun` and `CurrentStatus` fields to `CompleteResponse`.
- `cmd/uncomplete.go` — Same fix applied. Added `DryRun` and `CurrentStatus` fields to `UncompleteResponse`.

**Approach rationale:** Followed the existing pattern from `progress.go` for consistency across all status-changing commands.

**Alternatives considered:**
- Extract a shared dry-run helper — not chosen because each command has slightly different response structs and messages.

## Regression Test

**Test file:** `cmd/complete_test.go`
**Test names:** `TestRunCompleteDryRunJSON`, `TestRunCompleteDryRunMarkdown`, `TestRunUncompleteDryRunJSON`

**What it verifies:** Dry-run output is valid JSON when `--format json` is used, and contains expected markdown when `--format markdown` is used. Also verifies the file is not modified.

**Run command:** `go test -run "TestRunCompleteDryRunJSON|TestRunCompleteDryRunMarkdown|TestRunUncompleteDryRunJSON" -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/complete.go` | Added format-aware dry-run output; added `DryRun`/`CurrentStatus` fields to `CompleteResponse` |
| `cmd/uncomplete.go` | Added format-aware dry-run output; added `DryRun`/`CurrentStatus` fields to `UncompleteResponse` |
| `cmd/complete_test.go` | Added 3 regression tests for dry-run JSON/markdown output |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Code formatted with `make fmt`

## Prevention

**Recommendations to avoid similar bugs:**
- When adding a new code path (like dry-run), ensure it respects the same format dispatch as the primary path
- Consider a linter rule or code review checklist item: "Does every user-facing output path honour `--format`?"

## Related

- T-725: `complete --dry-run --format json` emits plain text instead of JSON
