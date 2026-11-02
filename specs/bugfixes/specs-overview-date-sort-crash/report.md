# Bugfix Report: Specs Overview Date Sort Crashes on Unknown Dates

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1885

## Description of the Issue

`./scripts/specs-overview.py --sort date` crashed with
`TypeError: can't compare offset-naive and offset-aware datetimes` whenever the
results mixed specs with a Git creation date and specs without one (untracked,
outside a Git repository, or without history).

**Reproduction steps:**

1. Create an untracked spec, e.g. `<tmp>/untracked/tasks.md`, outside the repo
2. `./scripts/specs-overview.py --dirs specs <tmp> --rune ./rune --sort date`
3. The script crashes with the `TypeError` above

Directly: `sort_specs([SpecInfo(created=datetime.fromisoformat("2026-01-01T00:00:00+00:00")), SpecInfo(created=None)], "date")`
raises the same error.

**Impact:** Medium. `--sort date` is unusable for any mixed set of specs.

## Investigation Summary

- **Symptoms examined:** `TypeError` from `sorted` in `sort_specs`.
- **Code inspected:** `get_creation_date`, `sort_specs`.
- **Hypotheses tested:** None needed; the mismatch is direct.

## Discovered Root Cause

`get_creation_date` parses Git `%aI` output, which always carries a UTC offset, so it
returns timezone-aware datetimes, or `None` when there is no history. `sort_specs`
replaced `None` with `datetime.min`, which is naive. Python refuses to order naive and
aware datetimes.

**Defect type:** Type mismatch in sort key.

## Resolution for the Issue

**Changes made:**
- `scripts/specs-overview.py` `sort_specs` - the date key is now
  `(created is not None, created.timestamp() or 0.0)`. Unknown dates sort first (as
  `datetime.min` intended), known dates compare as POSIX timestamps.

**Approach rationale:** The tuple keeps unknown dates apart from real ones, and
timestamps compare correctly across different UTC offsets without needing a naive or
aware placeholder.

**Alternatives considered:**
- Use an aware `datetime.min.replace(tzinfo=timezone.utc)` placeholder - works for
  Git dates, but still crashes if a naive date ever reaches the key.

## Regression Test

**Test file:** `scripts/test_specs_overview.py`
**Test class:** `TestSortByDate`

**What it verifies:** Sorting mixed known/unknown dates does not raise and puts
unknown first; known dates with different offsets order by actual instant, with
unknown last under `reverse=True`.

**Run command:** `python3 -m unittest discover -s scripts -p 'test_*.py'`

## Affected Files

| File | Change |
|------|--------|
| `scripts/specs-overview.py` | Date sort key |
| `scripts/test_specs_overview.py` | Regression tests (new) |
| `scripts/README.md` | Note where unknown dates sort |

## Verification

**Automated:**
- [x] Regression tests pass (raised `TypeError` before the fix)
- [x] No Go code changed. `make test` passes except `TestCreateBackup/error_handling_for_read_failures`,
  which fails in the root-run cloud container (a chmod-unreadable file is still readable by root);
  unrelated to this change
- [x] `ruff check scripts/test_specs_overview.py` passes (pre-existing ruff findings in
  `specs-overview.py` style rules were left alone)

**Manual verification:**
- `--dirs specs <tmp-with-untracked-spec> --sort date` crashes on main and now
  lists the untracked spec with `unknown` date.

## Prevention

- Avoid mixing naive sentinels with aware datetimes in sort keys.

## Related

- T-1884 (rune failures reported as empty specs, same script)
