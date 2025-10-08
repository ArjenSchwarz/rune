# rune JSON API Schema

This document describes the JSON schema for the rune batch operations API.

## Batch Request Schema

### BatchRequest

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "file": {
      "type": "string",
      "description": "Path to the task file"
    },
    "requirements_file": {
      "type": "string",
      "default": "requirements.md",
      "description": "Path to requirements file for linking tasks to acceptance criteria"
    },
    "operations": {
      "type": "array",
      "items": {"$ref": "#/definitions/Operation"},
      "minItems": 1,
      "description": "Array of operations to execute"
    },
    "dry_run": {
      "type": "boolean",
      "default": false,
      "description": "If true, validate operations without applying changes"
    }
  },
  "required": ["file", "operations"],
  "definitions": {
    "Operation": {
      "type": "object",
      "properties": {
        "type": {
          "type": "string",
          "enum": ["add", "remove", "update"],
          "description": "Type of operation to perform"
        },
        "id": {
          "type": "string",
          "pattern": "^[1-9]\\d*(\\.[1-9]\\d*)*$",
          "description": "Task ID (required for remove, update operations)"
        },
        "parent": {
          "type": "string",
          "pattern": "^[1-9]\\d*(\\.[1-9]\\d*)*$",
          "description": "Parent task ID (optional for add operation)"
        },
        "phase": {
          "type": "string",
          "description": "Phase name (optional for add operation, creates phase if it doesn't exist)"
        },
        "title": {
          "type": "string",
          "maxLength": 500,
          "description": "Task title (required for add, optional for update)"
        },
        "status": {
          "type": "integer",
          "enum": [0, 1, 2],
          "description": "Task status: 0=Pending, 1=InProgress, 2=Completed"
        },
        "details": {
          "type": "array",
          "items": {
            "type": "string",
            "maxLength": 1000
          },
          "description": "Array of detail strings"
        },
        "references": {
          "type": "array",
          "items": {
            "type": "string",
            "maxLength": 500
          },
          "description": "Array of reference strings"
        },
        "requirements": {
          "type": "array",
          "items": {
            "type": "string",
            "pattern": "^[1-9]\\d*(\\.[1-9]\\d*)*$"
          },
          "description": "Array of requirement IDs (e.g., [\"1.1\", \"1.2\", \"2.3\"])"
        }
      },
      "required": ["type"],
      "allOf": [
        {
          "if": {"properties": {"type": {"const": "add"}}},
          "then": {"required": ["title"]}
        },
        {
          "if": {"properties": {"type": {"enum": ["remove", "update"]}},
          "then": {"required": ["id"]}
        },
      ]
    }
  }
}
```

## Batch Response Schema

### BatchResponse

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object", 
  "properties": {
    "success": {
      "type": "boolean",
      "description": "Whether all operations succeeded"
    },
    "applied": {
      "type": "integer",
      "minimum": 0,
      "description": "Number of operations successfully applied"
    },
    "errors": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Array of error messages (empty if successful)"
    },
    "preview": {
      "type": "string",
      "description": "Preview of resulting markdown (only in dry_run mode)"
    }
  },
  "required": ["success", "applied", "errors"]
}
```

## Search Results Schema

### SearchResult

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "query": {
      "type": "string",
      "description": "Original search query"
    },
    "matches": {
      "type": "array",
      "items": {"$ref": "#/definitions/TaskMatch"}
    },
    "total": {
      "type": "integer",
      "minimum": 0,
      "description": "Total number of matches found"
    }
  },
  "definitions": {
    "TaskMatch": {
      "type": "object", 
      "properties": {
        "id": {
          "type": "string",
          "pattern": "^[1-9]\\d*(\\.[1-9]\\d*)*$"
        },
        "title": {
          "type": "string"
        },
        "status": {
          "type": "integer",
          "enum": [0, 1, 2]
        },
        "details": {
          "type": "array",
          "items": {"type": "string"}
        },
        "references": {
          "type": "array",
          "items": {"type": "string"}
        },
        "requirements": {
          "type": "array",
          "items": {"type": "string"}
        },
        "path": {
          "type": "array",
          "items": {"type": "string"},
          "description": "Hierarchical path to this task"
        },
        "parent": {
          "type": "object",
          "properties": {
            "id": {"type": "string"},
            "title": {"type": "string"}
          },
          "description": "Parent task information (if applicable)"
        }
      },
      "required": ["id", "title", "status", "path"]
    }
  }
}
```

## Task List Schema

### TaskList (Full JSON export)

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "title": {
      "type": "string",
      "description": "Task list title"
    },
    "tasks": {
      "type": "array",
      "items": {"$ref": "#/definitions/Task"},
      "description": "Root level tasks"
    },
    "requirements_file": {
      "type": "string",
      "description": "Path to requirements file (if set)"
    },
    "file_path": {
      "type": "string",
      "description": "Source file path"
    },
    "modified": {
      "type": "string",
      "format": "date-time",
      "description": "Last modification timestamp"
    }
  },
  "definitions": {
    "Task": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "pattern": "^[1-9]\\d*(\\.[1-9]\\d*)*$"
        },
        "title": {
          "type": "string",
          "maxLength": 500
        },
        "status": {
          "type": "integer", 
          "enum": [0, 1, 2]
        },
        "details": {
          "type": "array",
          "items": {
            "type": "string",
            "maxLength": 1000
          }
        },
        "references": {
          "type": "array",
          "items": {
            "type": "string",
            "maxLength": 500
          }
        },
        "requirements": {
          "type": "array",
          "items": {
            "type": "string",
            "pattern": "^[1-9]\\d*(\\.[1-9]\\d*)*$"
          }
        },
        "children": {
          "type": "array",
          "items": {"$ref": "#/definitions/Task"}
        },
        "parent_id": {
          "type": "string",
          "pattern": "^[1-9]\\d*(\\.[1-9]\\d*)*$"
        }
      },
      "required": ["id", "title", "status", "details", "references", "children"]
    }
  }
}
```

## Operation Examples

### Add Operation

```json
{
  "type": "add",
  "title": "Implement user authentication",
  "details": [
    "Research authentication libraries",
    "Implement login/logout endpoints",
    "Add session management"
  ],
  "references": ["auth-spec.md", "security-requirements.md"],
  "requirements": ["1.1", "1.2", "2.3"]
}
```

### Add Subtask Operation

```json
{
  "type": "add",
  "parent": "1",
  "title": "Add OAuth integration",
  "details": ["Configure OAuth providers", "Implement OAuth flow"]
}
```

### Add Task to Phase Operation

```json
{
  "type": "add",
  "title": "Setup development environment",
  "phase": "Implementation",
  "details": ["Install dependencies", "Configure build tools"]
}
```

### Status Update via Update Operation

```json
{
  "type": "update",
  "id": "1.2",
  "status": 2
}
```

### Update Task Operation

```json
{
  "type": "update",
  "id": "2",
  "title": "Enhanced user management",
  "details": [
    "User profile management",
    "Role-based permissions",
    "Account deactivation"
  ],
  "references": ["user-management-spec.md"],
  "requirements": ["3.1", "3.2"]
}
```

### Remove Operation

```json
{
  "type": "remove",
  "id": "3.1"
}
```

### Complete Batch Request with Requirements

```json
{
  "file": "tasks.md",
  "requirements_file": "specs/requirements.md",
  "operations": [
    {
      "type": "add",
      "title": "Implement authentication system",
      "details": ["Setup OAuth providers", "Configure JWT tokens"],
      "requirements": ["1.1", "1.2"]
    },
    {
      "type": "update",
      "id": "2",
      "requirements": ["2.1", "2.2", "2.3"]
    }
  ]
}
```

**Field Descriptions:**
- `requirements_file` - Path to requirements file for all operations (default: "requirements.md")
- `requirements` - Array of requirement IDs that link to acceptance criteria in the requirements file

## Error Response Examples

### Validation Error

```json
{
  "success": false,
  "applied": 0,
  "errors": [
    "operation 1: add operation requires title",
    "operation 3: task 999 not found"
  ]
}
```

### Successful Batch Response

```json
{
  "success": true,
  "applied": 5,
  "errors": []
}
```

### Dry Run Response

```json
{
  "success": true,
  "applied": 3,
  "errors": [],
  "preview": "# Project Tasks\n\n- [ ] 1. Setup environment\n  - [ ] 1.1. Install dependencies\n- [-] 2. Implementation\n- [x] 3. Testing\n"
}
```