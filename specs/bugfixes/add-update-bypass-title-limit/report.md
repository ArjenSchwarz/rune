# Bugfix Report: add-update-bypass-title-limit

**Date:** 2026-04-16
**Status:** Fixed

## Description of the Issue

The 500-character task title limit was only enforced in batch operations (`batch.go`) and `Task.Validate()`, but not in the direct add/update code paths. This meant CLI `add` and `update` commands could persist oversized titles while batch calls rejected them.

**Reproduction steps:**
1. Create a task file: `rune create test.md --title "Test"`
2. Add a task with a 501+ character title: `rune add test.md --title "<501 chars>"`
3. The task is accepted despite exceeding the limit

**Impact:** Inconsistent validation across entry points; oversized titles could be persisted through non-batch operations.

## Investigation Summary

- **Symptoms examined:** `validateTaskInput` in `operations.go` only checked for null bytes, not title length
- **Code inspected:** `operations.go` (validateTaskInput, AddTask, UpdateTask, AddTaskWithOptions, UpdateTaskWithOptions), `batch.go` (hardcoded 500 checks), `task.go` (Task.Validate)
- **Hypotheses tested:** Confirmed that `Task.Validate()` was never called in the add/update paths after setting the title

## Discovered Root Cause

`validateTaskInput()` contained a comment "Length validation is handled by Task.Validate()" but `Task.Validate()` was never invoked in the add/update operation paths. The batch operations had their own separate inline length checks.

**Defect type:** Missing validation

**Why it occurred:** The validation was split across multiple locations with an incorrect assumption that `Task.Validate()` would be called downstream.

**Contributing factors:** Lack of a single shared validation constant — the limit was hardcoded as `500` in three separate locations.

## Resolution for the Issue

**Changes made:**
- `internal/task/operations.go:16` - Added `MaxTitleLength = 500` constant
- `internal/task/operations.go:390-396` - Added length check to `validateTaskInput()`, removed misleading comment
- `internal/task/task.go:137-138` - Use `MaxTitleLength` constant instead of hardcoded `500`
- `internal/task/batch.go:234-235,274-275` - Use `MaxTitleLength` constant instead of hardcoded `500`

**Approach rationale:** Centralizing the validation in `validateTaskInput()` ensures all add/update code paths enforce the limit consistently. Introducing a named constant eliminates magic numbers.

**Alternatives considered:**
- Calling `Task.Validate()` after each mutation — heavier and would require constructing a Task object before validation in some paths.

## Regression Test

**Test file:** `internal/task/operations_test.go`
**Test name:** `TestTitleLengthValidation`

**What it verifies:** All four entry points (AddTask, UpdateTask, AddTaskWithOptions, UpdateTaskWithOptions) reject titles exceeding MaxTitleLength and accept titles at exactly MaxTitleLength.

**Run command:** `go test -run TestTitleLengthValidation -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/operations.go` | Added `MaxTitleLength` constant; added length check to `validateTaskInput` |
| `internal/task/task.go` | Use `MaxTitleLength` constant in `Task.Validate()` |
| `internal/task/batch.go` | Use `MaxTitleLength` constant in batch validation |
| `internal/task/operations_test.go` | Added `TestTitleLengthValidation` regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`make test`)

## Prevention

**Recommendations to avoid similar bugs:**
- Use named constants for all validation limits to enable single-point-of-change
- Validate inputs at the shared helper level rather than at individual call sites
- Integration tests should cover all entry points for the same validation rule
