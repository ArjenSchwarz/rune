# Bugfix Report: Task-Shaped Details Become Child Tasks

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1999

## Description of the Issue

`rune update tasks.md 1 --details '[ ] 99. Injected child'` was accepted and written as `  - [ ] 99. Injected child`. The next read turned it into child task `1.1`, with the parent's details empty.

**Impact:** Medium. Accepted input changed the task hierarchy and task count.

## Discovered Root Cause

**Defect type:** Missing validation.

`validateDetails` (`internal/task/operations.go`) already rejected details that would be read back as metadata (T-1788), but not details that the parser reads as a task line. `renderTask` writes details verbatim, and `parseDetailsAndChildren` tries task syntax first.

The parser also treats any `- [..]` line that is not a valid task as an invalid task line and fails, so a detail like `[link](x.md) text` produced a file rune could not read again.

## Resolution for the Issue

**Changes made:**
- `internal/task/operations.go` - `validateDetails` runs the parser's own `parseTaskLine` on the detail as it will be written (`"- " + detail`). If that reads as a task, or as malformed task syntax, the detail is rejected with an error naming it.

**Approach rationale:** Using the parser's classifier keeps validation and parsing in agreement, the same way metadata collisions are handled. Rejecting is simpler and more predictable than inventing an escape for details.

**Behaviour change:** Details starting with `[` that contain a later `]` (for example `[link](x.md) text` or `[ ]`) are now rejected. Before this change they were accepted but left the file unparsable. Details that merely contain brackets later in the text (`see [link](x.md)`) are unaffected.

## Regression Test

**Test file:** `internal/task/task_shaped_details_regression_test.go`
**Test name:** `TestTaskShapedDetailsRejected`

**Run command:** `go test ./internal/task -run TaskShaped`
