# Requirements: Renumber Command

## Introduction

This feature adds a `renumber` command to rune that automatically fixes task numbering in a task file. This is particularly useful when tasks are manually reordered (e.g., moving a phase earlier in the project) and the hierarchical IDs need to be recalculated to maintain proper sequential order.

## Requirements

### 1. Renumber Command

**User Story:** As a developer, I want to renumber tasks in a file, so that hierarchical IDs remain sequential after manually reordering tasks.

**Acceptance Criteria:**

1. <a name="1.1"></a>The system SHALL provide a `renumber` command that accepts a file path as input
2. <a name="1.2"></a>The system SHALL read the task file and parse the task hierarchy
3. <a name="1.3"></a>The system SHALL recalculate all task IDs using hierarchical numbering (1, 1.1, 1.2, 2, 2.1...) sequentially across the entire file
4. <a name="1.4"></a>The system SHALL preserve the task hierarchy (parent-child relationships)
5. <a name="1.5"></a>The system SHALL update the file in-place with the corrected numbering
6. <a name="1.6"></a>The system SHALL maintain all task metadata (title, status, phase information, requirements links)
7. <a name="1.7"></a>The system SHALL preserve phase markers in their original positions in the file
8. <a name="1.8"></a>The system SHALL use WriteFileWithPhases when phase markers are detected
9. <a name="1.9"></a>The system SHALL preserve YAML front matter when present
10. <a name="1.10"></a>The system SHALL use atomic file operations (write to .tmp, then rename) to prevent file corruption

### 2. Error Handling

**User Story:** As a developer, I want clear error messages when renumbering fails, so that I can understand and fix issues with the task file.

**Acceptance Criteria:**

1. <a name="2.1"></a>The system SHALL validate that the input file exists before attempting to renumber
2. <a name="2.2"></a>The system SHALL validate that the file is a valid task file format before renumbering
3. <a name="2.3"></a>The system SHALL report parse errors with specific line numbers and descriptions
4. <a name="2.4"></a>The system SHALL validate file size does not exceed the 10MB limit
5. <a name="2.5"></a>The system SHALL validate the file path is within the working directory (security constraint)
6. <a name="2.6"></a>The system SHALL validate task count does not exceed 10000
7. <a name="2.7"></a>The system SHALL validate hierarchy depth does not exceed 10 levels
8. <a name="2.8"></a>The system SHALL leave the original file unmodified if any error occurs during processing
9. <a name="2.9"></a>The system SHALL clean up temporary files if the renumbering operation fails

### 3. Backup Management

**User Story:** As a developer, I want automatic backups when renumbering, so that I can recover if something goes wrong.

**Acceptance Criteria:**

1. <a name="3.1"></a>The system SHALL automatically create a backup file with .bak extension before writing changes
2. <a name="3.2"></a>The system SHALL preserve the original file's permissions in the backup
3. <a name="3.3"></a>The system SHALL inform the user about the backup file location in the output
4. <a name="3.4"></a>The system SHALL handle backup file collisions by overwriting existing .bak files
5. <a name="3.5"></a>The system SHALL create the backup before any modifications are written to disk

### 4. Gap Handling and Renumbering Logic

**User Story:** As a developer, I want automatic gap filling, so that task IDs are always sequential after renumbering.

**Acceptance Criteria:**

1. <a name="4.1"></a>The system SHALL renumber all tasks to create sequential IDs (e.g., 1, 2, 3 instead of 1, 2, 5)
2. <a name="4.2"></a>The system SHALL renumber phases, tasks, and all subtasks regardless of hierarchy depth
3. <a name="4.3"></a>The system SHALL maintain parent-child relationships when recalculating IDs
4. <a name="4.4"></a>The system SHALL preserve task order as it appears in the file

### 5. Output Format Support

**User Story:** As a developer, I want flexible output formats, so that I can integrate renumbering into different workflows.

**Acceptance Criteria:**

1. <a name="5.1"></a>The system SHALL support a --format flag with values: table, markdown, json
2. <a name="5.2"></a>The system SHALL default to table format when --format is not specified
3. <a name="5.3"></a>The system SHALL display the total number of tasks in the file
4. <a name="5.4"></a>The system SHALL display the backup file location
5. <a name="5.5"></a>The system SHALL indicate successful completion of the renumbering operation
6. <a name="5.6"></a>The system SHALL include these fields in JSON output: task_count, backup_file, success
7. <a name="5.7"></a>The system SHALL format table output consistently with other rune commands

### 6. Requirements Link Handling

**User Story:** As a developer, I want the renumber command to preserve requirement links without validation, so that renumbering is fast and straightforward.

**Acceptance Criteria:**

1. <a name="6.1"></a>The system SHALL preserve requirement links (e.g., [Req 1.1]) exactly as they appear in task descriptions
2. <a name="6.2"></a>The system SHALL NOT validate whether requirement links are valid or broken
3. <a name="6.3"></a>The system SHALL NOT modify requirement link references during renumbering

**Design Note:** Requirement links are preserved as-is to keep the command focused on structural renumbering. Updating links would require parsing task descriptions and understanding link semantics, adding significant complexity. This trade-off prioritizes performance and simplicity over automatic link maintenance. Users can manually fix broken cross-references after renumbering if needed.

### 7. Edge Case Handling

**User Story:** As a developer, I want predictable behavior for edge cases, so that the renumber command handles unusual files correctly.

**Acceptance Criteria:**

1. <a name="7.1"></a>The system SHALL successfully process empty files (no tasks) and report task_count=0
2. <a name="7.2"></a>The system SHALL successfully process files containing only phase markers
3. <a name="7.3"></a>The system SHALL report an error if the task hierarchy is malformed (e.g., task 1.1 exists but task 1 does not)
4. <a name="7.4"></a>The system SHALL report an error if duplicate task IDs are detected
5. <a name="7.5"></a>The system SHALL report an error if disk space is insufficient to write the file
