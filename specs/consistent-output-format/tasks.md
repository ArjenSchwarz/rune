---
references:
    - requirements.md
    - design.md
    - decision_log.md
---
# Consistent Output Format

## Foundation

- [x] 1. Create cmd/format.go with shared utility functions
  - Add outputJSON, outputMarkdownMessage, outputMessage, and verboseStderr functions
  - Requirements: [8.1](requirements.md#8.1), [8.2](requirements.md#8.2), [8.3](requirements.md#8.3)

- [x] 2. Add unit tests for format utilities
  - Test outputJSON, outputMarkdownMessage, verboseStderr with different inputs

## Mutation Commands

- [x] 3. Add format support to complete.go
  - Add CompleteResponse struct
  - Add format switch for JSON/markdown/table output
  - Move verbose to stderr when JSON
  - Requirements: [6.1](requirements.md#6.1), [6.2](requirements.md#6.2)

- [x] 4. Add format support to uncomplete.go
  - Add UncompleteResponse struct
  - Add format switch for output
  - Requirements: [6.1](requirements.md#6.1), [6.2](requirements.md#6.2)

- [x] 5. Add format support to progress.go
  - Add ProgressResponse struct
  - Add format switch for output
  - Requirements: [6.1](requirements.md#6.1), [6.2](requirements.md#6.2)

- [x] 6. Add format support to add.go
  - Add AddResponse struct
  - Add format switch for output
  - Requirements: [6.1](requirements.md#6.1), [6.2](requirements.md#6.2)

- [x] 7. Add format support to remove.go
  - Add RemoveResponse struct
  - Add format switch for output
  - Requirements: [6.1](requirements.md#6.1), [6.2](requirements.md#6.2)

- [x] 8. Add format support to update.go
  - Add UpdateResponse struct
  - Add format switch for output
  - Requirements: [6.1](requirements.md#6.1), [6.2](requirements.md#6.2)

- [x] 9. Add format support to create.go
  - Add CreateResponse struct
  - Add format switch for output
  - Requirements: [6.4](requirements.md#6.4)

- [x] 10. Add format support to add_phase.go
  - Add AddPhaseResponse struct
  - Add format switch for output
  - Requirements: [6.1](requirements.md#6.1), [6.2](requirements.md#6.2)

- [x] 11. Add format support to add_frontmatter.go
  - Add AddFrontmatterResponse struct
  - Add format switch for output
  - Requirements: [6.1](requirements.md#6.1), [6.2](requirements.md#6.2)

## Read Commands

- [x] 12. Fix next.go empty state output
  - Replace fmt.Println All tasks complete with format-aware output
  - Replace hardcoded {} with proper JSON structure
  - Fix phase mode empty state output
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.3](requirements.md#3.3), [3.4](requirements.md#3.4)

- [x] 13. Fix list.go empty state output
  - Add format-aware empty state handling
  - Return empty array for JSON when no tasks
  - Requirements: [4.3](requirements.md#4.3), [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3)

- [x] 14. Fix find.go empty state output
  - Add format-aware no matches handling
  - Return empty array for JSON when no matches
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3)

## Align Existing

- [x] 15. Add success field to renumber.go JSON output
  - Update RenumberResponse to include success field
  - Ensure JSON structure matches conventions
  - Requirements: [7.1](requirements.md#7.1), [7.2](requirements.md#7.2), [7.3](requirements.md#7.3)

- [x] 16. Verify batch.go format consistency
  - Review batch output and ensure it follows conventions
  - Add success field if missing

- [x] 17. Document has_phases as JSON-only
  - Update has_phases command help text
  - Ensure non-JSON format flags are ignored gracefully
  - Requirements: [9.1](requirements.md#9.1), [9.2](requirements.md#9.2), [9.3](requirements.md#9.3)

## Testing

- [x] 18. Add format-specific integration tests for mutation commands
  - Test complete, uncomplete, progress with --format json
  - Test add, remove, update with --format json
  - Test create with --format json

- [x] 19. Add empty state integration tests
  - Test next all-complete with --format json
  - Test list empty with --format json
  - Test find no-matches with --format json

- [x] 20. Add verbose + JSON integration tests
  - Verify verbose output goes to stderr when --format json
  - Verify stdout contains only valid JSON

- [x] 21. Run full test suite and fix any failures
  - Run make check
  - Fix any linting or test failures
