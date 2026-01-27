# Decision Log: Task Dependencies and Streams

## Decision 1: Hybrid Storage Approach for Metadata

**Date**: 2025-01-27
**Status**: accepted

### Context

The feature requires storing several types of metadata on tasks: stable IDs, dependencies (blocked-by), streams, and owners. We needed to decide whether to store these as HTML comments (hidden) or as list items (visible), considering both human readability and system requirements.

### Decision

Use a hybrid approach: stable IDs are stored as HTML comments (hidden, system-managed), while dependencies, streams, and owners are stored as visible list items (user-editable).

### Rationale

Stable IDs are purely internal identifiers that users should never need to see or edit. Storing them as HTML comments keeps them out of the way while preserving them in the markdown. Dependencies, streams, and owners are user-facing concepts that users may want to manually set or modify, so they should be visible and follow the existing pattern of Details, References, and Requirements.

### Alternatives Considered

- **All HTML comments**: Would hide useful information from users viewing raw markdown
- **All list items**: Would expose internal stable IDs unnecessarily, cluttering the view

### Consequences

**Positive:**
- Users can manually edit streams, dependencies, and owners
- Stable IDs are protected from accidental modification
- Consistent with existing rune patterns for metadata

**Negative:**
- Two different storage mechanisms to implement and maintain
- Parser must handle both formats

---

## Decision 2: No Auto-assignment of Stable IDs to Existing Tasks

**Date**: 2025-01-27
**Status**: accepted (supersedes earlier draft decision)

### Context

Existing task files don't have stable IDs. We needed to decide whether to auto-assign IDs on parse or only assign IDs to newly created tasks.

### Decision

Only generate stable IDs for tasks created via rune commands (add, batch). Existing tasks without stable IDs are treated as "legacy tasks" that continue to work but cannot be referenced in dependencies.

### Rationale

Auto-assigning IDs on parse would cause unexpected file modifications during read operations, create issues with read-only files, and introduce potential race conditions when multiple processes parse the same file. Keeping parse operations side-effect-free is a cleaner design.

### Alternatives Considered

- **Auto-assign on parse**: Rejected due to side effects on read operations, read-only file issues, and concurrent access concerns
- **Generate on first dependency creation**: Adds complexity and unexpected behavior

### Consequences

**Positive:**
- Parse operations remain read-only (no side effects)
- Works correctly with read-only files
- No concurrent access issues during parsing
- Simpler mental model: new tasks get IDs, old tasks don't

**Negative:**
- Legacy tasks cannot be dependency targets
- Users must recreate tasks via rune commands if they want them to be dependency targets
- Two classes of tasks: those with stable IDs and those without

---

## Decision 3: Warn and Allow on Dependent Task Deletion

**Date**: 2025-01-27
**Status**: accepted

### Context

When a task that other tasks depend on is deleted, we needed to decide how to handle the orphaned references.

### Decision

Display a warning to the user but proceed with the deletion, removing the deleted task from all blocked-by lists.

### Rationale

Preventing deletion would be too restrictive and frustrating for users. Silent removal would hide potentially important information. A warning balances user awareness with workflow flexibility.

### Alternatives Considered

- **Error and prevent**: Too restrictive, forces users to manually update dependencies first
- **Silent removal**: Could lead to confusion about why tasks suddenly became unblocked

### Consequences

**Positive:**
- Users are informed about the impact of their action
- Workflow isn't blocked
- References are cleaned up automatically

**Negative:**
- Warning may be ignored
- Automatic cleanup could unblock tasks unexpectedly

---

## Decision 4: Stable IDs with Title Hints for Blocked-by References

**Date**: 2025-01-27
**Status**: accepted

### Context

Blocked-by references in the markdown need to identify which tasks a task depends on. We needed to choose between using hierarchical IDs (1.1, 1.2) which are human-readable but change on renumber, or stable IDs which are permanent but opaque.

### Decision

Use stable IDs with title hints in the format `blocked-by:XXXXXX (Task Title)`.

### Rationale

Stable IDs ensure references don't break when tasks are renumbered. Adding the title hint makes the raw markdown human-readable despite using opaque IDs. The title hint is informational only and not parsed for reference resolution.

### Alternatives Considered

- **Pure stable IDs**: Robust but completely opaque in raw markdown
- **Hierarchical IDs**: Human-readable but fragile - would require updating all references on any renumber operation

### Consequences

**Positive:**
- References survive renumbering operations
- Raw markdown remains understandable
- Title hints help users understand dependencies at a glance

**Negative:**
- Title hints may become stale if task titles are edited (accepted as known limitation)
- Slightly more complex parsing to handle the format

---

## Decision 5: Title Hints Are Not Auto-Updated

**Date**: 2025-01-27
**Status**: accepted

### Context

When a task title changes, the title hints in blocked-by references that point to that task could become stale. We needed to decide whether to automatically update these hints.

### Decision

Title hints are set at dependency creation time and are never automatically updated. They are purely informational.

### Rationale

Automatically updating title hints would require scanning the entire file for references on every title change, adding complexity and potential for unexpected file modifications. The stable ID is the authoritative reference; the title hint is just a convenience for human readers.

### Alternatives Considered

- **Auto-update title hints**: Adds complexity, performance overhead, and unexpected file changes
- **Remove title hints entirely**: Would make raw markdown harder to understand

### Consequences

**Positive:**
- Simple implementation
- No cascading file changes on title edits
- Predictable behavior

**Negative:**
- Title hints can become stale and misleading
- Users may need to manually update hints if they want them accurate

---

## Decision 6: Numeric Stream Identifiers

**Date**: 2025-01-27
**Status**: accepted

### Context

Streams are used to partition work across agents. We needed to decide whether streams should have names (strings) or just numeric identifiers.

### Decision

Streams are identified by positive integers only (1, 2, 3, etc.).

### Rationale

Numeric streams are simpler to implement and sufficient for the use case. The primary purpose is work partitioning for agents, not human categorization. Agents can easily iterate through numbered streams.

### Alternatives Considered

- **Named streams**: More human-friendly but adds complexity without clear benefit for agent orchestration

### Consequences

**Positive:**
- Simple implementation
- Easy to iterate (stream 1, stream 2, etc.)
- No naming conventions to enforce

**Negative:**
- Less descriptive than names like "backend" or "frontend"
- Users must remember what each stream number represents

---

## Decision 7: Stream-Level Claiming with --stream --claim

**Date**: 2025-01-27
**Status**: accepted

### Context

We needed to define how the `--claim` flag interacts with the `--stream` flag on the `next` command.

### Decision

When `--stream N --claim AGENT_ID` is used together, claim all currently ready tasks in stream N. When `--claim AGENT_ID` is used alone, claim only the single next ready task from any stream.

### Rationale

Stream-level claiming allows an orchestrator to assign an entire stream to a subagent at once, avoiding repeated back-and-forth. Single-task claiming without a stream supports simpler workflows where stream partitioning isn't needed.

### Alternatives Considered

- **Always claim single task**: Would require more round-trips between orchestrator and subagent
- **Always claim entire stream**: Would require specifying stream even for simple single-agent workflows

### Consequences

**Positive:**
- Efficient stream handoff to subagents
- Flexible for both simple and complex workflows
- Clear semantic distinction based on flag combination

**Negative:**
- Two different behaviors for `--claim` depending on context
- Users must understand the distinction

---

## Decision 8: Derived Stream List

**Date**: 2025-01-27
**Status**: accepted

### Context

We needed to decide whether streams should be defined upfront (in frontmatter or configuration) or derived from task assignments.

### Decision

Streams are purely derived from task assignments. No upfront definition is required.

### Rationale

This keeps the system simple and flexible. Users can start using streams immediately by adding `Stream: 2` to a task without any setup. The `streams` command shows whatever streams exist based on current assignments.

### Alternatives Considered

- **Upfront definition in frontmatter**: Adds ceremony without clear benefit
- **Predefined stream configuration**: Overly rigid for dynamic task management

### Consequences

**Positive:**
- Zero configuration required
- Streams appear and disappear naturally based on task state
- Simple mental model

**Negative:**
- No way to pre-define stream names or properties
- Empty streams don't exist (might be surprising)

---

## Decision 9: Basic Circular Dependency Detection

**Date**: 2025-01-27
**Status**: accepted

### Context

During requirements review, the question arose whether to detect circular dependencies (A depends on B, B depends on A) which would cause tasks to be permanently blocked.

### Decision

Implement basic cycle detection when adding or updating dependencies. The system will detect both self-references and longer dependency chains that form cycles, returning an error before the cycle is created.

### Rationale

While initially deferred, circular dependencies create permanently blocked tasks with no path to resolution. The cost of implementing basic cycle detection is low compared to the user confusion and debugging effort when cycles occur. A simple depth-first search during dependency updates is sufficient.

### Alternatives Considered

- **Defer cycle detection**: Would allow permanently blocked tasks with no clear error message
- **Warn but allow**: Could confuse users who don't understand why tasks never become ready

### Consequences

**Positive:**
- Prevents permanently blocked tasks
- Clear error messages when cycles would be created
- Users understand immediately when a dependency is invalid

**Negative:**
- Slight overhead when adding dependencies (graph traversal)
- More complex implementation than ignoring cycles

---

## Decision 10: Concurrent Access as Orchestrator Responsibility

**Date**: 2025-01-27
**Status**: accepted

### Context

When multiple agents attempt to claim tasks simultaneously, race conditions could result in duplicate work or conflicts. We needed to decide whether to implement file locking/optimistic locking or document the limitation.

### Decision

Document that concurrent claim operations may conflict and that the orchestrator is responsible for coordinating agent access to avoid duplicate claims. No file locking or optimistic locking is implemented.

### Rationale

Implementing true concurrency control in a file-based system adds significant complexity (file locking, retry logic, conflict resolution). For the primary use case of orchestrator-managed agents, the orchestrator can coordinate access by ensuring only one agent claims at a time or by assigning different streams to different agents. The claiming mechanism still writes in a single file operation, which provides basic atomicity at the filesystem level.

### Alternatives Considered

- **File locking**: Platform-dependent, complex to implement correctly, could cause deadlocks
- **Optimistic locking with version checks**: Adds complexity, requires read-modify-check-write pattern

### Consequences

**Positive:**
- Simple implementation
- No platform-specific locking code
- Fits the orchestrator-managed agent model well

**Negative:**
- Uncoordinated agents could claim the same task
- Users must understand the limitation when designing multi-agent workflows

---
