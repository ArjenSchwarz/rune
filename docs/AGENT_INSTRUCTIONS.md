# Agent Instructions for rune

This guide provides optimal workflows for AI agents using rune to manage hierarchical task lists.

## Creating a Task List

### Basic Workflow (Hierarchical Tasks)

```bash
# 1. Create new task file
rune create "Project Name" --file tasks.md

# 2. Add main phases/categories
rune add tasks.md --title "Planning Phase"
rune add tasks.md --title "Implementation Phase"
rune add tasks.md --title "Testing Phase"

# 3. Add detailed tasks under each phase
rune add tasks.md --title "Requirements gathering" --parent 1
rune add tasks.md --title "Architecture design" --parent 1
rune add tasks.md --title "Setup development environment" --parent 2
rune add tasks.md --title "Implement core features" --parent 2
```

### Basic Workflow (Using Phases)

```bash
# 1. Create new task file
rune create "Project Name" --file tasks.md

# 2. Add phase headers
rune add-phase tasks.md "Planning"
rune add-phase tasks.md "Implementation"
rune add-phase tasks.md "Testing"

# 3. Add tasks to specific phases
rune add tasks.md --title "Requirements gathering" --phase "Planning"
rune add tasks.md --title "Architecture design" --phase "Planning"
rune add tasks.md --title "Setup development environment" --phase "Implementation"
rune add tasks.md --title "Implement core features" --phase "Implementation"
```

### Advanced: Batch Creation (Hierarchical)

For complex task lists, use JSON batch operations for atomic creation:

```bash
# Create batch-setup.json
cat > batch-setup.json << 'EOF'
{
  "file": "project.md",
  "operations": [
    {
      "type": "add",
      "title": "Planning Phase",
      "details": ["Define requirements", "Create timeline"]
    },
    {
      "type": "add",
      "title": "Implementation Phase"
    },
    {
      "type": "add",
      "parent": "1",
      "title": "Requirements analysis",
      "references": ["requirements.md"]
    },
    {
      "type": "add",
      "parent": "1",
      "title": "Technical design"
    },
    {
      "type": "add",
      "parent": "2",
      "title": "Core development",
      "details": ["API implementation", "Database setup"]
    }
  ],
  "dry_run": false
}
EOF

# Execute batch creation
rune batch project.md --operations batch-setup.json
```

### Advanced: Batch Creation with Phases

Use phases for better organization in batch operations:

```bash
# Create batch-phased-setup.json
cat > batch-phased-setup.json << 'EOF'
{
  "file": "project.md",
  "operations": [
    {
      "type": "add",
      "title": "Requirements analysis",
      "phase": "Planning",
      "details": ["Define requirements", "Create timeline"],
      "references": ["requirements.md"]
    },
    {
      "type": "add",
      "title": "Technical design",
      "phase": "Planning"
    },
    {
      "type": "add",
      "title": "Core development",
      "phase": "Implementation",
      "details": ["API implementation", "Database setup"]
    },
    {
      "type": "add",
      "title": "Integration work",
      "phase": "Implementation"
    },
    {
      "type": "add",
      "title": "Unit testing",
      "phase": "Testing",
      "details": ["Test coverage > 80%"]
    }
  ],
  "dry_run": false
}
EOF

# Execute batch creation - phases are auto-created
rune batch project.md --operations batch-phased-setup.json
```

## Marking Groups of Tasks as Done

### Option 1: Individual Commands (Simple)

```bash
# Mark specific tasks complete
rune complete tasks.md 1.1
rune complete tasks.md 1.2  
rune complete tasks.md 2.1
```

### Option 2: Batch Operations (Recommended)

For marking multiple tasks as done efficiently:

```bash
# Create completion batch file
cat > mark-complete.json << 'EOF'
{
  "file": "tasks.md",
  "operations": [
    {"type": "update", "id": "1.1", "status": 2},
    {"type": "update", "id": "1.2", "status": 2},
    {"type": "update", "id": "1.3", "status": 2},
    {"type": "update", "id": "2.1", "status": 2}
  ],
  "dry_run": false
}
EOF

# Execute batch completion
rune batch tasks.md --operations mark-complete.json
```

### Option 3: Find and Complete Pattern

Use search to identify tasks, then batch complete:

```bash
# Find all pending tasks in a specific phase
rune find tasks.md "Phase 1" --status pending --format json > pending-tasks.json

# Process the JSON to create batch completion operations
# (This would require additional scripting to parse the JSON and create batch operations)
```

## Status Values Reference

- `0` = Pending `[ ]`
- `1` = In Progress `[-]`  
- `2` = Completed `[x]`

## Best Practices for Agents

### 1. Use Descriptive Hierarchies
```bash
# Good: Clear hierarchy
rune add tasks.md --title "Backend Development" 
rune add tasks.md --title "Database Setup" --parent 1
rune add tasks.md --title "API Implementation" --parent 1

# Better: Include details and references
rune add tasks.md --title "Database Setup" --parent 1 \
  --details "Install PostgreSQL,Create schemas,Set up migrations" \
  --references "db-design.md"
```

### 2. Batch Operations for Efficiency
```bash
# Instead of 10+ individual commands, use one batch operation
rune batch tasks.md --operations batch-file.json
```

### 3. Use Dry Run for Validation
```bash
# Always test complex batch operations first
rune batch tasks.md --operations complex-changes.json --dry-run
```

### 4. Progress Tracking Pattern
```bash
# Mark task as in-progress when starting
rune progress tasks.md 2.1

# Add implementation details as you work
rune update tasks.md 2.1 --details "API endpoints implemented,Tests added,Documentation updated"

# Mark complete when finished
rune complete tasks.md 2.1
```

### 5. Search for Status Updates
```bash
# Find all completed tasks
rune find tasks.md "" --status completed --format table

# Find all pending tasks in a specific area
rune find tasks.md "authentication" --status pending

# Get JSON output for programmatic processing
rune find tasks.md "" --status pending --format json
```

## Error Prevention

### Validate Before Batch Operations
```bash
# Use dry-run to preview changes
rune batch tasks.md --operations changes.json --dry-run

# Check current state before making changes
rune list tasks.md --format table
```

### Handle Renumbering
When removing tasks, remember that IDs will be renumbered:
```bash
# If you remove task 2, task 3 becomes task 2, task 4 becomes task 3, etc.
# Plan removal operations carefully in batch operations
```

### Atomic Operations
All batch operations are atomic - they either all succeed or all fail, preventing partial updates.

## Working with Phases

### Get Next Phase Tasks
```bash
# Get all pending tasks from the next incomplete phase
rune next tasks.md --phase

# This is useful for focusing on phase-by-phase execution
```

### Phase-Aware Operations
```bash
# Check if file has phases (for conditional logic)
rune has-phases tasks.md  # Exit code 0 if phases exist, 1 if not
rune has-phases tasks.md --verbose  # Get JSON with phase names

# Add task to specific phase (creates phase if needed)
rune add tasks.md --title "New task" --phase "Implementation"

# Add phase header manually
rune add-phase tasks.md "New Phase Name"

# List tasks with phase information in table format
rune list tasks.md --format table
```

### Phases vs Hierarchical Tasks
- **Phases**: Organize using H2 headers (`## Phase Name`), tasks numbered sequentially across phases
- **Hierarchical**: Organize using parent-child relationships, nested task IDs (1.1, 1.2.1)
- **Can combine both**: Use phases for high-level organization, hierarchical tasks within phases

## Programmatic Phase Detection

For automation workflows that need to handle phased and non-phased files differently:

```bash
# Check if file has phases (JSON output)
rune has-phases tasks.md

# Example output for file with phases:
# {"hasPhases":true,"count":3,"phases":[]}

# Example output for file without phases:
# {"hasPhases":false,"count":0,"phases":[]}

# Get phase names with --verbose
rune has-phases tasks.md --verbose
# {"hasPhases":true,"count":3,"phases":["Planning","Implementation","Testing"]}

# Use in scripts for conditional workflows
if rune has-phases tasks.md > /dev/null 2>&1; then
    # File has phases - use phase-specific workflow
    rune next tasks.md --phase
else
    # File has no phases - use regular workflow
    rune next tasks.md
fi
```

## Multi-Agent Parallel Execution

rune supports parallel task execution across multiple agents using streams and the claiming mechanism.

### Setting Up Streams

Partition tasks into streams for parallel work distribution:

```bash
# Assign tasks to different streams
rune add tasks.md --title "Backend API development" --stream 1
rune add tasks.md --title "Frontend UI development" --stream 2
rune add tasks.md --title "Database optimization" --stream 1
rune add tasks.md --title "UI testing" --stream 2

# Check stream status
rune streams tasks.md
rune streams tasks.md --json  # For programmatic use
```

### Claiming Tasks

Agents claim tasks to indicate they're working on them:

```bash
# Claim a single next ready task (orchestrator assigns to agent)
rune next tasks.md --claim "agent-1"

# Claim all ready tasks in a specific stream
rune next tasks.md --stream 2 --claim "agent-2"

# Agent releases a task when done or blocked
rune update tasks.md 3 --release
```

### Orchestrator Workflow

A typical orchestrator workflow:

```bash
# 1. Check available streams
rune streams tasks.md --available --json

# 2. Assign streams to agents
rune next tasks.md --stream 1 --claim "agent-backend"
rune next tasks.md --stream 2 --claim "agent-frontend"

# 3. Monitor progress
rune list tasks.md --format json

# 4. When agent completes a task, more tasks may become ready
rune complete tasks.md 1
rune streams tasks.md --json  # Check for newly available work
```

### Batch Operations for Multi-Agent Setup

Set up streams and dependencies in a single atomic operation:

```bash
cat > setup-parallel.json << 'EOF'
{
  "file": "tasks.md",
  "operations": [
    {
      "type": "add",
      "title": "Initialize project",
      "stream": 1
    },
    {
      "type": "add",
      "title": "Backend development",
      "stream": 1,
      "blocked_by": ["1"]
    },
    {
      "type": "add",
      "title": "Frontend development",
      "stream": 2,
      "blocked_by": ["1"]
    },
    {
      "type": "add",
      "title": "Integration testing",
      "stream": 1,
      "blocked_by": ["2", "3"]
    }
  ]
}
EOF

rune batch tasks.md --operations setup-parallel.json
```

## Task Dependencies

Tasks can declare dependencies on other tasks that must complete before they become "ready".

### Adding Dependencies

```bash
# Create a task with dependencies
rune add tasks.md --title "Build API" --blocked-by "1,2"

# Add dependencies to existing task
rune update tasks.md 4 --blocked-by "1,2,3"
```

### Understanding Ready vs Blocked

- **Ready**: All dependencies completed, task can be started
- **Blocked**: One or more dependencies not completed

```bash
# Get only ready tasks (dependencies satisfied)
rune next tasks.md

# See which tasks are blocked
rune streams tasks.md --json  # Shows ready/blocked counts per stream
```

### Dependency Resolution Workflow

```bash
# 1. Check what's ready to work on
rune next tasks.md --format json

# 2. Complete a task
rune complete tasks.md 1

# 3. This may unblock other tasks - check what's now ready
rune next tasks.md --format json
```

### Batch Operations with Dependencies

```bash
cat > with-dependencies.json << 'EOF'
{
  "file": "tasks.md",
  "operations": [
    {
      "type": "add",
      "title": "Setup database",
      "details": ["Install PostgreSQL", "Create schemas"]
    },
    {
      "type": "add",
      "title": "Build API layer",
      "blocked_by": ["1"]
    },
    {
      "type": "add",
      "title": "Build UI layer",
      "blocked_by": ["1"]
    },
    {
      "type": "add",
      "title": "Integration tests",
      "blocked_by": ["2", "3"]
    }
  ]
}
EOF

rune batch tasks.md --operations with-dependencies.json
```

## Quick Reference

```bash
# Create task list
rune create "Title" --file name.md

# Add task (hierarchical)
rune add file.md --title "Task" [--parent ID] [--details "a,b,c"] [--references "x.md,y.md"]

# Add task to phase
rune add file.md --title "Task" --phase "Phase Name" [--details "a,b,c"]

# Add task with stream and dependencies
rune add file.md --title "Task" --stream 2 --blocked-by "1,2" --owner "agent-1"

# Add phase header
rune add-phase file.md "Phase Name"

# Check for phases
rune has-phases file.md [--verbose]

# Get next task or next phase
rune next file.md [--phase] [--stream N] [--claim AGENT_ID]

# Mark complete/in-progress/pending
rune complete file.md ID
rune progress file.md ID
rune uncomplete file.md ID

# Stream management
rune streams file.md [--available] [--json]
rune update file.md ID --stream N
rune update file.md ID --owner "agent-1"
rune update file.md ID --release

# Batch operations (most efficient for multiple changes)
rune batch file.md --operations batch.json [--dry-run]

# Search and filter
rune find file.md "pattern" [--status STATUS] [--format FORMAT]
rune list file.md --stream N --owner "agent-1"

# View current state
rune list file.md [--format table|json|markdown]
```