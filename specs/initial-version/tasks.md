# Go-Tasks Initial Version Implementation Tasks

## 1. Project Setup and Core Data Structures

- [x] 1.1. Initialize Go module and create basic project structure
  - Create go.mod with module name `github.com/arjenschwarz/go-tasks`
  - Set up internal/task package directory
  - Set up cmd directory for CLI commands
  - Add .gitignore for Go projects
  - References: Design Package Structure section

- [x] 1.2. Create unit tests for core data models
  - Write tests for Task struct validation and operations
  - Write tests for TaskList struct and its methods
  - Write tests for Status enum conversions
  - Create test fixtures in testdata directory
  - References: Requirements 2.1-2.6

- [x] 1.3. Implement core data models (Task and TaskList structs)
  - Create task.go with Task struct (ID, Title, Status, Details, References, Children, ParentID)
  - Create TaskList struct (Title, Tasks, FilePath, Modified)
  - Implement Status enum (Pending, InProgress, Completed)
  - Add helper methods for task traversal and lookup
  - References: Requirements 2.1-2.6, Design Core Data Models section

## 2. Markdown Parser Implementation

- [x] 2.1. Create comprehensive parser unit tests
  - Test valid markdown parsing with various formats
  - Test malformed content error reporting (per Decision #2)
  - Test hierarchy building from indentation
  - Test task status detection ([ ], [-], [x])
  - Test details and references extraction
  - References: Requirements 3.1-3.8

- [x] 2.2. Implement markdown parser
  - Create parse.go with ParseMarkdown and ParseFile functions
  - Implement line-by-line parsing with indentation tracking
  - Extract task titles, status, details, and references
  - Build hierarchical task structure with parent-child relationships
  - Report errors for malformed content without auto-correction
  - References: Requirements 3.1-3.3, Decision #2

## 3. Markdown Renderer Implementation

- [x] 3.1. Create renderer unit tests
  - Test consistent formatting output
  - Test 2-space indentation per hierarchy level
  - Test round-trip operations (parse → render → parse)
  - Test empty task list handling
  - References: Requirements 3.4-3.8

- [x] 3.2. Implement markdown renderer
  - Create render.go with RenderMarkdown function
  - Implement consistent 2-space indentation
  - Format tasks with hierarchical numbering (1, 1.1, 1.2.1)
  - Format details as bullet points with proper spacing
  - Format references with "References: " prefix
  - References: Requirements 3.4-3.7, 7.1-7.7

## 4. Task Operations and Mutations

- [x] 4.1. Create unit tests for task operations
  - Test AddTask at all hierarchy levels
  - Test RemoveTask with automatic renumbering
  - Test UpdateStatus for all three states
  - Test UpdateTask for title, details, and references
  - Test parent-child relationship integrity
  - References: Requirements 1.1-1.7

- [x] 4.2. Implement task mutation methods
  - Create operations.go with TaskList methods
  - Implement AddTask with automatic ID assignment
  - Implement RemoveTask with renumbering logic
  - Implement UpdateStatus for task state changes
  - Implement UpdateTask for content modifications
  - References: Requirements 1.3-1.6, 2.6

## 5. Search and Query Functionality

- [x] 5.1. Create search and filter unit tests
  - Test find command with title content matching
  - Test case-sensitive and case-insensitive search
  - Test filtering by status (pending, in-progress, completed)
  - Test filtering by hierarchy level
  - Test searching within details and references
  - References: Requirements 6.1-6.7

- [x] 5.2. Implement search and query methods
  - Create search.go with Find and Filter functions
  - Implement pattern matching for task titles
  - Add status filtering capabilities
  - Add hierarchy level filtering
  - Include parent context in search results
  - References: Requirements 6.1-6.7

## 6. CLI Foundation and Basic Commands

- [x] 6.1. Set up Cobra CLI structure
  - Create cmd/root.go with root command setup
  - Add global flags (verbose, format, dry-run)
  - Configure command hierarchy
  - Add version information
  - References: Requirements 4.1, Decision #4

- [x] 6.2. Implement create command tests and functionality
  - Test file creation with specified title
  - Test validation of file paths
  - Implement cmd/create.go using Cobra
  - Generate new task files with clean structure
  - References: Requirements 1.1, 4.2

- [x] 6.3. Implement list command tests and functionality
  - Test table, markdown, and JSON output formats
  - Test filtering options
  - Implement cmd/list.go with go-output/v2 integration
  - Support multiple output formats
  - References: Requirements 4.3, Design Output Integration section

## 7. Task Manipulation Commands

- [x] 7.1. Implement add command tests and functionality
  - Test adding tasks with parent specification
  - Test validation of parent IDs
  - Implement cmd/add.go
  - Support adding tasks at any hierarchy level
  - References: Requirements 1.4, 4.4

- [x] 7.2. Implement complete and uncomplete command tests and functionality
  - Test status changes for all states
  - Test error handling for invalid task IDs
  - Implement cmd/complete.go and cmd/uncomplete.go
  - Update task status with validation
  - References: Requirements 1.3, 4.5

- [x] 7.3. Implement update command tests and functionality
  - Test updating title, details, and references
  - Test partial updates
  - Implement cmd/update.go
  - Modify task content independently
  - References: Requirements 1.6, 4.6

- [x] 7.4. Implement remove command tests and functionality
  - Test task removal with renumbering
  - Test removing tasks with children
  - Implement cmd/remove.go
  - Handle automatic ID renumbering
  - References: Requirements 1.5, 4.7

## 8. Search Command Implementation

- [ ] 8.1. Implement find command tests and functionality
  - Test search with various patterns
  - Test JSON output format
  - Test filtering options integration
  - Implement cmd/find.go
  - Return hierarchical context for results
  - References: Requirements 6.1-6.7, 4.9

## 9. JSON API and Batch Operations

- [ ] 9.1. Create batch operations unit tests
  - Test atomic batch execution (all succeed or all fail)
  - Test validation before applying changes
  - Test dry-run mode
  - Test JSON schema validation
  - Test error reporting for invalid operations
  - References: Requirements 5.1-5.6, Decision #12

- [ ] 9.2. Implement batch command and JSON API
  - Implement cmd/batch.go for batch operations
  - Create JSON schema structures
  - Implement validation for all operation types
  - Implement atomic transaction pattern
  - Add comprehensive error reporting
  - References: Requirements 4.8, 5.1-5.6

## 10. File Operations and Security

- [ ] 10.1. Create file operations and security tests
  - Test file size limits (10MB maximum)
  - Test path traversal protection
  - Test atomic write operations
  - Test input sanitization
  - Test concurrent access safety
  - References: Design Security Considerations section, Decision #13

- [ ] 10.2. Implement secure file operations
  - Add file path validation
  - Implement atomic writes with temp files
  - Add input sanitization for user content
  - Enforce resource limits
  - Add concurrency protection for single process
  - References: Design Security Considerations section, Decision #3

## 11. Integration Testing and Documentation

- [ ] 11.1. Create comprehensive integration tests
  - Test complete workflows (create → add → update → remove)
  - Test large file handling (100+ tasks)
  - Test all CLI commands end-to-end
  - Test JSON API with complex batch operations
  - Test error handling and recovery
  - References: Design Testing Strategy section

- [ ] 11.2. Add documentation and examples
  - Create README.md with usage instructions
  - Add example task files in examples directory
  - Document JSON API schema
  - Add code comments for exported functions
  - Create CLI help text for all commands
  - References: Requirements 4.1-4.9

## 12. Final Integration and Polish

- [ ] 12.1. Wire all components together
  - Ensure all CLI commands use core functionality
  - Verify all error paths are handled
  - Test complete application flow
  - Add version information to CLI
  - References: All requirements

- [ ] 12.2. Performance validation and optimization
  - Benchmark parse/render operations
  - Verify sub-second response for 100+ tasks
  - Profile memory usage
  - Optimize only if targets aren't met
  - References: Design Performance Targets section