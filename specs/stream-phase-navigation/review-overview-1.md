# PR Review Overview - Iteration 1

**PR**: #28 | **Branch**: feature/stream-phase-navigation | **Date**: 2026-02-06

## Valid Issues

### Code-Level Issues

(none)

### PR-Level Issues

#### Issue 1: Add input validation for negative streamFlag

- **Type**: discussion comment
- **Reviewer**: @claude
- **Comment**: "Add validation in `cmd/next.go` to fail fast for `streamFlag < 0`"
- **Validation**: Valid. Negative values currently fall through all `streamFlag > 0` checks and are treated as 0 (not specified). While benign, returning an explicit error for negative values improves UX and makes intent clear. The suggested check `< 0` is correct since 0 means "not specified".

## Invalid/Skipped Issues

### Issue A: FilterByStream non-recursive for nested stream tasks

- **Location**: `internal/task/next.go:324`
- **Reviewer**: @chatgpt-codex-connector
- **Comment**: "FilterByStream is non-recursive. Stream tasks nested under non-matching parents are missed."
- **Reason**: By design. Stream assignment is a top-level task concern. Subtasks belong to their parent's work unit and are not independently assigned to different streams. `extractPhasesWithTaskRanges` intentionally only associates top-level tasks with phases. This is consistent with the hierarchical task model.

### Issue B: Potential index nil dereference in runNextWithClaim

- **Location**: `cmd/next.go:183`
- **Reviewer**: @claude
- **Comment**: "The code assumes index is never nil. Add nil-safety check or document the invariant."
- **Reason**: `BuildDependencyIndex` (called at line 164) always constructs and returns a non-nil `*DependencyIndex`. A nil check would be dead code. The invariant is guaranteed by the function signature.

### Issue C: Missing function comment for getReadyTasks

- **Location**: `cmd/next.go:247-270`
- **Reviewer**: @claude
- **Comment**: "This important helper function lacks a doc comment."
- **Reason**: The function already has a doc comment at lines 243-246 describing its behavior, including the three criteria (Pending status, blockers completed, no owner). The reviewer appears to have missed it.

### Issue D: Double file parsing performance

- **Location**: `cmd/next.go:517,538`
- **Reviewer**: @claude
- **Comment**: "File is parsed twice - once in the command handler and once in FindNextPhaseTasksForStream."
- **Reason**: Marked as "Nice to Have (Follow-up PRs)" by the reviewer. Performance impact is negligible for typical files. The stateless CLI design makes this acceptable. Skipped for this iteration.
