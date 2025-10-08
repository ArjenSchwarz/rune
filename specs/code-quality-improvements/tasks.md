---
references:
    - specs/code-quality-improvements/requirements.md
    - specs/code-quality-improvements/design.md
    - specs/code-quality-improvements/decision_log.md
---
# Code Quality Improvements Tasks

## Phase 1: Helper Functions Consolidation

- [x] 1. Create cmd/helpers.go with shared utility functions
  - Create new file cmd/helpers.go
  - Add checkbox constants (checkboxPending, checkboxInProgress, checkboxCompleted)
  - Implement formatStatus() function
  - Implement formatStatusMarkdown() function
  - Implement getTaskLevel() function
  - Implement countAllTasks() function
  - Ensure proper package imports
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.3](requirements.md#1.3), [1.4](requirements.md#1.4), [1.5](requirements.md#1.5)

- [x] 2. Create cmd/helpers_test.go with unit tests
  - Create test file cmd/helpers_test.go
  - Write TestFormatStatus with cases for all status types
  - Write TestFormatStatusMarkdown with cases for all status types
  - Write TestGetTaskLevel with cases for different ID depths
  - Write TestCountAllTasks with flat and nested hierarchies
  - Ensure all tests pass
  - Requirements: [1.8](requirements.md#1.8)

- [x] 3. Update cmd/list.go to use shared helpers
  - Remove formatStatus() function definition from cmd/list.go
  - Remove getTaskLevel() function definition from cmd/list.go
  - Remove countAllTasks() function definition from cmd/list.go
  - Verify all call sites now use helpers from cmd/helpers.go
  - Run tests to ensure no breakage
  - Requirements: [1.7](requirements.md#1.7), [1.9](requirements.md#1.9)

- [x] 4. Update cmd/next.go to use shared helpers
  - Remove formatStatusMarkdown() function definition from cmd/next.go
  - Remove checkbox constants from cmd/next.go
  - Verify all call sites now use helpers from cmd/helpers.go
  - Run tests to ensure no breakage
  - Requirements: [1.7](requirements.md#1.7), [1.9](requirements.md#1.9)

## Phase 2: Remove Custom String Contains

- [ ] 5. Replace custom string contains in internal/task/autocomplete_test.go
  - Remove containsString() function definition
  - Remove stringContains() function definition
  - Replace all usages with strings.Contains()
  - Verify strings package is imported
  - Run tests to ensure all assertions still work
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.5](requirements.md#2.5)

- [ ] 6. Replace custom string contains in cmd/integration_test.go
  - Remove containsString() function definition at line 1070
  - Replace all 108 usages with strings.Contains()
  - Verify strings package is imported
  - Run integration tests to ensure all assertions still work
  - Requirements: [2.4](requirements.md#2.4), [2.5](requirements.md#2.5)

## Phase 3: Parse Function Refactoring

- [ ] 7. Rename parseTaskLineWithError to parseTaskLine
  - In internal/task/parse.go, rename parseTaskLineWithError() to parseTaskLine()
  - Update function signature to return (Task, bool, error)
  - Remove old parseTaskLine() wrapper function
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2)

- [ ] 8. Update parse.go:155 parseTasksAtLevel caller
  - Update line 155 call from parseTaskLineWithError to parseTaskLine
  - Verify error handling remains correct (already handles errors)
  - Run parse tests to ensure no breakage
  - Requirements: [3.3](requirements.md#3.3), [3.5](requirements.md#3.5)

- [ ] 9. Update parse.go:255 parseDetailsAndChildren caller
  - Update line 255 to call parseTaskLine with error handling
  - Add error check: if err != nil, return with formatted error
  - Maintain existing ok check for task detection
  - Run parse tests to ensure errors are properly propagated
  - Requirements: [3.3](requirements.md#3.3), [3.5](requirements.md#3.5)

- [ ] 10. Update parse.go:417 ExtractPhaseMarkers caller
  - Update line 417 to call parseTaskLine
  - Use pattern: if _, ok, err := parseTaskLine(line); err == nil && ok
  - Ensure malformed tasks are silently skipped (current behavior)
  - Run phase-related tests to ensure no breakage
  - Requirements: [3.3](requirements.md#3.3), [3.5](requirements.md#3.5)

- [ ] 11. Update or remove tests for old parseTaskLine
  - Review existing tests that test parseTaskLine error handling
  - Remove or update tests specific to error-discarding behavior
  - Ensure parseTaskLine error handling is tested
  - Verify test coverage is maintained
  - Requirements: [3.4](requirements.md#3.4), [3.6](requirements.md#3.6)

## Phase 4: ID Validation Simplification

- [ ] 12. Consolidate ID validation to single IsValidID function
  - In internal/task/task.go, update isValidID() to become the implementation for IsValidID()
  - Remove the old IsValidID() wrapper function
  - Keep single IsValidID() as the public API
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.4](requirements.md#4.4)

- [ ] 13. Update all isValidID call sites to use IsValidID
  - Update internal/task/parse.go:374
  - Update internal/task/task.go:126
  - Update internal/task/task.go:131
  - Update internal/task/batch.go:29
  - Update internal/task/operations.go:73
  - Update internal/task/operations.go:408
  - Run all tests to ensure no breakage
  - Requirements: [4.3](requirements.md#4.3), [4.5](requirements.md#4.5)

## Phase 5: Test File Reorganization

- [ ] 14. Create batch_helpers_test.go for shared test utilities
  - Create internal/task/batch_helpers_test.go
  - Move StatusPtr helper function
  - Move any other shared test setup functions
  - Ensure no Test* functions in this file
  - Requirements: [5.4](requirements.md#5.4)

- [ ] 15. Create batch_add_test.go with addition operation tests
  - Create internal/task/batch_add_test.go
  - Use .claude/scripts/move_code_section.py to move 11 test functions from batch_test.go
  - Move: TestExecuteBatch_SingleAdd, TestExecuteBatch_MultipleOperations, TestExecuteBatch_ComplexOperations
  - Move: TestExecuteBatch_PositionInsertionSingle, TestExecuteBatch_PositionInsertionHierarchical
  - Move: TestExecuteBatch_PositionInsertionValidation, TestExecuteBatch_PositionInsertionMultiple
  - Move: TestExecuteBatch_PositionInsertionWithOtherOperations, TestExecuteBatch_PositionInsertionAtomicFailure
  - Move: TestExecuteBatch_PositionInsertionDryRun, TestExecuteBatch_AddWithRequirements
  - Run tests to ensure all moved tests pass
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.5](requirements.md#5.5), [5.7](requirements.md#5.7)
  - References: .claude/scripts/move_code_section.py

- [ ] 16. Create batch_update_test.go with update operation tests
  - Create internal/task/batch_update_test.go
  - Use .claude/scripts/move_code_section.py to move 13 test functions from batch_test.go
  - Move: TestExecuteBatch_UnifiedUpdateWithStatus, TestExecuteBatch_UnifiedUpdateOperations
  - Move: TestExecuteBatch_UnifiedUpdateStatusOnly, TestExecuteBatch_UnifiedUpdateTitleAndStatus
  - Move: TestExecuteBatch_UnifiedUpdateDetailsAndReferencesWithoutStatus
  - Move: TestExecuteBatch_UnifiedUpdateEmptyOperation, TestExecuteBatch_UnifiedUpdateTitleLengthValidation
  - Move: TestExecuteBatch_UnifiedUpdateAutoCompleteTriggers, TestExecuteBatch_UpdateStatusOperationInvalid
  - Move: TestExecuteBatch_UpdateWithRequirements, TestExecuteBatch_AutoCompleteSimpleHierarchy
  - Move: TestExecuteBatch_AutoCompleteMultiLevel, TestExecuteBatch_AutoCompletePartialCompletion
  - Run tests to ensure all moved tests pass
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.5](requirements.md#5.5), [5.7](requirements.md#5.7)
  - References: .claude/scripts/move_code_section.py

- [ ] 17. Create batch_validation_test.go with validation tests
  - Create internal/task/batch_validation_test.go
  - Use .claude/scripts/move_code_section.py to move 10 test functions from batch_test.go
  - Move: TestExecuteBatch_ValidationFailures, TestExecuteBatch_AtomicFailure, TestExecuteBatch_DryRun
  - Move: TestBatchRequest_JSONSerialization, TestBatchResponse_JSONSerialization
  - Move: TestValidateOperation_EdgeCases, TestExecuteBatch_RequirementsValidation
  - Move: TestExecuteBatch_AtomicBehaviorWithInvalidRequirements, TestBatchRequest_RequirementsFile
  - Move: TestExecuteBatch_AutoCompleteErrorHandling
  - Run tests to ensure all moved tests pass
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.5](requirements.md#5.5), [5.7](requirements.md#5.7)
  - References: .claude/scripts/move_code_section.py

- [ ] 18. Create batch_operations_test.go with complex operation tests
  - Create internal/task/batch_operations_test.go
  - Use .claude/scripts/move_code_section.py to move 7 test functions from batch_test.go
  - Move: TestExecuteBatch_AutoCompleteWithMixedOperations
  - Move: TestExecuteBatch_AutoCompleteSameParentMultipleTimes, TestExecuteBatch_AutoCompleteDryRun
  - Move: TestExecuteBatch_AutoCompleteComplexScenario, TestExecuteBatch_PhaseAddOperation
  - Move: TestExecuteBatch_PhaseDuplicateHandling, TestExecuteBatch_MixedPhaseOperations
  - Run tests to ensure all moved tests pass
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.5](requirements.md#5.5), [5.7](requirements.md#5.7)
  - References: .claude/scripts/move_code_section.py

- [ ] 19. Remove original batch_test.go or update with remaining tests
  - After moving all 41 test functions, check if batch_test.go is empty
  - If empty, remove batch_test.go
  - If any tests remain, verify they're intentionally kept
  - Run all batch tests to ensure complete coverage
  - Requirements: [5.5](requirements.md#5.5), [5.6](requirements.md#5.6)

- [ ] 20. Evaluate and optionally split parse_test.go
  - Review parse_test.go test organization (1175 lines, 16 functions)
  - If splitting improves organization, create parse_basic_test.go (9 functions)
  - If splitting improves organization, create parse_frontmatter_test.go (7 functions)
  - Use .claude/scripts/move_code_section.py for moving test functions if splitting
  - Create parse_helpers_test.go for writeTestFile helper if needed
  - Run all parse tests to ensure no breakage
  - Requirements: [5.3](requirements.md#5.3), [5.4](requirements.md#5.4), [5.5](requirements.md#5.5), [5.7](requirements.md#5.7)
  - References: .claude/scripts/move_code_section.py

## Phase 6: Final Validation

- [ ] 21. Verify all unit tests pass
  - Run make test to execute all unit tests
  - Verify zero failures
  - Check that all test files are properly organized
  - Requirements: [6.1](requirements.md#6.1)

- [ ] 22. Verify all integration tests pass
  - Run INTEGRATION=1 make test-integration
  - Verify zero failures
  - Ensure CLI behavior is unchanged
  - Requirements: [6.2](requirements.md#6.2)

- [ ] 23. Run code quality checks
  - Run golangci-lint and verify zero issues
  - Run make modernize and verify no changes needed
  - Run make fmt to ensure proper formatting
  - Requirements: [6.3](requirements.md#6.3), [6.4](requirements.md#6.4), [6.7](requirements.md#6.7)

- [ ] 24. Verify test coverage baseline
  - Run make test-coverage
  - Verify coverage is maintained at 70-80% baseline
  - Document any coverage changes
  - Requirements: [6.5](requirements.md#6.5)

- [ ] 25. Validate CLI output remains identical
  - Test rune commands against example files
  - Compare output before and after refactoring
  - Verify list, next, add, update, batch commands produce same output
  - Document validation results
  - Requirements: [6.6](requirements.md#6.6)
