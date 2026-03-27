# Bugfix Report: empty-phase-subtasks

**Date:** 2026-03-27
**Status:** Fixed

## Description of the Issue

When `AddTaskToPhase` is called with a `parentID` (adding a subtask), the function unconditionally runs its phase-finding/creation logic before checking whether the task is a subtask. If the specified phase does not exist, a phantom empty phase marker is appended to the file. Even when the phase exists, the insert-position calculation runs unnecessarily for subtasks.

**Reproduction steps:**
1. Create a task file with a Planning phase and two tasks
2. Run `rune add --parent 1 --phase "Phase X" --title "Child"` where "Phase X" does not exist
3. Observe that the file now contains an empty `## Phase X` marker at the end with no top-level tasks

**Impact:** Files accumulate phantom phase markers that confuse both users and programmatic consumers. The phantom phases have no tasks and break the expected document structure.

## Investigation Summary

- **Symptoms examined:** Phantom `## Phase X` headers appearing at end of file after adding subtasks with `--phase`
- **Code inspected:** `AddTaskToPhase` in `internal/task/operations.go` (lines 519-662), phase rendering in `render.go`, CLI integration in `cmd/add.go`
- **Hypotheses tested:** Confirmed that the phase-creation code path (lines 550-562) runs unconditionally before the `parentID` check (line 600)

## Discovered Root Cause

**Defect type:** Logic error — missing early return for subtask case

**Why it occurred:** The function was structured with phase logic first (find/create phase, calculate insert position) and parent-handling second. When `parentID` is set, the subtask is added via `tl.AddTask(parentID, title, "")` which ignores the calculated `insertPosition`, but the phase marker mutation has already happened.

**Contributing factors:** Subtasks inherit their parent's phase implicitly (they are nested under a top-level task that belongs to a phase). The function did not account for this — it treated all tasks the same for phase purposes.

## Resolution for the Issue

**Changes made:**
- `internal/task/operations.go:538-549` - Added early-return block: when `parentID` is set, skip all phase-finding/creation logic, add the subtask via `tl.AddTask`, and write the file with the original unmodified phase markers.

**Approach rationale:** Subtasks belong to whatever phase their parent task is in. The `--phase` flag is meaningless for subtasks since phase membership is determined by root-level task position. The simplest and safest fix is to bypass all phase mutation when adding a subtask.

**Alternatives considered:**
- Reject `--phase` with `--parent` at the CLI level — would be a breaking change and overly restrictive
- Validate phase exists before proceeding — wouldn't fix the core issue of unnecessary phase creation for subtasks

## Regression Test

**Test file:** `internal/task/phase_operations_test.go`
**Test name:** `TestAddTaskToPhaseSubtaskNoPhantomPhase`

**What it verifies:** Three scenarios:
1. Subtask with non-existent phase → no phantom phase marker created
2. Subtask with existing phase → no extra markers or formatting changes
3. Subtask with non-existent phase in a file with no phases → no phantom phase created

**Run command:** `go test -run TestAddTaskToPhaseSubtaskNoPhantomPhase -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/operations.go` | Early-return for subtasks in `AddTaskToPhase` |
| `internal/task/phase_operations_test.go` | Regression tests for T-569 |
| `specs/bugfixes/empty-phase-subtasks/report.md` | This report |

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

**Manual verification:**
- Tested that `AddTaskToPhase` with `parentID` no longer creates phantom phases

## Prevention

**Recommendations to avoid similar bugs:**
- Handle special cases (subtasks) at the top of functions before running the general case logic
- Consider adding a lint check that warns when `parentID` and `phaseName` are both non-empty at the CLI level

## Related

- T-569: Prevent AddTaskToPhase from creating empty phases for subtasks
- T-371: Previous fix for phase marker updates (related `AddTaskToPhase` logic)
