# Bugfix Report: Remove Reports Wrong Title

**Date:** 2026-04-16
**Status:** Fixed
**Ticket:** T-801

## Description of the Issue

After removing a task that is not the last in the list, the `rune remove` command prints the title of the task that shifted into the removed task's position, instead of the title of the task that was actually removed.

**Reproduction steps:**
1. Create a task file with task 1 titled `Blocker` and task 2 titled `Dependent`
2. Run `rune remove <file> 1`
3. Observe the command prints `Removed task 1: Dependent` instead of `Removed task 1: Blocker`

**Impact:** Low severity — file mutation is correct, but user-facing output is misleading. Affects all output formats (table, JSON, markdown).

## Investigation Summary

- **Symptoms examined:** Output after remove shows wrong task title
- **Code inspected:** `cmd/remove.go` — the `runRemove` function
- **Hypotheses tested:** Pointer aliasing after slice mutation confirmed as root cause

## Discovered Root Cause

In `cmd/remove.go`, `targetTask := tl.FindTask(taskID)` returns a pointer into the `tl.Tasks` slice. After `tl.RemoveTaskWithPhases(taskID, content)` mutates and renumbers the slice, the pointer refers to whatever task shifted into that memory position — typically the next task in the original list.

**Defect type:** Use-after-mutation (stale pointer into mutated slice)

**Why it occurred:** The task info (title, child count) was read from the pointer after the slice was mutated, rather than being captured by value beforehand.

**Contributing factors:** Go slices are reference types; removing an element and compacting the slice causes subsequent elements to shift, making any pre-existing pointers into the slice unreliable.

## Resolution for the Issue

**Changes made:**
- `cmd/remove.go:94-96` — Capture `removedTitle` and `childCount` as value copies before calling `RemoveTaskWithPhases`
- `cmd/remove.go:117-151` — Replace all references to `targetTask.Title` with the captured `removedTitle` variable

**Approach rationale:** Minimal, surgical fix — capture the needed values by value before the mutation. No changes to the task engine or data structures required.

**Alternatives considered:**
- Making `RemoveTaskWithPhases` return removed task info — rejected as unnecessarily invasive change to the API

## Regression Test

**Test file:** `cmd/remove_test.go`
**Test name:** `TestRemoveReportsCorrectTitleAfterDeletion`

**What it verifies:** When removing task 1 (titled "Blocker") from a file with task 2 (titled "Dependent"), the output contains "Blocker" and does not contain "Dependent". Tests all three output formats.

**Run command:** `go test -run TestRemoveReportsCorrectTitleAfterDeletion -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/remove.go` | Capture title/childCount by value before mutation |
| `cmd/remove_test.go` | Add regression test for T-801 |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Code formatted (`make fmt`)

## Prevention

**Recommendations to avoid similar bugs:**
- When reading data from a struct pointer that will be mutated, capture needed fields by value before the mutation
- Consider returning removed-item metadata from mutation methods to avoid the pattern entirely
