---
references:
    - specs/task-requirements-linking/requirements.md
    - specs/task-requirements-linking/design.md
    - specs/task-requirements-linking/decision_log.md
---
# Task Requirements Linking Implementation

## Core Data Structure Changes

- [x] 1. Update Task struct with Requirements field
  - Add Requirements []string field to Task struct in internal/task/task.go
  - Add RequirementsFile string field to TaskList struct
  - Add DefaultRequirementsFile constant ("requirements.md")
  - Add json tags with omitempty for both fields
  - Ensure validation supports hierarchical requirement IDs matching pattern
  - References: Requirements 1.1, 1.2, 1.3, 1.4, 1.5

- [x] 2. Write unit tests for Task struct Requirements field
  - Test valid single requirement ID
  - Test valid multiple requirement IDs
  - Test invalid requirement ID format
  - Test empty requirements array
  - Test hierarchical requirement IDs (1.1, 1.2.3, etc.)
  - References: Requirements 1.1, 1.2, 1.3, 1.5

## Parsing Implementation

- [x] 3. Implement parseRequirements helper function
  - Add in internal/task/parse.go after parseReferences function
  - Extract requirement IDs from markdown links [ID](file#ID)
  - Handle comma-separated requirement links
  - Return requirement IDs array and requirements file path
  - Use compiled regex pattern requirementLinkPattern for performance
  - Reuse existing isValidID function for validation
  - References: Requirements 4.2, 4.3, Design parseRequirements implementation

- [x] 4. Write unit tests for parseRequirements function
  - Test single requirement link parsing
  - Test multiple comma-separated requirement links
  - Test malformed links (no markdown syntax)
  - Test extracting requirements file path from links
  - Test whitespace handling in requirement IDs
  - References: Requirements 4.2, 4.3, 4.4

- [x] 5. Add requirements parsing to parseDetailsAndChildren
  - Add parsing logic for lines starting with 'Requirements: ' or '*Requirements: '
  - Call parseRequirements helper to extract IDs and file path
  - Set task.Requirements array
  - Set taskList.RequirementsFile if not already set
  - Treat malformed requirement lines as plain text details
  - References: Requirements 4.1, 4.4, Decision 5

- [x] 6. Write unit tests for requirements parsing in markdown
  - Test parsing tasks with Requirements detail lines
  - Test requirements extraction from markdown
  - Test RequirementsFile extraction from links
  - Test malformed requirements treated as plain text
  - Test round-trip parsing preservation
  - References: Requirements 4.1, 4.5

## Rendering Implementation

- [x] 7. Update renderTask function signature for requirements file
  - Change renderTask signature to accept reqFile string parameter
  - Update recursive calls to pass reqFile parameter
  - No other changes to existing rendering logic yet
  - References: Design renderTask signature change

- [x] 8. Implement requirements rendering in renderTask
  - Add requirements rendering before references section
  - Format as '  - Requirements: [ID](file#ID), ...'
  - Use plain text format (no italic formatting)
  - Generate comma-separated markdown links
  - Only render if task.Requirements is not empty
  - References: Requirements 2.1, 2.2, 2.3, Decision 9

- [x] 9. Update RenderMarkdown to pass requirements file
  - Determine requirements file from tl.RequirementsFile or default
  - Pass reqFile parameter to renderTask calls
  - Apply same logic to RenderMarkdownWithPhases
  - References: Design RenderMarkdown changes

- [x] 10. Write unit tests for requirements rendering
  - Test rendering tasks with requirements
  - Test markdown link format [ID](file#ID)
  - Test multiple comma-separated requirements
  - Test positioning before references
  - Test plain text format without italics
  - Test round-trip parse-render-parse preservation
  - References: Requirements 2.1, 2.2, 2.3, 4.5

## CLI Command Updates

- [x] 11. Add --requirements flag to add command
  - Add addRequirements string variable in cmd/add.go
  - Add --requirements flag accepting comma-separated requirement IDs
  - Implement parseRequirementIDs helper function
  - Validate requirement ID format using isValidID
  - Update task.Requirements after AddTask call
  - Return clear error messages for invalid format
  - References: Requirements 5.1, 5.4

- [x] 12. Add --requirements-file flag to add command
  - Add addRequirementsFile string variable in cmd/add.go
  - Add --requirements-file flag to specify requirements file path
  - Set tl.RequirementsFile from flag or default to DefaultRequirementsFile
  - Document default behavior in flag description
  - References: Requirements 3.1, 3.2, 5.2

- [x] 13. Write unit tests for add command requirements flags
  - Test --requirements flag parsing
  - Test --requirements-file flag
  - Test validation error for invalid requirement IDs
  - Test default requirements file behavior
  - Test comma-separated requirement IDs
  - References: Requirements 5.1, 5.2, 5.4

## Update Command Changes

- [x] 14. Modify UpdateTask signature to accept requirements
  - Change UpdateTask signature in internal/task/operations.go
  - Add requirements []string parameter
  - Update function to handle nil vs empty slice (nil = no change, empty = clear)
  - Update all existing call sites to pass nil for requirements
  - References: Requirements 5.3, 5.6, Decision 10

- [x] 15. Add requirements flags to update command
  - Add updateRequirements string variable in cmd/update.go
  - Add clearRequirements bool variable
  - Add --requirements flag to replace requirements
  - Add --clear-requirements flag to clear requirements
  - Implement validation using isValidID
  - Call UpdateTask with new requirements parameter
  - References: Requirements 5.3, 5.4, 5.6

- [x] 16. Write unit tests for update command requirements flags
  - Test --requirements flag updates requirements
  - Test --clear-requirements flag clears requirements
  - Test validation errors for invalid IDs
  - Test nil vs empty slice behavior
  - References: Requirements 5.3, 5.6

## Batch Operations Support

- [x] 17. Add Requirements field to Operation struct
  - Add Requirements []string field to Operation in internal/task/batch.go
  - Add RequirementsFile string field to BatchRequest struct
  - Add json tags with omitempty
  - References: Requirements 6.3, 6.4

- [x] 18. Add requirements validation to batch operations
  - Validate requirement ID format in validateOperation for add/update ops
  - Use existing validateTaskIDFormat function
  - Return clear error messages for invalid IDs
  - References: Requirements 6.5

- [x] 19. Update applyOperation to handle requirements
  - Pass requirements to UpdateTask in add operation
  - Pass requirements to UpdateTask in update operation
  - Handle empty requirements array correctly
  - References: Requirements 6.3, 6.4

- [x] 20. Add requirements_file support to batch command
  - In cmd/batch.go runBatch function
  - Set tl.RequirementsFile from BatchRequest.RequirementsFile
  - Default to DefaultRequirementsFile if not specified
  - References: Requirements 6.1, 6.2

- [x] 21. Write unit tests for batch requirements operations
  - Test add operation with requirements field
  - Test update operation with requirements field
  - Test requirements validation in batch
  - Test atomic behavior with invalid requirements
  - Test requirements_file field in BatchRequest
  - References: Requirements 6.1, 6.2, 6.3, 6.4, 6.5

## JSON API Support

- [ ] 22. Verify JSON output includes requirements fields
  - Verify Task.Requirements field appears in JSON output
  - Verify TaskList.RequirementsFile field appears in JSON output
  - Standard Go marshaling should handle this automatically
  - Write test to verify JSON structure matches expectations
  - References: Requirements 7.1, 7.2

## Integration Testing

- [ ] 23. Run integration test for complete requirements workflow
  - Create task file
  - Add task with --requirements and --requirements-file flags
  - Verify requirements rendered as markdown links
  - Parse file and verify Requirements field populated
  - Update requirements via batch command
  - Verify changes persisted correctly
  - Test round-trip preservation
  - References: Requirements 4.5, Design integration tests

## Documentation

- [ ] 24. Update README.md with requirements feature documentation
  - Add Requirements section after References documentation
  - Document --requirements flag with examples
  - Document --requirements-file flag and default behavior
  - Show rendered markdown format for requirement links
  - Explain difference between Requirements and References
  - Include examples for add, update, and clear operations
  - References: Requirements 8.1, 8.2, 8.3, 8.5

- [ ] 25. Update docs/json-api.md with requirements fields
  - Document requirements field in Task JSON structure
  - Document requirements_file field in TaskList JSON structure
  - Add batch operation examples with requirements
  - Show complete JSON examples
  - References: Requirements 8.4

## Final Validation

- [ ] 26. Run complete test suite and validate coverage
  - Run make test to execute all unit tests
  - Run make test-integration to execute integration tests
  - Run make test-coverage to verify >80% coverage for new code
  - Fix any failing tests
  - Ensure backward compatibility with existing task files
  - References: Design test coverage goals

## Code Quality

- [ ] 27. Run linters and format code
  - Run make fmt to format all Go code
  - Run make lint to check for issues
  - Run make modernize to apply modern Go patterns
  - Fix any linter warnings or errors
  - References: CLAUDE.md development commands
