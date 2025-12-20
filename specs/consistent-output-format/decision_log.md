# Decision Log: Consistent Output Format

## Decision 1: Include Mutation Commands in Scope

**Date**: 2025-12-19
**Status**: accepted

### Context

The original scope only covered read commands (next, list, find, renumber). However, mutation commands (complete, add, remove, update, etc.) have the same issue - they output plain text confirmation messages regardless of the format flag. For full programmatic use, mutation confirmations also need structured output.

### Decision

Include all mutation commands in scope: complete, uncomplete, progress, add, remove, update, renumber, create, add_phase, add_frontmatter.

### Rationale

If the goal is programmatic use, automation scripts need structured confirmation of every operation, not just read operations. Excluding mutations would leave the feature incomplete.

### Alternatives Considered

- **Read-only commands only**: Focus on next, list, find - Rejected because it leaves mutation confirmations inconsistent, requiring special-case parsing in automation scripts.

### Consequences

**Positive:**
- Complete programmatic support for all operations
- Consistent user experience across all commands

**Negative:**
- Larger implementation scope
- More commands to update and test

---

## Decision 2: Keep Success Field in JSON Responses

**Date**: 2025-12-19
**Status**: accepted

### Context

External review suggested removing the `success` field since exit codes already signal success/failure. The question was whether to keep explicit `success` in the JSON response or rely solely on exit codes.

### Decision

Keep the `success` field in all JSON responses.

### Rationale

Having explicit `success` in the response makes the JSON self-describing. While exit codes are the primary signal, having success in the payload is useful for logging, debugging, and cases where the exit code isn't easily accessible (e.g., piped output).

### Alternatives Considered

- **Remove success field**: Rely on exit codes only - Rejected because it makes the JSON less self-describing and harder to debug in logs.

### Consequences

**Positive:**
- Self-describing responses
- Easier debugging and logging
- Works in contexts where exit code isn't accessible

**Negative:**
- Slight redundancy with exit codes
- Marginally larger JSON payloads

---

## Decision 3: Type-Appropriate Empty Values

**Date**: 2025-12-19
**Status**: accepted

### Context

Empty state responses needed consistent handling. The question was whether to use `null` universally or use type-appropriate empty values.

### Decision

Use type-appropriate empty values: `[]` for list operations, `null` for single-item operations.

### Rationale

This matches industry practice (GitHub CLI, kubectl) and makes responses more predictable. List operations always return arrays (empty or populated), single-item operations return the item or null.

### Alternatives Considered

- **Use null universally**: All empty states return `data: null` - Rejected because it requires consumers to check for both null and empty array for list operations.

### Consequences

**Positive:**
- Predictable response shapes per command type
- Easier client-side parsing (no null-vs-array checks)
- Matches industry standards

**Negative:**
- Slightly more complex server-side logic

---

## Decision 4: has_phases Remains JSON-Only

**Date**: 2025-12-19
**Status**: accepted

### Context

The `has_phases` command was identified as outputting only JSON. The question was whether to add table/markdown support or document it as intentionally JSON-only.

### Decision

Keep `has_phases` JSON-only and document this as intentional.

### Rationale

The command is designed for programmatic detection (scripts checking if a file has phases). Human-readable output provides no value for this use case.

### Alternatives Considered

- **Add full format support**: Implement table/markdown output - Rejected because it adds complexity for no practical benefit.

### Consequences

**Positive:**
- Simpler implementation
- Clear documentation of intended use

**Negative:**
- Inconsistency with other commands (mitigated by documentation)

---

## Decision 5: Errors to Stderr as Plain Text

**Date**: 2025-12-19
**Status**: accepted

### Context

Error messages could either respect the format flag (outputting JSON errors) or follow standard CLI convention (plain text to stderr).

### Decision

Error messages go to stderr as plain text regardless of format flag.

### Rationale

This is the standard Unix/CLI convention. Exit codes and stderr are the established mechanisms for error signaling. Mixing JSON errors with JSON data on stdout would complicate parsing.

### Alternatives Considered

- **JSON errors to stdout**: Output `{"success": false, "error": "..."}` - Rejected because it breaks CLI conventions and complicates stdout parsing.

### Consequences

**Positive:**
- Follows established conventions
- Clean separation of success data (stdout) and errors (stderr)
- Works with existing shell scripting patterns

**Negative:**
- Requires separate stderr handling for full automation

---

## Decision 6: Verbose Output to Stderr with JSON Format

**Date**: 2025-12-19
**Status**: accepted

### Context

When `--verbose` and `--format json` are both set, verbose messages (like "Using task file: X") would break JSON parsing if written to stdout.

### Decision

When JSON format is requested, verbose messages go to stderr to preserve stdout parseability.

### Rationale

Stdout must contain only valid JSON when JSON format is requested. Verbose output is informational and belongs on stderr in this context.

### Alternatives Considered

- **Include verbose in JSON**: Add a `verbose` field to JSON output - Rejected because it changes the response structure based on verbose flag, complicating consumers.
- **Disable verbose with JSON**: Silently ignore verbose flag with JSON - Rejected because it hides useful debugging information.

### Consequences

**Positive:**
- JSON output remains parseable
- Verbose information still available
- Consistent with error handling pattern

**Negative:**
- Verbose output location differs by format (stdout for table/markdown, stderr for JSON)

---

## Decision 7: Markdown Empty State Format

**Date**: 2025-12-19
**Status**: accepted

### Context

Empty state messages in markdown format needed a consistent representation. Options included plain paragraphs, blockquotes, or other markdown structures.

### Decision

Use blockquote format: `> Message here`

### Rationale

Blockquotes visually distinguish informational messages from task content. They're valid markdown and render distinctively in markdown viewers.

### Alternatives Considered

- **Plain paragraph**: Just the message text - Rejected because it's indistinguishable from task content.
- **Italics**: `*Message here*` - Rejected because it's less visually distinct than blockquotes.

### Consequences

**Positive:**
- Clear visual distinction
- Valid, standard markdown
- Easy to parse programmatically if needed

**Negative:**
- Slightly more verbose than plain text

---

## Decision 8: Command-Specific Response Types Over Unified Helper

**Date**: 2025-12-19
**Status**: accepted

### Context

The initial design proposed a centralized OutputHelper with generic response types (MutationResponse, ListResponse, SingleItemResponse) that all commands would use. Design review identified that commands like `renumber`, `batch`, `create`, and `next --phase` have fundamentally different data structures that don't fit generic types well.

### Decision

Use command-specific response types instead of unified generic types. Each command defines its own response struct matching its natural data shape. Shared utilities are limited to small helper functions (outputJSON, verboseStderr).

### Rationale

Commands have different data needs. Forcing them into generic types either loses information or requires awkward mappings. Keeping response types local to commands is simpler and more maintainable. The core problem (respecting format flag) doesn't require unified types.

### Alternatives Considered

- **Centralized OutputHelper with generic types**: A single helper managing all output with MutationResponse, ListResponse, SingleItemResponse types - Rejected because commands have different data shapes that don't fit cleanly into generic types.
- **Hybrid approach**: Unified envelope (success, message) with command-specific data - Rejected because it's still trying to force structure where flexibility is needed.

### Consequences

**Positive:**
- Each command's response matches its data naturally
- No awkward type mappings or forced generics
- Easier to understand and maintain
- Changes to one command don't affect others

**Negative:**
- Less standardization across commands (mitigated by common conventions)
- Response types defined in multiple files
