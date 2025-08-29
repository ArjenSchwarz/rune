# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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