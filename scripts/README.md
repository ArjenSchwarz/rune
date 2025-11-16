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

### Options

```
--dirs [DIR...]         Directories to scan for specs (default: specs .kiro/specs)
--rune PATH             Path to rune binary (default: ./rune)
--format {table,markdown,json}
                        Output format (default: table)
--sort {name,date,completion,type}
                        Sort by field (default: name)
--reverse               Reverse sort order
```

### Examples

#### Table Output (default)

```
Spec                             Type        Created       Status        Total   Done  Remaining      %
=====================================================================================================
initial-version                  full        2024-10-15    ✓ Complete      100    100          0  100.0%
task-requirements-linking        full        2024-11-01    ✓ Complete       27     27          0  100.0%
github-action                    full        2025-11-07    ⚠ In Progress    15     12          3   80.0%
batch-operations-simplification  full        2024-11-05    ○ Pending        20      0         20    0.0%
```

#### Markdown Output

```
| Spec | Type | Created | Status | Total | Done | Remaining | % |
|------|------|---------|--------|------:|-----:|----------:|--:|
| initial-version | full | 2024-10-15 | ✓ Complete | 100 | 100 | 0 | 100.0% |
| task-requirements-linking | full | 2024-11-01 | ✓ Complete | 27 | 27 | 0 | 100.0% |
| github-action | full | 2025-11-07 | ⚠ In Progress | 15 | 12 | 3 | 80.0% |
```

#### JSON Output

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
