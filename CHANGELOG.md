# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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