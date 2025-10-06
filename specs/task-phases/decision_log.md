# Task Phases Feature - Decision Log

## Overview
This document tracks key decisions made during the requirements phase for the task-phases feature.

## Decisions Made

### D1: Phase Creation Method
**Decision:** Implement `add-phase` command for creating new phases
**Date:** 2025-09-03
**Rationale:** User requested programmatic phase creation rather than manual markdown editing only
**Alternatives Considered:** Manual editing only
**Impact:** Requires new CLI command implementation

### D2: Task Placement Strategy  
**Decision:** Use `--phase` flag on `add` command to specify target phase
**Date:** 2025-09-03
**Rationale:** User requested ability to place tasks in specific phases during creation
**Alternatives Considered:** Always add to last phase, prompt for phase selection
**Impact:** Modifies existing `add` command interface

### D3: Phase Management Scope
**Decision:** No phase-specific operations (list, move, remove) in initial version
**Date:** 2025-09-03
**Rationale:** User specified "not right now" for these features
**Alternatives Considered:** Full phase CRUD operations
**Impact:** Reduces initial implementation scope

### D4: Empty Phase Handling
**Decision:** Preserve empty phases without automatic removal
**Date:** 2025-09-03
**Rationale:** User preference to maintain phase structure even when empty
**Alternatives Considered:** Auto-remove empty phases
**Impact:** Phase headers persist regardless of task content

### D5: Phase Ordering
**Decision:** Phases can be freely rearranged by users in markdown files
**Date:** 2025-09-03
**Rationale:** Maintains flexibility for manual document organization
**Alternatives Considered:** Fixed phase ordering, system-managed ordering
**Impact:** No ordering constraints in parser or renderer

### D6: Table Display Format
**Decision:** Add "Phase" column to table output when phases are present
**Date:** 2025-09-03
**Rationale:** User requested phase column for clear organization visibility
**Alternatives Considered:** Section headers, inline phase names
**Impact:** Modifies table rendering logic

### D7: Next Command Enhancement
**Decision:** Add `--phase` flag to `next` command to retrieve all tasks from next phase
**Date:** 2025-09-03
**Rationale:** User requested phase-aware task progression
**Alternatives Considered:** Separate command, phase name parameter
**Impact:** Enhances existing `next` command with phase logic

### D8: Optional Phase Support
**Decision:** Phases are completely optional - no impact on non-phase documents
**Date:** 2025-09-03
**Rationale:** User emphasized phases should not affect existing workflows
**Alternatives Considered:** Always show phase information
**Impact:** Conditional rendering and processing based on phase presence

### D9: Batch Command Phase Support
**Decision:** Support phases in batch operations with same behavior as individual commands
**Date:** 2025-09-03
**Rationale:** User requested consistency between individual and batch operations
**Alternatives Considered:** Phase-only batch operations, separate batch commands
**Impact:** Batch JSON operations need phase field support, phase creation logic

## Technical Design Decisions

### TD1: Phase Header Format
**Decision:** Use H2 markdown headers (`## Phase Name`) for phase boundaries
**Date:** 2025-09-03
**Rationale:** Standard markdown format, visually distinct, easy to parse
**Alternatives Considered:** H1 headers, special markers, YAML front matter
**Impact:** Parser must detect H2 headers as phase boundaries

### TD2: Task ID Continuity
**Decision:** Maintain sequential task numbering across all phases
**Date:** 2025-09-03
**Rationale:** Preserves existing ID format and ensures unique identifiers
**Alternatives Considered:** Restart numbering per phase, phase-prefixed IDs
**Impact:** ID renumbering logic must account for phase boundaries

## Questions Resolved

### Q1: Next Phase Definition (RESOLVED)
**Decision:** Return tasks from the first phase in document order containing pending tasks
**Date:** 2025-09-03
**Rationale:** Consistent with main `next` command behavior - find first available match
**Impact:** Next command searches phases sequentially from document start

### Q2: Non-Existent Phase Behavior (RESOLVED)  
**Decision:** Automatically create new phases when referenced in `add --phase`
**Date:** 2025-09-03
**Rationale:** User preference for automatic creation reduces friction
**Impact:** No error handling needed, phase creation is seamless

### Q3: Duplicate Phase Names (RESOLVED)
**Decision:** Use first occurrence when multiple phases have same name
**Date:** 2025-09-03
**Rationale:** Consistent with document order precedence, predictable behavior
**Impact:** Parser stops at first match, ignores subsequent duplicates

### Q4: Mixed Content Task Placement (RESOLVED)
**Decision:** Tasks without `--phase` flag are added at end of document
**Date:** 2025-09-03
**Rationale:** Simple, predictable placement that doesn't interfere with phase organization
**Impact:** Non-phased tasks consistently appear after all phase content

## Implementation Notes

- Phase detection during parsing should be efficient and non-intrusive
- Backward compatibility is critical - existing files must work unchanged
- Empty phase preservation requires careful handling in file operations
- Table rendering needs conditional logic for phase column display

## Design Decisions

### DD1: Phase Data Structure Design (REVISED)
**Decision:** No persistent phase storage - phases determined by position only
**Date:** 2025-09-03
**Rationale:** Eliminates data redundancy, no synchronization issues, simpler implementation
**Alternatives Considered:** Store phases in TaskList, add Phase field to Task (too complex, redundant)
**Impact:** True backward compatibility, no model changes required

### DD2: Task-Phase Association (REVISED)
**Decision:** Phase association calculated on-demand based on document position
**Date:** 2025-09-03
**Rationale:** Single source of truth (the document), no data to keep in sync
**Alternatives Considered:** Store phase name in Task struct (requires synchronization)
**Impact:** Slightly slower lookups but much simpler and more reliable

### DD3: Parser Enhancement Strategy
**Decision:** Detect H2 headers during parsing but don't store them in model
**Date:** 2025-09-03
**Rationale:** Phases are transient markers, not persistent data
**Alternatives Considered:** Store phase info in TaskList (creates synchronization issues)
**Impact:** Minimal parser changes, phases naturally preserved in markdown

### DD4: Phase Auto-Creation Location
**Decision:** Append auto-created phases at end of document
**Date:** 2025-09-03
**Rationale:** Predictable behavior, simple implementation, preserves existing structure
**Alternatives Considered:** Insert alphabetically, use predefined order (too complex)
**Impact:** Users know exactly where new phases will appear

### DD5: Manual Phase Reordering
**Decision:** Out of scope - assume phases stay in position
**Date:** 2025-09-03
**Rationale:** Rarely needed, adds significant complexity for edge case
**Alternatives Considered:** Full renumbering on reorder (complex and expensive)
**Impact:** Simplified implementation, documented limitation

### DD6: Next Phase Algorithm
**Decision:** First phase with pending tasks, starting from document beginning
**Date:** 2025-09-03
**Rationale:** Simple, predictable, matches linear workflow
**Alternatives Considered:** Track current context (too complex for value provided)
**Impact:** Clear behavior users can understand