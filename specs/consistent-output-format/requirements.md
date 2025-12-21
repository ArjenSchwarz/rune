# Consistent Output Format

## Introduction

The rune CLI provides a `--format` flag (`-f`) supporting three output formats: `table` (default), `markdown`, and `json`. Currently, many commands bypass this flag in certain scenarios, outputting plain text regardless of the requested format. This is problematic for programmatic use where JSON output is expected but plain text is returned.

This feature ensures all commands respect the format flag consistently, providing structured responses for all output scenarios including empty results, success confirmations, and informational messages.

## Scope

### Commands Affected
- **Read commands:** `next`, `list`, `find`
- **Mutation commands:** `complete`, `uncomplete`, `progress`, `add`, `remove`, `update`, `renumber`, `create`, `add_phase`, `add_frontmatter`

### Exclusions
- The `has_phases` command intentionally remains JSON-only as it's designed for programmatic detection
- Error messages continue to use stderr as plain text (standard CLI convention)

---

## Requirements

### 1. JSON Response Structure

**User Story:** As a developer using rune programmatically, I want consistent JSON structures across all commands, so that I can parse output reliably without per-command special cases.

**Acceptance Criteria:**

1. <a name="1.1"></a>All JSON responses SHALL include a `success` field (boolean) indicating operation outcome

2. <a name="1.2"></a>All JSON responses MAY include a `message` field (string) for human-readable context

3. <a name="1.3"></a>JSON responses from list-type operations (list, find) SHALL use an array for the `data` field, returning `[]` when empty

4. <a name="1.4"></a>JSON responses from single-item operations (next) SHALL use `null` for the `data` field when no item exists

5. <a name="1.5"></a>JSON responses from list-type operations SHALL include a `count` field (integer) indicating the number of items returned

### 2. Empty State Responses

**User Story:** As a developer using rune programmatically, I want empty state messages returned in my requested format, so that I can parse all output consistently without special-casing plain text messages.

**Acceptance Criteria:**

1. <a name="2.1"></a>WHEN the format flag is set to `json` AND a command has no results to display, the system SHALL output a structured JSON response following the structure defined in requirement 1

2. <a name="2.2"></a>WHEN the format flag is set to `markdown` AND a command has no results to display, the system SHALL output the message as a markdown paragraph prefixed with `> ` (blockquote)

3. <a name="2.3"></a>WHEN the format flag is set to `table` AND a command has no results to display, the system SHALL output the message as plain text (current behavior)

### 3. Next Command Format Compliance

**User Story:** As a developer, I want the `next` command to respect my format choice in all scenarios, so that I can reliably parse its output in automation scripts.

**Acceptance Criteria:**

1. <a name="3.1"></a>WHEN `rune next` is called with `--format json` AND all tasks are complete, the system SHALL output `{"success": true, "message": "All tasks are complete!", "data": null}`

2. <a name="3.2"></a>WHEN `rune next --phase` is called with `--format json` AND no pending tasks exist in any phase, the system SHALL output a structured JSON response instead of plain text

3. <a name="3.3"></a>The system SHALL NOT output hardcoded `{}` for empty phase results

4. <a name="3.4"></a>All output paths in the `next` command SHALL route through format-aware output functions

### 4. List Command Format Compliance

**User Story:** As a developer, I want the `list` command output to be consistent regardless of format chosen, so that switching formats doesn't require changes to my parsing logic structure.

**Acceptance Criteria:**

1. <a name="4.1"></a>WHEN `rune list` produces JSON output, the system SHALL output valid JSON to stdout with `count` and `data` fields

2. <a name="4.2"></a>WHEN `rune list` produces markdown output, the system SHALL output valid markdown to stdout

3. <a name="4.3"></a>Empty task lists with `--format json` SHALL return `{"success": true, "message": "No tasks found", "count": 0, "data": []}`

### 5. Find Command Format Compliance

**User Story:** As a developer, I want search results from `find` to respect my format choice, so that I can integrate search functionality into automated workflows.

**Acceptance Criteria:**

1. <a name="5.1"></a>WHEN `rune find` returns no matches with `--format json`, the system SHALL output `{"success": true, "message": "No matching tasks found", "count": 0, "data": []}`

2. <a name="5.2"></a>WHEN `rune find` returns no matches with `--format markdown`, the system SHALL output `> No matching tasks found`

3. <a name="5.3"></a>All search result output paths SHALL respect the format flag

### 6. Mutation Command Format Compliance

**User Story:** As a developer automating task management, I want mutation commands to return structured confirmations, so that I can verify operations completed successfully without parsing plain text.

**Acceptance Criteria:**

1. <a name="6.1"></a>WHEN a mutation command (complete, uncomplete, progress, add, remove, update) succeeds with `--format json`, the system SHALL output a JSON response with `success: true` and relevant operation details

2. <a name="6.2"></a>WHEN a mutation command succeeds with `--format markdown`, the system SHALL output a markdown-formatted confirmation

3. <a name="6.3"></a>The `renumber` command with `--format json` SHALL output a JSON response including `task_count` and `backup_file` fields

4. <a name="6.4"></a>The `create` command with `--format json` SHALL output a JSON response including the created file path

5. <a name="6.5"></a>All mutation command output SHALL route through format-aware output functions

### 7. Renumber Command Format Compliance

**User Story:** As a developer, I want the `renumber` command to provide consistent output across formats, so that I can verify renumbering operations programmatically.

**Acceptance Criteria:**

1. <a name="7.1"></a>WHEN `rune renumber` completes with `--format json`, the system SHALL output a structured JSON response with `success`, `task_count`, and `backup_file` fields

2. <a name="7.2"></a>WHEN `rune renumber` completes with `--format markdown`, the system SHALL output markdown-formatted results

3. <a name="7.3"></a>The renumber command SHALL NOT mix output formats within a single invocation

### 8. Implementation Safeguards

**User Story:** As a maintainer, I want safeguards preventing format inconsistencies from being reintroduced, so that the codebase remains consistent over time.

**Acceptance Criteria:**

1. <a name="8.1"></a>All stdout output from commands SHALL route through format-aware output functions

2. <a name="8.2"></a>Direct `fmt.Print`, `fmt.Println`, or `fmt.Printf` to stdout SHALL NOT be used for command output in affected commands

3. <a name="8.3"></a>WHEN `--verbose` flag is set AND `--format json` is set, verbose messages SHALL be written to stderr to preserve JSON parseability on stdout

### 9. Excluded Commands

**User Story:** As a maintainer, I want clarity on which commands intentionally have limited format support, so that I can document exceptions appropriately.

**Acceptance Criteria:**

1. <a name="9.1"></a>The `has_phases` command SHALL remain JSON-only as it is designed for programmatic detection

2. <a name="9.2"></a>WHEN `has_phases` is called with a non-JSON format flag, the system SHALL ignore the flag and output JSON

3. <a name="9.3"></a>The `has_phases` command documentation SHALL clearly state it only supports JSON output

### 10. Error Handling Convention

**User Story:** As a developer, I want error messages to follow standard CLI conventions, so that I can distinguish errors from normal output.

**Acceptance Criteria:**

1. <a name="10.1"></a>Error messages SHALL be written to stderr as plain text regardless of the format flag

2. <a name="10.2"></a>The format flag SHALL only affect stdout content, not stderr

3. <a name="10.3"></a>Commands SHALL return non-zero exit codes on error, independent of output format
