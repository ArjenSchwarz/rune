# Bugfix Report: Specs Overview Reports Malformed Specs as Empty

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1884

## Description of the Issue

`scripts/specs-overview.py` reported a spec whose `tasks.md` rune could not parse as
a valid, empty spec. It printed no warning and exited 0, so malformed specs vanished
from progress reporting and automation accepted the overview as correct.

**Reproduction steps:**

1. Create `<dir>/bad/tasks.md` containing `- [?] 1. Invalid status`
2. `rune list <dir>/bad/tasks.md --format json` exits 1 with `invalid status: [?]`
3. `./scripts/specs-overview.py --dirs <dir> --rune ./rune --format json` exits 0,
   prints no warning, and reports `"status": "Empty"` with every count zero

**Impact:** Medium. Progress reports were wrong without any signal.

## Investigation Summary

- **Symptoms examined:** All-zero stats and `Empty` status for a file rune rejects.
- **Code inspected:** `get_task_stats`, `scan_specs`, `SpecInfo.status`, the output
  functions and `main` in `scripts/specs-overview.py`.
- **Hypotheses tested:** Whether rune exits 0 with an error payload — ruled out,
  rune exits 1 and writes the error plus usage text to stderr.

## Discovered Root Cause

`get_task_stats` only parsed stdout when `rune list` exited 0. Every non-zero exit
fell through to the default all-zero dict without a warning (the warning only fired on
Python exceptions). `SpecInfo.status` then derived `Empty` from `total == 0`, and
`main` had no failure path, so the script exited 0.

**Defect type:** Swallowed error / missing error state.

**Why it occurred:** The zero-count fallback served both "no tasks" and "could not
read tasks", so the two were indistinguishable downstream.

## Resolution for the Issue

**Changes made:**
- `scripts/specs-overview.py` `get_task_stats` - returns an `error` entry for a
  non-zero rune exit (first stderr line, which is rune's error message) or a Python
  exception (missing binary, bad JSON), and prints a warning to stderr in both cases.
- `SpecInfo` - new `error` field; `status` returns `Error` (symbol `✗`) when set.
- `scan_specs` - passes the error into `SpecInfo`.
- `output_json` - includes `error` for failed specs (field omitted otherwise, so the
  existing schema is unchanged for healthy specs).
- `main` - after printing the overview, exits 1 if any spec failed.

**Approach rationale:** The overview is still printed so healthy specs remain visible,
while the explicit `Error` state, stderr warning and non-zero exit make the failure
impossible to miss for humans and automation.

**Alternatives considered:**
- Abort on the first failing spec - hides the rest of the overview.
- Drop failed specs from output - same problem the ticket describes.

## Regression Test

**Test file:** `scripts/test_specs_overview.py`
**Test class:** `TestRuneFailureReported` (uses a fake rune that fails like the real one)

**What it verifies:** `get_task_stats` returns the rune error; `scan_specs` marks the
spec `Error` instead of `Empty`; `main` warns on stderr, emits `"status": "Error"` and
an `error` field, and exits non-zero; a missing rune binary is also an error.

**Run command:** `python3 -m unittest discover -s scripts -p 'test_*.py'`

## Affected Files

| File | Change |
|------|--------|
| `scripts/specs-overview.py` | Error state, warning, non-zero exit |
| `scripts/test_specs_overview.py` | Regression tests (new) |
| `scripts/README.md` | Document `Error` status and how to run tests |

## Verification

**Automated:**
- [x] Regression tests pass (failed before the fix)
- [x] No Go code changed. `make test` passes except `TestCreateBackup/error_handling_for_read_failures`,
  which fails in the root-run cloud container (a chmod-unreadable file is still readable by root);
  unrelated to this change
- [x] `ruff check scripts/test_specs_overview.py` passes (pre-existing ruff findings in
  `specs-overview.py` style rules were left alone)

**Manual verification:**
- Reran the ticket's reproduction with a real rune build: now exits 1, warns
  `invalid status: [?]`, and reports `"status": "Error"` with an `error` field.
- `./scripts/specs-overview.py` against this repo's own `specs/` still exits 0.

## Prevention

- Keep "no data" and "failed to get data" as separate states rather than sharing a
  default value.
- The script had no tests; `scripts/test_specs_overview.py` is a place to add them.
  It is not yet run by CI.

## Related

- T-1885 (date sort crash in the same script)
