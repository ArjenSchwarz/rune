---
references:
    - requirements.md
    - design.md
    - decision_log.md
---
# Task Dependencies and Streams Implementation

## Core Data Structures

- [x] 1. Extend Task struct with new fields
  - Add StableID, BlockedBy, Stream, Owner fields to Task struct in internal/task/task.go
  - Add json:"-" tag for StableID to hide from JSON output
  - Add GetEffectiveStream() helper function
  - Requirements: [1.3](requirements.md#1.3), [1.5](requirements.md#1.5), [2.1](requirements.md#2.1), [3.1](requirements.md#3.1), [3.3](requirements.md#3.3), [4.1](requirements.md#4.1)

- [x] 2. Implement error types and warning mechanism
  - Create error types: ErrNoStableID, ErrStableIDNotFound, ErrDuplicateStableID, ErrCircularDependency, ErrInvalidBlockedBy, ErrInvalidStream, ErrInvalidOwner
  - Create CircularDependencyError struct with Path field
  - Create Warning struct with Code, Message, TaskID fields
  - Define warning code constants
  - Requirements: [2.11](requirements.md#2.11), [2.14](requirements.md#2.14), [2.15](requirements.md#2.15), [2.16](requirements.md#2.16), [3.10](requirements.md#3.10), [4.9](requirements.md#4.9)

## Stable ID Generation

- [x] 3. Write unit tests for StableIDGenerator
  - Test uniqueness across 10,000 generations
  - Test base36 encoding correctness
  - Test 7-character length constraint
  - Test collision detection with existing IDs
  - Test crypto/rand seeding for new files
  - Test counter continuation from existing IDs
  - Requirements: [1.1](requirements.md#1.1), [1.4](requirements.md#1.4), [1.7](requirements.md#1.7), [1.8](requirements.md#1.8)

- [x] 4. Implement StableIDGenerator in internal/task/stable_id.go
  - Create StableIDGenerator struct with usedIDs map and counter
  - Implement NewStableIDGenerator with existing ID parsing
  - Implement Generate() with zero-padding and collision detection
  - Implement IsUsed() check
  - Use crypto/rand for initial seeding
  - Requirements: [1.1](requirements.md#1.1), [1.4](requirements.md#1.4), [1.7](requirements.md#1.7), [1.8](requirements.md#1.8)

- [x] 5. Write property-based tests for stable ID uniqueness
  - Use rapid framework for property testing
  - Test that all generated IDs in a list are unique
  - Test that IDs survive multiple generate cycles
  - Requirements: [1.7](requirements.md#1.7)

## Dependency Index

- [x] 6. Write unit tests for DependencyIndex
  - Test index building from task hierarchy
  - Test GetTask by stable ID
  - Test GetTaskByHierarchicalID
  - Test GetDependents returns correct tasks
  - Test IsReady with various blocker states
  - Test IsBlocked with partial completion
  - Test TranslateToHierarchical accuracy
  - Requirements: [2.4](requirements.md#2.4), [2.5](requirements.md#2.5), [2.6](requirements.md#2.6)

- [x] 7. Write unit tests for cycle detection
  - Test self-reference detection (A → A)
  - Test direct cycle detection (A → B → A)
  - Test indirect cycle detection (A → B → C → A)
  - Test valid chain passes (no false positives)
  - Test depth limit prevents stack overflow
  - Requirements: [2.14](requirements.md#2.14), [2.15](requirements.md#2.15), [2.16](requirements.md#2.16)

- [x] 8. Implement DependencyIndex in internal/task/dependencies.go
  - Create DependencyIndex struct with byStableID, byHierarchical, dependents maps
  - Implement BuildDependencyIndex from task list
  - Implement GetTask, GetTaskByHierarchicalID, GetDependents
  - Implement IsReady and IsBlocked
  - Implement TranslateToHierarchical
  - Requirements: [2.4](requirements.md#2.4), [2.5](requirements.md#2.5), [2.6](requirements.md#2.6)

- [x] 9. Implement cycle detection in DependencyIndex
  - Implement DetectCycle with DFS algorithm
  - Add maxDependencyDepth constant (1000)
  - Return cycle path for error messages
  - Handle self-reference as special case
  - Requirements: [2.14](requirements.md#2.14), [2.15](requirements.md#2.15), [2.16](requirements.md#2.16)

- [x] 10. Write property-based tests for cycle detection
  - Test that no cycles can be created through valid operations
  - Test that detected cycles are always real cycles
  - Requirements: [2.14](requirements.md#2.14), [2.15](requirements.md#2.15), [2.16](requirements.md#2.16)

## Stream Analysis

- [x] 11. Write unit tests for stream analysis
  - Test stream derivation from tasks
  - Test ready/blocked/active classification
  - Test default stream (1) assignment for unset tasks
  - Test FilterByStream correctness
  - Test available stream calculation
  - Test AnalyzeStreams with mixed streams
  - Requirements: [3.3](requirements.md#3.3), [3.4](requirements.md#3.4), [3.5](requirements.md#3.5)

- [x] 12. Implement stream functions in internal/task/streams.go
  - Create StreamStatus struct with ID, Ready, Blocked, Active fields
  - Create StreamsResult struct with Streams and Available fields
  - Implement AnalyzeStreams function
  - Implement FilterByStream function
  - Implement GetEffectiveStream helper
  - Requirements: [3.3](requirements.md#3.3), [3.4](requirements.md#3.4), [3.5](requirements.md#3.5), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3)

## Parser Extensions

- [x] 13. Write unit tests for parsing new metadata
  - Test stable ID extraction from HTML comments
  - Test Blocked-by parsing with title hints
  - Test case-insensitive metadata parsing
  - Test Stream value parsing
  - Test Owner parsing
  - Test legacy files without new fields
  - Requirements: [7.1](requirements.md#7.1), [7.2](requirements.md#7.2), [7.3](requirements.md#7.3), [7.7](requirements.md#7.7)

- [x] 14. Write negative parsing tests for malformed input
  - Test invalid stable ID format handling
  - Test Blocked-by with malformed references
  - Test Stream with non-integer value
  - Test Stream with zero or negative value
  - Test duplicate stable IDs warning
  - Requirements: [2.11](requirements.md#2.11), [3.10](requirements.md#3.10), [7.1](requirements.md#7.1)

- [x] 15. Extend parser in internal/task/parse.go
  - Add stableIDPattern regex for HTML comments
  - Add blockedByPattern, streamPattern, ownerPattern regexes
  - Add blockedByRefPattern for parsing references with title hints
  - Modify parseTaskLine to extract stable ID from title
  - Modify parseDetailsAndChildren to extract new metadata
  - Preserve parsing of legacy files
  - Requirements: [7.1](requirements.md#7.1), [7.2](requirements.md#7.2), [7.3](requirements.md#7.3), [7.6](requirements.md#7.6), [7.7](requirements.md#7.7)

## Renderer Extensions

- [x] 16. Write unit tests for rendering new metadata
  - Test stable ID inclusion in markdown output
  - Test Blocked-by formatting with title hints
  - Test Stream rendering (only when non-zero)
  - Test Owner rendering
  - Test JSON output excludes stable IDs
  - Test JSON BlockedBy uses hierarchical IDs
  - Requirements: [1.5](requirements.md#1.5), [7.1](requirements.md#7.1), [7.4](requirements.md#7.4), [7.5](requirements.md#7.5)

- [x] 17. Write property-based test for parse-render round-trip
  - Test that parse(render(tasks)) equals original tasks
  - Include all new metadata fields in test
  - Requirements: [7.4](requirements.md#7.4)

- [x] 18. Extend renderer in internal/task/render.go
  - Create RenderContext struct with RequirementsFile and DependencyIndex
  - Modify renderTask to include stable ID in HTML comment
  - Add formatBlockedByRefs helper for title hints
  - Render Blocked-by, Stream, Owner in correct order
  - Implement taskJSON struct for JSON marshaling
  - Implement MarshalTasksJSON with hierarchical ID translation
  - Requirements: [1.5](requirements.md#1.5), [7.1](requirements.md#7.1), [7.4](requirements.md#7.4), [7.5](requirements.md#7.5)

## Operations Extensions

- [x] 19. Write unit tests for extended add operation
  - Test AddTask generates stable ID
  - Test AddTask with stream option
  - Test AddTask with blocked-by option
  - Test AddTask with owner option
  - Test blocked-by validation (target must have stable ID)
  - Requirements: [1.1](requirements.md#1.1), [2.9](requirements.md#2.9), [2.12](requirements.md#2.12), [3.8](requirements.md#3.8), [4.4](requirements.md#4.4)

- [x] 20. Write unit tests for extended update operation
  - Test UpdateTask with stream modification
  - Test UpdateTask with blocked-by modification
  - Test UpdateTask with owner modification
  - Test UpdateTask with release flag
  - Test cycle detection on blocked-by update
  - Test invalid stream value rejection
  - Requirements: [2.8](requirements.md#2.8), [2.14](requirements.md#2.14), [3.7](requirements.md#3.7), [3.10](requirements.md#3.10), [4.4](requirements.md#4.4), [4.5](requirements.md#4.5)

- [x] 21. Write unit tests for extended remove operation
  - Test RemoveTask with dependents generates warning
  - Test dependent references are cleaned up
  - Test stable ID is not reused
  - Requirements: [1.6](requirements.md#1.6), [1.8](requirements.md#1.8), [2.7](requirements.md#2.7)

- [x] 22. Extend operations in internal/task/operations.go
  - Add AddOptions struct with Stream, BlockedBy, Owner fields
  - Modify AddTask to generate stable ID and apply options
  - Add UpdateOptions struct with Stream, BlockedBy, Owner, Release fields
  - Modify UpdateTask to validate and apply new options
  - Add resolveToStableIDs helper function
  - Modify RemoveTask to warn about and clean up dependents
  - Add removeFromBlockedByLists helper
  - Requirements: [1.1](requirements.md#1.1), [1.6](requirements.md#1.6), [2.7](requirements.md#2.7), [2.8](requirements.md#2.8), [2.9](requirements.md#2.9), [3.7](requirements.md#3.7), [3.8](requirements.md#3.8), [4.4](requirements.md#4.4), [4.5](requirements.md#4.5)

## Batch Operations Extensions

- [x] 23. Write unit tests for extended batch operations
  - Test batch add with stream, blocked_by, owner
  - Test batch update with new fields
  - Test batch with cycle detection
  - Test batch atomicity with dependency errors
  - Requirements: [2.10](requirements.md#2.10), [3.9](requirements.md#3.9), [4.7](requirements.md#4.7)

- [x] 24. Extend batch operations in internal/task/batch.go
  - Add Stream, BlockedBy, Owner, Release fields to Operation struct
  - Update executeAdd to pass new options
  - Update executeUpdate to handle new fields
  - Requirements: [2.10](requirements.md#2.10), [3.9](requirements.md#3.9), [4.7](requirements.md#4.7)

## Streams Command

- [x] 25. Write unit tests for streams command
  - Test streams output with multiple streams
  - Test --available flag filtering
  - Test --json output format
  - Test empty streams result
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4), [5.5](requirements.md#5.5), [5.6](requirements.md#5.6)

- [x] 26. Implement streams command in cmd/streams.go
  - Create streamsCmd with Use, Short, Long descriptions
  - Add --available and --json flags
  - Implement runStreams with stream analysis
  - Output table format by default
  - Output JSON with task ID arrays per stream
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4), [5.5](requirements.md#5.5), [5.6](requirements.md#5.6)

## Next Command Enhancements

- [x] 27. Write unit tests for next command enhancements
  - Test --stream filter returns only stream tasks
  - Test --claim without stream claims single task
  - Test --stream --claim claims all ready tasks in stream
  - Test claim sets status to in-progress and owner
  - Test already-claimed tasks are skipped
  - Test --phase with stream/dependency info
  - Test --phase --json includes streams summary
  - Requirements: [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [6.3](requirements.md#6.3), [6.4](requirements.md#6.4), [6.5](requirements.md#6.5), [6.6](requirements.md#6.6), [6.7](requirements.md#6.7), [6.8](requirements.md#6.8), [6.9](requirements.md#6.9), [6.10](requirements.md#6.10), [6.11](requirements.md#6.11), [6.12](requirements.md#6.12)

- [x] 28. Extend next command in cmd/next.go
  - Add --stream and --claim flags
  - Implement stream filtering with FilterByStream
  - Implement claimStreamTasks for --stream --claim
  - Implement claimSingleTask for --claim alone
  - Update phase output to include stream/dependency info
  - Update JSON output with streams summary
  - Handle no-ready-tasks case gracefully
  - Requirements: [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [6.3](requirements.md#6.3), [6.4](requirements.md#6.4), [6.5](requirements.md#6.5), [6.6](requirements.md#6.6), [6.7](requirements.md#6.7), [6.8](requirements.md#6.8), [6.9](requirements.md#6.9), [6.10](requirements.md#6.10), [6.11](requirements.md#6.11), [6.12](requirements.md#6.12)

## List Command Enhancements

- [x] 29. Write unit tests for list command enhancements
  - Test stream display when non-default streams exist
  - Test blocked-by display as hierarchical IDs
  - Test --stream filter
  - Test --owner filter
  - Test JSON output includes new fields
  - Requirements: [8.1](requirements.md#8.1), [8.2](requirements.md#8.2), [8.3](requirements.md#8.3), [8.4](requirements.md#8.4), [8.5](requirements.md#8.5)

- [x] 30. Extend list command in cmd/list.go
  - Add --stream and --owner filter flags
  - Conditionally display stream column
  - Display blocked-by as hierarchical IDs
  - Update JSON output with blockedBy, stream, owner
  - Requirements: [8.1](requirements.md#8.1), [8.2](requirements.md#8.2), [8.3](requirements.md#8.3), [8.4](requirements.md#8.4), [8.5](requirements.md#8.5)

## Add/Update Command Enhancements

- [x] 31. Write unit tests for add command enhancements
  - Test --stream flag sets stream
  - Test --blocked-by flag adds dependencies
  - Test --owner flag sets owner
  - Test blocked-by with legacy task target fails
  - Requirements: [2.9](requirements.md#2.9), [2.12](requirements.md#2.12), [3.8](requirements.md#3.8)

- [x] 32. Extend add command in cmd/add.go
  - Add --stream, --blocked-by, --owner flags
  - Pass options to AddTask
  - Handle validation errors
  - Requirements: [2.9](requirements.md#2.9), [2.12](requirements.md#2.12), [3.8](requirements.md#3.8)

- [x] 33. Write unit tests for update command enhancements
  - Test --stream flag updates stream
  - Test --blocked-by flag updates dependencies
  - Test --owner flag updates owner
  - Test --release flag clears owner
  - Test cycle detection error
  - Requirements: [2.8](requirements.md#2.8), [2.14](requirements.md#2.14), [3.7](requirements.md#3.7), [4.4](requirements.md#4.4), [4.5](requirements.md#4.5)

- [x] 34. Extend update command in cmd/update.go
  - Add --stream, --blocked-by, --owner, --release flags
  - Pass options to UpdateTask
  - Handle validation and cycle detection errors
  - Requirements: [2.8](requirements.md#2.8), [2.14](requirements.md#2.14), [3.7](requirements.md#3.7), [4.4](requirements.md#4.4), [4.5](requirements.md#4.5)

## Integration Tests

- [ ] 35. Write multi-agent workflow integration test
  - Create task file with streams and dependencies
  - Simulate agent claiming stream 1
  - Simulate agent claiming stream 2
  - Complete tasks and verify blocking resolution
  - Verify no duplicate claims
  - Requirements: [6.2](requirements.md#6.2), [6.4](requirements.md#6.4), [6.5](requirements.md#6.5), [6.13](requirements.md#6.13)

- [ ] 36. Write dependency chain resolution integration test
  - Create A → B → C → D dependency chain
  - Verify only A is ready initially
  - Complete tasks sequentially and verify readiness
  - Requirements: [2.4](requirements.md#2.4), [2.5](requirements.md#2.5)

- [ ] 37. Write backward compatibility integration test
  - Parse existing file without new fields
  - Verify all operations work
  - Add new task with stable ID
  - Verify mixed legacy/new tasks work
  - Requirements: [9.1](requirements.md#9.1), [9.2](requirements.md#9.2), [9.3](requirements.md#9.3), [9.4](requirements.md#9.4), [9.5](requirements.md#9.5), [9.6](requirements.md#9.6)

## Documentation

- [ ] 38. Update README.md with new features
  - Add Task Dependencies section
  - Add Work Streams section
  - Add streams command documentation
  - Update next command documentation

- [ ] 39. Update docs/AGENT_INSTRUCTIONS.md
  - Add Multi-Agent Parallel Execution section
  - Add Task Dependencies section
  - Update batch operation examples

- [ ] 40. Update docs/json-api.md
  - Add new Operation fields schema
  - Add new Task fields schema
  - Add StreamsResult and StreamStatus schemas
  - Add ClaimResult schema
  - Add Warning schema

- [ ] 41. Update command help text
  - Update cmd/next.go Long description
  - Update cmd/list.go Long description
  - Update cmd/add.go Long description
  - Update cmd/update.go Long description
  - Write cmd/streams.go Long description

- [ ] 42. Add example file for parallel agents
  - Create examples/parallel-agents.md with streams and dependencies example
