# Bugfix Report: Filtered Markdown Places Promoted Children Under the Wrong Phase

**Date:** 2026-11-02
**Status:** In progress
**Ticket:** T-1352

## Description of the Issue

`rune list --filter ... --format markdown` rendered a matching child under the
first phase when its non-matching parent was filtered out.

**Reproduction steps:**
1. Create a file with phases `Planning` (task 1, completed) and `Build`
   (task 2, completed, with pending child 2.1).
2. Run `rune list tasks.md --filter pending --format markdown`.
3. Observe `- [ ] 2.1. ...` rendered under `## Planning`, followed by an empty
   `## Build`.

**Impact:** Medium. Filtered markdown output misattributes tasks to phases.

## Investigation Summary

- **Symptoms examined:** Markdown output for the repro above, reproduced on
  main at 2ab5286.
- **Code inspected:** `cmd/list.go` (`filterTasksRecursive`, markdown
  rendering), `internal/task/render.go` (`RenderMarkdownWithPhases`).
- **Hypotheses tested:** The filter correctly promotes 2.1 to the top level of
  the filtered list and keeps its original ID. The JSON renderer was not
  examined beyond confirming this bug is in the markdown renderer.

## Discovered Root Cause

`RenderMarkdownWithPhases` looks up each rendered task's position with
`positionOf[tl.Tasks[i].ID]`, where `positionOf` only holds the top-level IDs
of the phase source. A promoted child's ID (`2.1`) is not in the map, so the
lookup returns Go's zero value, position 0, which is in the first phase.

**Defect type:** Logic error (missing-key lookup returning a zero value).

**Why it occurred:** The phase-source lookup added for T-698 assumed every
rendered top-level task is also top-level in the source; filter promotion
breaks that assumption.

## Resolution for the Issue

_To be filled in after the fix._

## Regression Test

**Test file:** `internal/task/render_phase_test.go`
**Test name:** `TestRenderMarkdownWithPhases_PromotedChildKeepsPhase`

**What it verifies:** A promoted child `2.1` renders under its parent's
`Build` phase, after the empty `Planning` header.

**Run command:** `go test ./internal/task -run PromotedChildKeepsPhase`

## Related

- T-1352, T-698
