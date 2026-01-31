# Batch Add-Phase Operation

## Overview

Add an "add-phase" operation type to the batch processing system. This allows agents and scripts to create phase headers through the JSON batch API, matching the functionality available via the `add-phase` CLI command. Agents frequently attempt to use this operation type, expecting it to exist alongside "add", "update", and "remove".

## Requirements

- The batch system MUST support an "add-phase" operation type that creates a new phase header
- The operation MUST accept the phase name in the "phase" field (reusing existing field)
- The operation MUST validate the phase name using `ValidatePhaseName()` (non-empty after trimming)
- The system MUST create the phase at the end of the document, after all existing content
- The system MUST update phase markers when creating a new phase
- The batch operation MUST be atomic with other operations in the same batch
- The system MUST NOT fail when creating a phase with a name that already exists (matches CLI behavior)
- The batch detection logic MUST route add-phase operations to the phase-aware execution path

## JSON Format

```json
{"type": "add-phase", "phase": "Planning"}
```

The "phase" field contains the name of the phase to create. This reuses the existing `Phase` field in the `Operation` struct with context-dependent meaning:
- For "add" operations: the phase to add the task TO
- For "add-phase" operations: the name of the phase to CREATE

## Implementation Approach

**Files to modify:**

1. `internal/task/batch.go`
   - Add `addPhaseOperation = "add-phase"` constant (after line 15)
   - Add validation case in `validateOperation()` (lines 225-298): check `op.Phase` is non-empty using `ValidatePhaseName()`
   - Add execution case in `applyOperationWithPhases()` (starts line 644) to create phase and update markers
   - Note: Only the phase-aware path needs modification; batch command routes add-phase to this path

2. `cmd/batch.go`
   - Update `hasPhaseOps` detection (around line 106-108) to also check for add-phase operation type
   - Update command documentation examples to include add-phase operation

3. Rune skill documentation (`~/.claude/skills/rune/SKILL.md`)
   - Update skill documentation to include add-phase as a supported batch operation

4. `internal/task/batch_operations_test.go`
   - Add test cases following existing map-based table pattern (see `TestExecuteBatch_PhaseAddOperation`)

**Execution behavior:**
- Operations execute in the order they appear in the JSON array
- An add-phase followed by an add to that phase will work (phase is created first)
- New phase marker uses `AfterTaskID` of the last task in the file, or empty string if no tasks exist

**Pattern to follow:**
- Match CLI `add-phase` behavior from `cmd/add_phase.go` (lines 59-63 for validation, 104-112 for appending)
- Use `addPhase(name)` from `internal/task/phase.go:15-19` for header creation
- Update `phaseMarkers` slice with new `PhaseMarker{Name: name, AfterTaskID: lastTaskID}`

**Dependencies:**
- `ValidatePhaseName()` in `internal/task/validation.go:8-15`
- `addPhase()` in `internal/task/phase.go:15-19`
- Existing `PhaseMarker` struct and phase marker management
- `RenderMarkdownWithPhases()` in `internal/task/render.go:158-241` for output

**Out of Scope:**
- Inserting phases at specific positions (only append to end)
- Phase renaming or deletion via batch operations
- Preventing duplicate phase names (matches existing CLI behavior)

## Risks and Assumptions

- **Risk:** Phase marker positioning when adding to file with tasks but no phases | **Mitigation:** Set `AfterTaskID` to the last task's ID; if no tasks exist, use empty string
- **Risk:** Batch ordering expectations | **Mitigation:** Document that operations execute in array order, so add-phase before add-to-phase works
- **Assumption:** The "phase" field semantic overload (create vs target) is acceptable since operation type provides context
- **Assumption:** Creating a phase at end of file is the expected behavior (matches CLI `add-phase` command)
- **Prerequisite:** Existing phase infrastructure (PhaseMarker, addPhase, WriteFileWithPhases) is functioning correctly
