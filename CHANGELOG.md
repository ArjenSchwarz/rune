# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Enhanced next command specification to include task details and task-level references in output
  - New requirement 1.10 for including task details (multi-line descriptions or notes) in output
  - New requirement 1.11 for including task-level references in addition to front matter references
  - New task 11 "Enhance next command to include task details and task-level references" with subtasks for testing and implementation

### Changed
- Updated requirement 1.6 to specify returning task details and task-level references
- Updated requirements 5.3-5.6 to include both task details and reference types in all output formats
- Updated design document examples to show tasks with details and references
- Updated TaskWithContext structure documentation to clarify included fields
- Renumbered tasks 11-13 to 12-14 to accommodate new task 11

### Added
- **Git discovery integration with all commands**: Complete integration of git branch-based file discovery
  - Updated all commands (add, complete, find, list, progress, remove, uncomplete, update) to support optional filenames
  - Automatic filename resolution using git discovery when no explicit file is provided
  - New shared filename resolution helper across commands for consistent behavior
  - Enhanced command argument patterns to support both traditional and git-discovery modes
- **Comprehensive integration tests for git discovery**: End-to-end testing of git discovery workflow
  - Git repository setup and branch-based file discovery testing
  - Integration tests covering list, next, complete, find, and add commands with git discovery
  - Error handling tests for missing files and invalid branch states
  - Explicit file override testing to ensure precedence works correctly

### Changed
- **Configuration defaults updated**: Git discovery now enabled by default for better user experience
  - Default configuration sets Discovery.Enabled to true
  - Default template uses "{branch}/tasks.md" pattern for simpler branch-based workflows
  - Updated tests and specifications to match enabled-by-default behavior

### Fixed
- **Configuration test alignment**: Fixed test expectations to match new default configuration
  - Updated TestLoadConfig and TestDefaultConfig to expect discovery enabled by default
  - Aligned specification documentation with actual implementation behavior

### Added
- **Reference rendering support**: Enhanced output formats to include FrontMatter references
  - Added references section to table output format with dedicated references table
  - Preserved FrontMatter structure in markdown rendering using SerializeWithFrontMatter
  - Added FormatTaskListReferences helper function for table reference formatting
  - Comprehensive test coverage for reference rendering across all output formats
  - Updated list command table rendering to pass TaskList instead of just title string

### Added
- Auto-completion functionality for batch operations that automatically completes parent tasks when all children are completed
  - Tracking and reporting of auto-completed tasks in batch operation responses  
  - Visual feedback for auto-completed tasks in batch command output with 🎯 emoji indicators
  - Integration with existing auto-completion infrastructure from complete command
  - Comprehensive test suite covering complex hierarchy scenarios and error handling
- **Auto-completion of parent tasks**: Automatically marks parent tasks as complete when all children are done
  - Recursive parent checking up the task hierarchy
  - Multi-level auto-completion support (e.g., completing grandparents when all descendants are done)
  - Cycle detection for safety with maximum depth protection
  - Integration with complete command to display auto-completed parent tasks
  - Comprehensive test coverage for all auto-completion scenarios
  - New internal/task/autocomplete.go with AutoCompleteParents functionality
- **Next command implementation**: Complete next task workflow functionality
  - Finds first incomplete task using depth-first traversal algorithm  
  - Supports git-based file discovery when no filename is provided
  - Multiple output formats: table (default), JSON, markdown with reference documents
  - Comprehensive error handling, validation, and edge case management
  - Full test coverage including unit tests and integration tests (363 lines)
  - cmd/next.go and cmd/next_test.go implementing next task workflow requirements

### Changed
- Enhanced `BatchResponse` struct to include `AutoCompleted` field for tracking auto-completed parent tasks
- Updated batch operation execution to call auto-completion after status updates
- Improved deep copy functionality in batch operations to preserve front matter metadata
- **Code quality improvements**: Replace magic strings with constants in list command format switch

### Fixed
- Static analysis issues: extracted string constant for `update_status` operations and replaced manual loop with `copy()` function
- **Linting issues**: Use formatJSON and formatMarkdown constants instead of string literals in cmd/list.go

### Added
- Next task finding algorithm implementation for workflow support
  - Created internal/task/next.go with FindNextIncompleteTask function
  - TaskWithContext structure for tasks with their incomplete subtasks
  - Depth-first traversal logic for finding first incomplete task
  - Helper functions for filtering incomplete children and checking work status
  - Comprehensive test suite with edge cases including depth protection
  - Support for pending and in-progress states as incomplete work
- Front matter parsing integration into task file processing
  - Updated ParseFile to set FilePath and handle front matter extraction
  - Modified parseContent to extract and process YAML front matter before task parsing
  - Added FrontMatter field to TaskList structure for storing references and metadata
  - Preserved backward compatibility with files without front matter
  - Comprehensive test suite for front matter parsing with various edge cases
  - Tests for unclosed blocks, invalid YAML, and different front matter configurations

### Added
- Git branch discovery functionality for automated task file location
  - New internal/config/discovery.go with DiscoverFileFromBranch function
  - Git branch detection using rev-parse with timeout and error handling
  - Template substitution for {branch} placeholder in file paths
  - Branch name sanitization to prevent command injection vulnerabilities
  - Special git state detection (detached HEAD, rebase/merge) with fallback handling
  - Comprehensive unit tests with mock git command testing and integration tests
- Front matter parsing infrastructure for task file metadata
  - New internal/task/references.go with YAML front matter parsing capabilities
  - FrontMatter struct supporting references and arbitrary metadata fields
  - ParseFrontMatter function for extracting YAML from markdown files
  - SerializeWithFrontMatter function for combining front matter with content
  - Robust error handling for unclosed blocks and invalid YAML syntax
  - Comprehensive test suite covering edge cases and validation scenarios
- Code modernization improvements
  - Updated interface{} usage to any type throughout codebase
  - Applied Go modernization patterns for better language compatibility
- Configuration management infrastructure
  - Complete internal/config package with YAML configuration loading
  - Support for .go-tasks.yml in current directory and ~/.config/go-tasks/config.yml
  - Configuration precedence handling with default fallback
  - GitDiscovery configuration structure for branch-based file discovery
  - Comprehensive test suite with precedence and error handling validation
- Next-task-workflow implementation tasks document
  - Detailed 296-line implementation plan in specs/next-task-workflow/tasks.md
  - Comprehensive task breakdown covering configuration, git discovery, and CLI integration
  - Step-by-step implementation guide with requirements references

### Enhanced
- Batch operations functionality
  - Add operations now support details and references in batch mode
  - Enhanced applyOperation to handle details and references for newly added tasks
  - Automatic task ID resolution for updating newly added tasks with additional metadata

### Changed
- Modernized Go code patterns throughout codebase
  - Updated loop patterns to use `for i := range n` syntax
  - Replaced `strings.Split()` with `strings.SplitSeq()` in range loops
  - Applied modern Go conventions following language modernization guidelines

### Added
- Next task workflow specification and design documentation
  - Comprehensive requirements document defining sequential task management capabilities
  - Detailed design document with technical architecture and implementation approach
  - Decision log documenting key design choices for workflow features
- Enhanced Claude Code configuration
  - Updated CLAUDE.md with project-specific guidance for development commands
  - Added integration test commands and code quality tooling instructions

### Added
- Comprehensive project documentation and examples
  - Complete README.md with installation, usage guide, and command reference
  - Agent instruction documentation for AI integration patterns
  - Task creation guide with format specifications and validation rules
  - JSON API documentation with batch operations and schema definitions
  - Example files demonstrating simple, project, and complex task structures
  - Integration test suite with comprehensive workflow testing
- Claude Code configuration updates
  - Added integration test command allowlists for test automation
  - Enhanced development tooling configuration

### Added
- Comprehensive file operations and security test suite (612 lines)
  - File size limit validation tests (10MB maximum with multiple test cases)
  - Path traversal protection tests with malicious path detection
  - Atomic write operation tests ensuring data integrity
  - Input sanitization tests for null bytes and control characters
  - Concurrent access safety tests for multi-goroutine scenarios
  - Task ID validation tests preventing invalid ID formats (e.g., leading zeros)
- Enhanced security measures in task operations
  - Resource limits enforcement (MaxTaskCount: 10000, MaxHierarchyDepth: 10, MaxDetailLength: 1000)
  - Input validation for all task creation and update operations
  - File path validation with null byte and control character detection
  - Validation functions for task input, details, and references with length restrictions
- Task management CLI commands implementation
  - Add command for creating new tasks with optional parent hierarchy support
  - Complete command for marking tasks as completed with [x] status
  - Progress command for marking tasks as in-progress with [-] status  
  - Remove command for deleting tasks with automatic ID renumbering
  - Uncomplete command for marking completed tasks back to pending status
  - Update command for modifying task titles and details
- Comprehensive CLI command unit tests
  - Full test coverage for all task management commands
  - Dry-run functionality testing for safe operations
  - Parent-child relationship validation tests
  - Error handling and edge case tests
- Complete CLI implementation with Cobra framework integration
  - Root command with global flags (verbose, format, dry-run) and version information
  - Create command for generating new task markdown files with title specification
  - List command with multiple output formats (table, markdown, JSON) and filtering options
  - Input validation and security checks for file path operations
- Main executable entry point for the go-tasks CLI application
- Comprehensive CLI command unit tests
  - Full test coverage for create and list commands
  - File creation and parsing validation tests
  - Path traversal security validation tests
  - Output format validation tests (JSON, markdown, table)
- Go module dependencies integration
  - Cobra CLI framework for command structure
  - go-output/v2 library for formatted table output
  - Complete dependency management with go.sum
- Task operations module with core task management functionality
  - AddTask method for adding tasks with parent-child hierarchy support
  - RemoveTask method with automatic ID renumbering for consistency
  - UpdateStatus method for changing task status (Pending/InProgress/Completed)
  - UpdateTask method for modifying task title, details, and references
  - FindTask method for searching tasks by ID in the hierarchy
- Comprehensive test suite for task operations
  - Full test coverage for all CRUD operations
  - Parent-child relationship integrity validation
  - ID renumbering tests for task removal scenarios
  - Edge case handling and error validation
- Task parser module for reading and parsing markdown task files
  - Comprehensive parser with support for hierarchical task structures
  - Validation for indentation, status markers, and task formatting
  - Support for task details and references parsing
- Parser unit tests with comprehensive coverage
  - Test fixtures for simple, complex, and malformed task files
  - Edge case testing for various formatting issues
  - Performance testing with large task lists
- Makefile for development tooling
  - Test targets (unit, integration, coverage)
  - Code quality targets (lint, fmt, modernize)
  - Development utilities (mod-tidy, benchmark, clean)
- GolangCI-lint configuration for code quality enforcement
  - Standard linters enabled with custom rules
  - Formatter configuration for automatic interface{} to any conversion
- Initial Go module setup for go-tasks project
- Project specifications and documentation structure
  - Comprehensive project idea and implementation plan
  - Detailed requirements document with user stories and acceptance criteria
  - Decision log template for tracking design decisions
  - Out-of-scope documentation to define project boundaries
- Claude Code settings configuration
- Complete initial version design documentation
  - Comprehensive technical design document with architecture overview
  - Component specifications and data models
  - Implementation priorities and testing strategy
  - Security considerations and performance targets
- Decision log entries #14 for design simplification
  - Simplified package structure to 2 packages (cmd/ and internal/task/)
  - Removed unnecessary interfaces and premature optimizations
  - Direct implementation approach for better maintainability
- External research documentation for go-output/v2 library integration
  - Complete API documentation for table formatting capabilities
  - Usage patterns for AI agent implementation
  - Thread-safe document generation with preserved key ordering
- Implementation tasks document for initial version
  - Comprehensive task breakdown with 12 major sections
  - Unit test-first approach for all components
  - Detailed subtasks with requirement references
  - Clear dependencies and implementation order

### Changed
- Improved task ID validation regex to prevent invalid ID patterns
  - Updated regex pattern to disallow task IDs starting with zero (e.g., "01", "0.1" now invalid)
  - Ensures consistent ID numbering with natural numbers only
- Refactored task module structure for better separation of concerns
  - Moved task operations from task.go to dedicated operations.go file
  - Reorganized tests into operations_test.go to match new module structure
- Updated task document to mark completed items for project setup and core data structures

### Added
- Git ignore file for Go projects with standard exclusions
- Core task data models implementation in internal/task package
  - Task struct with hierarchical ID support, validation, and status tracking
  - TaskList struct with task management operations (add, remove, update, find)
  - Status enum implementation with string parsing and formatting
- Comprehensive unit tests for task package
  - 100% test coverage for all core functionality
  - Table-driven tests for all methods and edge cases
  - Tests for hierarchical task operations and ID renumbering
- Markdown renderer implementation for task lists
  - RenderMarkdown function for converting TaskList to markdown format
  - Consistent 2-space indentation per hierarchy level
  - Support for task status, details, and references rendering
  - RenderJSON function for JSON output format
- Comprehensive renderer unit tests
  - Tests for empty, simple, and hierarchical task lists
  - Tests for task details and references formatting
  - Indentation validation tests ensuring 2-space consistency
  - Round-trip tests verifying parse → render → parse integrity
- Task search and query functionality implementation
  - Find method for pattern-based searching in task titles, details, and references
  - Filter method for task filtering by status, depth, parent ID, and title patterns
  - QueryOptions for configurable search behavior (case sensitivity, search scope)
  - QueryFilter for flexible filtering criteria with multiple conditions
- Comprehensive search functionality unit tests
  - Case-sensitive and case-insensitive search pattern tests
  - Tests for searching in task details and references
  - Filter tests for status, depth, and parent ID filtering
  - Complex multi-criteria filtering tests with nested task hierarchies
- Find command implementation with pattern-based task searching
  - Support for pattern matching in task titles, details, and references
  - Multiple output formats (table, JSON, markdown) for search results
  - Filtering options for status, hierarchy depth, and parent task ID
  - Case-sensitive and case-insensitive search modes with include-parent option
- Comprehensive find command unit tests
  - Full test coverage for all search patterns and filtering options
  - Output format validation tests for table, JSON, and markdown formats
  - Advanced feature testing including hierarchy depth and parent ID filtering
  - Edge case testing with empty files and special characters in patterns
- Batch command for executing multiple task operations atomically
  - Batch processing functionality with dry-run support and JSON/table output formats
  - Operation validation and atomic execution guarantees
  - Support for add, remove, update_status, and update operations in batch mode
  - CLI support for batch operations via stdin, file, or string input
  - Comprehensive test coverage for batch operations and CLI integration