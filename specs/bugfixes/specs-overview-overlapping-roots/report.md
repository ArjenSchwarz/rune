# Bugfix Report: Specs Overview Duplicates Overlapping Roots

**Date:** 2026-11-04
**Status:** Fixed
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

**Changes made:**
- `scripts/specs-overview.py` `find_spec_directories` - skips a spec directory
  whose resolved path was already found. The first-seen path is kept, so output
  paths and names are unchanged. It takes an optional shared `seen` set.
- `scan_specs` passes an optional `seen` set through; `scan_multi_project`
  shares one set across every project it scans, so repeated or symlinked
  `--project-dirs` entries add no rows. Distinct projects resolve to distinct
  spec directories and are all kept.

**Approach rationale:** deduplicating at discovery stops duplicates before
`rune` and `git` run for them. Comparing resolved paths also catches equivalent
spellings (`specs`, `./specs`, `specs/../specs`, absolute paths, symlinks).
Review of the first version found that a set local to each `scan_specs` call
missed duplicates across `--project-dirs` entries, because each project is
scanned separately. Sharing the set covers both modes. Sorting is unchanged.

**Alternatives considered:**
- Normalising the roots and dropping nested ones - more code, and still needs a
  per-directory check for symlinked or equivalent roots.
- Deduplicating rows in `scan_specs` - runs `rune` and `git` for the duplicates
  first.
- Deduplicating resolved `--project-dirs` roots only - misses a project reached
  through a symlink inside another workspace.

## Regression Test

**Test file:** `scripts/test_specs_overview.py`
**Test class:** `TestOverlappingRoots`

**What it verifies:** nested roots and differently spelled roots yield each
spec directory once, `scan_specs` emits one row per spec, and
`scan_multi_project` with a repeated and a symlinked workspace keeps one row per
distinct project, with the first-seen path.

**Run command:** `python3 -m unittest discover -s scripts -p 'test_*.py'`

## Affected Files

| File | Change |
|------|--------|
| `scripts/specs-overview.py` | Deduplicate discovered spec directories by resolved path |
| `scripts/test_specs_overview.py` | `TestOverlappingRoots` regression tests |

## Verification

**Automated:**
- [x] Regression tests pass (`python3 -m unittest discover -s scripts -p 'test_*.py'`, 10 tests)
- [x] golangci-lint: 0 issues; flake8 reports no new findings. Ruff (global
  ruleset, no repo config) adds two UP045 hints for the new `Optional[set]`
  parameters, kept to match the file's existing `Optional` style.
- [ ] `make check` / `make test-integration`: Go suites fail only on
  `TestCreateBackup/error_handling_for_read_failures` and
  `TestIntegrationRenumber/renumber_write_failure`. Both fail identically on
  unmodified `main` (73e31eb) because the cloud container runs as root, which
  bypasses the permission denial those tests rely on. This change touches only
  Python.

**Manual verification:**
- The ticket repro (`--dirs specs specs/task-phases`) reports no duplicate
  paths; `--dirs specs ./specs` also has none.
- `--project-dirs <ws> <ws> <symlink-to-ws> --dirs specs` returns one row.
- `--dirs specs` JSON output is byte-identical before and after the fix.

## Related

- T-2011
