# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2026-02-06

### Added

- **Task Dependencies**: Tasks can declare dependencies on other tasks using `--blocked-by`
  - Dependencies stored using stable IDs that survive task renumbering
  - Automatic cycle detection prevents circular dependencies
  - Dependent references cleaned up when blocking tasks are removed
  - `next` command only returns "ready" tasks (all blockers completed)

- **Work Streams**: Partition tasks for parallel execution across multiple agents
  - `--stream N` flag on `add`, `update`, and `list` commands
  - `streams` command shows ready, blocked, and active task counts per stream
  - `--available` flag filters to streams with ready tasks
  - Cross-stream dependencies supported

- **Task Ownership**: Claim tasks for specific agents
  - `--owner AGENT_ID` flag on `add` and `update` commands
  - `--release` flag on `update` to clear ownership
  - `--owner` filter on `list` command (empty string for unowned tasks)

- **Task Claiming**: Atomic claim operations for multi-agent coordination
  - `next --claim AGENT_ID` claims the next ready task (sets in-progress + owner)
  - `next --stream N --claim AGENT_ID` claims all ready tasks in a stream
  - `next --phase --stream N --claim AGENT_ID` claims ready stream tasks from the appropriate phase

- **Stream-Aware Phase Navigation**: `next --phase --stream N` finds the first phase with ready tasks in the specified stream
  - Blocking status indicators in all output formats (JSON, table, markdown)
  - Backward compatible with existing `--phase` and `--stream` behaviors

- **Batch Add-Phase Operation**: New `add-phase` operation type for the batch JSON API to create phase headers programmatically

- **Streams Command**: `rune streams [file]` displays work stream status
  - `--available` flag shows only streams with ready tasks
  - `--json` flag outputs structured JSON for scripting

- **Consistent Output Format**: All commands now include `success` and `count` fields in JSON output, with verbose output directed to stderr when using JSON format

- **Install Target**: `make install` installs rune binary to `$GOPATH/bin`

### Changed

- **Dependencies**: Updated `go-output/v2` (v2.2.0 → v2.6.0), `cobra` (v1.9.1 → v1.10.2)
- **Smart Branch Discovery**: Branch prefix stripping now uses the first slash instead of the last
  - `feature/auth/oauth` strips to `auth/oauth` (previously `oauth`)
  - Full branch path is tried as fallback for backward compatibility
- **Default Discovery Template**: Changed from `{branch}/tasks.md` to `specs/{branch}/tasks.md`
  - Users can override in `.rune.yml` or `~/.config/rune/config.yml`
- List command conditionally shows Stream, BlockedBy, and Owner columns only when relevant data exists

### Fixed

- Phase marker corruption when removing tasks from phase-based files
- Batch remove operations now preserve phase boundaries and process in reverse order
- Negative `--stream` flag values now rejected with a clear error message

## [1.1.0] - 2025-11-12

### Added

- **Renumber Command**: Fix task numbering and maintain file consistency
  - Recalculates all task IDs sequentially while preserving hierarchy
  - Automatic backup creation with `.bak` extension before operations
  - Phase marker preservation with automatic `AfterTaskID` adjustment
  - YAML front matter preservation during renumbering
  - Atomic file write operations for data safety
  - Supports table, markdown, and JSON output formats

- **GitHub Action**: Official action for installing rune in GitHub workflows
  - Cross-platform support (Linux, macOS, Windows) with amd64/arm64 architectures
  - Version resolution with "latest" or specific version support
  - Automatic caching using GitHub Actions tool-cache
  - MD5 checksum verification for integrity
  - Outputs for version and installation path

### Changed

- **Test Suite Modernization**: Refactored to follow Go 2025 best practices
  - Converted slice-based table tests to map-based for better isolation
  - Split monolithic integration test file into focused test files by feature
  - Improved test maintainability and clarity

## [1.0.0] - 2025-10-08

### Added

- **Task Requirements Linking**: Link tasks to requirement acceptance criteria
  - `--requirements` flag on `add` and `update` commands for comma-separated requirement IDs
  - `--requirements-file` flag to specify requirements document (defaults to "requirements.md")
  - `--clear-requirements` flag to remove all requirements from a task
  - Requirements rendered as markdown links `[ID](file#ID)` in task output
  - Full batch operation support with requirements validation
  - Round-trip preservation in parse-render cycles

- **Task Phases**: Organize tasks under semantic H2 markdown headers
  - `add-phase` command to add phase headers to task files
  - `--phase` flag on `add` and `next` commands
  - `has-phases` command for programmatic phase detection
  - Phase information displayed in all output formats

- **Front Matter Support**: YAML front matter for metadata and references
  - `--reference` and `--meta` flags on `create` command
  - `add-frontmatter` command for existing files
  - References displayed in all output formats

- **Git Branch Discovery**: Automatic task file location based on git branch
  - Default `{branch}/tasks.md` pattern
  - Configuration via `.rune.yml` or `~/.config/rune/config.yml`

- **Next Task Workflow**: Sequential task management
  - `next` command finds first incomplete task via depth-first traversal
  - Task details and references in output

- **Auto-completion**: Parent tasks complete when all children are done
  - Recursive hierarchy checking
  - Visual feedback with 🎯 emoji

- **Position-based Insertion**: `--position` flag for precise task placement
  - Automatic ID renumbering

- **Core Task Management**: Complete CLI for hierarchical markdown task lists
  - CRUD operations: `create`, `add`, `update`, `remove`
  - Status management: `complete`, `progress`, `uncomplete`
  - Query operations: `list`, `find`
  - Batch operations: atomic multi-operation execution

### Changed

- Improved code organization with consolidated helper functions
- Refactored test suites into focused files by functionality
- Simplified ID validation and task parsing logic

### Security

- File size limit (10MB)
- Path traversal protection
- Input sanitization for null bytes and control characters
- Resource limits (10,000 tasks max, 10 levels deep, 1,000 char details)
- Branch name sanitization