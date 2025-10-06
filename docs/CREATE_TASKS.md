# Creating Task Lists with rune

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
rune create "Project Tasks" --file tasks.md

# Execute batch creation
rune batch tasks.md --operations create-tasks.json
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

## Creating Task Lists with Phases

Phases organize tasks into logical sections using H2 headers, making it easier to structure large projects.

### Option 1: Using add-phase Command

```bash
# Create task file
rune create "Project Tasks" --file tasks.md

# Add phase headers
rune add-phase tasks.md "Planning"
rune add-phase tasks.md "Implementation"
rune add-phase tasks.md "Testing"

# Add tasks to specific phases
rune add tasks.md --title "Define requirements" --phase "Planning"
rune add tasks.md --title "Create design documents" --phase "Planning"
rune add tasks.md --title "Setup development environment" --phase "Implementation"
rune add tasks.md --title "Implement core features" --phase "Implementation"
rune add tasks.md --title "Write unit tests" --phase "Testing"
```

### Option 2: Using Batch Operations with Phases

```bash
cat > create-phased-tasks.json << 'EOF'
{
  "file": "tasks.md",
  "operations": [
    {
      "type": "add",
      "title": "Define requirements",
      "phase": "Planning",
      "details": ["Gather stakeholder input", "Document requirements"]
    },
    {
      "type": "add",
      "title": "Create design documents",
      "phase": "Planning",
      "references": ["architecture.md"]
    },
    {
      "type": "add",
      "title": "Setup development environment",
      "phase": "Implementation",
      "details": ["Install dependencies", "Configure tools"]
    },
    {
      "type": "add",
      "title": "Implement core features",
      "phase": "Implementation"
    },
    {
      "type": "add",
      "title": "Write unit tests",
      "phase": "Testing",
      "details": ["Test coverage > 80%", "Integration tests"]
    }
  ]
}
EOF

# Create file and execute batch
rune create "Project Tasks" --file tasks.md
rune batch tasks.md --operations create-phased-tasks.json
```

### Result with Phases

This creates a phase-organized task file:

```markdown
# Project Tasks

## Planning

- [ ] 1. Define requirements
  - Gather stakeholder input
  - Document requirements
- [ ] 2. Create design documents
  - References: architecture.md

## Implementation

- [ ] 3. Setup development environment
  - Install dependencies
  - Configure tools
- [ ] 4. Implement core features

## Testing

- [ ] 5. Write unit tests
  - Test coverage > 80%
  - Integration tests
```

### Working with Phase-Based Tasks

```bash
# Get all tasks from the next incomplete phase
rune next tasks.md --phase

# Add more tasks to an existing phase
rune add tasks.md --title "Additional task" --phase "Implementation"

# View tasks with phase information
rune list tasks.md --format table
```