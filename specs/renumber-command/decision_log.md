# Decision Log: Renumber Command

## Decision 1: No User Confirmation Required
**Date**: 2025-11-11
**Decision**: The renumber command will execute immediately without asking for confirmation.
**Rationale**: User explicitly requested no confirmation dialog. Backup file provides safety net if issues occur.
**Alternatives Considered**:
- Always confirm with preview
- Optional --yes flag
**Status**: Accepted

## Decision 2: Automatic Backup Creation
**Date**: 2025-11-11
**Decision**: Always create a `.bak` file automatically before renumbering.
**Rationale**: Provides recovery mechanism if renumbering produces unexpected results. User can manually delete after verification.
**Alternatives Considered**:
- No backup (too risky)
- Optional --backup flag (adds complexity)
**Status**: Accepted

## Decision 3: Renumber Everything (Hierarchical Sequential)
**Date**: 2025-11-11
**Decision**: The command will renumber all tasks with hierarchical sequential numbering (1, 1.1, 1.2, 2, 2.1...) across phases, including all tasks and subtasks regardless of hierarchy depth. Numbering is global (continuous across the entire file) rather than per-phase.
**Rationale**: Maintains the existing hierarchical ID system where parent-child relationships are encoded in the task IDs. This is the established rune pattern and what the existing renumberTasks() function implements.
**Alternatives Considered**:
- Preserve phase boundaries (per-phase numbering 1.1, 1.2, 2.1, 2.2)
- Smart detection based on current state
**Status**: Accepted

## Decision 4: Support Multiple Output Formats
**Date**: 2025-11-11
**Decision**: Support --format flag with values: table (default), markdown, json. Output will include task count and backup file location.
**Rationale**: Enables integration with different workflows and programmatic use via JSON.
**Alternatives Considered**:
- Table format only (too limiting)
- Minimal output (less useful for debugging)
**Status**: Accepted

## Decision 5: Fill Numbering Gaps Automatically
**Date**: 2025-11-11
**Decision**: Always renumber to create sequential IDs (e.g., convert 1, 2, 5 to 1, 2, 3).
**Rationale**: Main purpose of the command is to fix numbering inconsistencies.
**Alternatives Considered**:
- Warn about gaps but let user decide
- Preserve intentional gaps
**Status**: Accepted

## Decision 6: No Requirement Link Validation or Updates
**Date**: 2025-11-11
**Decision**: Preserve requirement links exactly as they appear without validation or modification.
**Rationale**: Keeps the command focused on structural renumbering. Updating requirement links would require parsing task descriptions and understanding link semantics, adding significant complexity. Users can manually fix broken links after renumbering if needed.
**Trade-offs**: May result in broken cross-references if tasks with requirement links are renumbered. Acceptable because:
- Performance: Avoids parsing and rewriting all task descriptions
- Simplicity: Command stays focused on numbering structure
- User expectation: Renumber fixes structure, not content
**Status**: Accepted

## Decision 7: Backup Timing Strategy
**Date**: 2025-11-11
**Decision**: Backup file should be created after successful parsing but before any write operations.
**Rationale**: User wants backup to capture the original state before any modifications. Creating after parsing ensures we only backup valid files.
**Alternatives Considered**:
- Create backup after successful renumbering
- Create both before and after backups
**Status**: Accepted

## Decision 8: Leverage Existing renumberTasks() Function
**Date**: 2025-11-11
**Decision**: Use existing `TaskList.renumberTasks()` method (operations.go:168-173) for all renumbering logic.
**Rationale**: The existing implementation already handles hierarchical renumbering correctly. Used successfully by AddTask and RemoveTask operations. Tested extensively in existing test suite. No need to duplicate or modify this logic.
**Implementation Detail**: Make renumberTasks() method public by renaming to RenumberTasks() if currently private.
**Status**: Accepted

## Decision 9: Use Existing Validation Functions
**Date**: 2025-11-11
**Decision**: Use existing validation functions from operations.go instead of creating new ones.
**Rationale**: validateFilePath() already handles security constraints, ParseFileWithPhases() validates file format, and resource limits are checked in checkResourceLimits(). No need to create duplicate validation logic.
**Status**: Accepted

## Decision 10: Use go-output Library for Table Format
**Date**: 2025-11-11
**Decision**: Use github.com/ArjenSchwarz/go-output/v2 for table output formatting.
**Rationale**: Already used by list command, provides consistent table rendering across all commands, handles terminal width and formatting automatically.
**Status**: Accepted

## Decision 11: Implement Depth Validation in Parser
**Date**: 2025-11-11
**Decision**: Implement hierarchy depth validation during the parsing phase instead of adding a new GetMaxDepth() method.
**Rationale**: ParseFileWithPhases() already validates hierarchy structure. Parser can track depth while building the tree, avoiding extra tree traversal after parsing. Depth validation is primarily a parse-time concern.
**Status**: Accepted

## Decision 12: Export CountTotalTasks() Method
**Date**: 2025-11-11
**Decision**: Export existing countTotalTasks() method by capitalizing it to CountTotalTasks().
**Rationale**: Needed for output summary and resource limit validation. The method already exists (operations.go:444-460), just needs to be made public.
**Status**: Accepted

## Decision 13: Phase Marker Adjustment After Renumbering
**Date**: 2025-11-11
**Decision**: After renumberTasks() is called, adjust all PhaseMarker AfterTaskID values to reflect the new task IDs by extracting root task numbers and reformatting them.
**Rationale**: Phase markers store task IDs as strings. When renumberTasks() changes task IDs, the phase markers become stale and must be updated. This follows the same pattern as RemoveTaskWithPhases (operations.go:714-734).
**Implementation**: Use a simple approach that extracts the root task number from each AfterTaskID and reformats it, since renumberTasks() maintains task order.
**Status**: Accepted
