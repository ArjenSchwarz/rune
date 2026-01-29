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
        },
        "stream": {
          "type": "integer",
          "minimum": 1,
          "description": "Stream assignment for parallel execution (positive integer)"
        },
        "blocked_by": {
          "type": "array",
          "items": {
            "type": "string",
            "pattern": "^[1-9]\\d*(\\.[1-9]\\d*)*$"
          },
          "description": "Array of task IDs that must complete before this task (for add/update operations)"
        },
        "owner": {
          "type": "string",
          "description": "Agent identifier claiming the task (for add/update operations)"
        },
        "release": {
          "type": "boolean",
          "default": false,
          "description": "If true, clear the owner field (for update operations)"
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
        },
        "blockedBy": {
          "type": "array",
          "items": {
            "type": "string",
            "pattern": "^[1-9]\\d*(\\.[1-9]\\d*)*$"
          },
          "description": "Array of hierarchical task IDs that must complete before this task"
        },
        "stream": {
          "type": "integer",
          "minimum": 1,
          "description": "Stream assignment (1 if not explicitly set)"
        },
        "owner": {
          "type": "string",
          "description": "Agent identifier that owns this task"
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

## Streams Schema

### StreamsResult

Returned by `rune streams --json`:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "streams": {
      "type": "array",
      "items": {"$ref": "#/definitions/StreamStatus"},
      "description": "Status for each stream"
    },
    "available": {
      "type": "array",
      "items": {"type": "integer"},
      "description": "Stream IDs that have ready tasks"
    }
  },
  "definitions": {
    "StreamStatus": {
      "type": "object",
      "properties": {
        "id": {
          "type": "integer",
          "minimum": 1,
          "description": "Stream identifier"
        },
        "ready": {
          "type": "array",
          "items": {"type": "string"},
          "description": "Hierarchical IDs of tasks ready to start"
        },
        "blocked": {
          "type": "array",
          "items": {"type": "string"},
          "description": "Hierarchical IDs of tasks waiting on dependencies"
        },
        "active": {
          "type": "array",
          "items": {"type": "string"},
          "description": "Hierarchical IDs of tasks currently in progress"
        }
      },
      "required": ["id", "ready", "blocked", "active"]
    }
  }
}
```

### StreamsResult Example

```json
{
  "streams": [
    {
      "id": 1,
      "ready": ["1"],
      "blocked": ["2", "4"],
      "active": []
    },
    {
      "id": 2,
      "ready": [],
      "blocked": ["3"],
      "active": []
    }
  ],
  "available": [1]
}
```

## Claim Schema

### ClaimResult

Returned by `rune next --claim AGENT_ID --json`:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "success": {
      "type": "boolean",
      "description": "Whether the claim operation succeeded"
    },
    "count": {
      "type": "integer",
      "minimum": 0,
      "description": "Number of tasks claimed"
    },
    "stream": {
      "type": "integer",
      "minimum": 1,
      "description": "Stream ID if --stream flag was used"
    },
    "claimed": {
      "type": "array",
      "items": {"$ref": "#/definitions/ClaimedTask"},
      "description": "Tasks that were claimed"
    }
  },
  "required": ["success", "count", "claimed"],
  "definitions": {
    "ClaimedTask": {
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
          "type": "string",
          "enum": ["InProgress"],
          "description": "Always InProgress after claiming"
        },
        "stream": {
          "type": "integer",
          "minimum": 1
        },
        "owner": {
          "type": "string",
          "description": "The agent ID that claimed the task"
        },
        "blockedBy": {
          "type": "array",
          "items": {"type": "string"},
          "description": "Hierarchical IDs of blocking tasks (should be empty for claimed tasks)"
        }
      },
      "required": ["id", "title", "status", "stream", "owner"]
    }
  }
}
```

### ClaimResult Example

```json
{
  "success": true,
  "count": 2,
  "stream": 1,
  "claimed": [
    {
      "id": "1",
      "title": "Initialize project",
      "status": "InProgress",
      "stream": 1,
      "owner": "agent-1"
    },
    {
      "id": "3",
      "title": "Setup database",
      "status": "InProgress",
      "stream": 1,
      "owner": "agent-1",
      "blockedBy": []
    }
  ]
}
```

## Warning Schema

### Warning

Warnings are returned alongside successful operations when non-fatal issues are encountered:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "code": {
      "type": "string",
      "enum": [
        "invalid_stable_id",
        "missing_dependency",
        "duplicate_stable_id",
        "invalid_stream_value",
        "dependents_removed"
      ],
      "description": "Machine-readable warning code"
    },
    "message": {
      "type": "string",
      "description": "Human-readable warning message"
    },
    "taskId": {
      "type": "string",
      "pattern": "^[1-9]\\d*(\\.[1-9]\\d*)*$",
      "description": "Hierarchical ID of the affected task (if applicable)"
    }
  },
  "required": ["code", "message"]
}
```

### Response with Warnings Example

```json
{
  "success": true,
  "applied": 1,
  "errors": [],
  "warnings": [
    {
      "code": "dependents_removed",
      "message": "Removed dependency references from 2 task(s)",
      "taskId": "3"
    }
  ]
}
```

## Operation Examples with New Fields

### Add Task with Stream and Dependencies

```json
{
  "type": "add",
  "title": "Build API endpoints",
  "stream": 2,
  "blocked_by": ["1", "2"],
  "owner": "agent-backend"
}
```

### Update Task Dependencies

```json
{
  "type": "update",
  "id": "3",
  "blocked_by": ["1", "2"]
}
```

### Update Task Stream and Owner

```json
{
  "type": "update",
  "id": "4",
  "stream": 2,
  "owner": "agent-frontend"
}
```

### Release Task (Clear Owner)

```json
{
  "type": "update",
  "id": "4",
  "release": true
}
```

### Batch Setup for Parallel Agents

```json
{
  "file": "tasks.md",
  "operations": [
    {
      "type": "add",
      "title": "Initialize project",
      "stream": 1
    },
    {
      "type": "add",
      "title": "Backend development",
      "stream": 1,
      "blocked_by": ["1"]
    },
    {
      "type": "add",
      "title": "Frontend development",
      "stream": 2,
      "blocked_by": ["1"]
    },
    {
      "type": "add",
      "title": "Integration testing",
      "stream": 1,
      "blocked_by": ["2", "3"]
    }
  ]
}
```