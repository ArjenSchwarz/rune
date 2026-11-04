# Bugfix Report: Specs Overview Duplicates Overlapping Roots

**Date:** 2026-11-04
**Status:** Investigating
**Ticket:** T-2011

## Description of the Issue

`scripts/specs-overview.py` listed the same spec more than once when `--dirs`
contained overlapping roots: a root nested inside another, or one root written
two ways. Table, Markdown and JSON output all got duplicate rows, so anything
that totals the output counted those specs twice.

**Reproduction steps:**
1. `go build -o <tmp>/rune .`
2. `python3 scripts/specs-overview.py --dirs specs specs/task-phases --rune <tmp>/rune --format json | jq '[group_by(.path)[] | select(length > 1) | {path: .[0].path, occurrences: length}]'`
3. `specs/task-phases` is reported with `occurrences: 2`

`--dirs specs ./specs` duplicates every spec (16 duplicate groups on this
repository).

**Impact:** Low. Duplicate overview rows and inflated totals when roots overlap.

## Investigation Summary

- **Symptoms examined:** duplicate JSON rows that share a `path`.
- **Code inspected:** `find_spec_directories`, `scan_specs`, `scan_multi_project`.
- **Hypotheses tested:** none needed; the cause is direct.

## Discovered Root Cause

`find_spec_directories` runs `rglob("tasks.md")` once per root and appends
every parent directory it finds, with no deduplication. A directory under two
roots is appended twice, and `scan_specs` emits one row per entry.

**Defect type:** Missing deduplication.

**Why it occurred:** the default roots (`specs`, `.kiro/specs`) never overlap,
so the overlapping case was not considered.

**Contributing factors:** `pathlib.Path` equality is textual, so even an
equality check without `resolve()` would miss roots like `specs` and
`specs/../specs`.

## Resolution for the Issue

_Pending._

## Regression Test

**Test file:** `scripts/test_specs_overview.py`
**Test class:** `TestOverlappingRoots`

**What it verifies:** nested roots and differently spelled roots yield each
spec directory once, and `scan_specs` emits one row per spec.

**Run command:** `python3 -m unittest discover -s scripts -p 'test_*.py'`

## Related

- T-2011
