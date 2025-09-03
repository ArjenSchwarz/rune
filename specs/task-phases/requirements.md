# Task Phases Feature Requirements

## Introduction

This feature introduces hierarchical organization of tasks using phases - high-level sections that group related tasks under descriptive headers. Phases provide logical grouping of tasks while maintaining the existing task ID hierarchy and state management capabilities.

The feature enables users to structure their task lists with semantic sections (e.g., "Planning", "Implementation", "Testing") while preserving the sequential task numbering across the entire document.

## Requirements

### 1. Phase Header Support

**User Story:** As a user, I want to organize my tasks under phase headers, so that I can logically group related tasks and improve document readability.

**Acceptance Criteria:**
1.1. The system SHALL recognize markdown H2 headers (`## Phase Name`) as phase boundaries in task documents
1.2. The system SHALL preserve phase headers when parsing and rendering task lists
1.3. The system SHALL allow any text content for phase names without restrictions on naming conventions
1.4. The system SHALL maintain phase headers in their original position relative to tasks during file operations
1.5. The system SHALL support multiple phases within a single task document
1.6. The system SHALL handle documents with mixed content (phases with tasks and tasks without phases)

### 2. Task ID Continuity Across Phases

**User Story:** As a user, I want task IDs to continue sequentially across phases, so that I have a consistent numbering scheme throughout the document.

**Acceptance Criteria:**
2.1. The system SHALL maintain sequential task ID numbering across phase boundaries (e.g., Phase 1: tasks 1,2; Phase 2: tasks 3,4)
2.2. The system SHALL renumber all subsequent tasks when a task is removed from any phase
2.3. The system SHALL maintain hierarchical sub-task numbering within each phase (e.g., 3.1, 3.2 under task 3)
2.4. The system SHALL preserve the existing task ID format `^\d+(\.\d+)*$` across phases
2.5. The system SHALL handle task additions and removals correctly regardless of which phase contains the operation

### 3. Phase Creation and Management

**User Story:** As a user, I want to create new phases using a dedicated command, so that I can organize my task structure programmatically.

**Acceptance Criteria:**
3.1. The system SHALL provide an `add-phase` command to create new phase headers
3.2. The system SHALL add new phases as H2 markdown headers (`## Phase Name`) 
3.3. The system SHALL append new phases to the end of the document by default
3.4. The system SHALL preserve empty phases without automatically removing them
3.5. The system SHALL allow users to manually rearrange phases in the markdown file
3.6. The system SHALL handle documents where phases are mixed with non-phased tasks

### 4. Phase-Aware Task Operations

**User Story:** As a user, I want to add tasks to specific phases, so that I can organize work within logical groupings.

**Acceptance Criteria:**
4.1. The system SHALL support a `--phase` flag for the `add` command to specify target phase
4.2. The system SHALL add tasks to the specified phase when the --phase flag is provided
4.3. The system SHALL automatically create new phases when a non-existent phase name is specified with --phase flag
4.4. The system SHALL add tasks to the end of the document when no --phase flag is specified
4.5. The system SHALL use the first occurrence when duplicate phase names exist
4.6. The system SHALL support removing tasks from any phase while maintaining phase structure
4.7. The system SHALL support updating task content and state within phases
4.8. The system SHALL preserve phase headers when performing batch operations
4.9. The system SHALL handle task state changes (pending/in-progress/completed) within phases

### 5. Phase Display and Filtering

**User Story:** As a user, I want to view tasks with phase information, so that I can understand the organizational structure.

**Acceptance Criteria:**
5.1. The system SHALL display a "Phase" column in table output format when phases are present
5.2. The system SHALL include phase context in JSON output for programmatic access
5.3. The system SHALL maintain phase information in all supported output formats (table, markdown, JSON)
5.4. The system SHALL show empty cells in the Phase column for tasks that are not within any phase
5.5. The system SHALL clearly indicate which phase each task belongs to in all output formats

### 6. Batch Command Phase Support

**User Story:** As a user, I want to perform batch operations on tasks within specific phases, so that I can efficiently manage phase-organized work.

**Acceptance Criteria:**
6.1. The system SHALL support phase-aware operations in batch JSON commands
6.2. The system SHALL allow specifying target phases for "add" operations in batch files
6.3. The system SHALL automatically create phases referenced in batch operations that don't exist
6.4. The system SHALL use first occurrence when batch operations reference duplicate phase names
6.5. The system SHALL preserve phase structure during batch operations
6.6. The system SHALL include phase information in batch operation responses when applicable
6.7. The system SHALL handle mixed batch operations (some with phases, some without) correctly

### 7. Next Command Phase Support

**User Story:** As a user, I want to retrieve all tasks from the next phase, so that I can focus on the upcoming work phase.

**Acceptance Criteria:**
7.1. The system SHALL support a `--phase` flag for the `next` command
7.2. The system SHALL return all pending tasks from the next phase when `--phase` flag is provided
7.3. The system SHALL determine the "next phase" as the first phase in document order containing pending tasks
7.4. The system SHALL return an appropriate message if no next phase with pending tasks exists
7.5. The system SHALL maintain existing `next` command behavior when `--phase` flag is not used

### 8. Backward Compatibility and Optional Phase Support

**User Story:** As a user with existing task files, I want phase support to be completely optional, so that my current workflow remains unaffected.

**Acceptance Criteria:**
8.1. The system SHALL continue to work with existing task files that do not use phases
8.2. The system SHALL not require phase headers for normal task operations
8.3. The system SHALL maintain existing task ID behavior for documents without phases
8.4. The system SHALL handle mixed documents (some tasks in phases, some not) gracefully
8.5. The system SHALL preserve all existing CLI command functionality for non-phase documents
8.6. The system SHALL not display Phase column in table output when no phases are present in the document
8.7. The system SHALL not include phase information in JSON output when phases are not used