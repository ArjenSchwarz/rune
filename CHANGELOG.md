# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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