# Requirements: Stream-Aware Phase Navigation

## Introduction

This feature improves how the `next` command handles the combination of `--phase` and `--stream` flags. Currently, `--phase --stream N` finds the first phase with ANY pending tasks and then filters to stream N, which returns "no tasks available" when that phase lacks stream N tasks—even if later phases have stream N work available.

The improved behavior finds the first phase that has **ready tasks in the specified stream**, making multi-stream workflows practical where different phases may have different stream distributions.

## Definitions

| Term | Definition |
|------|------------|
| Phase | An H2 markdown header (`## Name`) that groups tasks. Tasks belong to the phase whose header most recently precedes them in document order. |
| Pending | A task with status `[ ]` (the `Pending` status value). |
| Ready | A task that is Pending, has no owner assigned, and has all blocked-by dependencies completed. |
| Blocked | A task that has incomplete dependencies per `DependencyIndex.IsBlocked()`. |
| Stream | A positive integer assigned to tasks via `Stream: N` metadata. Tasks without explicit stream assignment default to stream 1. |
| Document Order | The sequential position of elements as they appear in the markdown file from top to bottom. |

## Requirements

### 1. Stream-Aware Phase Discovery

**User Story:** As an agent working in a specific stream, I want `next --phase --stream N` to find the first phase with actionable work in my stream, so that I can get tasks even when earlier phases don't have work for my stream.

**Acceptance Criteria:**

1. <a name="1.1"></a>WHEN `--phase` and `--stream N` flags are both provided, the system SHALL find the first phase (in document order) that contains at least one ready task in stream N
2. <a name="1.2"></a>IF no phase contains ready tasks in stream N, the system SHALL return "No ready tasks found in stream N"
3. <a name="1.3"></a>WHEN a matching phase is found, the system SHALL return all tasks in stream N from that phase, including blocked tasks
4. <a name="1.4"></a>The system SHALL return tasks in document order within the selected phase
5. <a name="1.5"></a>Stream filtering SHALL apply to individual tasks based on their assigned or effective stream (defaulting to stream 1 if unassigned)
6. <a name="1.6"></a>Tasks that appear before the first phase header SHALL be excluded when `--phase` flag is used
7. <a name="1.7"></a>WHEN the document has no phase headers, `--phase --stream N` SHALL return "No ready tasks found in stream N" (phases are required for phase-based navigation)

### 2. Backward Compatibility

**User Story:** As an existing user of the `next --phase` command, I want the behavior to remain unchanged when I don't specify a stream, so that my existing workflows continue to work.

**Acceptance Criteria:**

1. <a name="2.1"></a>WHEN only `--phase` is provided (no `--stream`), the system SHALL maintain current behavior: find the first phase with any pending tasks (status != Completed)
2. <a name="2.2"></a>WHEN only `--stream N` is provided (no `--phase`), the system SHALL maintain current behavior: find the first ready task in stream N across all phases
3. <a name="2.3"></a>The system SHALL NOT change the output format for existing flag combinations

### 3. Dependency-Aware Phase Selection

**User Story:** As an agent, I want the system to find phases with actionable work, so that I'm directed to phases where I can make progress.

**Acceptance Criteria:**

1. <a name="3.1"></a>WHEN selecting which phase to return, the system SHALL only consider a phase viable if it has at least one ready task in stream N
2. <a name="3.2"></a>IF a phase has stream N tasks but none are ready (all blocked, owned, or not pending), the system SHALL skip that phase and check subsequent phases
3. <a name="3.3"></a>The system SHALL use the existing `DependencyIndex.IsBlocked()` mechanism to determine task blocking status
4. <a name="3.4"></a>The system SHALL treat tasks with non-existent blockers as blocked (existing safety behavior)
5. <a name="3.5"></a>Blocking status SHALL be evaluated using the dependency index regardless of stream assignment—a task in stream 2 can be blocked by a task in stream 1
6. <a name="3.6"></a>WHEN returning tasks from a selected phase, the output SHALL include blocking status for each task so agents can identify actionable vs waiting tasks

### 4. Claim Integration

**User Story:** As an agent, I want `--phase --stream N --claim AGENT_ID` to claim all ready tasks in my stream from the appropriate phase, so that I can reserve work atomically.

**Acceptance Criteria:**

1. <a name="4.1"></a>WHEN `--phase`, `--stream N`, and `--claim AGENT_ID` are combined, the system SHALL first find the appropriate phase using stream-aware discovery
2. <a name="4.2"></a>The system SHALL then claim all ready tasks in stream N from that phase
3. <a name="4.3"></a>The system SHALL set claimed tasks' status to InProgress and owner to AGENT_ID
4. <a name="4.4"></a>IF no claimable tasks exist, the system SHALL return "No ready tasks to claim in stream N"

### 5. Output Consistency

**User Story:** As a user, I want consistent output regardless of flag combination, so that I can parse and use the results reliably.

**Acceptance Criteria:**

1. <a name="5.1"></a>The system SHALL include the phase name in output when `--phase` flag is used
2. <a name="5.2"></a>The system SHALL include stream information in JSON output for all returned tasks
3. <a name="5.3"></a>The system SHALL include dependency/blocker information in JSON output
4. <a name="5.4"></a>The table and markdown output formats SHALL display the same task set as JSON output

## Example Scenarios

### Scenario 1: Basic Stream Filtering

Given a task file with:
```markdown
## Phase A
- [ ] 1. Task A1
  Stream: 1
- [ ] 2. Task A2
  Stream: 1

## Phase B
- [ ] 3. Task B1
  Stream: 1
- [ ] 4. Task B2
  Stream: 2
- [ ] 5. Task B3
  Stream: 2
```

| Command | Result |
|---------|--------|
| `next --phase` | Returns Task A1, A2 (first phase with pending tasks) |
| `next --phase --stream 1` | Returns Task A1, A2 (Phase A has ready stream 1 tasks) |
| `next --phase --stream 2` | Returns Task B2, B3 (Phase A skipped—no stream 2 tasks; Phase B has stream 2 tasks) |
| `next --stream 2` | Returns Task B2 (first ready task in stream 2, ignores phase boundaries) |

### Scenario 2: Intra-Phase Dependencies

Given a task file with:
```markdown
## Phase A
- [ ] 1. Task A1 <!-- id:abc1234 -->
  Stream: 2
- [ ] 2. Task A2
  Stream: 2
  Blocked-by: abc1234 (Task A1)
```

| Command | Result |
|---------|--------|
| `next --phase --stream 2` | Returns Task A1 AND A2 (both in stream 2; A2 marked as blocked) |
| `next --phase --stream 2 --claim agent` | Claims only Task A1 (A2 is blocked, cannot be claimed) |

### Scenario 3: All Stream Tasks Blocked

Given a task file with:
```markdown
## Phase A
- [ ] 1. Task A1 <!-- id:abc1234 -->
  Stream: 1

## Phase B
- [ ] 2. Task B1
  Stream: 2
  Blocked-by: abc1234 (Task A1)
```

| Command | Result |
|---------|--------|
| `next --phase --stream 2` | "No ready tasks found in stream 2" (Phase B has stream 2 but all are blocked) |

## Non-Requirements

- Verbose output showing skipped phases is not required
- Phase-aware behavior for `--stream N` without `--phase` is not required
- Returning tasks from multiple phases in a single call is not required
- File-level locking for concurrent access is out of scope; concurrent claim operations may result in last-write-wins behavior (existing limitation)
- Distinguishing "stream N has no ready tasks" from "stream N does not exist" is not required (streams are implicitly defined by task assignments)
- Limiting the number of tasks claimed with `--claim` is not required (existing behavior); consider `--limit` flag for future enhancement
- Error message text is not part of the stable API; scripts should check exit codes rather than parsing messages
