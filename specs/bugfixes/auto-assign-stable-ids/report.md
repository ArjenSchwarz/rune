# Bugfix Report: Auto-assign Stable IDs to Dependency Targets

**Date:** 2026-02-16
**Status:** Fixed

## Description of the Issue

When a task is referenced as a `blocked_by` target but has no stable ID, both `validateExtendedFields` and `resolveToStableIDs` return `ErrNoStableID`. This causes batch operations with dependencies to fail unless every task in the chain already has a stable ID (which only happens when a task is created with extended fields like `stream`, `blocked_by`, or `owner`).

**Reproduction steps:**
1. Create a task list with `rune create tasks.md --title "My Tasks"`
2. Add a plain task: `rune add tasks.md --title "Foundation task"`
3. Try to add a dependent task: `rune add tasks.md --title "Dependent task" --blocked-by 1`
4. Observe error: `task does not have a stable ID (legacy task)`

**Impact:** Users must add a gratuitous extended field (e.g., `--stream "1"`) to root tasks just to make dependency chains work. Integration tests worked around this by always adding `--stream "1"`.

## Investigation Summary

- **Symptoms examined:** `ErrNoStableID` returned when referencing a task without extended fields as a dependency target
- **Code inspected:** `internal/task/operations.go` (`resolveToStableIDs`), `internal/task/batch.go` (`validateExtendedFields`)
- **Hypotheses tested:** Single hypothesis — the code should auto-assign stable IDs on demand rather than erroring, since stable IDs are designed to be generated when extended features are used

## Discovered Root Cause

Both `resolveToStableIDs` (operations.go:1148) and `validateExtendedFields` (batch.go:324) check `task.StableID == ""` and immediately return `ErrNoStableID`. The stable ID generation machinery (`NewStableIDGenerator` + `collectStableIDs`) already exists but was not invoked in these paths.

**Defect type:** Missing feature — auto-assignment on demand

**Why it occurred:** The original implementation treated tasks without stable IDs as "legacy" tasks that couldn't participate in dependencies, rather than auto-assigning IDs when needed.

**Contributing factors:** Stable IDs are only generated when `AddTaskWithOptions` is called with extended fields. Tasks created without any extended fields (the common case for root tasks) never get stable IDs assigned.

## Resolution for the Issue

**Changes made:**
- `internal/task/operations.go:1148-1155` — `resolveToStableIDs` now generates a stable ID via `NewStableIDGenerator(tl.collectStableIDs())` when `task.StableID == ""`
- `internal/task/batch.go:324-332` — `validateExtendedFields` applies the same auto-assignment pattern
- `internal/task/errors.go:12` — Removed `ErrNoStableID` (no longer returned from any call site)

**Approach rationale:** `FindTask` returns `*Task` (pointer into the slice), so assigning `task.StableID = newID` mutates the actual task in the list. This matches the existing design principle that stable IDs are generated on demand when extended features are used.

**Alternatives considered:**
- Requiring users to explicitly assign stable IDs before using dependencies — rejected as unnecessary friction; the ID generation is deterministic and collision-free
- Auto-assigning stable IDs to all tasks at parse time — rejected as too broad; only tasks that participate in dependencies need them

## Regression Test

**Test file:** `internal/task/batch_extended_test.go`
**Test name:** `TestExecuteBatch_BlockedByAutoAssignsStableID`

**What it verifies:** A batch operation that adds two tasks where the second depends on the first (neither has extended fields initially) succeeds, with the first task receiving an auto-assigned stable ID.

**Run command:** `go test -run TestExecuteBatch_BlockedByAutoAssignsStableID -v ./internal/task`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/operations.go` | `resolveToStableIDs` auto-generates stable IDs instead of returning error |
| `internal/task/batch.go` | `validateExtendedFields` auto-generates stable IDs instead of returning error |
| `internal/task/errors.go` | Removed `ErrNoStableID` |
| `internal/task/operations_extended_test.go` | Updated 2 tests from error to success expectations |
| `internal/task/batch_extended_test.go` | Updated `DependencyOnLegacyTask` test, added `BlockedByAutoAssignsStableID` test |
| `cmd/add_test.go` | Updated legacy task test to expect success with auto-assigned stable ID |
| `cmd/update_test.go` | Updated legacy task test to expect success with auto-assigned stable ID |
| `cmd/integration_batch_test.go` | Removed `--stream "1"` workaround from dependency chain test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`make check`)
- [x] Integration tests pass (`INTEGRATION=1 make test-all`)

## Prevention

**Recommendations to avoid similar bugs:**
- When adding validation that rejects a state, consider whether the system can fix the state automatically instead
- Design extended features to be fully opt-in without requiring workarounds on related tasks

## Related

- Transit ticket: T-59
- PR: https://github.com/ArjenSchwarz/rune/pull/34
