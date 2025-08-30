# Creating Task Lists with go-tasks

## Create Task File and Structure in One Command

Use batch operations to create a complete task hierarchy atomically:

### 1. Create the batch operations file

```bash
cat > create-tasks.json << 'EOF'
{
  "file": "tasks.md",
  "operations": [
    {
      "type": "add",
      "title": "Planning Phase"
    },
    {
      "type": "add",
      "title": "Implementation Phase"
    },
    {
      "type": "add",
      "title": "Testing Phase"
    },
    {
      "type": "add",
      "parent": "1",
      "title": "Requirements gathering"
    },
    {
      "type": "add",
      "parent": "1",
      "title": "Architecture design"
    },
    {
      "type": "add",
      "parent": "2",
      "title": "Core development"
    },
    {
      "type": "add",
      "parent": "2",
      "title": "Integration work"
    },
    {
      "type": "add",
      "parent": "3",
      "title": "Unit testing"
    },
    {
      "type": "add",
      "parent": "3",
      "title": "Integration testing"
    }
  ]
}
EOF
```

### 2. Create the task file and execute batch operations

```bash
# Create empty task file
go-tasks create "Project Tasks" --file tasks.md

# Execute batch creation
go-tasks batch tasks.md --operations create-tasks.json
```

## Result

This creates a structured task file:

```markdown
# Project Tasks

- [ ] 1. Planning Phase
  - [ ] 1.1. Requirements gathering
  - [ ] 1.2. Architecture design
- [ ] 2. Implementation Phase
  - [ ] 2.1. Core development
  - [ ] 2.2. Integration work
- [ ] 3. Testing Phase
  - [ ] 3.1. Unit testing
  - [ ] 3.2. Integration testing
```