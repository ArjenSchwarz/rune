# Bugfix Report: Dependency IsReady Ignores Missing Blockers

**Date:** 2026-03-04
**Status:** Fixed

## Description of the Issue

The `DependencyIndex.IsReady` method treated tasks with missing (non-existent) blocker IDs as ready, instead of not-ready. When a task had a `blocked_by` reference to a stable ID that did not exist in the dependency index, `IsReady` returned `true`.

**Reproduction steps:**
1. Create a task with a `blocked_by` referencing a stable ID not present in the task list (e.g., a deleted task or a typo)
2. Build the dependency index and call `IsReady` on that task
3. Observe that `IsReady` returns `true` even though the blocker cannot be resolved

**Impact:** Tasks with invalid or stale dependency references could be incorrectly reported as ready, potentially allowing work to proceed on tasks whose actual prerequisites were unknown or unmet.

## Investigation Summary

The ticket description pointed directly to the defective method. Code inspection confirmed the issue.

- **Symptoms examined:** `IsReady` returning `true` for tasks with unresolvable blocker IDs
- **Code inspected:** `internal/task/dependencies.go`, specifically the `IsReady` method
- **Hypotheses tested:** Single root cause -- the `continue` statement on the nil-blocker branch was identified immediately

## Discovered Root Cause

In `IsReady`, when `idx.GetTask(blockerID)` returned `nil` (blocker not found), the code executed `continue` instead of `return false`. This caused the loop to skip the missing blocker and proceed to the next one. If all blockers were missing, or if the remaining blockers were completed, the method returned `true`.

**Defect type:** Logic error (control flow)

**Why it occurred:** The comment correctly stated the intended behaviour ("we consider the task as not ready") but the code did the opposite (`continue` instead of `return false`).

**Contributing factors:** No test case existed for tasks referencing non-existent blocker IDs, so the mismatch between comment and code was not caught.

## Resolution for the Issue

**Changes made:**
- `internal/task/dependencies.go:90` - Changed `continue` to `return false` in the nil-blocker branch of `IsReady`

**Approach rationale:** The comment already described the correct conservative behaviour. The fix aligns the code with the documented intent.

**Alternatives considered:**
- Silently ignore missing blockers (treat as completed) - Rejected because this is unsafe; missing references should block rather than unblock

## Regression Test

**Test file:** `internal/task/dependencies_test.go`
**Test names:** `TestDependencyIndex_IsReady/blocked_by_missing_blocker`, `TestDependencyIndex_IsReady/blocked_by_missing_and_completed`, `TestDependencyIndex_IsBlocked/blocked_by_nonexistent`

**What it verifies:**
- A task blocked by a non-existent stable ID is not ready
- A task blocked by both a completed task and a non-existent stable ID is not ready
- `IsBlocked` correctly returns `true` for tasks with non-existent blockers

**Run command:** `go test -run 'TestDependencyIndex_IsReady|TestDependencyIndex_IsBlocked' ./internal/task/ -v`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/dependencies.go` | Changed `continue` to `return false` for nil blocker lookup |
| `internal/task/dependencies_test.go` | Added 3 test cases for missing blocker scenarios |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When writing a comment that describes intended behaviour, immediately verify the code matches the comment
- Add test cases for nil/missing/invalid lookup results whenever building index-based resolution logic

## Related

- Transit ticket: T-337
