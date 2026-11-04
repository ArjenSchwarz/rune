"""Regression tests for specs-overview.py.

Run with: python3 -m unittest discover -s scripts -p 'test_*.py'
"""

import contextlib
import importlib.util
import io
import stat
import sys
import tempfile
import unittest
from datetime import datetime
from pathlib import Path
from unittest import mock

_SCRIPT = Path(__file__).with_name("specs-overview.py")
_spec = importlib.util.spec_from_file_location("specs_overview", _SCRIPT)
specs_overview = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(specs_overview)


def make_spec(name, created, error=None):
    return specs_overview.SpecInfo(
        name=name,
        path=Path(name),
        spec_type="full",
        created=created,
        total=0,
        pending=0,
        in_progress=0,
        completed=0,
        error=error,
    )


class FakeRuneTest(unittest.TestCase):
    """Base class providing a temp spec dir and a fake rune binary."""

    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)
        spec_dir = self.root / "specs" / "broken"
        spec_dir.mkdir(parents=True)
        (spec_dir / "tasks.md").write_text("# T\n\n- [?] 1. Invalid status\n")
        self.spec_dir = spec_dir

        # Fake rune that fails the way `rune list` does on an invalid status.
        self.rune = self.root / "fake-rune"
        self.rune.write_text(
            "#!/bin/sh\n"
            "echo 'Error: failed to read task file: line 3: invalid status: [?]' >&2\n"
            "exit 1\n"
        )
        self.rune.chmod(self.rune.stat().st_mode | stat.S_IEXEC)

    def tearDown(self):
        self._tmp.cleanup()


class TestRuneFailureReported(FakeRuneTest):
    """T-1884: a failing `rune list` must not be reported as an empty spec."""

    def test_get_task_stats_reports_error(self):
        with contextlib.redirect_stderr(io.StringIO()):
            stats = specs_overview.get_task_stats(self.spec_dir, str(self.rune))
        self.assertIn("invalid status: [?]", stats.get("error") or "")

    def test_scan_marks_spec_as_error_not_empty(self):
        with contextlib.redirect_stderr(io.StringIO()):
            specs = specs_overview.scan_specs([str(self.root / "specs")], str(self.rune))
        self.assertEqual(len(specs), 1)
        self.assertEqual(specs[0].status, "Error")
        self.assertIn("invalid status", specs[0].error)

    def test_main_warns_and_exits_nonzero(self):
        argv = [
            "specs-overview.py",
            "--dirs", str(self.root / "specs"),
            "--rune", str(self.rune),
            "--format", "json",
        ]
        out, err = io.StringIO(), io.StringIO()
        with (
            mock.patch.object(sys, "argv", argv),
            contextlib.redirect_stdout(out),
            contextlib.redirect_stderr(err),
            self.assertRaises(SystemExit) as ctx,
        ):
            specs_overview.main()
        self.assertNotEqual(ctx.exception.code, 0)
        self.assertIn("invalid status", err.getvalue())
        self.assertIn('"status": "Error"', out.getvalue())
        self.assertIn('"error":', out.getvalue())

    def test_missing_rune_binary_is_error(self):
        missing = str(self.root / "does-not-exist")
        with contextlib.redirect_stderr(io.StringIO()):
            stats = specs_overview.get_task_stats(self.spec_dir, missing)
        self.assertTrue(stats.get("error"))


class TestSortByDate(unittest.TestCase):
    """T-1885: sorting by date must handle specs without a known date."""

    def test_mixed_known_and_unknown_dates(self):
        known = make_spec("known", datetime.fromisoformat("2026-01-01T00:00:00+00:00"))
        unknown = make_spec("unknown", None)
        result = specs_overview.sort_specs([known, unknown], "date")
        self.assertEqual([s.name for s in result], ["unknown", "known"])

    def test_known_dates_ordered_across_offsets(self):
        # 10:00+11:00 is 23:00 UTC the previous day, so it sorts before 00:00+00:00.
        later = make_spec("later", datetime.fromisoformat("2026-01-01T00:00:00+00:00"))
        earlier = make_spec("earlier", datetime.fromisoformat("2026-01-01T10:00:00+11:00"))
        unknown = make_spec("unknown", None)
        result = specs_overview.sort_specs([later, unknown, earlier], "date", reverse=True)
        self.assertEqual([s.name for s in result], ["later", "earlier", "unknown"])


if __name__ == "__main__":
    unittest.main()
