# Bugfix Report: Find --parent "" Cannot Filter Top-Level Tasks

**Date:** 2026-03-13
**Status:** Fixed

## Description of the Issue

The `find` command advertises `--parent ""` for filtering top-level tasks, but setting `--parent ""` is a no-op. Top-level tasks cannot be isolated from nested results.

**Reproduction steps:**
1. Create a task file with both top-level and nested tasks
2. Run `rune find <file> --pattern <broad-pattern> --parent ""`
3. Observe that all matching tasks are returned, not just top-level ones

**Impact:** Users and agents cannot filter find results to top-level tasks only, despite the flag's documentation suggesting this is supported.

## Investigation Summary

- **Symptoms examined:** `--parent ""` returns the same results as omitting `--parent` entirely
- **Code inspected:** `cmd/find.go` — `runFind` function and `applyAdditionalFilters`
- **Hypotheses tested:** The empty string is ambiguous — it serves as both the flag's default ("not set") and the desired filter value ("no parent")

## Discovered Root Cause

Two guard clauses prevent empty-string parent filtering from working:

1. In `runFind` (line 106): `if statusFilter != "" || maxDepth > 0 || parentIDFilter != ""` — when `parentIDFilter` is `""`, this condition is false (unless other filters are active), so `applyAdditionalFilters` is never called.

2. In `applyAdditionalFilters` (line 146): `if parentIDFilter != "" && t.ParentID != parentIDFilter` — even if the function is reached, the empty-string check skips the filter.

**Defect type:** Logic error — flag default value collision with valid filter value

**Why it occurred:** The empty string `""` was used as both the zero value (meaning "flag not set") and as a meaningful filter value (meaning "match tasks with no parent"). No mechanism existed to distinguish the two cases.

**Contributing factors:** Cobra flags always have a default value. String flags default to `""`, making it impossible to detect explicit `--parent ""` through value inspection alone.

## Resolution for the Issue

**Changes made:**
- `cmd/find.go:106` — Use `cmd.Flags().Changed("parent")` to detect explicit flag usage
- `cmd/find.go:107-108` — Pass `parentFilterSet` bool to `applyAdditionalFilters`
- `cmd/find.go:130` — Add `parentFilterSet bool` parameter to function signature
- `cmd/find.go:148` — Replace `parentIDFilter != ""` guard with `parentFilterSet` check

**Approach rationale:** Cobra's `Changed()` method reliably detects whether a flag was explicitly set on the command line, regardless of its value. This cleanly separates "not set" from "set to empty string."

**Alternatives considered:**
- Using a sentinel value like `"*"` for "not set" — fragile and could collide with actual task IDs
- Using a separate `--top-level` boolean flag — adds unnecessary API surface when `--parent ""` already communicates the intent

## Regression Test

**Test file:** `cmd/find_test.go`
**Test names:** `TestFindParentEmptyFilterTopLevel`, `TestFindParentFilterNotSet`

**What it verifies:**
- `TestFindParentEmptyFilterTopLevel`: When `parentFilterSet` is true and `parentIDFilter` is `""`, only top-level tasks (ParentID == "") are returned
- `TestFindParentFilterNotSet`: When `parentFilterSet` is false, no parent filtering occurs even if `parentIDFilter` is `""`

**Run command:** `go test -run TestFindParent -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/find.go` | Use `cmd.Flags().Changed("parent")` to detect explicit flag usage; pass `parentFilterSet` to filter function |
| `cmd/find_test.go` | Add regression tests; update existing call sites with new parameter |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed that `applyAdditionalFilters` with `parentFilterSet=true` and `parentIDFilter=""` returns only top-level tasks
- Confirmed that `applyAdditionalFilters` with `parentFilterSet=false` returns all tasks regardless of parent

## Prevention

**Recommendations to avoid similar bugs:**
- When a flag's default value is also a meaningful filter value, always use `cmd.Flags().Changed()` to distinguish "not set" from "set to default"
- Document in code comments when a flag has this ambiguity

## Related

- Transit ticket: T-414
