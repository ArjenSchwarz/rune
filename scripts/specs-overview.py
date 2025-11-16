#!/usr/bin/env python3
"""
Specs Overview - Aggregate view of all specifications in a project.

Scans directories for spec folders containing tasks.md files and generates
an overview table with statistics and metadata.
"""

import argparse
import json
import os
import subprocess
import sys
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import List, Optional


@dataclass
class SpecInfo:
    """Information about a single specification."""

    name: str
    path: Path
    spec_type: str  # "full" or "smolspec"
    created: Optional[datetime]
    total: int
    pending: int
    in_progress: int
    completed: int

    @property
    def status(self) -> str:
        """Derive status from task counts."""
        if self.total == 0:
            return "Empty"
        elif self.completed == self.total:
            return "Complete"
        elif self.pending == self.total:
            return "Pending"
        else:
            return "In Progress"

    @property
    def status_symbol(self) -> str:
        """Get symbol for status."""
        if self.status == "Complete":
            return "✓"
        elif self.status == "In Progress":
            return "⚠"
        elif self.status == "Pending":
            return "○"
        else:
            return "−"

    @property
    def completion_pct(self) -> float:
        """Calculate completion percentage."""
        if self.total == 0:
            return 0.0
        return (self.completed / self.total) * 100

    @property
    def remaining(self) -> int:
        """Calculate remaining tasks."""
        return self.total - self.completed


def find_spec_directories(root_dirs: List[str]) -> List[Path]:
    """Find all directories containing tasks.md files."""
    spec_dirs = []

    for root in root_dirs:
        root_path = Path(root)
        if not root_path.exists():
            continue

        # Recursively find all tasks.md files
        for tasks_file in root_path.rglob("tasks.md"):
            spec_dir = tasks_file.parent
            spec_dirs.append(spec_dir)

    return sorted(spec_dirs)


def detect_spec_type(spec_dir: Path) -> str:
    """Detect whether this is a full spec or smolspec."""
    has_requirements = (spec_dir / "requirements.md").exists()
    has_design = (spec_dir / "design.md").exists()
    has_smolspec = (spec_dir / "smolspec.md").exists()

    if has_requirements and has_design:
        return "full"
    elif has_smolspec:
        return "smolspec"
    else:
        return "unknown"


def get_creation_date(spec_dir: Path) -> Optional[datetime]:
    """Get creation date from git history."""
    tasks_file = spec_dir / "tasks.md"

    try:
        result = subprocess.run(
            [
                "git",
                "log",
                "--follow",
                "--format=%aI",
                "--reverse",
                "--",
                str(tasks_file),
            ],
            cwd=spec_dir.parent,
            capture_output=True,
            text=True,
            check=False,
        )

        if result.returncode == 0 and result.stdout.strip():
            # Get first line (earliest commit)
            first_line = result.stdout.strip().split("\n")[0]
            return datetime.fromisoformat(first_line.replace("Z", "+00:00"))

    except Exception:
        pass

    return None


def get_task_stats(spec_dir: Path, rune_path: str) -> dict:
    """Get task statistics using rune list --output json."""
    tasks_file = spec_dir / "tasks.md"

    try:
        result = subprocess.run(
            [rune_path, "list", str(tasks_file), "--output", "json"],
            capture_output=True,
            text=True,
            check=False,
        )

        if result.returncode == 0:
            data = json.loads(result.stdout)
            return data.get("stats", {})

    except Exception as e:
        print(f"Warning: Failed to get stats for {spec_dir}: {e}", file=sys.stderr)

    return {"total": 0, "pending": 0, "in_progress": 0, "completed": 0}


def scan_specs(root_dirs: List[str], rune_path: str) -> List[SpecInfo]:
    """Scan all spec directories and gather information."""
    spec_dirs = find_spec_directories(root_dirs)
    specs = []

    for spec_dir in spec_dirs:
        # Get relative name from the first root dir that matches
        name = None
        for root in root_dirs:
            root_path = Path(root)
            if root_path.exists():
                try:
                    name = str(spec_dir.relative_to(root_path))
                    break
                except ValueError:
                    continue

        if name is None:
            name = spec_dir.name

        spec_type = detect_spec_type(spec_dir)
        created = get_creation_date(spec_dir)
        stats = get_task_stats(spec_dir, rune_path)

        spec = SpecInfo(
            name=name,
            path=spec_dir,
            spec_type=spec_type,
            created=created,
            total=stats.get("total", 0),
            pending=stats.get("pending", 0),
            in_progress=stats.get("in_progress", 0),
            completed=stats.get("completed", 0),
        )

        specs.append(spec)

    return specs


def sort_specs(specs: List[SpecInfo], sort_by: str, reverse: bool = False) -> List[SpecInfo]:
    """Sort specs by the specified field."""
    if sort_by == "name":
        return sorted(specs, key=lambda s: s.name, reverse=reverse)
    elif sort_by == "date":
        return sorted(
            specs,
            key=lambda s: s.created or datetime.min,
            reverse=reverse,
        )
    elif sort_by == "completion":
        return sorted(specs, key=lambda s: s.completion_pct, reverse=reverse)
    elif sort_by == "type":
        return sorted(specs, key=lambda s: s.spec_type, reverse=reverse)
    else:
        return specs


def output_table(specs: List[SpecInfo]):
    """Output specs as a formatted table."""
    if not specs:
        print("No specs found.")
        return

    # Calculate column widths
    max_name = max(len(s.name) for s in specs)
    max_name = max(max_name, len("Spec"))

    # Print header
    print(
        f"{'Spec':<{max_name}}  {'Type':<10}  {'Created':<12}  {'Status':<12}  "
        f"{'Total':>5}  {'Done':>5}  {'Remaining':>9}  {'%':>6}"
    )
    print("=" * (max_name + 80))

    # Print rows
    for spec in specs:
        created_str = spec.created.strftime("%Y-%m-%d") if spec.created else "unknown"
        status_str = f"{spec.status_symbol} {spec.status}"

        print(
            f"{spec.name:<{max_name}}  {spec.spec_type:<10}  {created_str:<12}  "
            f"{status_str:<12}  {spec.total:>5}  {spec.completed:>5}  "
            f"{spec.remaining:>9}  {spec.completion_pct:>5.1f}%"
        )


def output_markdown(specs: List[SpecInfo]):
    """Output specs as a markdown table."""
    if not specs:
        print("No specs found.")
        return

    # Print header
    print("| Spec | Type | Created | Status | Total | Done | Remaining | % |")
    print("|------|------|---------|--------|------:|-----:|----------:|--:|")

    # Print rows
    for spec in specs:
        created_str = spec.created.strftime("%Y-%m-%d") if spec.created else "unknown"
        status_str = f"{spec.status_symbol} {spec.status}"

        print(
            f"| {spec.name} | {spec.spec_type} | {created_str} | {status_str} | "
            f"{spec.total} | {spec.completed} | {spec.remaining} | {spec.completion_pct:.1f}% |"
        )


def output_json(specs: List[SpecInfo]):
    """Output specs as JSON."""
    data = []
    for spec in specs:
        data.append(
            {
                "name": spec.name,
                "path": str(spec.path),
                "type": spec.spec_type,
                "created": spec.created.isoformat() if spec.created else None,
                "status": spec.status,
                "stats": {
                    "total": spec.total,
                    "pending": spec.pending,
                    "in_progress": spec.in_progress,
                    "completed": spec.completed,
                    "remaining": spec.remaining,
                },
                "completion_pct": spec.completion_pct,
            }
        )

    print(json.dumps(data, indent=2))


def main():
    parser = argparse.ArgumentParser(
        description="Generate overview of project specifications"
    )
    parser.add_argument(
        "--dirs",
        nargs="+",
        default=["specs", ".kiro/specs"],
        help="Directories to scan for specs (default: specs .kiro/specs)",
    )
    parser.add_argument(
        "--rune",
        default="./rune",
        help="Path to rune binary (default: ./rune)",
    )
    parser.add_argument(
        "--format",
        choices=["table", "markdown", "json"],
        default="table",
        help="Output format (default: table)",
    )
    parser.add_argument(
        "--sort",
        choices=["name", "date", "completion", "type"],
        default="name",
        help="Sort by field (default: name)",
    )
    parser.add_argument(
        "--reverse",
        action="store_true",
        help="Reverse sort order",
    )

    args = parser.parse_args()

    # Scan specs
    specs = scan_specs(args.dirs, args.rune)

    # Sort specs
    specs = sort_specs(specs, args.sort, args.reverse)

    # Output in requested format
    if args.format == "table":
        output_table(specs)
    elif args.format == "markdown":
        output_markdown(specs)
    elif args.format == "json":
        output_json(specs)


if __name__ == "__main__":
    main()
