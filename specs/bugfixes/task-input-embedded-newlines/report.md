# Bugfix Report: task-input-embedded-newlines

**Date:** 2026-04-16
**Status:** Fixed
**Transit:** T-781

## Description of the Issue

The `containsNullByte` helper function in `internal/task/operations.go` explicitly allowed `\n` and `\r` characters through validation. Since `validateTaskInput`, `validateDetails`, and `validateReferences` all relied on this helper, embedded newlines could be injected into task titles, details, and references. When rendered to markdown, these newlines break line structure, corrupting the task file.

**Reproduction steps:**
1. Call `AddTask` with a title containing `\n` (e.g., `"line1\nline2"`)
2. Render the task list to markdown
3. Observe the title splits across multiple lines, breaking the markdown task format

**Impact:** Malformed markdown files could be generated from validated input, leading to data corruption on re-parse.

## Investigation Summary

- **Symptoms examined:** Newlines in titles/details/references pass validation but break markdown rendering
- **Code inspected:** `internal/task/operations.go` — `containsNullByte`, `validateTaskInput`, `validateDetails`, `validateReferences`, `validateOwner`
- **Hypotheses tested:** `validateOwner` already correctly rejects `\n`/`\r`, confirming the inconsistency is in the shared helper

## Discovered Root Cause

`containsNullByte` (line 440) had an allowlist for `\t`, `\n`, and `\r`:
```go
if r == 0 || (r < 32 && r != '\t' && r != '\n' && r != '\r') {
```

This meant newlines and carriage returns were treated as safe, but they break single-line markdown fields.

**Defect type:** Missing validation

**Why it occurred:** The helper was originally designed to catch null bytes and "dangerous" control characters, but `\n`/`\r` were considered benign whitespace rather than markdown-structure-breaking characters.

**Contributing factors:** `validateOwner` was implemented separately with its own stricter check, so the inconsistency went unnoticed.

## Resolution for the Issue

**Changes made:**
- `internal/task/operations.go:440` — Removed `\n` and `\r` from the allowlist in `containsNullByte`, so only `\t` is permitted among control characters

**Approach rationale:** Single-point fix in the shared helper ensures all callers (`validateTaskInput`, `validateDetails`, `validateReferences`, and path validation) consistently reject embedded newlines. This aligns with `validateOwner`'s existing behaviour.

**Alternatives considered:**
- Add separate newline checks in each validator — rejected because it duplicates logic and risks future validators forgetting the check

## Regression Test

**Test file:** `internal/task/operations_test.go`
**Test name:** `TestEmbeddedNewlinesRejected`

**What it verifies:** `\n`, `\r`, and `\r\n` are rejected in titles (AddTask/UpdateTask), details, and references. Also verifies tabs and normal text are still accepted.

**Run command:** `go test -run TestEmbeddedNewlinesRejected -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/operations.go` | Remove `\n`/`\r` from `containsNullByte` allowlist |
| `internal/task/operations_test.go` | Add `TestEmbeddedNewlinesRejected` regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Code formatted with `make fmt`

## Prevention

**Recommendations to avoid similar bugs:**
- Validation helpers should default to rejecting control characters and explicitly allowlist only what's needed
- Any new field validators should use the shared helper rather than implementing their own checks
