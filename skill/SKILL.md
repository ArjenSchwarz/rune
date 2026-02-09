---
name: rune-tasks
description: Manage hierarchical task lists using the rune CLI tool. Create, update, and organize tasks with phases, subtasks, status tracking, task dependencies, and work streams for multi-agent parallel execution.
---

# Rune Task Management Skill

Manage hierarchical task lists using the `rune` CLI tool.

## Command Reference

### Creating and Listing
- `rune create [file] --title "Title"` - Initialize task file (required)
- `rune create [file] --title "Title" --reference "file.md"` - With references (repeatable)
- `rune list [file]` - Display tasks (supports `--filter`, `--format`, `--stream N`, `--owner "name"`)
- `rune next [file]` - Get next incomplete task
- `rune next [file] --one` - Show only first incomplete path (single task chain)
- `rune next [file] --phase` - Get all tasks from next phase
- `rune next [file] --stream N --claim "agent-id"` - Claim ready tasks in stream
- `rune streams [file]` - Show stream status (`--available`, `--json`)
- `rune find [file] --pattern "term"` - Search tasks

### Task Management
- `rune add [file] --title "Task"` - Add task (options: `--parent`, `--phase`, `--stream`, `--blocked-by`, `--owner`)
- `rune complete [file] [task-id]` - Mark completed (positional ID)
- `rune progress [file] [task-id]` - Mark in-progress
- `rune uncomplete [file] [task-id]` - Mark pending
- `rune update [file] [task-id] --title "New"` - Update task (options: `--details`, `--stream`, `--blocked-by`, `--owner`, `--release`)
- `rune remove [file] [task-id]` - Remove task and subtasks

### Organization
- `rune add-phase [file] "Phase Name"` - Add phase header
- `rune has-phases [file]` - Check if file uses phases
- `rune renumber [file]` - Recalculate IDs (creates .bak, use `--dry-run` to preview)
- `rune add-frontmatter [file] --reference "file.md"` - Add references (repeatable)
- `rune add-frontmatter [file] --meta "key:value"` - Add metadata

### Batch Operations
Execute multiple operations atomically:
```bash
rune batch [file] --input '{"file":"tasks.md","operations":[...]}'
```

**Operation types:**
- `add` - Required: `title`. Optional: `parent`, `phase`, `stream`, `blocked_by`, `owner`, `requirements`, `requirements_file`
- `update` - Required: `id`. Optional: `title`, `status` (0/1/2), `details`, `references`, `stream`, `blocked_by`, `owner`, `release`
- `remove` - Required: `id`
- `add-phase` - Required: `name`

**Status values:** 0=pending, 1=in-progress, 2=completed

**Important:** Array fields (`references`, `requirements`, `blocked_by`) must be JSON arrays, not comma-separated strings.

Use hierarchical task IDs (e.g., "1", "2.1") in `blocked_by`; rune resolves to stable IDs internally.

Use `"dry_run": true` to preview changes without applying.

## Key Concepts

### Task Status
- `[ ]` Pending | `[-]` In-progress | `[x]` Completed

### Phases
H2 headers (`## Phase Name`) that group tasks. Tasks numbered globally across phases.

### Task Dependencies
Tasks can block other tasks using `--blocked-by "1,2"`. A task is "ready" only when all blockers are completed. Circular dependencies are rejected. Stable IDs (7-char alphanumeric, hidden as HTML comments) survive renumbering.

### Work Streams
Partition tasks for parallel execution. Default stream is 1. Use `rune streams` to see ready/blocked/active counts per stream.

### Task Ownership
Agents claim tasks via `--claim` or `--owner`. Claimed tasks show owner; use `--release` to unclaim.

### Git Integration
With discovery enabled, omit filename to auto-discover from branch:
```yaml
# .rune.yml or ~/.config/rune/config.yml
discovery:
  enabled: true
  template: "specs/{branch}/tasks.md"
```
Branch prefixes like `feature/` are stripped automatically.

## Workflow Guidelines

### Single Agent
1. `rune list --filter pending` to see work
2. `rune next` to identify next task
3. Mark in-progress when starting, completed when done
4. Use batch operations for multiple related changes

### Multi-Agent
1. `rune streams --available` to find streams with ready tasks
2. `rune next --stream N --claim "agent-id"` to claim stream work
3. Each agent works independently within its stream
4. Complete tasks to unblock dependents in other streams
5. Use `--release` when giving up a task

## Best Practices

1. Use `rune batch` for related changes (atomic)
2. Keep task status current
3. Use `--blocked-by` to define execution order
4. Partition independent work into streams
5. Use `--dry-run` to preview changes
6. Use phases for workflow stages
7. Keep hierarchy shallow (avoid deep nesting)

## Error Handling

- **Task not found**: Use `rune list` to see valid IDs
- **Circular dependency**: Restructure dependencies to break cycle
- **Phase not found**: Create with `rune add-phase` first
- **Batch validation failure**: Fix invalid operation and retry (entire batch fails if any operation invalid)
- **Git discovery failure**: Check config, template path, and that you're on a feature branch

## Response Format

When managing tasks:
1. Explain operations to perform
2. Execute rune commands
3. Show results
4. Confirm what was accomplished

For detailed batch examples and patterns, read `references/patterns.md`.
