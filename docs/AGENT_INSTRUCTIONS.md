# Agent Instructions for rune

This guide provides optimal workflows for AI agents using rune to manage hierarchical task lists.

## Creating a Task List

### Basic Workflow

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

### Advanced: Batch Creation

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

## Quick Reference

```bash
# Create task list
rune create "Title" --file name.md

# Add task
rune add file.md --title "Task" [--parent ID] [--details "a,b,c"] [--references "x.md,y.md"]

# Mark complete/in-progress/pending  
rune complete file.md ID
rune progress file.md ID
rune uncomplete file.md ID

# Batch operations (most efficient for multiple changes)
rune batch file.md --operations batch.json [--dry-run]

# Search and filter
rune find file.md "pattern" [--status STATUS] [--format FORMAT]

# View current state
rune list file.md [--format table|json|markdown]
```