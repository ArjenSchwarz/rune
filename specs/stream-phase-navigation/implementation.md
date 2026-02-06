# Implementation Explanation: Stream-Aware Phase Navigation

## Beginner Level

### What Changed

rune is a command-line tool that manages task lists stored as markdown files. Tasks can be organized into **phases** (sections marked with `##` headers) and assigned to **streams** (numbered work lanes for parallel agents).

Before this change, running `rune next --phase --stream 2` would find the first phase with *any* pending tasks and then filter to stream 2. If that first phase had no stream 2 tasks, you'd get "no tasks available" — even if a later phase had stream 2 work ready.

Now the command finds the first phase that actually has **ready** stream 2 tasks, skipping phases where stream 2 either doesn't exist or is entirely blocked.

### Why It Matters

When multiple AI agents work in parallel on different streams, each agent needs to find actionable work in their stream. Without this change, agents would get stuck returning empty results when earlier phases happened to not include their stream — making multi-stream workflows impractical.

### Key Concepts

- **Phase**: A section in a task file under an `## H2 Header` — phases are worked through in order
- **Stream**: A numbered lane (1, 2, 3...) that lets multiple agents work in parallel on different task groups
- **Ready task**: Pending, no one owns it, and nothing is blocking it
- **Blocked task**: Has dependencies that aren't completed yet

---

## Intermediate Level

### Changes Overview

| File | Purpose |
|------|---------|
| `internal/task/next.go` | New `FindNextPhaseTasksForStream` + `hasReadyTaskInStream` functions |
| `cmd/next.go` | Modified `runNextPhase` and `runNextWithClaim` to use stream-aware discovery; added blocking status to all output formats |
| `internal/task/next_test.go` | 15 unit test cases for `FindNextPhaseTasksForStream`, 10 for `hasReadyTaskInStream` |
| `cmd/next_test.go` | Unit tests for table/markdown/JSON blocking output and phase+stream+claim combinations |
| `cmd/integration_test.go` | 7 integration tests exercising end-to-end scenarios via the compiled binary |

### Implementation Approach

**Core function** — `FindNextPhaseTasksForStream(filepath, stream)`:
1. Parses the markdown file and builds a `DependencyIndex`
2. Extracts phases by scanning for H2 headers and associating top-level tasks
3. Iterates phases in document order, calling `hasReadyTaskInStream` for each
4. On the first match, returns ALL non-completed stream tasks from that phase (including blocked ones, per Decision 5)
5. Returns `nil` if no phase qualifies or if the document has no phases

**Phase selection vs. task return** — The distinction matters: selection uses strict "ready" semantics (pending + no owner + not blocked), but the returned set includes blocked tasks so agents can see the full scope of upcoming work.

**Backward compatibility** — `runNextPhase` conditionally delegates: when `streamFlag > 0`, it calls the new function; otherwise it calls the existing `FindNextPhaseTasks`. No existing behavior changes.

**Output enhancements** — Blocking status is computed at render time using the `DependencyIndex`, not stored in the task data:
- JSON: `blocked` boolean + `blockedBy` array with hierarchical task IDs
- Table: `(ready)` or `(blocked)` appended to status
- Markdown: `(blocked by: 1, 2)` appended to task title

### Trade-offs

- **File parsed twice in `runNextWithClaim`**: The file is parsed once in the command handler (for the full task list) and again inside `FindNextPhaseTasksForStream`. This duplicates work but keeps the API simple — the alternative would be passing pre-parsed data, coupling the internal function to the caller's parse context.
- **Returns all stream tasks vs. only ready**: Decision 5 chose visibility over minimalism. Agents get the full picture of what's coming, with blocking metadata to distinguish actionable from waiting tasks.
- **No verbose skip output**: Decision 4 chose silent skipping over logging which phases were skipped, keeping automated workflows clean.

---

## Expert Level

### Technical Deep Dive

**`hasReadyTaskInStream`** (`internal/task/next.go:259`): Filters tasks by stream via `FilterByStream`, then checks the triple condition (Pending + no Owner + !IsBlocked). Short-circuits on first match. Nil-safe on the index parameter.

**`FindNextPhaseTasksForStream`** (`internal/task/next.go:287`): The function is self-contained — it reads the file, parses markdown, builds the dependency index, extracts phase structure, and iterates. This isolation means it can be called from any context without setup, but at the cost of redundant parsing when the caller also needs the full task list.

**Phase extraction** reuses `extractPhasesWithTaskRanges` which only associates top-level tasks (no `.` in task ID) with phases. Children are included through their parent's `Children` slice. Stream filtering via `FilterByStream` handles `GetEffectiveStream` (defaulting stream 0 → 1).

**The `runNextWithClaim` path** (`cmd/next.go:169`): When all three flags are set (`--phase --stream --claim`), the handler uses `FindNextPhaseTasksForStream` to discover the phase, then filters `phaseResult.Tasks` through the *outer* dependency index (built from the separately-parsed full task list) to identify claimable tasks. This works correctly because both parses read the same file and the index lookups are by stable ID, which is deterministic.

**Blocking status rendering** is decoupled from task storage. `TranslateToHierarchical` converts stable IDs (7-char alphanumeric like `abc1234`) to hierarchical IDs (like `1`, `2.3`) for user-facing output. This means output shows task numbers users recognize, not internal identifiers.

### Architecture Impact

- The `FindNextPhaseTasksForStream` function follows the same pattern as `FindNextPhaseTasks` — file-path-based, self-contained, returns `*PhaseTasksResult`. No new types were needed.
- Blocking status output functions (`formatStatusWithBlocking`, `renderPhaseTaskMarkdownWithBlocking`, `outputPhaseTasksJSONWithStreams`) are additions that augment existing rendering without modifying it. The original `outputPhaseTasksJSON` was removed as dead code.
- The `DependencyIndex` remains the single source of truth for blocking checks — no parallel or redundant blocking logic was introduced.

### Potential Issues

- **Double parse performance**: `runNextPhase` and `runNextWithClaim` both parse the file in their handler AND call `FindNextPhaseTasksForStream` which parses it again internally. For typical task files this is negligible, but it's O(2N) where O(N) would suffice. A future refactor could accept pre-parsed data.
- **Cross-phase dependency awareness**: A task in Phase C blocked by a task in Phase A is correctly handled — the `DependencyIndex` spans all tasks, not just the current phase. This means phase selection correctly skips phases where all stream tasks are blocked by work in earlier phases.
- **Concurrent access**: The claim operation is not atomic at the file level. Two agents claiming simultaneously could result in last-write-wins. This is a pre-existing limitation documented in the non-requirements.
- **Empty phase after filtering**: If a phase has stream tasks but they're all completed, `len(result) == 0` triggers `continue` to check the next phase. This edge case is correctly handled.

---

## Completeness Assessment

### Fully Implemented

| Requirement | Status |
|---|---|
| R1: Stream-Aware Phase Discovery (all 7 acceptance criteria) | Complete |
| R2: Backward Compatibility (all 3 acceptance criteria) | Complete |
| R3: Dependency-Aware Phase Selection (all 6 acceptance criteria) | Complete |
| R4: Claim Integration (all 4 acceptance criteria) | Complete |
| R5: Output Consistency (all 4 acceptance criteria) | Complete |

### Test Coverage

- 15 unit test cases for `FindNextPhaseTasksForStream` covering: single phase, missing stream, blocked tasks, no phases, empty phases, mixed states, owned tasks, in-progress tasks, completed tasks, default stream, multiple ready tasks, invalid stream
- 10 unit test cases for `hasReadyTaskInStream` covering: ready, owned, blocked, in-progress, completed, no stream match, multiple with one ready, nil index, empty list, default stream
- 7 integration tests via compiled binary: stream-aware navigation, blocked output, blocked-skip, claim, no-phases error, existing phase behavior, existing stream behavior
- Backward compatibility verified for `--phase` alone and `--stream` alone

### Not Implemented (by design — listed as non-requirements)

- Verbose skip output showing which phases were bypassed
- Phase-aware behavior for `--stream` without `--phase`
- Multi-phase return in a single call
- Concurrent access locking
- Stream existence validation
- Claim limiting
