# rune

![Rune logo](docs/images/rune-logo-small.jpg)

A standalone Go command-line tool designed for AI agents and developers to create and manage hierarchical markdown task lists with consistent formatting.

## Features

- **CRUD Operations**: Create, read, update, and delete hierarchical task structures
- **Consistent Formatting**: Generate standardized markdown regardless of input variations
- **Status Management**: Track tasks as pending `[ ]`, in-progress `[-]`, or completed `[x]`
- **Hierarchical Structure**: Support nested tasks with automatic ID management (1, 1.1, 1.2.1)
- **Batch Operations**: Execute multiple operations atomically via JSON API
- **Search & Filtering**: Find tasks by content, status, hierarchy level, or parent
- **Multiple Output Formats**: View tasks as tables, markdown, or JSON
- **Next Task Discovery**: Automatically find the next incomplete task to work on
- **Git Branch Integration**: Automatic file discovery based on current git branch
- **Reference Documents**: Link task files to related documentation and resources
- **Automatic Parent Completion**: Parent tasks auto-complete when all subtasks are done
- **Phase Organization**: Group tasks under H2 headers for logical organization
- **Phase Detection**: Programmatically check if files contain phases with JSON output
- **AI Agent Optimized**: Structured JSON API with comprehensive error reporting

## Installation

```bash
go install github.com/ArjenSchwarz/rune@latest
```

Or build from source:

```bash
git clone https://github.com/ArjenSchwarz/rune.git
cd rune
make build
```

## Quick Start

```bash
# Create a new task file
rune create "My Project Tasks" --file tasks.md

# Add some tasks
rune add tasks.md --title "Setup development environment"
rune add tasks.md --title "Write documentation" --parent 1

# Mark a task as completed
rune complete tasks.md 1.1

# List tasks in table format
rune list tasks.md --format table

# Search for tasks
rune find tasks.md "documentation" --status pending
```

## Command Reference

### create - Create New Task File

Create a new task file with the specified title.

```bash
rune create [title] --file [filename]
```

**Examples:**
```bash
rune create "Sprint Planning" --file sprint.md
rune create "Project Tasks" --file project-tasks.md
```

### list - Display Tasks

Display tasks in various formats (table, markdown, or JSON).

```bash
rune list [file] [options]
```

**Options:**
- `--format [table|markdown|json]` - Output format (default: table)
- `--status [pending|in-progress|completed]` - Filter by status
- `--depth [number]` - Maximum hierarchy depth to show

**Examples:**
```bash
rune list tasks.md --format table
rune list tasks.md --format json --status pending
rune list tasks.md --depth 2
```

### add - Add New Task

Add a new task or subtask to the file.

```bash
rune add [file] --title [title] [options]
```

**Options:**
- `--title [text]` - Task title (required)
- `--parent [id]` - Parent task ID for subtasks
- `--phase [name]` - Add task to specified phase (creates phase if it doesn't exist)
- `--details [text,...]` - Comma-separated detail points
- `--references [ref,...]` - Comma-separated references
- `--requirements [id,...]` - Comma-separated requirement IDs (e.g., "1.1,1.2,2.3")
- `--requirements-file [path]` - Path to requirements file (default: requirements.md)

**Examples:**
```bash
rune add tasks.md --title "Implement authentication"
rune add tasks.md --title "Add unit tests" --parent 1
rune add tasks.md --title "Setup database" --phase "Development"
rune add tasks.md --title "Review code" --details "Check logic,Verify tests" --references "coding-standards.md"
rune add tasks.md --title "Implement login" --requirements "1.1,1.2" --requirements-file "specs/requirements.md"
```

### complete - Mark Task Complete

Mark a task as completed `[x]`.

```bash
rune complete [file] [task-id]
```

**Examples:**
```bash
rune complete tasks.md 1
rune complete tasks.md 2.1
```

### uncomplete - Mark Task Pending

Mark a task as pending `[ ]`.

```bash
rune uncomplete [file] [task-id]
```

### progress - Mark Task In Progress

Mark a task as in-progress `[-]`.

```bash
rune progress [file] [task-id]
```

### update - Modify Task

Update task title, details, or references.

```bash
rune update [file] [task-id] [options]
```

**Options:**
- `--title [text]` - New task title
- `--details [text,...]` - Replace all details
- `--references [ref,...]` - Replace all references
- `--requirements [id,...]` - Replace all requirements
- `--clear-requirements` - Clear all requirements from the task

**Examples:**
```bash
rune update tasks.md 1 --title "New title"
rune update tasks.md 2.1 --details "Step 1,Step 2,Step 3"
rune update tasks.md 3 --references "spec.md,api-docs.md"
rune update tasks.md 4 --requirements "2.1,2.2"
rune update tasks.md 5 --clear-requirements
```

### remove - Delete Task

Remove a task and all its subtasks. Remaining tasks are automatically renumbered.

```bash
rune remove [file] [task-id]
```

**Examples:**
```bash
rune remove tasks.md 2
rune remove tasks.md 1.3
```

### find - Search Tasks

Search for tasks by content, with filtering options.

```bash
rune find [file] [pattern] [options]
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
rune find tasks.md "authentication" --format json
rune find tasks.md "test" --in-details --status pending
rune find tasks.md "api" --parent 2 --max-depth 3
```

### next - Get Next Incomplete Task

Retrieve the next incomplete task from your task list using intelligent depth-first traversal.

```bash
rune next [file] [options]
```

**Options:**
- `--format [table|markdown|json]` - Output format (default: table)
- `--phase` - Get all pending tasks from the next incomplete phase

**Examples:**
```bash
# Get next task (uses git branch discovery if configured)
rune next

# Get next task from specific file
rune next tasks.md

# Get all tasks from next phase with incomplete work
rune next tasks.md --phase

# Output in JSON format
rune next --format json

# Output in markdown format
rune next --format markdown
```

**How it works:**
- Finds the first task with incomplete work (task itself or any subtask not completed)
- Uses depth-first traversal through the task hierarchy
- Returns the task and all its incomplete subtasks for focused work
- With `--phase` flag, returns all pending tasks from the first phase containing incomplete work
- Includes task details and references in the output
- Supports git branch-based file discovery when configured

### add-phase - Add Phase Header

Add a new phase header to organize tasks into logical sections.

```bash
rune add-phase [file] [name]
```

**Examples:**
```bash
# Add phase using git discovery for file
rune add-phase "Planning"

# Add phase to specific file
rune add-phase tasks.md "Implementation"
rune add-phase tasks.md "Testing"
```

**How it works:**
- Adds a markdown H2 header (`## Phase Name`) to organize tasks
- Phase is appended to the end of the document
- Tasks can then be added to the phase using `rune add --phase "Phase Name"`
- Phases are optional - tasks can exist outside of any phase

### has-phases - Check for Phase Headers

Check if a task file contains phase headers, returning JSON output suitable for scripting.

```bash
rune has-phases [file] [options]
```

**Options:**
- `--verbose, -v` - Include phase names in the output

**Exit Codes:**
- `0` - File contains phases
- `1` - File does not contain phases or error occurred

**Examples:**
```bash
# Check if file has phases using git discovery
rune has-phases

# Check specific file
rune has-phases tasks.md

# Get detailed output with phase names
rune has-phases tasks.md --verbose

# Use in shell scripts
if rune has-phases tasks.md > /dev/null 2>&1; then
    echo "File has phases"
else
    echo "File has no phases"
fi
```

**JSON Output Format:**
```json
{
  "hasPhases": true,
  "count": 2,
  "phases": ["Planning", "Implementation"]
}
```

**Fields:**
- `hasPhases` - Boolean indicating if phases exist
- `count` - Number of phases found in the file
- `phases` - Array of phase names (only included when `--verbose` is used, otherwise empty array)

**How it works:**
- Scans the file for H2 markdown headers (`## Phase Name`)
- Returns JSON output for easy parsing by scripts and automation
- Exit code allows for simple conditional checks in shell scripts
- Useful for determining if phase-specific operations are available

### batch - Execute Multiple Operations

Execute multiple operations atomically from JSON input.

```bash
rune batch [file] --operations [json-file] [options]
```

**Options:**
- `--operations [file]` - JSON file containing operations
- `--dry-run` - Preview changes without applying them

**Examples:**
```bash
rune batch tasks.md --operations batch-ops.json
rune batch tasks.md --operations updates.json --dry-run
```

## JSON API Schema

### Batch Operations Request

```json
{
  "file": "tasks.md",
  "requirements_file": "specs/requirements.md",
  "operations": [
    {
      "type": "add",
      "parent": "1",
      "title": "New task",
      "details": ["Detail 1", "Detail 2"],
      "references": ["doc.md"],
      "requirements": ["1.1", "1.2"]
    },
    {
      "type": "add",
      "title": "Setup database",
      "phase": "Development"
    },
    {
      "type": "update",
      "id": "2",
      "status": 2
    },
    {
      "type": "update",
      "id": "3",
      "title": "Updated title",
      "details": ["New detail"],
      "references": ["updated-doc.md"],
      "requirements": ["2.1", "2.2"]
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
- `add` - Add new task (requires `title`, optional `parent`, `phase`)
- `remove` - Delete task (requires `id`)
- `update` - Modify task content (requires `id`, optional `title`, `details`, `references`, `status`)

**Phase Support:**
- Include `"phase": "Phase Name"` in add operations to target a specific phase
- If the phase doesn't exist, it will be automatically created
- Phase is applied to the operation and cannot be combined with `parent` parameter

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

## Configuration

rune supports configuration files to customize behavior, including git branch-based file discovery.

### Configuration File Locations

Configuration is loaded in the following order of precedence:

1. `./.rune.yml` (project-local configuration)
2. `~/.config/rune/config.yml` (user-global configuration)

### Configuration Schema

```yaml
# Example configuration file
discovery:
  enabled: true
  template: "{branch}/tasks.md"
```

**Configuration Options:**

- `discovery.enabled` (boolean) - Enable/disable git branch-based file discovery (default: true)
- `discovery.template` (string) - Path template for branch-based files (default: "{branch}/tasks.md")

### Git Branch Discovery

When enabled, rune automatically discovers task files based on your current git branch:

**Examples:**
- Branch `feature/auth` with template `specs/{branch}/tasks.md` → `specs/feature/auth/tasks.md`
- Branch `bugfix/login` with template `tasks/{branch}.md` → `tasks/bugfix/login.md`
- Branch `main` with template `{branch}-tasks.md` → `main-tasks.md`

**Requirements:**
- Must be in a git repository
- Git must be available in PATH
- Target file must exist
- Works with branch names containing slashes (treated as path separators)

**Special Cases:**
- Detached HEAD: Requires explicit filename
- During rebase/merge: Requires explicit filename
- Non-git directory: Requires explicit filename

### Default Behavior

If no configuration file exists, rune uses these defaults:
- Git discovery enabled
- Template: `{branch}/tasks.md`

## File Format

rune supports two file formats: plain markdown and markdown with YAML front matter.

### Basic Markdown Format

rune generates consistent markdown with the following structure:

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

### Extended Format with YAML Front Matter

Task files can include YAML front matter for metadata and reference documents:

```markdown
---
references:
  - ./docs/architecture.md
  - ./specs/api-specification.yaml
  - ../shared/database-schema.sql
metadata:
  project: backend-api
  created: 2024-01-30
---
# Project Tasks

## Planning

- [ ] 1. Setup development environment
  - This involves setting up the complete development stack
  - including Docker containers and environment variables.
  - References: ./setup-guide.md, ./docker-compose.yml
  - [x] 1.1. Install dependencies
  - [ ] 1.2. Configure database
    - Create database schema and initial migrations.
    - Make sure to use the latest PostgreSQL version.
    - References: ./db/migrations/

## Implementation

- [x] 2. Implement authentication
- [ ] 3. Build API endpoints
  - [ ] 3.1. User endpoints
  - [ ] 3.2. Product endpoints
```

**Front Matter Fields:**

- `references` (array) - List of reference document paths
  - Can be relative or absolute paths
  - Included in all task retrieval commands
  - Path validation for security (no directory traversal)
  - File existence not validated (paths stored as-is)

- `metadata` (object) - Optional metadata fields
  - Extensible for future features
  - Not processed by current commands
  - Preserved when modifying tasks

**Reference Documents vs Task References:**

- **Front Matter References**: Apply to the entire task file, included in all outputs
- **Task References**: Apply to specific tasks, shown with task details

### Requirements

Link tasks to specific requirement acceptance criteria using the `--requirements` flag:

```bash
rune add tasks.md --title "Implement login" --requirements "1.1,1.2,2.3"
```

Specify a custom requirements file:

```bash
rune add tasks.md --title "Implement login" --requirements "1.1,1.2" --requirements-file "specs/requirements.md"
```

Requirements are rendered as clickable markdown links:

```markdown
- [ ] 1. Implement login
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2)
```

Update requirements:

```bash
rune update tasks.md 1 --requirements "3.1,3.2"
```

Clear requirements:

```bash
rune update tasks.md 1 --clear-requirements
```

**Requirements vs References:**

- **Requirements**: Links to acceptance criteria with automatic link generation pointing to requirement anchors
- **References**: Free-form text without link generation

### Using Phases to Organize Tasks

Phases provide a way to organize tasks into logical sections using H2 markdown headers. This is optional but useful for structuring larger projects.

**Basic Phase Usage:**

```markdown
# Project Tasks

## Planning

- [ ] 1. Define requirements
- [ ] 2. Create design documents

## Implementation

- [ ] 3. Set up project structure
- [ ] 4. Implement core features

## Testing

- [ ] 5. Write unit tests
- [ ] 6. Perform integration testing
```

**Key Features:**
- Task IDs continue sequentially across phases (1, 2, 3... regardless of phase boundaries)
- Phases are created with `rune add-phase "Phase Name"` or automatically when using `--phase` flag
- Tasks can be added to phases with `rune add --title "Task" --phase "Phase Name"`
- Use `rune next --phase` to get all tasks from the next incomplete phase
- Phase information appears in table and JSON output when phases are present
- Phases are completely optional - tasks work normally without them

## Examples

See the [`examples/`](examples/) directory for sample task files and common usage patterns:

- [`examples/simple.md`](examples/simple.md) - Basic task list
- [`examples/project.md`](examples/project.md) - Software project with phases
- [`examples/complex.md`](examples/complex.md) - Deep hierarchy with all features
- [`examples/batch-operations.json`](examples/batch-operations.json) - Sample batch operations (hierarchical)
- [`examples/batch-operations-phases.json`](examples/batch-operations-phases.json) - Sample batch operations with phases
- [`examples/phases/`](examples/phases/) - Phase-based task organization examples

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

rune follows strict error reporting without auto-correction:

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

## Troubleshooting

### Common Issues

**"Failed to read task file" errors:**
- Verify the file path exists and is readable
- Check file permissions (read/write access required)
- Ensure file size is under 10MB limit

**Git discovery not working:**
- Confirm you're in a git repository: `git status`
- Check that git is installed and in PATH: `git --version`
- Verify the target file exists at the computed path
- Use `--verbose` flag to see the resolved file path: `rune next --verbose`

**"No filename specified and git discovery failed" error:**
- Either specify a filename explicitly: `rune next tasks.md`
- Or configure git discovery in `.rune.yml`
- Or ensure you're in a git repository with the expected file structure

**Configuration file not loading:**
- Verify YAML syntax with an online YAML validator
- Check file permissions on config directories
- Use absolute paths if relative paths don't work
- Ensure config file is in expected location (`./.rune.yml` or `~/.config/rune/config.yml`)

**"All tasks are complete" when tasks remain:**
- Check task completion status - both parent and all children must be `[x]` to be considered complete
- Use `rune list --status pending` to see incomplete tasks
- Verify task syntax matches expected format

**Front matter parsing errors:**
- Ensure YAML front matter is properly delimited with `---` lines
- Validate YAML syntax (proper indentation, no tabs)
- Check that front matter appears at the very beginning of the file

### Debug Options

**Verbose Mode:**
```bash
rune next --verbose
# Shows file resolution, task parsing details, and discovery logic
```

**Validate Configuration:**
```bash
# Test git discovery
rune next --format json
# Check if correct file is being used

# Manual file specification to bypass discovery
rune next /full/path/to/tasks.md
```

### Performance Issues

**Large Files:**
- Files over 1MB may have slower parse times
- Consider splitting large task lists into multiple files
- Use `--depth` flag to limit hierarchy traversal: `rune list --depth 2`

**Deep Task Hierarchies:**
- Maximum recommended depth: 10 levels
- Deep hierarchies may impact performance and readability
- Consider flattening structure or using multiple files

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

- **Issues**: [GitHub Issues](https://github.com/ArjenSchwarz/rune/issues)
- **Discussions**: [GitHub Discussions](https://github.com/ArjenSchwarz/rune/discussions)
- **Documentation**: [Wiki](https://github.com/ArjenSchwarz/rune/wiki)