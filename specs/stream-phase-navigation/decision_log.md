# Decision Log: Stream-Aware Phase Navigation

## Decision 1: Phase Selection Algorithm

**Date**: 2025-02-05
**Status**: accepted

### Context

When `--phase --stream N` is specified, the system needs to determine which phase to return tasks from. The current implementation finds the first phase with ANY pending tasks and then filters by stream, which fails when that phase lacks the requested stream.

### Decision

Find the first phase (in document order) that contains at least one unblocked pending task in stream N.

### Rationale

This approach ensures agents working in specific streams always get actionable work when it exists, regardless of how streams are distributed across phases. It follows the principle of least surprise for multi-stream workflows.

### Alternatives Considered

- **Return all phases with stream N tasks**: Would change the semantics of `--phase` from "single phase" to "multiple phases" - Rejected to maintain conceptual consistency
- **Require explicit phase name with stream**: Would add complexity and require users to know phase names - Rejected for usability

### Consequences

**Positive:**
- Multi-stream workflows become practical
- Agents get work when it exists in their stream
- Maintains single-phase semantics of `--phase` flag

**Negative:**
- Slightly more complex phase discovery logic
- May skip phases entirely for some streams (by design)

---

## Decision 2: Backward Compatibility

**Date**: 2025-02-05
**Status**: accepted

### Context

Existing users rely on `--phase` without `--stream` to get the first phase with any pending work. Changing this behavior could break existing workflows.

### Decision

Maintain current behavior when `--stream` is not specified. The stream-aware logic only activates when both flags are provided.

### Rationale

Backward compatibility prevents disruption to existing users and scripts. The new behavior is additive, not a replacement.

### Alternatives Considered

- **Always use stream-aware logic**: Would change behavior for all users - Rejected to avoid breaking changes
- **Add new flag like --stream-aware**: Would add flag proliferation - Rejected for simplicity

### Consequences

**Positive:**
- Zero impact on existing workflows
- Clear mental model: stream filter adds stream awareness

**Negative:**
- Two slightly different code paths for phase discovery

---

## Decision 3: Dependency Handling

**Date**: 2025-02-05
**Status**: accepted

### Context

When finding the next phase for a stream, the system must decide whether to include blocked tasks (tasks with incomplete dependencies) in the count of "available tasks."

### Decision

Only count unblocked tasks when determining if a phase has work available for a stream. Blocked tasks are excluded from both the phase selection and the returned results.

### Rationale

Returning blocked tasks is not actionable - the agent cannot work on them until dependencies complete. This aligns with the user's workflow where phases represent sequential work stages.

### Alternatives Considered

- **Include blocked tasks with dependency info**: Shows full picture but returns non-actionable items - Rejected because it doesn't match the "give me work I can do" use case
- **Separate flag for blocked task handling**: Adds complexity - Rejected for simplicity

### Consequences

**Positive:**
- Only actionable work is returned
- Agents don't need to filter blocked tasks themselves
- Matches sequential phase workflow expectations

**Negative:**
- May return "no tasks" when tasks exist but are blocked (intentional)

---

## Decision 4: No Verbose Skip Information

**Date**: 2025-02-05
**Status**: accepted

### Context

When phases are skipped because they lack stream N tasks, the system could inform the user about which phases were skipped.

### Decision

Silently skip phases without the requested stream. No verbose output about skipped phases.

### Rationale

The user wants results, not a report on what was skipped. Verbose output would clutter the response and is rarely useful in automated workflows.

### Alternatives Considered

- **Always show skipped phases**: Adds noise to output - Rejected
- **Show via --verbose flag**: Adds complexity for marginal benefit - Rejected

### Consequences

**Positive:**
- Clean, focused output
- Simpler implementation

**Negative:**
- Less visibility into why a particular phase was chosen (acceptable tradeoff)

---

## Decision 5: Return All Stream Tasks Including Blocked

**Date**: 2025-02-05
**Status**: accepted

### Context

When `--phase --stream N` finds a matching phase, should it return only ready (unblocked) tasks, or all tasks in stream N from that phase including blocked ones?

### Decision

Return all tasks in stream N from the selected phase, including blocked tasks. Include blocking status in output so agents can distinguish actionable from waiting tasks.

### Rationale

Returning all stream tasks gives agents visibility into the full scope of work in their stream for that phase. Agents can see intra-phase dependencies and understand what's coming next. The blocking status in output allows agents to filter for actionable work if needed.

### Alternatives Considered

- **Return only ready tasks**: Simpler, only actionable items returned - Rejected because agents lose visibility into upcoming work and intra-phase dependencies
- **Separate flag for blocked inclusion**: Would add complexity - Rejected for simplicity

### Consequences

**Positive:**
- Agents see full scope of their stream's work in the phase
- Intra-phase dependencies are visible
- Agents can plan ahead knowing what's blocked and why

**Negative:**
- Output may include non-actionable items (mitigated by blocking status indicator)
- Claim operations must still filter to ready tasks only

---
