# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-10-07

### Added

- **Task Phases**: Organize tasks under semantic H2 markdown headers (phases) for better project structure
  - `add-phase` command to add phase headers to task files
  - `--phase` flag on `add` command to add tasks to specific phases (auto-creates phases if needed)
  - `--phase` flag on `next` command to retrieve all pending tasks from the next phase with work
  - `has-phases` command for programmatic phase detection with JSON output
  - Phase information displayed in table/JSON/markdown output when present
  - Batch operations support phase field for adding tasks to specific phases

- **Front Matter Support**: Add YAML front matter to task files for metadata and references
  - `--reference` and `--meta` flags on `create` command for adding front matter during file creation
  - `add-frontmatter` command to add metadata and references to existing task files
  - References displayed in all output formats (table, JSON, markdown)
  - Front matter preserved during all task operations

- **Git Branch Discovery**: Automatic task file location based on git branch
  - Enabled by default with `{branch}/tasks.md` pattern
  - All commands support optional filename (auto-discovers if not provided)
  - Configuration via `.rune.yml` or `~/.config/rune/config.yml`
  - Override with explicit filename when needed

- **Next Task Workflow**: Sequential task management for focused work
  - `next` command finds first incomplete task using depth-first traversal
  - Task details and references included in output
  - Multiple output formats (table, JSON, markdown)
  - Git discovery integration for automatic file location

- **Auto-completion**: Automatically completes parent tasks when all children are done
  - Recursive parent checking up the task hierarchy
  - Multi-level auto-completion support
  - Visual feedback with 🎯 emoji for auto-completed tasks
  - Works in both `complete` command and batch operations

- **Position-based Task Insertion**: Insert tasks at specific positions with `--position` flag
  - Works with top-level tasks and subtasks (e.g., `--position "1.2"`)
  - Automatic renumbering of existing tasks
  - Available in both `add` command and batch operations

- **Core Task Management**: Complete CLI for hierarchical markdown task lists
  - `create` command to generate new task files
  - `add` command for adding tasks with parent-child relationships
  - `list`, `find` commands with filtering and multiple output formats
  - `complete`, `progress`, `uncomplete` for status management
  - `update`, `remove` commands with automatic ID renumbering
  - `batch` command for atomic multi-operation execution

### Security

- File size limit enforcement (10MB maximum)
- Path traversal protection with validation
- Input sanitization for null bytes and control characters
- Resource limits (10,000 tasks max, 10 levels deep, 1,000 char details)
- Branch name sanitization to prevent command injection