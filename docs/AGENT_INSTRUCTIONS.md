# Agent Instructions for go-tasks

This guide provides optimal workflows for AI agents using go-tasks to manage hierarchical task lists.

## Creating a Task List

### Basic Workflow

```bash
# 1. Create new task file
go-tasks create "Project Name" --file tasks.md

# 2. Add main phases/categories
go-tasks add tasks.md --title "Planning Phase"
go-tasks add tasks.md --title "Implementation Phase"  
go-tasks add tasks.md --title "Testing Phase"

# 3. Add detailed tasks under each phase
go-tasks add tasks.md --title "Requirements gathering" --parent 1
go-tasks add tasks.md --title "Architecture design" --parent 1
go-tasks add tasks.md --title "Setup development environment" --parent 2
go-tasks add tasks.md --title "Implement core features" --parent 2
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
go-tasks batch project.md --operations batch-setup.json
```

## Marking Groups of Tasks as Done

### Option 1: Individual Commands (Simple)

```bash
# Mark specific tasks complete
go-tasks complete tasks.md 1.1
go-tasks complete tasks.md 1.2  
go-tasks complete tasks.md 2.1
```

### Option 2: Batch Operations (Recommended)

For marking multiple tasks as done efficiently:

```bash
# Create completion batch file
cat > mark-complete.json << 'EOF'
{
  "file": "tasks.md",
  "operations": [
    {"type": "update_status", "id": "1.1", "status": 2},
    {"type": "update_status", "id": "1.2", "status": 2},
    {"type": "update_status", "id": "1.3", "status": 2},
    {"type": "update_status", "id": "2.1", "status": 2}
  ],
  "dry_run": false
}
EOF

# Execute batch completion
go-tasks batch tasks.md --operations mark-complete.json
```

### Option 3: Find and Complete Pattern

Use search to identify tasks, then batch complete:

```bash
# Find all pending tasks in a specific phase
go-tasks find tasks.md "Phase 1" --status pending --format json > pending-tasks.json

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
go-tasks add tasks.md --title "Backend Development" 
go-tasks add tasks.md --title "Database Setup" --parent 1
go-tasks add tasks.md --title "API Implementation" --parent 1

# Better: Include details and references
go-tasks add tasks.md --title "Database Setup" --parent 1 \
  --details "Install PostgreSQL,Create schemas,Set up migrations" \
  --references "db-design.md"
```

### 2. Batch Operations for Efficiency
```bash
# Instead of 10+ individual commands, use one batch operation
go-tasks batch tasks.md --operations batch-file.json
```

### 3. Use Dry Run for Validation
```bash
# Always test complex batch operations first
go-tasks batch tasks.md --operations complex-changes.json --dry-run
```

### 4. Progress Tracking Pattern
```bash
# Mark task as in-progress when starting
go-tasks progress tasks.md 2.1

# Add implementation details as you work
go-tasks update tasks.md 2.1 --details "API endpoints implemented,Tests added,Documentation updated"

# Mark complete when finished
go-tasks complete tasks.md 2.1
```

### 5. Search for Status Updates
```bash
# Find all completed tasks
go-tasks find tasks.md "" --status completed --format table

# Find all pending tasks in a specific area
go-tasks find tasks.md "authentication" --status pending

# Get JSON output for programmatic processing
go-tasks find tasks.md "" --status pending --format json
```

## Error Prevention

### Validate Before Batch Operations
```bash
# Use dry-run to preview changes
go-tasks batch tasks.md --operations changes.json --dry-run

# Check current state before making changes
go-tasks list tasks.md --format table
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
go-tasks create "Title" --file name.md

# Add task
go-tasks add file.md --title "Task" [--parent ID] [--details "a,b,c"] [--references "x.md,y.md"]

# Mark complete/in-progress/pending  
go-tasks complete file.md ID
go-tasks progress file.md ID
go-tasks uncomplete file.md ID

# Batch operations (most efficient for multiple changes)
go-tasks batch file.md --operations batch.json [--dry-run]

# Search and filter
go-tasks find file.md "pattern" [--status STATUS] [--format FORMAT]

# View current state
go-tasks list file.md [--format table|json|markdown]
```