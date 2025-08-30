# Next Task Workflow - Decision Log

## Design Decisions

### 1. Next Task Algorithm (2024-01-30)
**Decision**: The "next" command will find the first incomplete task at ANY level in the hierarchy, not just level-1 tasks.
**Rationale**: This provides more granular task management and ensures no incomplete work is missed regardless of nesting level.
**Implications**: The algorithm will need to traverse the entire task tree depth-first until finding the first incomplete task.

### 2. Task Completion Definition (2024-01-30)
**Decision**: A task is considered complete only when both the task itself AND all its subtasks are marked as completed.
**Rationale**: This ensures thorough task completion and prevents marking parent tasks as done when work remains in subtasks.
**Implications**: Users must explicitly mark both parent and child tasks as complete.

### 3. In-Progress Task Handling (2024-01-30)
**Decision**: Tasks marked as "in-progress" `[-]` are treated the same as "pending" `[ ]` tasks when finding the next task.
**Rationale**: In-progress tasks still represent incomplete work that needs attention.
**Implications**: The next task algorithm will consider both `[ ]` and `[-]` as incomplete states.

### 4. Git Branch Path Configuration (2024-01-30)
**Decision**: The path pattern for git branch discovery will be configurable via a configuration file.
**Rationale**: Different projects and teams may have different organizational structures for their task files.
**Implications**: Need to implement configuration file parsing and management.

### 5. Git Availability Handling (2024-01-30)
**Decision**: If git is not available or the current directory is not a git repository, the command will error out and require explicit file specification.
**Rationale**: This provides clear feedback and avoids ambiguous fallback behavior.
**Implications**: Users in non-git environments must always specify the file explicitly.

### 6. Branch Names with Slashes (2024-01-30)
**Decision**: Branch names containing slashes (e.g., `feature/auth/login`) will be treated as paths, creating nested directories.
**Rationale**: This maintains consistency with filesystem path conventions and allows natural organization of feature branches.
**Implications**: The pattern `specs/{branch}/tasks.md` with branch `feature/auth` becomes `specs/feature/auth/tasks.md`.

### 7. Reference Documents Storage (2024-01-30)
**Decision**: Reference documents will be stored in a special Markdown section at the bottom of the task file.
**Rationale**: This keeps references within the task file itself without requiring front matter parsing, maintaining pure Markdown compatibility.
**Implications**: Need to define a specific section format (e.g., `## References`) and parse it appropriately.

### 8. Reference Document Output (2024-01-30)
**Decision**: Reference documents will show only file paths, not content, and will always be included in command output.
**Rationale**: This keeps output concise while providing necessary context about available documentation.
**Implications**: All task retrieval commands will include reference paths in their output format.

### 9. Path Security and Validation (2024-01-30)
**Decision**: No validation or restriction on reference file paths - absolute paths, relative paths, and parent directory traversal are all allowed.
**Rationale**: The tool only stores and returns path strings without reading the files, eliminating security risks. Users are responsible for managing their own reference paths.
**Implications**: The tool will not validate path existence or accessibility, simply storing and returning the paths as provided.

## Open Questions

None at this time - all critical design decisions have been made.

## Additional Decisions

### 10. Automatic Parent Task Completion (2024-01-30)
**Decision**: When a subtask is marked as completed through the complete or batch commands, the system will automatically check if the parent task can be marked as completed (all subtasks done) and do so if applicable.
**Rationale**: This reduces manual work and ensures task hierarchy consistency, making the workflow more efficient for developers.
**Implications**: The completion logic needs to recursively check up the task hierarchy and update multiple tasks in a single operation.

### 11. Front Matter for References (2024-01-30)
**Decision**: Use YAML front matter instead of markdown References section for storing reference documents and metadata.
**Rationale**: Front matter provides structured, parseable metadata that's clearly separated from content. It's less fragile than string parsing, supports extensibility for future metadata needs, and is a well-established pattern in static site generators and documentation tools.
**Implications**: Task files will use `---` delimited YAML blocks at the beginning. No migration needed as this is a new feature.

## Future Considerations

- Consider adding a flag to optionally include reference file content in output
- Consider adding validation warnings (non-blocking) for non-existent reference paths
- Consider supporting glob patterns in reference paths for multiple file inclusion