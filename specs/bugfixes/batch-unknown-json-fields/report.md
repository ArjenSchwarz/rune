# Bugfix Report: Unknown Batch JSON Fields Can Turn Dry Runs into Live Writes

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1844

## Description of the Issue

`rune batch` ignored unknown JSON fields. A misspelled `"dryRun": true` left `dry_run` false, so the batch was applied and saved.

**Reproduction steps:**
1. Create `tasks.md` with one task.
2. Run `rune batch --input '{"file":"tasks.md","dryRun":true,"operations":[{"type":"add","title":"Unexpected live write"}]}' --format json`.
3. Output reports `success: true`, `applied: 1`, and the task is written to the file.

**Impact:** High. Typos in request or operation fields silently changed semantics, including turning a preview into a write.

## Investigation Summary

- **Code inspected:** `cmd/batch.go` (request parsing), `internal/task/batch.go` (`BatchRequest`, `Operation` JSON tags).
- **Hypotheses tested:** `json.Unmarshal` drops unknown keys by design; nothing else validated field names.

## Discovered Root Cause

The request was decoded with permissive `json.Unmarshal`, so unknown keys at any level were discarded.

**Defect type:** Missing validation

## Resolution for the Issue

**Changes made:**
- `cmd/batch.go` - New `decodeBatchRequest` uses `json.Decoder` with `DisallowUnknownFields` (applies to nested operations too) and then requires EOF, so trailing data after the object is still rejected as `json.Unmarshal` did. Decoding happens before the task file is read, so nothing is mutated on rejection.

**Alternatives considered:**
- Accepting `dryRun` as an alias - rejected: fixes one typo, not the class of bug.

## Regression Test

**Test file:** `cmd/batch_test.go`
**Test name:** `TestBatchCommand_RejectsUnknownJSONFields`

**What it verifies:** `dryRun` at request level, a misspelled operation field (`parnet`) and trailing JSON are rejected with an error and the file is left unchanged.

**Run command:** `go test -run TestBatchCommand_RejectsUnknownJSONFields ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/batch.go` | Strict request decoding |
| `cmd/batch_test.go` | Regression test |
| `docs/json-api.md` | Documented strict decoding |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (pre-existing root-permission failures excepted, see T-1789 report)
- [x] Linters/validators pass

**Manual verification:**
- Built binary: the reproduction exits 1 with `json: unknown field "dryRun"`, and the file is unchanged. A valid `dry_run` request still exits 0.

## Prevention

- Decode user-supplied JSON requests strictly; a silently ignored field is indistinguishable from a default.
