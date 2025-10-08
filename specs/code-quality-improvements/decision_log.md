# Decision Log: Code Quality Improvements

## Overview
This document tracks all decisions made during the requirements phase for the code-quality-improvements feature.

## Decisions

### Decision 1: Feature Name
**Date:** 2025-10-08
**Decision:** Use "code-quality-improvements" as the feature name
**Rationale:** Accurately describes the scope of work - internal code quality refactoring without changing external behavior
**Alternatives Considered:** None, user approved initial proposal

### Decision 2: Backwards Compatibility Approach
**Date:** 2025-10-08
**Decision:** No concern for internal backwards compatibility; only CLI output must remain identical
**Rationale:** These are internal refactorings. As long as the tool's output is the same, internal changes are acceptable
**Impact:** Allows more aggressive refactoring of internal APIs and function signatures

### Decision 3: Test File Splitting Scope
**Date:** 2025-10-08
**Decision:** Split only the most problematic files (batch_test.go at 2549 lines); optionally split parse_test.go
**Rationale:** Focus on files that are significantly oversized; avoid unnecessary churn on files that are acceptable size
**Target Files:**
- `internal/task/batch_test.go` (2549 lines, 41 functions) - required
- `internal/task/parse_test.go` (1175 lines, 16 functions) - optional if improves organization
- `cmd/integration_test.go` (4129 lines, 3 functions) - explicitly out of scope due to its specific integration testing purpose

### Decision 4: Parse Function Naming Strategy
**Date:** 2025-10-08
**Decision:** Rename `parseTaskLineWithError` to `parseTaskLine`, remove the old error-discarding `parseTaskLine`, update all callers
**Rationale:** Makes error handling explicit and idiomatic; removes the non-standard "WithError" suffix
**Alternatives Considered:**
- Keep both with better naming (rejected - still maintains duplication)
- Use `mustParseTaskLine` pattern (rejected - doesn't fit the use case where errors should be handled)

### Decision 5: Helper Functions Location
**Date:** 2025-10-08
**Decision:** Place shared helper functions in `cmd/helpers.go` rather than `internal/task`
**Rationale:** Simpler approach that avoids potential import cycle issues; functions are command-level utilities
**Alternatives Considered:**
- `internal/task` package - rejected due to potential import cycles and these being display/formatting functions, not core task logic
- `internal/cmdhelpers` package - rejected as unnecessary complexity when `cmd/helpers.go` works fine

### Decision 6: Test Coverage Requirements
**Date:** 2025-10-08
**Decision:** All new functions must have tests; existing coverage baseline (70-80%) must be maintained
**Rationale:** Ensures quality while allowing for some flexibility (tests for removed functions can be removed/replaced)
**Specific Requirements:**
- Tests for functions being removed (like old `parseTaskLine`) should be removed
- Tests for functionality that's being consolidated should be updated, not duplicated
- New helper functions in `cmd/helpers.go` need new tests in `cmd/helpers_test.go`

### Decision 7: Implementation Approach
**Date:** 2025-10-08
**Decision:** Implement all improvements as a single cohesive change
**Rationale:** Changes are related and relatively small; single PR easier to review than multiple related PRs
**Alternatives Considered:**
- Separate PRs per improvement - rejected as unnecessary overhead for related refactorings

### Decision 8: Test File Size Threshold
**Date:** 2025-10-08
**Decision:** Target 800-1000 lines per test file (per language guidelines)
**Rationale:** Language guidelines specify 500-800 line threshold, but also mention "where practical"; 800-1000 is a reasonable practical target for this codebase
**Note:** Integration test file is excluded as it has only 3 test functions despite 4129 lines

### Decision 9: String Contains Function Scope
**Date:** 2025-10-08
**Decision:** Include `cmd/integration_test.go` in scope for removing custom string contains functions
**Rationale:** Consistency - if we're removing custom implementations, do it everywhere
**Initial Error:** Requirements initially only mentioned `autocomplete_test.go`, but review found instances in `integration_test.go` as well

### Decision 10: Format Status Function Consolidation
**Date:** 2025-10-08
**Decision:** Evaluate whether `formatStatusMarkdown()` can be combined with `formatStatus()` with a parameter
**Rationale:** Both serve similar purposes (formatting status enum) but for different output formats (string vs markdown checkbox)
**Implementation Note:** Decision deferred to implementation phase - may be better to keep separate if they serve truly different purposes

## Questions Raised During Review

### Question 1: Are helper functions actually duplicated?
**Status:** Resolved
**Answer:** No, initial investigation was incomplete. Functions exist only in `cmd/list.go`, but `cmd/next.go` imports and uses them. The issue is about consolidating for reuse, not eliminating duplication.
**Resolution:** Changed requirement title from "Eliminate Duplicate Helper Functions" to "Consolidate Shared Helper Functions"

### Question 2: Should formatStatusMarkdown be combined with formatStatus?
**Status:** Deferred to implementation
**Answer:** Needs evaluation during implementation - they serve similar purposes but for different output formats
**Action:** Added conditional requirement (1.5-1.6) to evaluate and combine if appropriate

### Question 3: What about import cycles when moving functions?
**Status:** Addressed
**Answer:** Using `cmd/helpers.go` (same package) avoids import cycle issues
**Documentation:** Added technical notes section explaining import cycle prevention

## Review Findings

### Design Critic Review
**Key Issues Identified:**
- False premise about duplication (resolved)
- Vague placement criteria (resolved by choosing `cmd/helpers.go`)
- Wishy-washy conditional logic (resolved for validation functions)
- Missing import cycle analysis (addressed in technical notes)

### Peer Review Validation
**Key Findings:**
- Confirmed need for concrete criteria (added 800-1000 line threshold)
- Validated technical approach for helper function consolidation
- Identified missing scope items (integration_test.go for string contains)
- Supported decision to use `cmd/helpers.go` over separate package

## Design Phase Decisions

### Decision 11: Checkbox Constants Location
**Date:** 2025-10-08
**Decision:** Move checkbox constants from `cmd/next.go` to `cmd/helpers.go` along with `formatStatusMarkdown` function
**Rationale:** The constants are only used by `formatStatusMarkdown`, and both should be together in helpers.go
**Impact:** `cmd/next.go` will have constants removed; helpers.go will define them

### Decision 12: Format Status Functions - Keep Separate
**Date:** 2025-10-08
**Decision:** Keep `formatStatus()` and `formatStatusMarkdown()` as separate functions, do NOT combine with a boolean parameter
**Rationale:**
- Two functions with clear names is more readable than one function with a parameter
- Call sites make intent obvious: `formatStatus(x)` vs `formatStatusMarkdown(x)` is clearer than `formatStatus(x, true)`
- The functions serve different output contexts even though they convert the same enum
**Rejected Alternative:** `formatStatus(status task.Status, markdown bool)` - adds unnecessary complexity
**Decision Resolves:** Question 2 from requirements phase (was deferred, now decided)

### Decision 13: Parse Error Handling Strategy
**Date:** 2025-10-08
**Decision:** Handle parse errors differently based on context:
- `parseTasksAtLevel` (line 155): Propagate errors (abort parsing)
- `parseDetailsAndChildren` (line 255): Propagate errors (abort parsing)
- `ExtractPhaseMarkers` (line 417): Ignore errors (just skip malformed lines)
**Rationale:** Phase marker extraction is best-effort; task structure building requires strict validation
**Implementation:** `err == nil && ok` pattern in ExtractPhaseMarkers to silently skip errors

### Decision 14: Shared Test Helper Organization
**Date:** 2025-10-08
**Decision:** Create dedicated helper test files: `batch_helpers_test.go` and optionally `parse_helpers_test.go`
**Rationale:**
- Clear naming convention for test-only helpers
- Accessible to all test files in the package
- No Test* functions so won't be run as tests themselves
- Better than keeping in batch_test.go which would be confusing after split
**Rejected Alternative:** Keep helpers in batch_test.go - confusing after the main tests are removed

### Decision 15: Phase-Related Helpers
**Date:** 2025-10-08
**Decision:** Do NOT move `findDuplicatePhases()` to helpers.go
**Rationale:** Only used in one location (list command), not a general utility, moving would not improve code
**Principle:** Only consolidate functions that ARE or WILL BE used in multiple places

### Decision 16: Custom String Contains in integration_test.go
**Date:** 2025-10-08
**Decision:** Confirmed `cmd/integration_test.go:1070` contains `containsString()` with 108 usages
**Evidence:** Code inspection during design phase
**Action:** Include in scope for replacement with `strings.Contains()`

## Design Review Findings

### Critical Design Review (design-critic agent)
**Date:** 2025-10-08

**Issues Identified and Addressed:**
1. ✅ Import path verified (lowercase, matches go.mod: `github.com/arjenschwarz/rune`)
2. ✅ Checkbox constants location clarified (move to helpers.go)
3. ✅ Parse error handling strategy documented for all 3 call sites
4. ✅ Test split strategy clarified with batch_helpers_test.go approach
5. ✅ Custom string contains in integration_test.go confirmed (line 1070, 108 usages)
6. ✅ Phase-related helpers evaluated (decision: don't move findDuplicatePhases)
7. ✅ Format status function separation rationale expanded

**Pending for Implementation:**
- Performance impact of parse error handling (to be measured if concerns arise)
- Final validation of test categorization (may adjust groupings during implementation)

## Open Items
Design complete pending peer review and user approval.
