# Bugfix Report: renumber-drops-later-phase-markers

**Date:** 2025-07-16
**Status:** Fixed
**Ticket:** T-748

## Description of the Issue

The `rune renumber` command could silently drop phase markers (H2 headers) in
multi-phase files. In the worst case only the first phase header survived a
renumber; all subsequent phases were lost.

**Reproduction steps:**
1. Create a task file with 2+ phase headers and non-sequential task IDs.
2. Run `rune renumber <file>`.
3. Observe that later phase headers may be missing from the output.

**Impact:** Medium — data loss of phase structure in task files after
renumbering. The integration test logged this as a "known issue" with a
warning instead of failing.

## Investigation Summary

- **Symptoms examined:** Integration test
  `testRenumberWithPhases` used weak assertions (`len >= 1` instead of `== 2`)
  with a logged warning when the second phase marker was absent.
- **Code inspected:** `cmd/renumber.go` (conversion functions),
  `internal/task/render.go` (`RenderMarkdownWithPhases`),
  `internal/task/parse.go` (`ExtractPhaseMarkers`).
- **Hypotheses tested:**
  1. Position-based conversion drops out-of-bounds phases → **confirmed latent bug**.
  2. `extractTaskIDOrder` reads front matter and produces misaligned positions → **confirmed latent bug**.
  3. `RenderMarkdownWithPhases` silently skips markers with empty `AfterTaskID` in non-initial positions → **confirmed downstream effect**.

## Discovered Root Cause

Two latent defects in `cmd/renumber.go`:

1. **`extractTaskIDOrder` did not strip YAML front matter.** `ExtractPhaseMarkers`
   (used by `ParseFileWithPhases`) strips front matter before scanning, so the two
   functions could disagree on the set of task IDs. If front matter contained
   text matching the task-line regex, `extractTaskIDOrder` would produce extra
   entries, shifting all subsequent position mappings.

2. **`convertPhasePositionsToMarkers` silently dropped markers when position was
   out of bounds.** If a position exceeded `len(tl.Tasks)`, the marker's
   `AfterTaskID` was left as `""`. Markers with empty `AfterTaskID` in
   non-initial positions are never emitted by `RenderMarkdownWithPhases`,
   effectively deleting them from the file.

**Defect type:** Logic error + missing error recovery.

**Why it occurred:** The renumber pipeline was added in one commit and
subsequent changes (front matter support, phase rendering) introduced subtle
misalignments that the weak integration test did not catch.

## Resolution for the Issue

**Changes made:**
- `cmd/renumber.go` — `extractTaskIDOrder` now calls `task.StripFrontMatterLines`
  before scanning, aligning it with `ExtractPhaseMarkers`.
- `cmd/renumber.go` — `convertPhasePositionsToMarkers` now anchors out-of-bounds
  positions to the last task instead of silently producing an empty `AfterTaskID`.
- `cmd/integration_renumber_test.go` — Replaced weak `len >= 1` + warning
  assertions with strict `len == 2` checks for both phase markers.
- `cmd/renumber_test.go` — Added `TestRenumberPreservesAllPhaseMarkers` covering
  five scenarios (2-phase gaps, 3-phase gaps, phase-before-tasks, consecutive
  phases, idempotent renumber). Added `TestConvertPhasePositionsToMarkersOutOfBounds`
  for the out-of-bounds recovery. Added front-matter test case to
  `TestExtractTaskIDOrder`.

**Approach rationale:** Fix the root causes (front matter mismatch, silent
drop) and add comprehensive regression tests.

## Regression Test

**Test file:** `cmd/renumber_test.go`
**Test names:**
- `TestRenumberPreservesAllPhaseMarkers` — exercises 5 multi-phase scenarios
- `TestConvertPhasePositionsToMarkersOutOfBounds` — validates recovery from bad positions
- `TestExtractTaskIDOrder/with_front_matter` — validates front-matter stripping

**What it verifies:** After renumber, `ParseFileWithPhases` returns the same
number of phase markers as before, each with the correct name and updated
`AfterTaskID`.

**Run command:** `go test -run "TestRenumberPreservesAllPhaseMarkers|TestConvertPhasePositionsToMarkersOutOfBounds|TestExtractTaskIDOrder" -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/renumber.go` | Strip front matter in `extractTaskIDOrder`; recover gracefully in `convertPhasePositionsToMarkers` |
| `cmd/renumber_test.go` | Add regression tests for multi-phase renumbering and out-of-bounds positions |
| `cmd/integration_renumber_test.go` | Replace weak assertions with strict phase-marker count checks |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes (`make test-all`)
- [x] Code formatted (`make fmt`)

## Prevention

- When two functions scan the same content, ensure they apply the same
  pre-processing (e.g., front matter stripping).
- Avoid silently discarding data on unexpected conditions; anchor to a safe
  default or return an error.
- Use strict assertions in tests; "known issue" warnings mask real regressions.

## Related

- T-748: Renumber can drop later phase markers in multi-phase files
