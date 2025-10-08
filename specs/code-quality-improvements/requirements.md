# Code Quality Improvements Requirements

## Introduction

This feature addresses code quality issues identified during a comprehensive codebase review. The improvements focus on consolidating shared utility functions, removing unnecessary custom implementations in favor of standard library functions, improving naming consistency, and reorganizing oversized test files. All changes are internal refactorings that maintain the same external behavior and tool output while improving maintainability and code clarity.

## Requirements

### 1. Consolidate Shared Helper Functions

**User Story:** As a developer, I want shared utility functions to exist in a centralized location, so that they can be easily reused across commands and maintained in one place.

**Acceptance Criteria:**

1. <a name="1.1"></a>The system SHALL create a new file `cmd/helpers.go` for shared command-level utility functions
2. <a name="1.2"></a>The system SHALL move `formatStatus()` from `cmd/list.go` to `cmd/helpers.go`
3. <a name="1.3"></a>The system SHALL move `getTaskLevel()` from `cmd/list.go` to `cmd/helpers.go`
4. <a name="1.4"></a>The system SHALL move `countAllTasks()` from `cmd/list.go` to `cmd/helpers.go`
5. <a name="1.5"></a>The system SHALL evaluate whether `formatStatusMarkdown()` in `cmd/next.go` can be combined with `formatStatus()` or should remain separate
6. <a name="1.6"></a>IF `formatStatusMarkdown()` can be unified with `formatStatus()`, the system SHALL combine them into a single function with a parameter to control output format
7. <a name="1.7"></a>The system SHALL update all references in `cmd/list.go` and `cmd/next.go` to use the shared implementations
8. <a name="1.8"></a>The system SHALL add unit tests for all helper functions in `cmd/helpers_test.go`
9. <a name="1.9"></a>The system SHALL maintain existing test coverage for functionality that uses these helpers

### 2. Remove Custom String Contains Implementation

**User Story:** As a developer, I want to use standard library functions instead of custom reimplementations, so that the code is more maintainable and easier to understand.

**Acceptance Criteria:**

1. <a name="2.1"></a>The system SHALL remove the custom `containsString()` function from `internal/task/autocomplete_test.go`
2. <a name="2.2"></a>The system SHALL remove the custom `stringContains()` function from `internal/task/autocomplete_test.go`
3. <a name="2.3"></a>The system SHALL replace all usages of custom string contains functions in `internal/task/autocomplete_test.go` with `strings.Contains()` from the standard library
4. <a name="2.4"></a>The system SHALL identify and replace any custom string contains functions in `cmd/integration_test.go` with `strings.Contains()`
5. <a name="2.5"></a>The system SHALL maintain all existing test functionality and assertions after the replacement
6. <a name="2.6"></a>The system SHALL verify no other custom string utility functions duplicate standard library functionality across the codebase

### 3. Improve Task Parsing Function Naming

**User Story:** As a developer, I want consistent and idiomatic function naming for task parsing, so that error handling is clear and the API is easier to understand.

**Acceptance Criteria:**

1. <a name="3.1"></a>The system SHALL rename `parseTaskLineWithError()` to `parseTaskLine()` in `internal/task/parse.go`
2. <a name="3.2"></a>The system SHALL remove the existing `parseTaskLine()` function that discards errors
3. <a name="3.3"></a>The system SHALL update all callers of the old `parseTaskLine()` to handle errors from the renamed function
4. <a name="3.4"></a>The system SHALL remove or update tests specific to the error-discarding `parseTaskLine()` function
5. <a name="3.5"></a>The system SHALL ensure all parsing errors are properly handled or propagated by callers
6. <a name="3.6"></a>The system SHALL maintain or improve test coverage for task line parsing functionality

### 4. Simplify ID Validation Functions

**User Story:** As a developer, I want a single clear implementation of ID validation, so that the code is simpler and easier to maintain.

**Acceptance Criteria:**

1. <a name="4.1"></a>The system SHALL consolidate the unexported `isValidID()` and exported `IsValidID()` functions into a single implementation
2. <a name="4.2"></a>The system SHALL keep `IsValidID()` as the public API function
3. <a name="4.3"></a>The system SHALL update all internal callers currently using `isValidID()` to use `IsValidID()`
4. <a name="4.4"></a>The system SHALL remove the now-redundant `isValidID()` function
5. <a name="4.5"></a>The system SHALL ensure all existing ID validation tests continue to pass

### 5. Split Oversized Test Files

**User Story:** As a developer, I want test files organized by functional areas with reasonable sizes (800-1000 lines), so that tests are easier to navigate and maintain.

**Acceptance Criteria:**

1. <a name="5.1"></a>The system SHALL split `internal/task/batch_test.go` (2549 lines, 41 test functions) into multiple files organized by batch operation type
2. <a name="5.2"></a>The system SHALL create separate test files: `batch_add_test.go`, `batch_update_test.go`, `batch_remove_test.go`, and `batch_validation_test.go`
3. <a name="5.3"></a>The system SHALL evaluate `internal/task/parse_test.go` (1175 lines, 16 test functions) and split it into `parse_basic_test.go` and `parse_frontmatter_test.go` if it improves organization
4. <a name="5.4"></a>The system SHALL keep shared test helper functions in a location accessible to all split test files
5. <a name="5.5"></a>The system SHALL maintain all existing test functionality after splitting
6. <a name="5.6"></a>The system SHALL ensure all tests continue to pass after the split
7. <a name="5.7"></a>The system SHALL aim for individual test files to be between 800-1000 lines where practical

### 6. Quality Assurance

**User Story:** As a developer, I want all refactoring changes to be validated against quality standards, so that improvements don't introduce regressions.

**Acceptance Criteria:**

1. <a name="6.1"></a>The system SHALL pass all existing unit tests after all changes
2. <a name="6.2"></a>The system SHALL pass all integration tests (INTEGRATION=1) after all changes
3. <a name="6.3"></a>The system SHALL pass `golangci-lint` with zero issues
4. <a name="6.4"></a>The system SHALL pass `make modernize` with no changes needed
5. <a name="6.5"></a>The system SHALL maintain current test coverage baseline of 70-80% as measured by `make test-coverage`
6. <a name="6.6"></a>The system SHALL produce identical CLI output from the rune tool for all commands when tested against example files
7. <a name="6.7"></a>The system SHALL ensure all code follows the Go language rules defined in `language-rules/go.md`

## Success Criteria

The code quality improvements will be considered successful when:

1. All shared helper functions are consolidated in `cmd/helpers.go`
2. No custom reimplementations of standard library functions exist in test files
3. Function naming follows idiomatic Go patterns with proper error handling
4. Test files are organized and reasonably sized (targeting 800-1000 lines per file)
5. All tests pass and coverage baseline is maintained
6. The tool produces identical output before and after changes
7. Code passes all linters and quality checks

## Out of Scope

- Refactoring test files already under 1000 lines that are well-organized
- Changes to external APIs or CLI command interfaces
- Performance optimizations beyond code organization
- Adding new features or functionality
- Modifying `cmd/integration_test.go` (4129 lines) as it serves a specific integration testing purpose with only 3 test functions and is appropriately sized for its scope

## Technical Notes

### Import Cycle Prevention

When moving functions to `cmd/helpers.go`, care must be taken to avoid import cycles. Since `cmd/helpers.go` will be in the same package as other command files, no import cycle issues are expected. However, if helper functions need to import from other `cmd` files, this should be flagged and resolved.

### Baseline Metrics

Current state before refactoring:
- Test coverage: 70-80% (per project guidelines)
- Largest test file: `cmd/integration_test.go` (4129 lines, 3 functions - out of scope)
- Second largest: `internal/task/batch_test.go` (2549 lines, 41 functions - in scope)
- Files to split: 2 confirmed (`batch_test.go`, optionally `parse_test.go`)
- Custom string functions to remove: 2-4 instances across 2 files

### Testing Strategy

- Existing tests for moved functions should be moved to `cmd/helpers_test.go`
- Integration tests should validate that CLI output remains identical
- Unit tests should verify all error handling paths in refactored parsing code
