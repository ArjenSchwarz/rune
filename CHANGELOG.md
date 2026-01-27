# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Documentation

- **README.md**: Document task dependencies, work streams, and task ownership features
  - Add Task Dependencies section with blocked-by usage and storage format
  - Add Work Streams section with stream assignment and status checking
  - Add Task Ownership section with claiming and filtering by owner
  - Update streams command documentation with options and examples
  - Update next command documentation with --stream and --claim flags
  - Update list command documentation with --stream and --owner filters
  - Update add command documentation with --stream, --blocked-by, --owner flags
  - Update update command documentation with --stream, --blocked-by, --owner, --release flags

- **docs/AGENT_INSTRUCTIONS.md**: Add multi-agent workflow guidance
  - Add Multi-Agent Parallel Execution section with stream setup and orchestrator patterns
  - Add Task Dependencies section with ready vs blocked explanation
  - Update batch operation examples with streams and dependencies
  - Update Quick Reference with new command flags

- **docs/json-api.md**: Document new JSON schemas for dependencies and streams
  - Add stream, blocked_by, owner, release fields to Operation schema
  - Add blockedBy, stream, owner fields to Task schema
  - Add StreamsResult and StreamStatus schemas for streams command output
  - Add ClaimResult schema for next --claim output
  - Add Warning schema for non-fatal operational issues
  - Add operation examples with new fields

- **cmd/list.go**: Enhance Long description with filtering and column display documentation

- **examples/parallel-agents.md**: Add example task file demonstrating multi-agent setup with streams and dependencies

### Fixed

- **examples/parallel-agents.md**: Fix invalid markdown format that prevented parsing
  - Stream metadata now uses correct list item format (`- Stream: N`) instead of HTML comment attributes
  - Stable IDs updated to valid 7-character alphanumeric format (e.g., `bknd002` instead of `backend001`)
  - Removed non-task content (introductory paragraph and H2 header) that caused parser errors

### Added

- **Add Command Enhancements**: Extended add command for task dependencies and streams
  - `--stream N` flag assigns task to a specific work stream (positive integer)
  - `--blocked-by IDs` flag sets task dependencies (comma-separated task IDs)
  - `--owner AGENT` flag claims the task for a specific agent
  - Blocked-by references are validated (target tasks must have stable IDs)
  - Uses `AddTaskWithOptions` internally when extended options are specified
  - Unit tests for all new flag combinations and error handling

- **Update Command Enhancements**: Extended update command for task dependencies and streams
  - `--stream N` flag updates task's stream assignment
  - `--blocked-by IDs` flag updates task dependencies (comma-separated task IDs)
  - `--owner AGENT` flag updates task owner
  - `--release` flag clears the task owner (releases the task)
  - Cycle detection prevents circular dependencies on blocked-by updates
  - Uses `UpdateTaskWithOptions` internally when extended options are specified
  - Unit tests for stream, blocked-by, owner, release flags and cycle detection

- **List Command Enhancements**: Extended list command for task dependencies and streams
  - `--stream N` flag filters tasks by stream number using `GetEffectiveStream()`
  - `--owner NAME` flag filters tasks by owner (use empty string for unowned tasks)
  - Stream column conditionally displayed only when non-default streams exist in the file
  - BlockedBy column displayed as hierarchical IDs (not stable IDs) for user readability
  - Owner column displayed when any task has an owner
  - JSON output includes blockedBy, stream, and owner fields for all tasks
  - Combined filtering: `--filter pending --stream 2 --owner alice`
  - Unit tests for all new filter options and display enhancements

- **Next Command Stream and Claim Support**: Extend next command for parallel agent coordination
  - `--stream N` flag filters tasks to a specific stream using `GetEffectiveStream()`
  - `--claim AGENT_ID` claims the next ready task by setting status to in-progress and owner
  - `--stream N --claim AGENT_ID` combination claims ALL ready tasks in the specified stream
  - Phase JSON output now includes `streams_summary` section with ready/blocked/active/available per stream
  - Phase output includes stream and dependency metadata (blockedBy) for each task
  - Stream filtering supported in phase mode via `--phase --stream N`
  - Claim operations atomically write updated task status and owner to file
  - Unit tests for stream filtering, claim operations, and combined stream+claim scenarios

- **Extended Batch Operations**: Batch API now supports task dependencies and streams
  - `Operation` struct extended with Stream, BlockedBy, Owner, and Release fields
  - Batch add operations support all new fields via `AddTaskWithOptions`
  - Batch update operations support all new fields via `UpdateTaskWithOptions`
  - Validation includes stream range, blocked-by existence, owner format, and cycle detection
  - Atomic failure: invalid operations cause entire batch to fail
  - Phase-aware operations also support extended fields
  - Unit tests for all extended batch scenarios

- **Extended Task Operations**: Support for stream, blocked-by, and owner options in add/update/remove operations
  - `AddOptions` struct with Position, Phase, Stream, BlockedBy, and Owner fields
  - `UpdateOptions` struct with Stream, BlockedBy, Owner, and Release fields
  - `AddTaskWithOptions()` generates stable ID and applies extended options
  - `UpdateTaskWithOptions()` validates and applies stream, blocked-by, owner, and release options
  - `RemoveTaskWithDependents()` warns about and cleans up dependent task references
  - `resolveToStableIDs()` converts hierarchical IDs to stable IDs with validation
  - `removeFromBlockedByLists()` removes stable ID from all BlockedBy lists
  - `collectStableIDs()` gathers all stable IDs from task hierarchy
  - `validateOwner()` checks owner strings for invalid control characters
  - Cycle detection via `DependencyIndex.DetectCycle()` on blocked-by updates
  - Unit tests for all extended operations including edge cases

- **Parser Extensions for Task Dependencies and Streams**: Parse new metadata fields from markdown
  - `stableIDCommentPattern` regex extracts stable IDs from HTML comments (`<!-- id:abc1234 -->`)
  - `blockedByPattern` regex parses Blocked-by metadata lines with case-insensitive matching
  - `streamPattern` regex parses Stream metadata lines (positive integers only)
  - `ownerPattern` regex parses Owner metadata lines
  - `blockedByRefPattern` extracts stable IDs from references with optional title hints
  - `extractStableIDFromTitle()` removes stable ID comment from task title
  - `parseBlockedByLine()`, `parseStreamLine()`, `parseOwnerLine()` helper functions
  - Lenient parsing: invalid formats are ignored without errors, supporting legacy files
  - Unit tests for all parsing scenarios (valid, invalid, case-insensitive, mixed)
  - Negative tests for malformed input handling

- **Stable ID Generator**: Generate unique 7-character base36 identifiers for tasks
  - `StableIDGenerator` struct with collision detection and counter continuation
  - `NewStableIDGenerator` seeds from existing IDs or crypto/rand for new files
  - `Generate()` produces unique IDs with zero-padding and uniqueness verification
  - `IsUsed()` check for ID collision detection
  - `IsValidStableID()` validation for 7-character lowercase alphanumeric format
  - Unit tests covering uniqueness, encoding, and counter continuation
  - Property-based tests using rapid framework for uniqueness guarantees

- **Dependency Index**: Fast lookup for dependency resolution and cycle detection
  - `DependencyIndex` struct with byStableID, byHierarchical, and dependents maps
  - `BuildDependencyIndex()` creates index from task list with recursive child indexing
  - `GetTask()` and `GetTaskByHierarchicalID()` for task lookup by ID type
  - `GetDependents()` returns tasks that depend on a given task
  - `IsReady()` and `IsBlocked()` for dependency status checking
  - `TranslateToHierarchical()` converts stable IDs to hierarchical IDs
  - `DetectCycle()` with DFS algorithm for circular dependency detection
  - Unit tests for index building, lookups, and dependency status
  - Property-based tests using rapid framework for cycle detection guarantees

- **Task Dependencies and Streams Core Data Structures**: Foundation for parallel agent execution
  - Extended Task struct with StableID, BlockedBy, Stream, and Owner fields
  - GetEffectiveStream() helper function returns stream 1 as default when not explicitly set
  - Error types for stable ID, dependency, stream, and owner validation (ErrNoStableID, ErrCircularDependency, ErrInvalidStream, ErrInvalidOwner, etc.)
  - CircularDependencyError struct for detailed cycle path information
  - Warning struct and codes for non-fatal issues during operations

- **Streams Command**: Display stream status for parallel agent orchestration
  - `rune streams [file]` command to display all work streams and their task counts
  - `--available` flag filters to only show streams with ready tasks
  - `--json` flag outputs structured JSON with task ID arrays per stream
  - Shows count of ready, blocked, and active tasks per stream
  - Filters out empty streams (all tasks completed) from output
  - Supports automatic file discovery via git branch when enabled

- **Stream Analysis**: Analyze work streams for parallel agent orchestration
  - `StreamStatus` struct with ID, Ready, Blocked, Active hierarchical task ID arrays
  - `StreamsResult` struct containing all streams and available stream IDs
  - `AnalyzeStreams()` computes stream status with ready/blocked/active task classification
  - `FilterByStream()` returns tasks belonging to a specific stream
  - Supports nested tasks, cross-stream dependencies, and owned task handling
  - Streams are sorted by ID in output for consistent ordering

- **Markdown Renderer Extensions**: Render new dependency and stream metadata to markdown
  - `RenderContext` struct to pass dependencies for rendering (requirements file, dependency index)
  - Stable IDs rendered as HTML comments after task title (`<!-- id:abc1234 -->`)
  - `formatBlockedByRefs()` renders Blocked-by with title hints for readability
  - Stream metadata rendered as `Stream: N` (only when explicitly set, not for default stream)
  - Owner metadata rendered as `Owner: agent-id` (only when non-empty)
  - Metadata ordering: Details, Blocked-by, Stream, Owner, Requirements, References
  - JSON output excludes StableID field (system-managed, not for external use)
  - Unit tests for all metadata rendering scenarios
  - Property-based tests ensuring parse-render round-trip preservation

- **Task Dependencies and Streams Specification**: Complete spec-driven design for parallel agent execution
  - Requirements document with 9 sections covering stable IDs, dependencies, streams, ownership, and backward compatibility
  - Design document with architecture diagrams, component interfaces, data models, and testing strategy
  - Decision log with 10 key architectural decisions (hybrid storage, no auto-assignment, cycle detection, etc.)
  - Implementation task list with 42 tasks across 13 phases following test-driven development

### Changed

- **Smart Branch Discovery**: Branch-based file discovery now intelligently strips branch prefixes
  - Branches like `specs/my-feature` or `feature/auth` now resolve correctly by trying the stripped name first (`my-feature`, `auth`)
  - Falls back to full branch name if stripped path doesn't exist
  - Single-component branches (e.g., `main`) avoid duplicate path attempts
  - Error messages now list all candidate paths that were tried

### Fixed

- **Batch Remove Phase Preservation**: Batch remove operations now correctly adjust phase markers after each removal to preserve phase boundaries
  - Phase-aware batch execution is used when the file has phase markers, even if no operations specify a phase
  - Batch removes process in reverse order (highest ID first) so users can specify original task IDs

### Added

- **Batch Remove Tests**: Tests verifying batch remove operations work with original task IDs and preserve phase boundaries
  - `TestExecuteBatch_MultipleRemovesOriginalIDs` - verifies multiple removes use original IDs
  - `TestExecuteBatch_RemovesWithAddsOriginalIDs` - verifies mixed add/remove operations
  - `TestExecuteBatchWithPhases_RemovePreservesPhases` - verifies phase preservation on single remove
  - `TestExecuteBatchWithPhases_MultipleRemovesPreservesPhases` - verifies phase preservation on multiple removes

- **Success and Count Fields in Read Commands**: Added `success` and `count` fields to JSON responses for read commands (`list`, `find`, `next`, `next --phase`) to comply with requirement 1.1 of the consistent output format specification
- **Integration Tests for Non-Empty JSON Responses**: Tests verifying `success` and `count` fields are present in non-empty JSON responses from read commands

- **Format-Specific Integration Tests**: End-to-end tests for consistent output format feature
  - Mutation commands JSON format tests (complete, uncomplete, progress, add, remove, update)
  - Create command JSON format tests with references and metadata
  - Empty state JSON format tests (next all-complete, list empty, find no-matches)
  - Verbose + JSON stderr separation tests to verify verbose output goes to stderr

- **Integration Tests for Task Dependencies and Streams**: End-to-end validation of multi-agent workflow capabilities
  - Multi-agent workflow test: parallel streams claiming, dependency blocking, cross-stream dependencies
  - Dependency chain resolution test: A → B → C → D chain validation, cycle and self-dependency prevention
  - Backward compatibility test: legacy files without stable IDs, mixed files with old/new task formats

- **Format Utilities**: Shared utility functions for consistent output format handling
  - `outputJSON` function for standardized JSON output to stdout
  - `outputMarkdownMessage` function for markdown blockquote messages
  - `outputMessage` function for plain text messages
  - `verboseStderr` function to write verbose output to stderr when JSON format is used

### Changed

- **Phase Marker Adjustment**: Extracted duplicate phase marker adjustment logic into `adjustPhaseMarkersForRemoval` helper function for maintainability
- **Renumber Command**: Refactored JSON output to use typed `RenumberResponse` struct for consistency with other commands
- **has-phases Command**: Updated help text to document that the command only outputs JSON and ignores the `--format` flag

### Removed

- Unused `outputJSON` function from list command (replaced by `outputJSONWithPhases`)

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