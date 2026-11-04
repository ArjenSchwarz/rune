# Bugfix Report: Requirements Lines Parsed Too Loosely

**Date:** 2026-11-02
**Status:** Fixed
**Tickets:** T-1848, T-1907

## Description of the Issue

Two defects in how `Requirements:` lines are read:

- **T-1848:** a requirements file path containing a comma lost its requirement meaning. `rune add tasks.md --title Linked --requirements 1.1 --requirements-file 'specs/requirements,v2.md'` writes `Requirements: [1.1](specs/requirements,v2.md#1.1)`, but `rune list --format json` then reported no requirements and showed the line as a detail.
- **T-1907:** a line with extra text, e.g. `Requirements: prefix [1.1](requirements.md#1.1) suffix`, was accepted as requirements. The next write (even `rune progress`) rewrote it as `Requirements: [1.1](requirements.md#1.1)`, deleting the extra text. Requirement 4.4 of `specs/task-requirements-linking` says malformed requirement lines stay plain details.

**Impact:** Medium. Requirement links were lost (T-1848) or user text was silently deleted (T-1907).

## Investigation Summary

- **Code inspected:** `parseRequirements`, `requirementLinkPattern` and `applyDetailLine` in `internal/task/parse.go`; requirement rendering in `internal/task/render.go`.
- **Finding:** `parseRequirements` split the value on every comma, including commas inside a link destination, then ran an unanchored link regex on each part. Parts that did not match were skipped, and any embedded link counted as a match.

## Discovered Root Cause

**Defect type:** Parser accepted a superset of the format and could not represent the format's own valid output.

**Why it occurred:** Splitting on commas assumed paths never contain commas; the unanchored, skip-on-failure matching assumed any recognisable link meant a valid line.

## Resolution for the Issue

**Changes made:**
- `internal/task/parse.go` - `parseRequirements` now consumes the value link by link: an anchored link pattern, then either the end of the value or a comma separator. Anything else (extra text, a non-link fragment, an invalid ID, a trailing comma) makes the whole line malformed, so it is kept verbatim as a plain detail. The mixed-file check from T-1973 (PR #112) is unchanged and still only applies to well-formed lines.
- `internal/task/parse_frontmatter_test.go` - the `mixed_valid_invalid` case expected the old lenient behaviour (valid links kept, the `invalid` fragment dropped). It now expects the line to be treated as malformed, as T-1907 and requirement 4.4 require.

**Behaviour change:** A line mixing valid links with junk, which used to give partial requirements, is now a detail. Because `validateDetails` uses the same classifier, such text can now be entered as a detail.

## Regression Test

**Test file:** `internal/task/requirements_roundtrip_regression_test.go`
**Test names:** `TestRequirementsPathWithCommaRoundTrip`, `TestMalformedRequirementsLinesStayDetails`, `TestValidRequirementsLinesStillParse`

**Run command:** `go test ./internal/task -run 'Requirements'`
