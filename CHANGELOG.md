# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- Fixed renumber command phase marker handling for out-of-order tasks
  - Phase markers now correctly maintain their positional relationship after renumbering
  - Added `extractTaskIDOrder()` to capture original task IDs from file before parser renumbers them
  - Converted phase marker adjustment from simple ID-based mapping to position-based tracking
  - Phase markers now reference task positions in file rather than task IDs, preventing misplacement when tasks are out of order
  - Replaced `adjustPhaseMarkersAfterRenumber()` with `convertPhaseMarkersToPositions()` and `convertPhasePositionsToMarkers()` for accurate position tracking
  - Updated unit tests to cover position-based conversion and task ID extraction
  - All existing integration tests continue to pass

### Changed

- Refactored test suite to follow Go 2025 best practices
  - Converted all slice-based table tests to map-based table tests for better test isolation and clearer test names
  - Split monolithic integration_test.go (4,818 lines) into focused test files by feature area
  - Created integration_helpers_test.go (67 lines) with shared test setup and helper functions
  - Created integration_batch_test.go (1,349 lines) for batch operations and position insertion tests
  - Created integration_phase_test.go (921 lines) for phase-related workflow tests
  - Created integration_renumber_test.go (738 lines) for renumber command integration tests
  - Created integration_requirements_test.go (416 lines) for requirements workflow tests
  - Reduced main integration_test.go to 1,527 lines containing core workflow tests
  - Updated test variable naming from `tt` to `tc` for consistency with Go conventions
  - Added descriptive test case names in all map-based tests (e.g., "empty slice", "pending", "root task")
  - Improved test maintainability with logical file groupings and independent test runners per file
- Simplified renumber command implementation
  - Removed custom `contains()` helper function wrapper, replaced 17 calls with direct `strings.Contains()` usage
  - Removed redundant `len(phaseMarkers) > 0` check before calling `adjustPhaseMarkersAfterRenumber()` as function handles empty slices correctly
  - Net reduction of 16 lines in code complexity while maintaining full test coverage and functionality

### Added

- Renumber command integration tests and documentation (Phase 7-8)
  - Integration test for end-to-end renumber workflow validating backup creation, task renumbering, status preservation, and atomic write behavior
  - Integration test for renumbering with phases verifying phase marker preservation and AfterTaskID updates
  - Integration test for front matter preservation ensuring YAML headers remain intact during renumbering
  - Integration test for write failure scenarios confirming original file remains untouched on errors
  - Integration test for symlink security documenting known ValidateFilePath limitation
  - Integration test for malformed phase markers handling edge cases gracefully
  - Integration test for large file handling testing 1000 tasks with 10-level hierarchy depth
  - README.md documentation for renumber command with usage examples, feature descriptions, and important notes
  - Enhanced cmd/renumber.go Long description with detailed features, workflow steps, and use cases
  - All tests passing with documented known issues for pre-existing limitations

- Renumber command comprehensive testing (Phase 5-6)
  - Unit tests for displaySummary() function covering table, markdown, and JSON output formats with stdout capture and validation
  - Unit tests for error handling covering parse errors, backup failures, and validation errors
  - Unit tests for edge cases covering empty files, phase-only files, and truly empty files
  - Test fixtures in examples/ directory: empty.md, phases_only.md, tasks_malformed.md, tasks_with_gaps.md
  - Error handling tests for invalid status markers, tab indentation, missing checkbox spaces
  - Edge case validation for task count=0, phase marker preservation, and backup creation
  - All output format tests verify correct field structure and content

- Renumber command unit tests (Phase 2-4)
  - Unit tests for createBackup() function covering content verification, permission preservation, backup overwriting, and error handling
  - Unit tests for validation phase covering ValidateFilePath, file size limits, task count limits, and validation order
  - Unit tests for adjustPhaseMarkersAfterRenumber() covering empty arrays, phases at beginning, phases after root tasks, phases after nested tasks, and multiple phases
  - Unit tests for getRootTaskNumber() helper function covering all task ID formats
  - Phase marker adjustment implementation in runRenumber() to update AfterTaskID values after renumbering
  - Helper functions adjustPhaseMarkersAfterRenumber() and getRootTaskNumber() for phase marker management

- Renumber command (Phase 1: Core Command Structure)
  - `renumber` command to fix task numbering by recalculating all task IDs sequentially
  - Automatic backup creation with .bak extension before renumbering operations
  - Support for multiple output formats (table, markdown, json) displaying task count, backup file location, and success status
  - Phase-aware renumbering that preserves phase markers in files
  - Atomic file write operations using temporary files for data safety
  - Exported task management functions for command-level access (ValidateFilePath, CountTotalTasks, RenumberTasks)
  - Resource limit validation (file size, task count) before renumbering
  - Command registration with cobra framework and integration with global --format flag

- Renumber command specification documentation
  - Requirements document covering renumbering logic, error handling, backup management, output formats, and edge cases
  - Design document with detailed architecture, component interfaces, phase marker adjustment logic, and implementation plan
  - Decision log tracking 13 architectural and implementation decisions
  - Implementation tasks organized in 8 phases (34 tasks total) covering core structure, backup, validation, renumbering, output, error handling, integration testing, and documentation
  - Hierarchical sequential numbering approach (1, 1.1, 1.2, 2, 2.1...) that maintains task hierarchy
  - Automatic backup creation with .bak extension before renumbering
  - Phase marker preservation and automatic AfterTaskID adjustment after renumbering
  - YAML front matter preservation during renumbering
  - Atomic write operations with temp file pattern for data safety
  - Support for multiple output formats (table, markdown, json)
  - Resource limit validation (10MB file size, 10,000 tasks, 10 hierarchy levels)
  - Path traversal protection and security constraints
  - Edge case handling (empty files, phase-only files, malformed hierarchies, duplicate IDs)

- GitHub Action specification documentation
  - Requirements document covering installation, versioning, platform support, caching, and integrity verification
  - Design document with simplified architecture emphasizing maintainability
  - Decision log tracking 21 architectural and implementation decisions
  - Implementation tasks organized in 4 phases (setup, core, testing, release)
- GitHub Action project setup (Phase 1)
  - TypeScript project initialization with GitHub Actions dependencies (@actions/core, @actions/tool-cache, @actions/github, @actions/exec)
  - Action metadata file (action.yml) with version and github-token inputs, version and path outputs
  - Jest test infrastructure with TypeScript support and coverage reporting
  - Build configuration with @vercel/ncc for bundling to dist/index.js
  - Development documentation and project structure
- GitHub Action core implementation (Phase 2)
  - `resolveVersion()` function for "latest" and exact version resolution via GitHub API
  - `getPlatformAsset()` function for platform detection across Linux/macOS/Windows on amd64/arm64
  - `verifyChecksum()` function for MD5 integrity verification with streaming file reading
  - `installRune()` orchestration function handling version resolution, cache checks, downloads, checksum verification, extraction, and PATH management
  - `main.ts` entry point with input handling, error catching, and output setting
  - Distribution bundle (dist/index.js) with all dependencies bundled for GitHub Actions execution
  - Unit test suite with 23 tests achieving 100% code coverage across all metrics
  - Support for .tar.gz extraction on Unix platforms and .zip extraction on Windows
  - Automatic cache management using GitHub Actions tool-cache with version and architecture isolation
  - Cross-platform chmod handling for binary executable permissions
- Integration test workflow (.github/workflows/test.yml) for GitHub Action
  - Multi-platform testing matrix (ubuntu-latest, macos-latest, windows-latest)
  - Version testing with both specific version (1.0.0) and latest
  - Cache behavior validation with repeated installations
  - Binary verification tests (PATH, --version output, binary existence)
  - Output validation tests (version and path outputs with format checks)
  - Functional testing (create, add, list operations)
  - Error handling tests (non-existent version, graceful failure, clear error messages)
- GitHub Action documentation and release preparation (Phase 4)
  - Comprehensive README with usage examples for all supported platforms (Ubuntu, macOS, Windows)
  - Input/output documentation with detailed tables
  - Example workflows demonstrating basic usage, specific versions, output usage, and matrix strategies
  - Caching behavior explanation with performance benefits
  - Integrity verification documentation
  - Troubleshooting guide for common issues (version not found, unsupported platform, checksum failures, rate limiting)
  - Development setup and contribution guidelines
  - GitHub Actions section in main README with quick start example and link to detailed documentation
  - Production bundle built and ready for release

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