# go-tasks

A standalone Go command-line tool designed for AI agents and developers to create and manage hierarchical markdown task lists with consistent formatting.

## Features

- **CRUD Operations**: Create, read, update, and delete hierarchical task structures
- **Consistent Formatting**: Generate standardized markdown regardless of input variations  
- **Status Management**: Track tasks as pending `[ ]`, in-progress `[-]`, or completed `[x]`
- **Hierarchical Structure**: Support nested tasks with automatic ID management (1, 1.1, 1.2.1)
- **Batch Operations**: Execute multiple operations atomically via JSON API
- **Search & Filtering**: Find tasks by content, status, hierarchy level, or parent
- **Multiple Output Formats**: View tasks as tables, markdown, or JSON
- **AI Agent Optimized**: Structured JSON API with comprehensive error reporting

## Installation

```bash
go install github.com/ArjenSchwarz/go-tasks@latest
```

Or build from source:

```bash
git clone https://github.com/ArjenSchwarz/go-tasks.git
cd go-tasks
make build
```

## Quick Start

```bash
# Create a new task file
go-tasks create "My Project Tasks" --file tasks.md

# Add some tasks
go-tasks add tasks.md --title "Setup development environment"
go-tasks add tasks.md --title "Write documentation" --parent 1

# Mark a task as completed
go-tasks complete tasks.md 1.1

# List tasks in table format
go-tasks list tasks.md --format table

# Search for tasks
go-tasks find tasks.md "documentation" --status pending
```

## Command Reference

### create - Create New Task File

Create a new task file with the specified title.

```bash
go-tasks create [title] --file [filename]
```

**Examples:**
```bash
go-tasks create "Sprint Planning" --file sprint.md
go-tasks create "Project Tasks" --file project-tasks.md
```

### list - Display Tasks

Display tasks in various formats (table, markdown, or JSON).

```bash
go-tasks list [file] [options]
```

**Options:**
- `--format [table|markdown|json]` - Output format (default: table)
- `--status [pending|in-progress|completed]` - Filter by status
- `--depth [number]` - Maximum hierarchy depth to show

**Examples:**
```bash
go-tasks list tasks.md --format table
go-tasks list tasks.md --format json --status pending
go-tasks list tasks.md --depth 2
```

### add - Add New Task

Add a new task or subtask to the file.

```bash
go-tasks add [file] --title [title] [options]
```

**Options:**
- `--title [text]` - Task title (required)
- `--parent [id]` - Parent task ID for subtasks
- `--details [text,...]` - Comma-separated detail points  
- `--references [ref,...]` - Comma-separated references

**Examples:**
```bash
go-tasks add tasks.md --title "Implement authentication"
go-tasks add tasks.md --title "Add unit tests" --parent 1
go-tasks add tasks.md --title "Review code" --details "Check logic,Verify tests" --references "coding-standards.md"
```

### complete - Mark Task Complete

Mark a task as completed `[x]`.

```bash
go-tasks complete [file] [task-id]
```

**Examples:**
```bash
go-tasks complete tasks.md 1
go-tasks complete tasks.md 2.1
```

### uncomplete - Mark Task Pending  

Mark a task as pending `[ ]`.

```bash
go-tasks uncomplete [file] [task-id]
```

### progress - Mark Task In Progress

Mark a task as in-progress `[-]`.

```bash
go-tasks progress [file] [task-id]
```

### update - Modify Task

Update task title, details, or references.

```bash
go-tasks update [file] [task-id] [options]
```

**Options:**
- `--title [text]` - New task title
- `--details [text,...]` - Replace all details
- `--references [ref,...]` - Replace all references

**Examples:**
```bash
go-tasks update tasks.md 1 --title "New title"
go-tasks update tasks.md 2.1 --details "Step 1,Step 2,Step 3"
go-tasks update tasks.md 3 --references "spec.md,api-docs.md"
```

### remove - Delete Task

Remove a task and all its subtasks. Remaining tasks are automatically renumbered.

```bash
go-tasks remove [file] [task-id]
```

**Examples:**
```bash
go-tasks remove tasks.md 2
go-tasks remove tasks.md 1.3
```

### find - Search Tasks

Search for tasks by content, with filtering options.

```bash
go-tasks find [file] [pattern] [options]
```

**Options:**
- `--case-sensitive` - Enable case-sensitive matching
- `--in-details` - Search within task details
- `--in-references` - Search within references
- `--status [status]` - Filter by task status
- `--parent [id]` - Filter by parent task ID
- `--max-depth [number]` - Maximum hierarchy depth
- `--format [table|json]` - Output format (default: table)

**Examples:**
```bash
go-tasks find tasks.md "authentication" --format json
go-tasks find tasks.md "test" --in-details --status pending
go-tasks find tasks.md "api" --parent 2 --max-depth 3
```

### batch - Execute Multiple Operations

Execute multiple operations atomically from JSON input.

```bash
go-tasks batch [file] --operations [json-file] [options]
```

**Options:**
- `--operations [file]` - JSON file containing operations
- `--dry-run` - Preview changes without applying them

**Examples:**
```bash
go-tasks batch tasks.md --operations batch-ops.json
go-tasks batch tasks.md --operations updates.json --dry-run
```

## JSON API Schema

### Batch Operations Request

```json
{
  "file": "tasks.md",
  "operations": [
    {
      "type": "add",
      "parent": "1", 
      "title": "New task",
      "details": ["Detail 1", "Detail 2"],
      "references": ["doc.md"]
    },
    {
      "type": "update_status",
      "id": "2",
      "status": 2
    },
    {
      "type": "update",
      "id": "3", 
      "title": "Updated title",
      "details": ["New detail"],
      "references": ["updated-doc.md"]
    },
    {
      "type": "remove",
      "id": "4"
    }
  ],
  "dry_run": false
}
```

**Operation Types:**
- `add` - Add new task (requires `title`, optional `parent`)
- `remove` - Delete task (requires `id`)
- `update_status` - Change task status (requires `id`, `status`)
- `update` - Modify task content (requires `id`, optional `title`, `details`, `references`)

**Status Values:**
- `0` = Pending `[ ]`
- `1` = In Progress `[-]`  
- `2` = Completed `[x]`

### Batch Operations Response

```json
{
  "success": true,
  "applied": 4,
  "errors": [],
  "preview": "# Task List\n\n- [x] 1. Updated task..."
}
```

### Search Results

```json
{
  "query": "authentication", 
  "matches": [
    {
      "id": "1.2",
      "title": "Implement authentication system",
      "status": 0,
      "details": ["OAuth integration", "Session management"],
      "references": ["auth-spec.md"],
      "path": ["1", "1.2"],
      "parent": {
        "id": "1",
        "title": "Backend Development"  
      }
    }
  ],
  "total": 1
}
```

## File Format

go-tasks generates consistent markdown with the following structure:

```markdown
# Project Title

- [ ] 1. First task
  - Implementation details
  - More details
  - References: spec.md, requirements.md
- [-] 2. Second task in progress
  - [ ] 2.1. Subtask
  - [x] 2.2. Completed subtask
- [x] 3. Completed task
```

**Format Rules:**
- Title as H1 header: `# Title`
- Tasks with hierarchical numbering: `1`, `1.1`, `1.2.1`
- Status indicators: `[ ]` pending, `[-]` in-progress, `[x]` completed
- 2-space indentation per hierarchy level
- Details as indented bullet points
- References with "References: " prefix, comma-separated

## Examples

See the [`examples/`](examples/) directory for sample task files and common usage patterns:

- [`examples/simple.md`](examples/simple.md) - Basic task list
- [`examples/project.md`](examples/project.md) - Software project with phases
- [`examples/complex.md`](examples/complex.md) - Deep hierarchy with all features
- [`examples/batch-operations.json`](examples/batch-operations.json) - Sample batch operations

## Development

### Prerequisites

- Go 1.21+
- Make
- golangci-lint (for linting)

### Building

```bash
# Run all checks (format, lint, test)
make check

# Build binary
make build

# Run tests
make test

# Run integration tests
make test-integration

# Generate coverage report
make test-coverage
```

### Testing

```bash
# Unit tests
make test

# Integration tests (requires INTEGRATION=1)
make test-integration

# All tests
make test-all

# Benchmarks
make benchmark
```

## Error Handling

go-tasks follows strict error reporting without auto-correction:

- **Malformed Files**: Reports syntax errors without attempting fixes
- **Invalid Operations**: Validates all parameters before applying changes
- **Atomic Batches**: All operations succeed or all fail (no partial updates)
- **Resource Limits**: Enforces limits on file size, task count, and hierarchy depth

## Security

- **Path Validation**: Prevents directory traversal attacks
- **Input Sanitization**: Validates all user input for safety
- **Resource Limits**: Prevents DoS through large files or deep hierarchies
- **Atomic Operations**: Ensures data consistency

## Performance

- **Sub-second Response**: Optimized for files with 100+ tasks
- **Memory Efficient**: Handles large task lists without excessive memory usage
- **Fast Search**: Efficient filtering and search algorithms

## License

MIT License. See [LICENSE](LICENSE) for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run `make check` to verify code quality
6. Submit a pull request

## Support

- **Issues**: [GitHub Issues](https://github.com/ArjenSchwarz/go-tasks/issues)
- **Discussions**: [GitHub Discussions](https://github.com/ArjenSchwarz/go-tasks/discussions)
- **Documentation**: [Wiki](https://github.com/ArjenSchwarz/go-tasks/wiki)