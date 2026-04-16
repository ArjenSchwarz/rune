# Bugfix Report: streams-available-json-stale-ids

**Date:** 2026-04-16
**Status:** Fixed
**Ticket:** T-703

## Description of the Issue

`rune streams <file> --available --json` could return an `available` array containing stream IDs that no longer appeared in the `streams` array after filtering. The `filterEmptyStreams` and `filterAvailableStreams` functions in `cmd/streams.go` copied the original `Available` slice verbatim instead of recomputing it from the filtered stream set.

**Reproduction steps:**
1. Construct a `StreamsResult` where `Available` includes a stream ID whose stream has no ready/blocked/active tasks.
2. Call `filterEmptyStreams` or `filterAvailableStreams`.
3. Observe `available` still contains the removed stream's ID.

**Impact:** Low-to-medium — downstream consumers parsing the JSON could attempt to use a stream ID from `available` that doesn't exist in `streams`, leading to incorrect agent scheduling or confusing output.

## Investigation Summary

- **Symptoms examined:** Both filter functions blindly copy `Available: result.Available`.
- **Code inspected:** `cmd/streams.go` lines 88–117 (`filterAvailableStreams`, `filterEmptyStreams`).
- **Hypotheses tested:** With the current `AnalyzeStreams` implementation, the stale IDs are unlikely to manifest end-to-end because `AnalyzeStreams` already computes `Available` consistently. However, the filter functions are independently wrong and would produce stale data given any upstream change.

## Discovered Root Cause

**Defect type:** Logic error — missing recomputation.

**Why it occurred:** Both filter functions were written to construct a new `StreamsResult` but simply assigned `Available: result.Available` instead of rebuilding `Available` from the streams that survived filtering.

**Contributing factors:** `Available` happens to be consistent after `AnalyzeStreams`, masking the filter-level bug in end-to-end tests.

## Resolution for the Issue

**Changes made:**
- `cmd/streams.go:88–101` — `filterAvailableStreams` now initialises `Available: []int{}` and appends each surviving stream's ID inline.
- `cmd/streams.go:104–119` — `filterEmptyStreams` now initialises `Available: []int{}` and appends each surviving stream's ID when it has ready tasks.

**Approach rationale:** Recomputing `Available` during the same loop that filters `Streams` is zero-cost and keeps the two fields permanently consistent by construction.

**Alternatives considered:**
- Adding a separate `recomputeAvailable` helper called after each filter — rejected because inlining is simpler and avoids a second pass.

## Regression Test

**Test file:** `cmd/streams_test.go`
**Test names:** `TestFilterEmptyStreamsRecomputesAvailable`, `TestFilterAvailableStreamsRecomputesAvailable`

**What they verify:** When given a `StreamsResult` with a deliberately stale `Available` array, the filter functions produce an `Available` array that only contains IDs present in the filtered `Streams`.

**Run command:** `go test -run "TestFilter(Empty|Available)StreamsRecomputes" -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/streams.go` | Recompute `Available` in both filter functions |
| `cmd/streams_test.go` | Add two regression tests for stale Available IDs |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes
- [x] Code formatted with `make fmt`

## Prevention

**Recommendations to avoid similar bugs:**
- When filtering a struct that contains derived/summary fields, always recompute those fields from the filtered data rather than copying from the original.
- Consider making `Available` a computed method on `StreamsResult` rather than a stored field, to eliminate the possibility of staleness.

## Related

- Transit ticket T-703
