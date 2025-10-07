# Decision Log: Task Requirements Linking Feature

## Decision 1: Separate Requirements Field vs Extending References
**Date:** 2025-10-07
**Status:** Accepted

**Context:**
Should we add a new Requirements field or extend the existing References field to support requirement linking?

**Decision:**
Add a separate Requirements field to the Task struct.

**Rationale:**
- Semantic clarity: Requirements and References serve different purposes
- Requirements link to acceptance criteria with automatic link generation
- References are free-form text without link generation
- Separating concerns makes the data model clearer
- Easier to query and filter tasks by requirements vs general references

## Decision 2: No Requirement ID Validation
**Date:** 2025-10-07
**Status:** Accepted

**Context:**
Should the system validate that requirement IDs actually exist in the requirements file?

**Decision:**
Do not validate requirement ID existence.

**Rationale:**
- User explicitly stated "the rest of the workflow will already ensure that things like anchors are present"
- Validation would add complexity
- Broken links are easy to detect when clicked
- Keeps the implementation simple per user directive
- External workflow handles anchor management

## Decision 3: TaskList as Single Source of Truth for Requirements File Path
**Date:** 2025-10-07
**Status:** Accepted

**Context:**
Where should the requirements file path be stored - in TaskList or extracted from individual task links?

**Decision:**
Store in TaskList.RequirementsFile as the single source of truth.

**Rationale:**
- Eliminates contradiction between storing path in TaskList vs extracting from links
- All tasks in a file share the same requirements file
- Simpler to manage and less error-prone
- Aligns with the default behavior (requirements.md in same directory)
- Path extracted from links during parsing, stored in TaskList structure in memory

## Decision 6: No Requirements File Configuration in Create Command
**Date:** 2025-10-07
**Status:** Accepted

**Context:**
Should the `create` command support `--requirements-file` flag?

**Decision:**
Do not add `--requirements-file` to the create command. Only support it in `add` command and as JSON field in `batch` command.

**Rationale:**
- Create command only creates empty task files - no tasks to link yet
- Adding requirements file path at creation would require storing it somewhere (front matter) which makes the markdown messier
- Requirements file path is only needed when actually adding tasks with requirements
- Keeps create command simple and focused on file creation
- Users can specify requirements file when they add first task with requirements

## Decision 7: Batch Command Uses JSON Field Instead of CLI Flag
**Date:** 2025-10-07
**Status:** Accepted

**Context:**
Should the batch command use a `--requirements-file` CLI flag or a JSON field?

**Decision:**
Use an optional "requirements_file" field in the batch JSON format, not a CLI flag.

**Rationale:**
- Batch operations already use JSON for all operation details
- JSON field allows different requirements files per batch operation if needed
- Keeps CLI flags minimal and consistent with batch command philosophy
- Defaults to "requirements.md" when field is omitted
- More flexible for programmatic use

## Decision 8: Simplified Requirements Scope
**Date:** 2025-10-07
**Status:** Accepted

**Context:**
Should we specify all implementation details, backward compatibility, and standard behaviors in requirements?

**Decision:**
Remove redundant requirements including entire backward compatibility section, standard JSON marshaling behaviors, and overly prescriptive formatting details.

**Rationale:**
- User directive: "Do not overcomplicate this design"
- Backward compatibility is implicit when adding optional fields
- Standard Go behaviors (omitempty, data preservation) don't need specification
- Consolidate formatting requirements into single high-level requirement
- Focus requirements on actual feature behavior, not implementation obviousness
- Reduced from 35+ acceptance criteria to ~20 focused requirements

## Decision 4: Link Format Uses Standard Markdown
**Date:** 2025-10-07
**Status:** Accepted

**Context:**
What format should requirement links use?

**Decision:**
Use standard markdown link format: `[ID](file#ID)`

**Rationale:**
- Standard markdown format for links with anchors
- ID appears in both text and anchor for clarity
- Compatible with all markdown renderers
- Follows existing patterns in the ecosystem
- Easy to parse and generate

## Decision 5: Error Handling for Malformed Requirements
**Date:** 2025-10-07
**Status:** Accepted

**Context:**
How should the parser handle malformed requirement lines?

**Decision:**
Treat malformed requirement lines as plain text details and continue parsing.

**Rationale:**
- Graceful degradation instead of failing
- Preserves user content even if format is wrong
- Consistent with parser philosophy of reporting errors without auto-correction
- Users can manually fix formatting if needed
- Doesn't block other parsing operations

## Decision 9: Plain Text Formatting for Requirements
**Date:** 2025-10-07
**Status:** Accepted

**Context:**
Should requirements be rendered with italic formatting like references (`*Requirements: ...*`) or plain text?

**Decision:**
Use plain text format without italics: `Requirements: [1.1](file#1.1)`

**Rationale:**
- Simplifies parsing - no need to handle asterisks
- Reduces risk of round-trip parsing bugs (parser/renderer synchronization)
- Keeps implementation simple per user directive
- References maintain italic formatting for visual differentiation
- Easier to parse consistently

## Decision 10: Modify UpdateTask Signature Directly
**Date:** 2025-10-07
**Status:** Accepted

**Context:**
How to add requirements parameter to task updates?

**Decision:**
Modify existing `UpdateTask()` signature to add requirements parameter. Update all call sites to pass `nil` for requirements if not modifying them.

**Rationale:**
- Tool is internal, so backward compatibility is not a concern
- Simpler to have one function instead of multiple
- All call sites can be easily updated
- Avoids unnecessary complexity
- Follows user directive: "don't overcomplicate things"
