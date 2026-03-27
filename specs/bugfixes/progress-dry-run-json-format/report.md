# Bugfix Report: progress-dry-run-json-format

**Date:** 2026-03-27
**Status:** Fixed
**Transit:** T-616

## Description of the Issue

When running `rune progress <file> <task-id> --dry-run --format json`, the output was plain text instead of structured JSON. The `--format` flag was completely ignored in dry-run mode.

**Reproduction steps:**
1. Create a task file with a pending task
2. Run `rune progress tasks.md 1 --dry-run --format json`
3. Observe plain text output like `Would mark task as in-progress...`

**Impact:** Any automation or scripting that relied on JSON output from dry-run would break, as the output was unparseable as JSON.

## Investigation Summary

- **Symptoms examined:** `--dry-run` always produced plain text regardless of `--format` value
- **Code inspected:** `cmd/progress.go` lines 84-92 (dry-run block), lines 105-126 (format switch)
- **Hypotheses tested:** Confirmed the early `return nil` in the dry-run block bypasses the format switch entirely

## Discovered Root Cause

The `runProgress` function's dry-run block (lines 84-92) used `fmt.Printf` for all output and returned early via `return nil`, completely bypassing the format-aware switch statement at lines 105-126.

**Defect type:** Logic error — early return bypassing format dispatch

**Why it occurred:** The dry-run path was added as a simple guard clause that exits before the format routing, rather than being integrated into the format-aware output pipeline.

**Contributing factors:** No tests existed for the combination of `--dry-run` with `--format` flags.

## Resolution for the Issue

**Changes made:**
- `cmd/progress.go:12-17` — Added `DryRun` and `CurrentStatus` fields to `ProgressResponse` struct
- `cmd/progress.go:84-101` — Replaced plain-text dry-run block with format-aware switch (JSON/markdown/table)

**Approach rationale:** Routes dry-run output through the same format dispatch pattern used by the non-dry-run path, ensuring all three formats (json, markdown, table) produce appropriate output.

**Alternatives considered:**
- Creating a separate `DryRunResponse` struct — unnecessary since `ProgressResponse` can accommodate both with `omitempty` tags

## Regression Test

**Test file:** `cmd/progress_test.go`
**Test names:** `TestRunProgressDryRunFormats`, `TestRunProgressDryRunDoesNotModifyFile`

**What it verifies:** Dry-run mode produces valid JSON (parseable, correct fields including `dry_run: true`), appropriate markdown, and table output. Also verifies the file is never modified during dry-run.

**Run command:** `go test -run TestRunProgressDryRun -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/progress.go` | Added `DryRun`/`CurrentStatus` to response struct; format-aware dry-run output |
| `cmd/progress_test.go` | New file with regression tests for dry-run format handling |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Code formatted with `make fmt`

## Prevention

**Recommendations to avoid similar bugs:**
- When adding dry-run support to commands, always route through the existing format switch rather than using early-return with `fmt.Printf`
- Add tests for the combination of `--dry-run` with each `--format` value
