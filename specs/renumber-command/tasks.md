---
references:
    - specs/renumber-command/requirements.md
    - specs/renumber-command/design.md
    - specs/renumber-command/decision_log.md
---
# Renumber Command Implementation

## Phase 1: Core Command Structure

- [x] 1. Export private functions from internal/task package
  - Export validateFilePath() by capitalizing to ValidateFilePath()
  - Export countTotalTasks() by capitalizing to CountTotalTasks()
  - Export renumberTasks() by capitalizing to RenumberTasks()
  - Update all internal references to use capitalized names
  - Requirements: [1.1](requirements.md#1.1)
  - References: internal/task/operations.go, internal/task/task.go

- [x] 2. Create renumber command file and cobra command definition
  - Create cmd/renumber.go file
  - Define renumberCmd cobra.Command with Use, Short, Long descriptions
  - Set Args to cobra.ExactArgs(1) for single file argument
  - Create runRenumber() function skeleton with error handling
  - Requirements: [1.1](requirements.md#1.1)
  - References: cmd/renumber.go

- [x] 3. Register renumber command with root command
  - Add command registration in cmd/root.go init() function
  - Verify --format flag is inherited from root
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2)
  - References: cmd/root.go, cmd/renumber.go

## Phase 2: Backup Functionality

- [x] 4. Write unit tests for createBackup() function
  - Test backup file creation with correct content
  - Test backup preserves file permissions
  - Test backup overwrites existing .bak files
  - Test error handling for read failures
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.4](requirements.md#3.4)
  - References: cmd/renumber_test.go

- [x] 5. Implement createBackup() function
  - Accept filePath and fileInfo parameters
  - Read original file content using os.ReadFile()
  - Create backup with .bak extension
  - Preserve original file permissions from fileInfo
  - Return backup path or error
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.4](requirements.md#3.4), [3.5](requirements.md#3.5)
  - References: cmd/renumber.go

- [x] 6. Add backup creation to runRenumber() Phase 4
  - Call createBackup() after successful parsing
  - Pass filePath and fileInfo from Phase 1 validation
  - Return error if backup creation fails
  - Store backupPath for use in summary output
  - Requirements: [3.1](requirements.md#3.1), [3.3](requirements.md#3.3), [3.5](requirements.md#3.5)
  - References: cmd/renumber.go

## Phase 3: Validation Integration

- [x] 7. Write unit tests for validation phase
  - Test ValidateFilePath rejects invalid paths
  - Test file size validation rejects files > 10MB
  - Test task count validation rejects > 10000 tasks
  - Test validation order (file size before parsing)
  - Requirements: [2.1](requirements.md#2.1), [2.4](requirements.md#2.4), [2.5](requirements.md#2.5), [2.6](requirements.md#2.6)
  - References: cmd/renumber_test.go

- [x] 8. Implement validation phase in runRenumber()
  - Phase 1: Call ValidateFilePath() on input path
  - Phase 1: Get fileInfo with os.Stat() and check file exists
  - Phase 1: Validate file size <= MaxFileSize before parsing
  - Phase 3: Call CountTotalTasks() and validate <= MaxTaskCount
  - Note that depth validation happens in ParseFileWithPhases per Decision 11
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.4](requirements.md#2.4), [2.5](requirements.md#2.5), [2.6](requirements.md#2.6), [2.7](requirements.md#2.7)
  - References: cmd/renumber.go

## Phase 4: Renumbering and Phase Marker Adjustment

- [x] 9. Write unit tests for adjustPhaseMarkersAfterRenumber()
  - Test with no phase markers (empty array)
  - Test with phase at beginning (AfterTaskID empty)
  - Test with phase after root task (AfterTaskID like "3")
  - Test with phase after nested task (AfterTaskID like "2.3")
  - Test that root task numbers are extracted correctly
  - Requirements: [1.7](requirements.md#1.7), [1.8](requirements.md#1.8)
  - References: cmd/renumber_test.go

- [x] 10. Implement getRootTaskNumber() helper function
  - Split taskID by "." to get parts
  - Parse first part as integer
  - Return root task number
  - Handle error cases gracefully
  - Requirements: [1.7](requirements.md#1.7)
  - References: cmd/renumber.go

- [x] 11. Implement adjustPhaseMarkersAfterRenumber() function
  - Create new slice for adjusted markers
  - Iterate through each marker
  - Skip markers with empty AfterTaskID (phases at beginning)
  - Extract root task number from AfterTaskID using getRootTaskNumber()
  - Reformat AfterTaskID to just root task number
  - Return adjusted markers
  - Requirements: [1.7](requirements.md#1.7), [1.8](requirements.md#1.8)
  - References: cmd/renumber.go

- [x] 12. Write unit tests for renumbering with various file types
  - Test simple file with no hierarchy
  - Test file with nested tasks (multiple levels)
  - Test file with phase markers
  - Test file with YAML front matter
  - Test that task order is preserved
  - Test that task metadata is preserved (status, details, references)
  - Requirements: [1.3](requirements.md#1.3), [1.4](requirements.md#1.4), [1.6](requirements.md#1.6), [1.7](requirements.md#1.7), [1.9](requirements.md#1.9), [4.4](requirements.md#4.4)
  - References: cmd/renumber_test.go

- [x] 13. Implement renumbering integration in runRenumber()
  - Phase 2: Parse file using ParseFileWithPhases()
  - Phase 5: Call RenumberTasks() on TaskList
  - Phase 5.5: Call adjustPhaseMarkersAfterRenumber() if phases exist
  - Phase 6: Call WriteFileWithPhases() if phases exist, otherwise WriteFile()
  - Use atomic write pattern from existing code
  - Requirements: [1.2](requirements.md#1.2), [1.3](requirements.md#1.3), [1.5](requirements.md#1.5), [1.7](requirements.md#1.7), [1.8](requirements.md#1.8), [1.10](requirements.md#1.10)
  - References: cmd/renumber.go

## Phase 5: Output Formatting

- [x] 14. Write unit tests for displaySummary() output formats
  - Test table format output structure
  - Test markdown format output structure
  - Test JSON format with correct fields (task_count, backup_file, success)
  - Test that backup file path is included in all formats
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4), [5.5](requirements.md#5.5), [5.6](requirements.md#5.6)
  - References: cmd/renumber_test.go

- [x] 15. Implement displaySummary() function
  - Accept taskList, backupPath, and format parameters
  - Implement table format using go-output library (default)
  - Implement markdown format with bullet points
  - Implement JSON format with task_count, backup_file, success fields
  - Display total task count from CountTotalTasks()
  - Display backup file location
  - Display success status
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4), [5.5](requirements.md#5.5), [5.6](requirements.md#5.6), [5.7](requirements.md#5.7)
  - References: cmd/renumber.go

- [x] 16. Add displaySummary() call to runRenumber() Phase 7
  - Call displaySummary() after successful file write
  - Pass taskList, backupPath, and format flag
  - Return any errors from displaySummary()
  - Requirements: [5.5](requirements.md#5.5)
  - References: cmd/renumber.go

## Phase 6: Error Handling and Edge Cases

- [x] 17. Write unit tests for error handling
  - Test file not found error
  - Test invalid path error
  - Test file too large error
  - Test parse error with line numbers
  - Test task count exceeds limit error
  - Test backup creation failure
  - Test write failure with cleanup
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [2.6](requirements.md#2.6), [2.8](requirements.md#2.8), [2.9](requirements.md#2.9)
  - References: cmd/renumber_test.go

- [x] 18. Add comprehensive error handling to runRenumber()
  - Wrap all errors with context using fmt.Errorf
  - Ensure error messages match design specification
  - Verify temp file cleanup on write failure (handled by WriteFile)
  - Test all error paths are covered
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [2.6](requirements.md#2.6), [2.8](requirements.md#2.8), [2.9](requirements.md#2.9)
  - References: cmd/renumber.go

- [x] 19. Write unit tests for edge cases
  - Test empty file (no tasks) returns task_count=0
  - Test file with only phase markers
  - Test malformed hierarchy error
  - Test duplicate task IDs error
  - Test disk space error handling
  - Requirements: [7.1](requirements.md#7.1), [7.2](requirements.md#7.2), [7.3](requirements.md#7.3), [7.4](requirements.md#7.4), [7.5](requirements.md#7.5)
  - References: cmd/renumber_test.go

- [x] 20. Create test fixtures for edge cases
  - Create examples/empty.md - empty file
  - Create examples/phases_only.md - only phase markers
  - Create examples/tasks_malformed.md - invalid hierarchy
  - Create examples/tasks_with_gaps.md - numbering gaps (1, 2, 5)
  - Requirements: [7.1](requirements.md#7.1), [7.2](requirements.md#7.2), [7.3](requirements.md#7.3)
  - References: examples/

## Phase 7: Integration Testing

- [ ] 21. Write integration test for end-to-end renumber workflow
  - Test complete workflow: validate -> parse -> backup -> renumber -> write
  - Verify backup file is created with correct content
  - Verify original file is updated with renumbered tasks
  - Verify atomic write behavior (temp file created and renamed)
  - Test with realistic file sizes and structures
  - Requirements: [1.5](requirements.md#1.5), [1.10](requirements.md#1.10), [3.1](requirements.md#3.1), [3.2](requirements.md#3.2)
  - References: cmd/integration_test.go

- [ ] 22. Write integration test for renumbering with phases
  - Create test file with phase markers
  - Run renumber command
  - Verify phase markers are preserved in correct positions
  - Verify AfterTaskID values are updated correctly
  - Verify tasks within phases are renumbered correctly
  - Requirements: [1.7](requirements.md#1.7), [1.8](requirements.md#1.8), [4.2](requirements.md#4.2)
  - References: cmd/integration_test.go

- [ ] 23. Write integration test for front matter preservation
  - Create test file with YAML front matter
  - Run renumber command
  - Verify front matter is preserved exactly
  - Verify tasks are renumbered correctly after front matter
  - Requirements: [1.9](requirements.md#1.9)
  - References: cmd/integration_test.go

- [ ] 24. Write integration test for write failure and cleanup
  - Simulate write failure scenario
  - Verify original file remains untouched
  - Verify temp file is cleaned up
  - Verify backup file exists
  - Requirements: [2.8](requirements.md#2.8), [2.9](requirements.md#2.9)
  - References: cmd/integration_test.go

- [ ] 25. Write integration test for symlink security
  - Create symlink pointing outside working directory
  - Attempt to renumber via symlink
  - Verify operation is rejected by ValidateFilePath
  - Verify error message indicates path traversal attempt
  - Requirements: [2.5](requirements.md#2.5)
  - References: cmd/integration_test.go

- [ ] 26. Write integration test for malformed phase markers
  - Create file with phase marker pointing to non-existent task
  - Run renumber command
  - Verify command handles gracefully
  - Document expected behavior
  - Requirements: [1.7](requirements.md#1.7)
  - References: cmd/integration_test.go

- [ ] 27. Write integration test for large file handling
  - Create test file near 10MB limit with ~9000 tasks
  - Test renumbering completes successfully
  - Create test file with 10-level hierarchy depth
  - Verify performance is acceptable
  - Requirements: [2.4](requirements.md#2.4), [2.6](requirements.md#2.6), [2.7](requirements.md#2.7)
  - References: cmd/integration_test.go

## Phase 8: Code Quality and Documentation

- [ ] 28. Run golangci-lint and fix any issues
  - Run make lint on cmd/renumber.go
  - Fix any linting issues
  - Ensure consistent error handling patterns
  - Verify all exported functions have comments
  - References: cmd/renumber.go

- [ ] 29. Run go fmt and ensure consistent formatting
  - Run make fmt on all modified files
  - Verify formatting matches project standards
  - References: cmd/renumber.go, cmd/renumber_test.go

- [ ] 30. Verify test coverage meets project standards
  - Run make test-coverage
  - Verify line coverage >= 80%
  - Verify all error paths are tested
  - Verify all edge cases are covered
  - References: cmd/renumber_test.go, cmd/integration_test.go

- [ ] 31. Update README.md with renumber command documentation
  - Add renumber command to command list
  - Add usage example showing basic renumbering
  - Add example showing renumbering with phases
  - Document backup file behavior (.bak creation)
  - Document interruption handling
  - Add note about requirement links not being updated
  - Requirements: [5.3](requirements.md#5.3), [5.4](requirements.md#5.4)
  - References: README.md

- [ ] 32. Add command documentation to cmd/renumber.go Long description
  - Document automatic backup creation
  - Document hierarchical numbering behavior
  - Document phase marker preservation
  - Document front matter preservation
  - Add usage examples in Long field
  - Requirements: [1.3](requirements.md#1.3), [1.7](requirements.md#1.7), [1.9](requirements.md#1.9), [3.1](requirements.md#3.1)
  - References: cmd/renumber.go

- [ ] 33. Run full test suite and verify all tests pass
  - Run make test for unit tests
  - Run make test-integration for integration tests
  - Verify all tests pass
  - Fix any failing tests

- [ ] 34. Manual testing with real task files
  - Test renumber with examples/simple.md
  - Test renumber with examples/project.md (has phases)
  - Verify output formats (table, json, markdown)
  - Verify backup files are created correctly
  - Test error scenarios (invalid path, file too large)
  - References: examples/simple.md, examples/project.md
