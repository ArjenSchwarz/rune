# Rune Scripts

This directory contains utility scripts for working with rune task files.

## specs-overview.py

Generate an overview of all specifications in your project.

### Purpose

The `specs-overview.py` script scans directories for specification folders and generates an aggregate view showing:
- Spec name and type (full spec vs smolspec)
- Creation date (from git history)
- Task statistics (total, completed, pending, in-progress)
- Completion percentage and status

### Spec Types

The script recognizes two types of specifications:

1. **Full Spec** - Contains:
   - `tasks.md`
   - `requirements.md`
   - `design.md`
   - Optionally: `decision_log.md`, `out-of-scope.md`

2. **Smolspec** - Contains:
   - `tasks.md`
   - `smolspec.md`

### Usage

#### Single Project Mode

```bash
# Basic usage (scans specs/ and .kiro/specs/ by default)
./scripts/specs-overview.py

# Scan custom directories
./scripts/specs-overview.py --dirs specs docs/specs

# Output as markdown table
./scripts/specs-overview.py --format markdown

# Output as JSON
./scripts/specs-overview.py --format json

# Sort by completion percentage
./scripts/specs-overview.py --sort completion --reverse

# Sort by creation date (newest first)
./scripts/specs-overview.py --sort date --reverse

# Use custom rune binary path
./scripts/specs-overview.py --rune /usr/local/bin/rune
```

#### Multi-Project Mode

Scan multiple projects in a workspace directory:

```bash
# Scan all projects in ~/workspace
# Each subdirectory is treated as a project
./scripts/specs-overview.py --project-dirs ~/workspace

# Scan multiple workspace directories
./scripts/specs-overview.py --project-dirs ~/workspace ~/other-projects

# Use custom spec directory names within each project
./scripts/specs-overview.py --project-dirs ~/workspace --dirs specs .specs

# Combine with other options
./scripts/specs-overview.py --project-dirs ~/workspace \
  --format markdown \
  --sort completion --reverse \
  --rune /usr/local/bin/rune
```

### Options

```
--dirs [DIR...]         Directories to scan for specs (default: specs .kiro/specs)
--project-dirs [DIR...] Directories containing multiple projects. Each subdirectory
                        will be treated as a separate project and scanned for --dirs
--rune PATH             Path to rune binary (default: ./rune)
--format {table,markdown,json}
                        Output format (default: table)
--sort {name,date,completion,type}
                        Sort by field (default: name)
--reverse               Reverse sort order
```

### Examples

#### Table Output (default)

**Single Project:**
```
Spec                             Type        Created       Status        Total   Done  Remaining      %
=====================================================================================================
initial-version                  full        2024-10-15    ✓ Complete      100    100          0  100.0%
task-requirements-linking        full        2024-11-01    ✓ Complete       27     27          0  100.0%
github-action                    full        2025-11-07    ⚠ In Progress    15     12          3   80.0%
batch-operations-simplification  full        2024-11-05    ○ Pending        20      0         20    0.0%
```

**Multi-Project:**
```
Project/Spec                     Type        Created       Status        Total   Done  Remaining      %
=====================================================================================================
rune/initial-version             full        2024-10-15    ✓ Complete      100    100          0  100.0%
rune/github-action               full        2025-11-07    ⚠ In Progress    15     12          3   80.0%
my-app/authentication            full        2024-09-20    ✓ Complete       42     42          0  100.0%
my-app/payment-integration       smolspec    2024-10-01    ⚠ In Progress    18     10          8   55.6%
another-project/api-redesign     full        2024-11-10    ○ Pending        30      0         30    0.0%
```

#### Markdown Output

**Single Project:**
```
| Spec | Type | Created | Status | Total | Done | Remaining | % |
|------|------|---------|--------|------:|-----:|----------:|--:|
| initial-version | full | 2024-10-15 | ✓ Complete | 100 | 100 | 0 | 100.0% |
| task-requirements-linking | full | 2024-11-01 | ✓ Complete | 27 | 27 | 0 | 100.0% |
| github-action | full | 2025-11-07 | ⚠ In Progress | 15 | 12 | 3 | 80.0% |
```

**Multi-Project:**
```
| Project | Spec | Type | Created | Status | Total | Done | Remaining | % |
|---------|------|------|---------|--------|------:|-----:|----------:|--:|
| rune | initial-version | full | 2024-10-15 | ✓ Complete | 100 | 100 | 0 | 100.0% |
| rune | github-action | full | 2025-11-07 | ⚠ In Progress | 15 | 12 | 3 | 80.0% |
| my-app | authentication | full | 2024-09-20 | ✓ Complete | 42 | 42 | 0 | 100.0% |
```

#### JSON Output

**Single Project:**
```json
[
  {
    "name": "initial-version",
    "path": "/path/to/specs/initial-version",
    "type": "full",
    "created": "2024-10-15T10:30:00+11:00",
    "status": "Complete",
    "stats": {
      "total": 100,
      "pending": 0,
      "in_progress": 0,
      "completed": 100,
      "remaining": 0
    },
    "completion_pct": 100.0
  }
]
```

**Multi-Project:**
```json
[
  {
    "name": "initial-version",
    "path": "/path/to/rune/specs/initial-version",
    "type": "full",
    "created": "2024-10-15T10:30:00+11:00",
    "status": "Complete",
    "stats": {
      "total": 100,
      "pending": 0,
      "in_progress": 0,
      "completed": 100,
      "remaining": 0
    },
    "completion_pct": 100.0,
    "project": "rune"
  },
  {
    "name": "authentication",
    "path": "/path/to/my-app/specs/authentication",
    "type": "full",
    "created": "2024-09-20T14:20:00+11:00",
    "status": "Complete",
    "stats": {
      "total": 42,
      "pending": 0,
      "in_progress": 0,
      "completed": 42,
      "remaining": 0
    },
    "completion_pct": 100.0,
    "project": "my-app"
  }
]
```

### Multi-Project Scanning

When using the `--project-dirs` flag, the script operates in multi-project mode:

1. **Discovery**: For each directory in `--project-dirs`, the script lists all immediate subdirectories
2. **Filtering**: Skips hidden directories (starting with `.`) and common non-project dirs (`node_modules`, `venv`, `__pycache__`)
3. **Scanning**: Within each project directory, looks for the spec directories specified by `--dirs`
4. **Labeling**: Tags each spec with its project name for easy identification

This allows you to maintain a workspace with multiple projects and get an aggregated view:

```
~/workspace/
  ├── rune/
  │   └── specs/
  │       ├── initial-version/tasks.md
  │       └── github-action/tasks.md
  ├── my-app/
  │   └── specs/
  │       ├── authentication/tasks.md
  │       └── payment-integration/tasks.md
  └── another-project/
      └── .kiro/specs/
          └── api-redesign/tasks.md
```

Running `./scripts/specs-overview.py --project-dirs ~/workspace` will find and aggregate all specs across all three projects.

### Requirements

- Python 3.7+
- `rune` binary (built and accessible)
- Git (for creation date extraction)

### How It Works

1. **Directory Scanning**: Recursively searches specified directories for `tasks.md` files
2. **Spec Detection**: Identifies spec type by checking for `requirements.md`, `design.md`, or `smolspec.md`
3. **Creation Date**: Extracts first commit date from git history using `git log --follow`
4. **Task Statistics**: Calls `rune list --output json` to get task counts and statistics
5. **Status Derivation**: Calculates overall status based on completion percentage:
   - **Complete** (✓): All tasks completed
   - **In Progress** (⚠): Some tasks completed, some pending
   - **Pending** (○): No tasks completed
   - **Empty** (−): No tasks in file

### Integration with Rune

This script leverages rune's JSON API to extract task statistics. The script depends on the enhanced JSON output that includes the `stats` object:

```json
{
  "title": "Spec Title",
  "tasks": [...],
  "stats": {
    "total": 27,
    "pending": 0,
    "in_progress": 0,
    "completed": 27
  }
}
```

This statistics feature was added to rune specifically to support this and similar aggregation tools.

### Extending the Script

To add custom directories to the default scan list, edit the script and modify the `--dirs` default:

```python
parser.add_argument(
    "--dirs",
    nargs="+",
    default=["specs", ".kiro/specs", "docs/architecture"],  # Add your dirs here
    help="Directories to scan for specs (default: specs .kiro/specs)",
)
```
