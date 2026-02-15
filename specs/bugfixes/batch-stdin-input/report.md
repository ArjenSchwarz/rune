# Bugfix Report: Batch Stdin Input

**Date:** 2026-02-16
**Status:** Fixed

## Description of the Issue

The `batch` command's `--input` flag treats every value as literal JSON. When users pass `--input -` (the conventional Unix idiom for "read from stdin"), it does not read stdin — it tries to parse the string `"-"` as JSON.

**Reproduction steps:**
1. Run `echo '{"operations":[{"type":"add","title":"New task"}]}' | rune batch tasks.md --input -`
2. Observe error: `parsing JSON input: invalid character '-' in numeric literal`

**Impact:** Users piping JSON into `rune batch` with an explicit `--input -` flag cannot use the command. The workaround is to omit `--input -` entirely and rely on implicit stdin, but that prevents using a positional file argument alongside stdin.

## Investigation Summary

- **Symptoms examined:** `--input -` produces a JSON parse error
- **Code inspected:** `cmd/batch.go` — the `runBatch` switch statement for input source selection
- **Hypotheses tested:** Single hypothesis — the switch evaluates `batchInput != ""` before checking for the `-` sentinel value

## Discovered Root Cause

The `runBatch` function's input-source switch (line 71) has three cases:
1. `batchInput != ""` — treat value as literal JSON
2. Positional arg — read from file
3. Default — read from stdin

When `--input -` is passed, case 1 matches because `"-"` is non-empty. The `-` convention is never checked.

**Defect type:** Missing input validation

**Why it occurred:** The original implementation assumed `--input` would only receive literal JSON strings, not the Unix `-` stdin marker.

**Contributing factors:** The stdin fallback (case 3) only triggers when no `--input` flag is set and no positional arg is given, so there was no path for explicit stdin via `--input -`.

## Resolution for the Issue

**Changes made:**
- `cmd/batch.go:72-79` — Added a new `batchInput == "-"` case before the `batchInput != ""` case that reads from `os.Stdin` via `io.ReadAll`
- `cmd/batch.go:270` — Updated `--input` flag description to document `-` as stdin marker
- `cmd/batch.go:278-280` — Added usage example for `--input -` with positional file arg

**Approach rationale:** Adding a dedicated case before the literal-JSON case is minimal and follows the existing pattern (the default case already reads stdin with `io.ReadAll`). It preserves positional file arg handling.

**Alternatives considered:**
- Checking for `-` inside the `batchInput != ""` case with an if/else — functionally equivalent but a dedicated switch case is clearer
- Making stdin detection automatic when input looks non-JSON — too magical, would break literal JSON strings that happen to start with certain characters

## Regression Test

**Test file:** `cmd/batch_test.go`
**Test name:** `TestBatchCommand_StdinViaDash`

**What it verifies:** Passing `--input -` with a positional file argument reads JSON from stdin (via an os.Pipe) and executes the batch operations correctly.

**Run command:** `go test -run TestBatchCommand_StdinViaDash -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/batch.go` | Added `batchInput == "-"` switch case, updated flag description and examples |
| `cmd/batch_test.go` | Added `TestBatchCommand_StdinViaDash` regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass (`make check`)

**Manual verification:**
- Confirmed the regression test fails before the fix and passes after

## Prevention

**Recommendations to avoid similar bugs:**
- When adding CLI flags that accept string values, consider the `-` stdin convention if the flag could plausibly receive piped input
- Document stdin support explicitly in flag descriptions

## Related

- Transit ticket: T-69
