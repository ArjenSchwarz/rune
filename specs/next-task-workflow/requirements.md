# Next Task Workflow Requirements

## Introduction

The next-task-workflow feature enhances the go-tasks CLI tool to better support developer workflows by providing intelligent task retrieval and document reference capabilities. This feature enables users to automatically retrieve the next actionable task from their task list along with any relevant reference documentation, making it easier for developers and AI agents to work through tasks systematically.

## Requirements

### 1. Next Task Retrieval Command

**User Story:** As a developer, I want to retrieve the next incomplete task from my task list, so that I can focus on the most immediate work without manually searching through the entire list.

**Acceptance Criteria:**
1.1. The system SHALL provide a new command called "next" that retrieves the next incomplete task
1.2. The system SHALL define "next task" as the first task at any level in the hierarchy where either the task itself or any of its subtasks are not marked as completed
1.3. The system SHALL traverse the task hierarchy in depth-first order to find the first incomplete task
1.4. The system SHALL treat both "pending" `[ ]` and "in-progress" `[-]` states as incomplete
1.5. The system SHALL consider a task complete only when both the task itself AND all its subtasks are marked as completed `[x]`
1.6. The system SHALL return the found task and all its subtasks (regardless of their completion status) when an incomplete task is found
1.7. The system SHALL support an optional filename parameter for specifying the task file
1.8. The system SHALL return a message indicating all tasks are complete when no incomplete tasks are found
1.9. The system SHALL preserve the existing output format options (table, markdown, JSON)

### 2. Git Branch-Based File Discovery

**User Story:** As a developer, I want the tool to automatically find my task file based on the current git branch, so that I can organize tasks by feature branch without specifying file paths.

**Acceptance Criteria:**
2.1. The system SHALL support configurable path templates for branch-based file discovery
2.2. The system SHALL read the path template from a configuration file (e.g., `.go-tasks.yml` or `~/.config/go-tasks/config.yml`)
2.3. The system SHALL automatically detect the current git branch when no filename is provided and git discovery is enabled
2.4. The system SHALL treat branch names with slashes as path separators (e.g., `feature/auth` becomes `feature/auth/`)
2.5. The system SHALL construct the file path using the configured template with branch name substitution
2.6. The system SHALL error out with a clear message if git is not available or the current directory is not a git repository
2.7. The system SHALL error out with a clear message if the branch-based file path doesn't exist
2.8. The system SHALL prioritize explicit filename parameters over automatic branch-based discovery
2.9. The system SHALL handle special git states (detached HEAD, rebasing) by requiring explicit file specification

### 3. Task List Reference Documents

**User Story:** As a developer, I want to define reference documents for my task list, so that important context and documentation is automatically included when retrieving tasks.

**Acceptance Criteria:**
3.1. The system SHALL support defining reference documents using YAML front matter at the beginning of the task file
3.2. The system SHALL use a standardized YAML front matter format delimited by `---` markers
3.3. The system SHALL parse reference entries as a YAML array under a `references` key
3.4. The system SHALL return only the file paths (not content) of reference documents when retrieving tasks
3.5. The system SHALL always include reference document paths in all output formats (table, markdown, JSON)
3.6. The system SHALL support multiple reference documents per task list
3.7. The system SHALL accept both relative and absolute file paths with basic security validation (preventing path traversal)
3.8. The system SHALL validate reference paths for security but not check existence or accessibility
3.9. The system SHALL preserve the front matter when modifying the task file through other commands
3.10. The system SHALL support optional metadata fields in the front matter for future extensibility

### 4. Automatic Parent Task Completion

**User Story:** As a developer, I want parent tasks to be automatically marked as complete when all their subtasks are completed, so that I don't have to manually update parent tasks.

**Acceptance Criteria:**
4.1. The system SHALL automatically check parent task completion status when a subtask is marked as completed
4.2. The system SHALL mark a parent task as completed when ALL of its subtasks are marked as completed
4.3. The system SHALL apply this check recursively up the task hierarchy (grandparent, great-grandparent, etc.)
4.4. The system SHALL trigger this behavior when using the "complete" command
4.5. The system SHALL trigger this behavior when using the "batch" command with complete operations
4.6. The system SHALL NOT automatically mark parent tasks as incomplete when a subtask is marked as incomplete
4.7. The system SHALL log or indicate when parent tasks are auto-completed in command output

### 5. Integration with Existing Commands

**User Story:** As a developer, I want reference documents to be included in all task retrieval operations, so that I always have access to relevant context.

**Acceptance Criteria:**
5.1. The system SHALL include reference document paths when using the "list" command
5.2. The system SHALL include reference document paths when using the "show" command  
5.3. The system SHALL include reference document paths when using the new "next" command
5.4. The system SHALL include reference document paths in JSON output as an array of path strings
5.5. The system SHALL include reference paths in table output as an additional section
5.6. The system SHALL include reference paths in markdown output under a "References" heading
5.7. The system SHALL maintain backward compatibility by gracefully handling task files without reference sections
5.8. The system SHALL apply git branch-based file discovery to all commands that accept a filename parameter when no file is specified

### 6. Configuration Management

**User Story:** As a developer, I want to configure the tool's behavior through configuration files, so that I can customize it for my project's needs.

**Acceptance Criteria:**
6.1. The system SHALL support configuration files in YAML format
6.2. The system SHALL check for configuration in the following order: `./.go-tasks.yml`, then `~/.config/go-tasks/config.yml`
6.3. The system SHALL support a configuration schema that includes git discovery settings and path templates
6.4. The system SHALL provide sensible defaults when no configuration file exists
6.5. The system SHALL validate configuration file syntax and report errors clearly
6.6. The system SHALL allow disabling git discovery through configuration

## Example Usage

### Task File with References

```markdown
---
references:
  - ./docs/architecture.md
  - ./specs/api-specification.yaml
  - ../shared/database-schema.sql
metadata:
  project: backend-api
  created: 2024-01-30
---
# Project Tasks

- [ ] 1. Setup development environment
  - [x] 1.1. Install dependencies
  - [ ] 1.2. Configure database
- [x] 2. Implement authentication
- [ ] 3. Build API endpoints
  - [ ] 3.1. User endpoints
  - [ ] 3.2. Product endpoints
```

### Configuration File Example

```yaml
# .go-tasks.yml
discovery:
  enabled: true
  template: "specs/{branch}/tasks.md"
```

### Command Examples

```bash
# Get next incomplete task (uses git branch discovery if configured)
go-tasks next

# Get next task from specific file
go-tasks next -f project-tasks.md

# List all tasks with references
go-tasks list

# Output with references in JSON format
go-tasks next --output json
```