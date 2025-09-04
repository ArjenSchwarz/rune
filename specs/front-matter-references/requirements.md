# Front Matter References Feature Requirements

## Introduction

This feature adds the ability to add front matter content through CLI commands in the rune tool. Currently, users cannot add front matter using the command-line interface, requiring manual markdown file editing to establish metadata, references, and other YAML properties.

## Requirements

### 1. Add Front Matter via Create Command

**User Story:** As a user, I want to add front matter content when creating new task files, so that I can establish metadata and references from the start without additional steps.

**Acceptance Criteria:**
1.1. The system SHALL extend the create command to accept --reference flags (repeatable) for reference paths
1.2. The system SHALL extend the create command to accept --meta flags in "key:value" format (repeatable) for metadata
1.3. The system SHALL support adding multiple references and metadata entries in a single create command
1.4. The system SHALL create the front matter section at the beginning of the new file
1.5. The system SHALL generate valid YAML front matter format
1.6. The system SHALL add references as YAML array entries under the "references" key

### 2. Add Front Matter to Existing Files

**User Story:** As a user, I want to add front matter content to existing task files through a dedicated command, so that I can add metadata and references to files after creation.

**Acceptance Criteria:**
2.1. The system SHALL provide an add-frontmatter command to add front matter to existing task files
2.2. The system SHALL accept --reference flags (repeatable) for reference paths
2.3. The system SHALL accept --meta flags in "key:value" format (repeatable) for metadata
2.4. The system SHALL create front matter section if it doesn't exist in the target file
2.5. The system SHALL append new references to existing reference arrays
2.6. The system SHALL merge new metadata entries with existing metadata, appending to arrays where applicable
2.7. The system SHALL only operate on files managed by the rune tool
2.8. The system SHALL provide feedback on successful addition of front matter