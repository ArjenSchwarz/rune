# Bugfix Report: next-phase-claim

**Date:** 2026-03-30
**Status:** Fixed

## Description of the Issue

Running `rune next --phase --claim AGENT` ignored the `--phase` flag and claimed only a single ready task instead of all ready tasks from the next phase.

**Reproduction steps:**
1. Create a task file with phases where the first phase has multiple ready tasks.
2. Run `rune next tasks.md --phase --claim agent-a`.
3. Observe only one task is claimed instead of all ready tasks from the next phase.

**Impact:** Agents using `--phase --claim` to batch-claim phase work received only one task, requiring repeated invocations and defeating the purpose of phase-based claiming.

## Investigation Summary

- **Symptoms examined:** `--phase --claim` with no `--stream` claimed exactly one task
- **Code inspected:** `cmd/next.go` — `runNextWithClaim()` switch statement (lines 191–240)
- **Hypotheses tested:** The switch statement had cases for `phaseFlag && streamFlag > 0`, `streamFlag > 0`, and a default — but no case for `phaseFlag` alone

## Discovered Root Cause

**Defect type:** Missing switch case (logic gap)

**Why it occurred:** The `runNextWithClaim` switch statement handled `--phase --stream --claim` and `--stream --claim`, but had no case for `--phase --claim` (without `--stream`). When `phaseFlag=true` and `streamFlag=0`, neither the first case (`phaseFlag && streamFlag > 0` → false) nor the second (`streamFlag > 0` → false) matched, so execution fell through to the `default` case which claims only the single first ready task.

**Contributing factors:** The `runNextPhase()` function (non-claim path) correctly handled `--phase` without `--stream`, but the claim path was not updated to mirror this when claim support was added.

## Resolution for the Issue

**Changes made:**
- `cmd/next.go:210-225` — Added `case phaseFlag:` between the `phaseFlag && streamFlag > 0` case and the `streamFlag > 0` case. This new case calls `task.FindNextPhaseTasks(filename)` to discover all pending tasks in the next phase, then filters to ready tasks using `isTaskReady`, matching the pattern used by the phase+stream case.

**Approach rationale:** Mirrors the existing `phaseFlag && streamFlag > 0` case structure but uses `FindNextPhaseTasks` (the non-stream variant), consistent with how `runNextPhase` handles the same flag combination in the non-claim path.

**Alternatives considered:**
- Refactoring the switch into if/else with shared phase resolution — rejected as higher risk for a targeted fix

## Regression Test

**Test file:** `cmd/next_test.go`
**Test names:**
- `TestNextCommandPhaseClaimClaimsAllReadyTasksInPhase`
- `TestNextCommandPhaseClaimExcludesBlockedTasks`

**What they verify:**
1. `--phase --claim` claims ALL ready tasks from the first phase (3 tasks), not just one
2. `--phase --claim` correctly excludes blocked tasks within the phase

**Run command:** `go test -run "TestNextCommandPhaseClaimClaimsAllReadyTasksInPhase|TestNextCommandPhaseClaimExcludesBlockedTasks" -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/next.go` | Added `case phaseFlag:` in `runNextWithClaim` switch |
| `cmd/next_test.go` | Added two regression tests for phase-only claim |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes (`make test`)
- [x] No lint regressions introduced

**Manual verification:**
- Confirmed regression tests fail before fix and pass after

## Prevention

**Recommendations to avoid similar bugs:**
- When adding flag combinations, enumerate all valid permutations and ensure each is handled
- Add a matrix-style test covering all flag combinations for multi-flag commands

## Related

- Transit ticket: T-637
