# Requirements: Task Dependencies and Streams

## Introduction

This feature extends rune to support task dependencies and parallel work streams, enabling multiple agents to work on tasks concurrently. The key capabilities are:

1. **Stable IDs**: Each task receives a persistent identifier that survives renumbering operations, stored as an HTML comment in the markdown
2. **Dependencies**: Tasks can declare which other tasks must complete before they become "ready", stored as visible list items with stable ID references and title hints
3. **Streams**: Tasks can be assigned to numbered streams for work partitioning, enabling parallel agent execution
4. **Ownership**: Tasks can be claimed by agents, tracking who is actively working on what
5. **Enhanced `next` command**: Stream filtering, atomic claiming, and phase-level task distribution for orchestrator agents

The feature uses a hybrid storage approach: stable IDs are hidden in HTML comments (system-managed), while dependencies, streams, and owners are visible list items (user-editable).

---

## Requirements

### 1. Stable Task Identifiers

**User Story:** As a developer, I want each task to have a stable identifier that doesn't change when tasks are renumbered, so that dependencies remain valid after task additions or removals.

**Acceptance Criteria:**

1. <a name="1.1"></a>The system SHALL generate a unique stable ID for each task when the task is created via rune commands (add, batch)
2. <a name="1.2"></a>The system SHALL parse and preserve existing tasks without stable IDs, treating them as legacy tasks
3. <a name="1.3"></a>The system SHALL store stable IDs as HTML comments immediately following the task checkbox line in the format `<!-- id:XXXXXX -->`
4. <a name="1.4"></a>The system SHALL generate stable IDs as lowercase alphanumeric strings of exactly 7 characters using base36 encoding
5. <a name="1.5"></a>The system SHALL never display stable IDs in command output (table, JSON, or markdown formats)
6. <a name="1.6"></a>The system SHALL preserve stable IDs across all task operations (update, complete, move)
7. <a name="1.7"></a>The system SHALL ensure stable IDs are unique within a task file
8. <a name="1.8"></a>The system SHALL never reuse a stable ID within the same file, even after task deletion
9. <a name="1.9"></a>Legacy tasks without stable IDs SHALL NOT be eligible as targets for blocked-by references
10. <a name="1.10"></a>Legacy tasks without stable IDs SHALL still be able to have dependencies on tasks with stable IDs

### 2. Task Dependencies

**User Story:** As an orchestrator agent, I want to declare that certain tasks depend on other tasks completing first, so that agents only work on tasks whose prerequisites are satisfied.

**Acceptance Criteria:**

1. <a name="2.1"></a>The system SHALL support a `Blocked-by` metadata field on tasks as a list item
2. <a name="2.2"></a>The system SHALL store blocked-by references using stable IDs with title hints in the format `blocked-by:XXXXXX (Task Title)`
3. <a name="2.3"></a>The system SHALL allow multiple blocked-by references on a single task, comma-separated
4. <a name="2.4"></a>The system SHALL consider a task "ready" only when all tasks in its blocked-by list have status "completed"
5. <a name="2.5"></a>The system SHALL consider a task "blocked" when any task in its blocked-by list is not completed
6. <a name="2.6"></a>The system SHALL translate stable IDs to current hierarchical IDs when displaying dependencies in output
7. <a name="2.7"></a>WHEN a task referenced in a blocked-by list is deleted, the system SHALL display a warning AND remove the reference from all blocked-by lists
8. <a name="2.8"></a>The system SHALL support adding dependencies via the `update` command with `--blocked-by` flag accepting hierarchical IDs
9. <a name="2.9"></a>The system SHALL support adding dependencies via the `add` command with `--blocked-by` flag accepting hierarchical IDs
10. <a name="2.10"></a>The system SHALL support modifying dependencies via batch operations
11. <a name="2.11"></a>WHEN a blocked-by reference contains an invalid or non-existent stable ID, the system SHALL display a warning and ignore the invalid reference
12. <a name="2.12"></a>WHEN adding a dependency on a task without a stable ID (legacy task), the system SHALL return an error explaining the limitation
13. <a name="2.13"></a>The system SHALL NOT update title hints when referenced task titles change (title hints are informational only, set at dependency creation time)
14. <a name="2.14"></a>The system SHALL detect circular dependencies when adding or updating blocked-by references
15. <a name="2.15"></a>WHEN a dependency would create a circular chain (A depends on B, B depends on A, or longer cycles), the system SHALL return an error describing the cycle
16. <a name="2.16"></a>WHEN attempting to add a self-referential dependency (task depends on itself), the system SHALL return an error

### 3. Work Streams

**User Story:** As an orchestrator agent, I want to partition tasks into numbered streams, so that I can assign different agents to work on different streams in parallel.

**Acceptance Criteria:**

1. <a name="3.1"></a>The system SHALL support a `Stream` metadata field on tasks as a list item
2. <a name="3.2"></a>The system SHALL store stream values as positive integers
3. <a name="3.3"></a>The system SHALL default to stream 1 for tasks without an explicit stream assignment
4. <a name="3.4"></a>The system SHALL derive the list of available streams from task assignments (no upfront definition required)
5. <a name="3.5"></a>The system SHALL consider a stream "available" when it contains at least one ready task that is not in-progress
6. <a name="3.6"></a>The system SHALL display stream assignments in task output when present
7. <a name="3.7"></a>The system SHALL support setting stream via the `update` command with `--stream` flag
8. <a name="3.8"></a>The system SHALL support setting stream via the `add` command with `--stream` flag
9. <a name="3.9"></a>The system SHALL support modifying stream via batch operations
10. <a name="3.10"></a>WHEN an invalid stream value is provided (zero, negative, or non-integer), the system SHALL return an error explaining valid stream values

### 4. Task Ownership and Claiming

**User Story:** As a subagent, I want to claim tasks I'm working on, so that other agents know which tasks are already being handled.

**Acceptance Criteria:**

1. <a name="4.1"></a>The system SHALL support an `Owner` metadata field on tasks as a list item
2. <a name="4.2"></a>The system SHALL store owner values as arbitrary strings (agent identifiers)
3. <a name="4.3"></a>The system SHALL display owner in task output when present
4. <a name="4.4"></a>The system SHALL support setting owner via the `update` command with `--owner` flag
5. <a name="4.5"></a>The system SHALL support clearing owner via the `update` command with `--release` flag, which removes the Owner metadata line entirely
6. <a name="4.6"></a>The system SHALL support filtering tasks by owner via the `list` command with `--owner` flag
7. <a name="4.7"></a>The system SHALL support modifying owner via batch operations
8. <a name="4.8"></a>The system SHALL validate owner strings contain only printable characters excluding newlines
9. <a name="4.9"></a>WHEN an owner string contains invalid characters, the system SHALL return an error

### 5. Stream Status Command

**User Story:** As an orchestrator agent, I want to see the status of all streams, so that I can decide how to distribute work across subagents.

**Acceptance Criteria:**

1. <a name="5.1"></a>The system SHALL provide a `streams` command that displays stream status
2. <a name="5.2"></a>The system SHALL display for each stream: stream ID, count of ready tasks, count of blocked tasks, count of active (in-progress) tasks
3. <a name="5.3"></a>The system SHALL list which streams are currently available (have ready tasks)
4. <a name="5.4"></a>The system SHALL support `--available` flag to filter output to only available streams
5. <a name="5.5"></a>The system SHALL support `--json` flag for machine-readable output
6. <a name="5.6"></a>The JSON output SHALL include arrays of hierarchical task IDs (not stable IDs) for ready, blocked, and active categories per stream

### 6. Enhanced Next Command with Stream Support

**User Story:** As a subagent, I want to claim tasks from my assigned stream (either all ready tasks in the stream, or just the next available task), so that I can work without conflicting with other agents.

**Acceptance Criteria:**

1. <a name="6.1"></a>WHEN `--stream N` is provided without `--claim`, the `next` command SHALL return only tasks from stream N
2. <a name="6.2"></a>WHEN `--stream N --claim AGENT_ID` is provided, the `next` command SHALL claim all ready tasks in stream N within a single file write operation
3. <a name="6.3"></a>WHEN `--claim AGENT_ID` is provided without `--stream`, the `next` command SHALL claim the single next ready task from any stream
4. <a name="6.4"></a>The claim operation SHALL set the task status to in-progress AND set the owner to the provided agent ID
5. <a name="6.5"></a>WHEN a task is already claimed (has owner and is in-progress), the system SHALL skip it when finding next ready tasks
6. <a name="6.6"></a>WHEN claiming a stream, the system SHALL only claim tasks that are currently ready (not blocked)
7. <a name="6.7"></a>The `next` command SHALL only return tasks that are ready (all blocked-by tasks completed) when dependencies exist
8. <a name="6.8"></a>WHEN `--phase` is used, the output SHALL include stream and dependency information for all tasks in the phase
9. <a name="6.9"></a>WHEN `--phase --json` is used, the output SHALL include a streams summary with ready, blocked, and active task IDs per stream
10. <a name="6.10"></a>WHEN claiming, the JSON output SHALL distinguish between claimed tasks and remaining blocked tasks
11. <a name="6.11"></a>WHEN claiming a stream with no ready tasks, the system SHALL return a message indicating no tasks were claimed and exit successfully
12. <a name="6.12"></a>The system SHALL support combining `--phase` and `--stream` flags to filter phase tasks to a specific stream
13. <a name="6.13"></a>Concurrent claim operations by multiple agents MAY result in conflicts; the orchestrator is responsible for coordinating agent access to avoid duplicate claims

### 7. Markdown Storage Format

**User Story:** As a developer, I want the markdown file to remain human-readable while storing all necessary metadata, so that I can understand and edit task files directly when needed.

**Acceptance Criteria:**

1. <a name="7.1"></a>The system SHALL store stable IDs as HTML comments on the same line as the task checkbox, immediately after the title, in the format `- [ ] 1. Task title <!-- id:XXXXXX -->`
2. <a name="7.2"></a>The system SHALL store Blocked-by, Stream, and Owner as list item children of the task, consistent with Details and References
3. <a name="7.3"></a>The system SHALL parse metadata list items case-insensitively (e.g., "Blocked-by", "blocked-by", "BLOCKED-BY")
4. <a name="7.4"></a>The system SHALL preserve the order of metadata list items and original casing when round-tripping through parse and render
5. <a name="7.5"></a>The system SHALL support blocked-by references with title hints in the format `Blocked-by: id1 (Title 1), id2 (Title 2)`
6. <a name="7.6"></a>The system SHALL parse title hints by matching the stable ID followed by parenthesized content, allowing parentheses within task titles
7. <a name="7.7"></a>Files without the new metadata fields SHALL parse successfully with default values (no dependencies, stream 1, no owner)

### 8. List Command Enhancements

**User Story:** As a user, I want to see stream assignments and dependencies when listing tasks, so that I can understand the task structure and parallelization opportunities at a glance.

**Acceptance Criteria:**

1. <a name="8.1"></a>The `list` command SHALL display stream assignments for tasks when streams other than the default (stream 1) are present in the file
2. <a name="8.2"></a>The `list` command SHALL display blocked-by dependencies as hierarchical IDs when dependencies exist
3. <a name="8.3"></a>The `list` command SHALL support filtering by stream via `--stream` flag
4. <a name="8.4"></a>The `list` command JSON output SHALL include stream, blockedBy, and owner fields for each task
5. <a name="8.5"></a>WHEN no tasks have non-default streams, the stream column SHALL be omitted from table output to reduce clutter

### 9. Backward Compatibility

**User Story:** As a user with existing task files, I want my files to continue working after upgrading, so that I don't lose any data or functionality.

**Acceptance Criteria:**

1. <a name="9.1"></a>The system SHALL parse existing task files without stable IDs, dependencies, streams, or owners without error
2. <a name="9.2"></a>The system SHALL NOT modify existing files during parse operations (read-only parse)
3. <a name="9.3"></a>The system SHALL preserve all existing task metadata (Details, References, Requirements) unchanged
4. <a name="9.4"></a>Existing commands (list, add, update, complete, find) SHALL continue to work with default behavior when new flags are not used
5. <a name="9.5"></a>The system SHALL not require any manual migration steps from users
6. <a name="9.6"></a>Tasks created before this feature (without stable IDs) SHALL continue to function for all non-dependency operations

---

## Resolved Questions

1. **Maximum streams:** No maximum needed - adds unnecessary constraint.
2. **List tasks by owner:** Already covered by requirement 4.6 (`--owner` flag on list command).
3. **Cycle detection:** Added as requirements 2.14-2.16.
4. **Concurrency handling:** Documented as orchestrator responsibility (requirement 6.13).
5. **Completed task counts in streams command:** Not needed - this information is available from the list command.

