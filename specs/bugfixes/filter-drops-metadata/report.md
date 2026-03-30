# Bugfix Report: filter-drops-metadata

**Date:** 2026-03-30
**Status:** Fixed
**Ticket:** T-630

## Description of the Issue

`TaskList.Filter` constructs partial `Task` copies in `filterTasks` (internal/task/search.go, lines ~165-178) and only keeps ID, Title, Status, Details, References, and ParentID. Fields like `StableID`, `Stream`, `BlockedBy`, `Owner`, and `Requirements` are omitted from filtered results.

**Reproduction steps:**
1. Build a task list with tasks that have `Stream`, `BlockedBy`, `Owner`, `StableID`, and `Requirements` populated.
2. Call `Filter` with any filter (even an empty one).
3. Observe that returned tasks have zero-values for all extended metadata fields.

**Impact:** Callers using `Filter` lose dependency/stream/ownership metadata even when the underlying tasks contain it. This can break downstream renderers or tooling that expects full task records.

## Investigation Summary

- **Symptoms examined:** Filtered tasks return with zeroed `StableID`, `Stream`, `BlockedBy`, `Owner`, `Requirements` fields.
- **Code inspected:** `internal/task/search.go` (`filterTasks` method), `internal/task/task.go` (Task struct definition).
- **Hypotheses tested:** Only one hypothesis needed — the explicit field-by-field copy omits the newer metadata fields.

## Discovered Root Cause

The `filterTasks` function creates result tasks using an explicit struct literal that lists only 6 of 11 non-Children fields. When `Requirements`, `StableID`, `BlockedBy`, `Stream`, and `Owner` were added to the Task struct, the Filter copy was not updated to include them.

**Defect type:** Incomplete struct copy (missing fields)

**Why it occurred:** The copy intentionally excludes `Children` to prevent non-matching descendants from leaking into results. However, the approach of listing fields explicitly means any new field added to the Task struct must also be manually added to the copy — and these five fields were missed.

**Contributing factors:** No test coverage for metadata preservation in filtered results. The field-by-field copy pattern is fragile when structs evolve.

## Resolution for the Issue

**Changes made:**
- `internal/task/search.go:171-178` — Copy the full task via struct assignment (`*task`), then zero out `Children` to maintain the existing exclusion semantics.

**Approach rationale:** Copying the entire struct and then clearing `Children` is both simpler and future-proof — any new fields added to Task will automatically be preserved without needing to update the filter copy logic.

**Alternatives considered:**
- Add the 5 missing fields explicitly to the struct literal — works but remains fragile; the same bug would recur when new fields are added.

## Regression Test

**Test file:** `internal/task/search_test.go`
**Test name:** `TestTaskList_Filter_PreservesExtendedMetadata`

**What it verifies:** That all extended metadata fields (Requirements, StableID, BlockedBy, Stream, Owner) are preserved in filtered results, while Children remain excluded.

**Run command:** `go test -run TestTaskList_Filter_PreservesExtendedMetadata -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/search.go` | Replace field-by-field copy with full struct copy + Children clear |
| `internal/task/search_test.go` | Add regression test for metadata preservation |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- Prefer whole-struct copies with selective field clearing over explicit field listing when the intent is "copy everything except X".
- Add metadata-preservation tests whenever Filter-like functions are introduced.

## Related

- Transit ticket T-630
