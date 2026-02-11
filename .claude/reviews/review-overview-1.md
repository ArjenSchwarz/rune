# PR Review Overview - Iteration 1

**PR**: #32 | **Branch**: fix/renumber-stableid-docs-and-test | **Date**: 2026-02-12

## Valid Issues

### Code-Level Issues

#### Issue 1: Assert renumbering against raw file IDs

- **File**: `cmd/renumber_test.go:1004`
- **Reviewer**: @chatgpt-codex-connector
- **Comment**: "This assertion does not actually verify that `runRenumber` rewrote IDs in the file: `task.ParseFileWithPhases` normalizes task IDs to sequential values during parsing, so the check will still pass even if the command leaves `5/10/15` unchanged on disk."
- **Validation**: Valid. The parser auto-renumbers during parsing (`parseTasksAtLevel`), so `result.Tasks[i].ID` will always be sequential regardless of what's on disk. The test already validates raw content for stable ID markers (lines 1046-1056); the same approach should verify hierarchical IDs were actually rewritten in the file.

### PR-Level Issues

#### Issue 2: Verify blocked-by values, not just count

- **Type**: discussion comment
- **Reviewer**: @claude
- **Comment**: "The blocked-by verification for Task 3 checks the count but not the actual values. Consider using `reflect.DeepEqual`."
- **Validation**: Valid. Lines 1023-1025 check `len(result.Tasks[2].BlockedBy) != 2` but don't verify the values are `["abc1234", "def5678"]`. Task 2's check (line 1020) already checks both count and value — Task 3 should be consistent.

## Invalid/Skipped Issues

### Issue A: CHANGELOG categorization

- **Location**: PR-level
- **Reviewer**: @claude
- **Comment**: "This is categorized as 'Fixed' but could arguably be 'Documentation' or 'Improved'."
- **Reason**: "Fixed" is appropriate here. The missing documentation was a defect that caused agents to incorrectly flag the command as unsafe. The test gap was also a defect. Both are fixes.

### Issue B: Add comment about Task 3 having no owner

- **Location**: PR-level
- **Reviewer**: @claude
- **Comment**: "Task 3 has no owner verification. Adding a comment would clarify this is intentional."
- **Reason**: The test structure already makes this clear — the input has no owner for task 3, and verifying absent fields for every task would add noise without value.

### Issue C: Codex review boilerplate

- **Location**: PR-level review
- **Reviewer**: @chatgpt-codex-connector
- **Comment**: Codex introduction/settings boilerplate
- **Reason**: No actionable feedback.
