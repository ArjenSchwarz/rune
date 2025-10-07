# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Task Requirements Linking: Update command implementation
  - `--requirements` flag added to `update` command for comma-separated requirement IDs (e.g., "1.1,1.2,2.3")
  - `--clear-requirements` flag added to clear all requirements from a task
  - Modified `UpdateTask` signature to accept requirements parameter (nil = no change, empty = clear)
  - Requirement ID validation using hierarchical ID pattern matching
  - Requirements display in dry-run mode showing current and new values
  - Comprehensive unit tests covering flag parsing, validation, clearing, and whitespace handling

- Task Requirements Linking: CLI command implementation
  - `--requirements` flag added to `add` command for comma-separated requirement IDs (e.g., "1.1,1.2,2.3")
  - `--requirements-file` flag added to `add` command to specify requirements file path
  - Requirement ID validation using existing hierarchical ID pattern matching
  - Automatic default to "requirements.md" when requirements provided without explicit file
  - `parseRequirementIDs` helper function for parsing comma-separated requirement strings
  - Comprehensive unit tests covering flag parsing, validation, and requirements file defaults

- Task Requirements Linking: Requirements rendering implementation
  - Updated `renderTask` function to accept requirements file parameter and render requirements
  - Requirements formatted as markdown links `[ID](file#ID)` with comma separation
  - Requirements appear before references section in task output
  - Default requirements file handling when not explicitly set
  - Support for nested tasks with requirements
  - Comprehensive unit tests covering rendering scenarios, round-trip parsing, link format validation, and positioning

- Task Requirements Linking: Requirements parsing implementation
  - `parseRequirements` helper function to extract requirement IDs from markdown links
  - Pattern matching for `[ID](file#ID)` format in Requirements detail lines
  - Support for comma-separated requirement links
  - Automatic extraction of requirements file path from first valid link
  - Integration with `parseDetailsAndChildren` to populate `task.Requirements` and `taskList.RequirementsFile`
  - Malformed requirement lines (plain text without markdown links) treated as regular details
  - Comprehensive unit tests covering single/multiple requirements, custom requirement files, whitespace handling, and error cases
  - Full integration tests verifying round-trip parsing with requirements preserved

- Task Requirements Linking: Core data structure implementation
  - `Requirements` field added to Task struct for linking requirement IDs
  - `RequirementsFile` field added to TaskList struct for specifying requirements document
  - `DefaultRequirementsFile` constant for default requirements file name ("requirements.md")
  - Validation for requirement IDs matching hierarchical pattern (e.g., "1", "1.2", "1.2.3")
  - Comprehensive unit tests covering valid and invalid requirement ID formats

- Task Requirements Linking feature specification
  - Decision log documenting design choices for linking tasks to requirement acceptance criteria
  - Comprehensive design document covering architecture, data models, and component interfaces
  - Requirements document with acceptance criteria for the feature
  - Tasks breakdown for implementation phases

### Changed

- Updated Claude Code settings to include codex-agent in pre-approved tools
- Cleaned up Claude Code settings by removing obsolete command approvals

### Fixed

- Fixed goconst linting issue by using existing formatJSON constant in version command
- Fixed revive linting issue by adding proper documentation comments for exported build variables

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