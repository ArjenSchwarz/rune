# Rune Command Patterns

Reference file with detailed examples. Load when the user needs specific batch operation examples or multi-agent workflow guidance.

## Task File Creation

### Basic Task File
```bash
rune create tasks.md --title "Project Name"
```

### Feature Task File with References
```bash
rune create specs/${feature_name}/tasks.md --title "Project Tasks" \
  --reference specs/${feature_name}/requirements.md \
  --reference specs/${feature_name}/design.md \
  --reference specs/${feature_name}/decision_log.md
```

## Batch Operations

### Adding Multiple Related Tasks
```bash
rune batch tasks.md --input '{
  "file": "tasks.md",
  "operations": [
    {"type": "add", "title": "Parent Task", "phase": "Phase Name"},
    {"type": "add", "title": "Subtask 1", "parent": "1"},
    {"type": "add", "title": "Subtask 2", "parent": "1"}
  ]
}'
```

### Marking Multiple Tasks Complete
```bash
rune batch tasks.md --input '{
  "file": "tasks.md",
  "operations": [
    {"type": "update", "id": "1.1", "status": 2},
    {"type": "update", "id": "1.2", "status": 2},
    {"type": "update", "id": "2.1", "status": 2}
  ]
}'
```

### Adding References and Requirements
```bash
rune batch tasks.md --input '{
  "file": "tasks.md",
  "operations": [
    {
      "type": "update",
      "id": "2.1",
      "references": ["docs/api-spec.md", "examples/usage.md"]
    },
    {
      "type": "add",
      "title": "Integration tests",
      "requirements": ["1.2", "1.3"]
    }
  ]
}'
```

### Creating Phases with Tasks Atomically
```bash
rune batch tasks.md --input '{
  "file": "tasks.md",
  "operations": [
    {"type": "add-phase", "name": "Implementation"},
    {"type": "add", "title": "Build core module", "phase": "Implementation"},
    {"type": "add", "title": "Add API endpoints", "phase": "Implementation"},
    {"type": "add-phase", "name": "Testing"},
    {"type": "add", "title": "Write unit tests", "phase": "Testing"},
    {"type": "add", "title": "Integration tests", "phase": "Testing"}
  ]
}'
```

### Setting Up Tasks with Dependencies and Streams
```bash
rune batch tasks.md --input '{
  "file": "tasks.md",
  "operations": [
    {"type": "add", "title": "Initialize project", "stream": 1},
    {"type": "add", "title": "Configure database", "stream": 1, "blocked_by": ["1"]},
    {"type": "add", "title": "Build API", "stream": 1, "blocked_by": ["2"]},
    {"type": "add", "title": "Build UI", "stream": 2, "blocked_by": ["1"]},
    {"type": "add", "title": "Write tests", "stream": 2, "blocked_by": ["3", "4"]}
  ]
}'
```

### Previewing Changes (Dry Run)
```bash
rune batch tasks.md --input '{
  "file": "tasks.md",
  "operations": [
    {"type": "add", "title": "New task"},
    {"type": "update", "id": "1", "status": 2}
  ],
  "dry_run": true
}'
```

## Multi-Agent Workflows

### Task Claiming
```bash
# Agent 1 claims all ready tasks in stream 1
rune next tasks.md --stream 1 --claim "agent-backend"

# Agent 2 claims all ready tasks in stream 2
rune next tasks.md --stream 2 --claim "agent-frontend"

# Check stream status
rune streams tasks.md

# Agent releases a task it can't complete
rune update tasks.md 3 --release
```

### Checking Available Work
```bash
# See which streams have ready tasks
rune streams tasks.md --available

# See all unowned pending tasks
rune list tasks.md --filter pending --owner ""

# Get JSON for programmatic processing
rune streams tasks.md --json
```

## Markdown Storage Format

Tasks with dependencies, streams, or owners are stored as:

```markdown
- [ ] 1. Initialize project <!-- id:abc1234 -->
  - Details about initialization
  - Stream: 1

- [ ] 2. Configure database <!-- id:def5678 -->
  - Blocked-by: abc1234 (Initialize project)
  - Stream: 1

- [-] 3. Build API <!-- id:ghi9012 -->
  - Blocked-by: def5678 (Configure database)
  - Stream: 1
  - Owner: agent-backend
```

- **Stable IDs**: HTML comments after title (system-managed)
- **Blocked-by, Stream, Owner**: List items under task (user-editable)
- **Title hints**: Dependency references include task titles for readability
