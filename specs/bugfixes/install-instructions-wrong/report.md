# Bugfix Report: install-instructions-wrong

**Date:** 2026-04-16
**Status:** Fixed
**Transit Ticket:** T-826

## Description of the Issue

The README and release notes contained incorrect `go install` commands using mixed-case `ArjenSchwarz/rune` instead of the lowercase `arjenschwarz/rune` that matches the Go module path declared in `go.mod`.

**Reproduction steps:**
1. Follow the install instructions: `go install github.com/ArjenSchwarz/rune@latest`
2. Go reports a module path mismatch because the module is registered as `github.com/arjenschwarz/rune`

**Impact:** Users cannot install rune using the documented `go install` command.

## Investigation Summary

- **Symptoms examined:** `go install` command in README uses `ArjenSchwarz/rune` (mixed case)
- **Code inspected:** `go.mod` (module path), `README.md`, and release notes
- **Hypotheses tested:** Confirmed Go module paths are case-sensitive; the module is registered as lowercase

## Discovered Root Cause

The `go install` commands in documentation used the GitHub username casing (`ArjenSchwarz`) instead of the Go module path casing (`arjenschwarz`). While GitHub URLs are case-insensitive, Go module paths are case-sensitive and must match `go.mod` exactly.

**Defect type:** Documentation error

**Why it occurred:** The GitHub username `ArjenSchwarz` has mixed case, which was used uniformly across all URLs and commands without distinguishing that `go install` requires the module path (lowercase).

**Contributing factors:** GitHub URLs work with either casing, masking the issue for non-Go-install references.

## Resolution for the Issue

**Changes made:**
- `README.md:27` - Changed `go install github.com/ArjenSchwarz/rune@latest` to `go install github.com/arjenschwarz/rune@latest`
- `docs/release_notes/RELEASE_NOTES_V1.1.0.md:92` - Fixed `go install` path to lowercase
- `docs/release_notes/RELEASE_NOTES_V1.2.0.md:121` - Fixed `go install` path to lowercase
- `docs/release_notes/RELEASE_NOTES_V1.3.0.md:51` - Fixed `go install` path to lowercase

**Approach rationale:** Only `go install` commands were changed since Go modules require exact case matching. GitHub URLs, GitHub Actions references, and other links correctly use the GitHub username casing (GitHub is case-insensitive for these).

**Alternatives considered:**
- Change all references to lowercase — not appropriate since GitHub Actions `uses:` references and URLs work correctly with the display casing

## Regression Test

Not applicable — this is a documentation-only fix with no testable code changes.

## Affected Files

| File | Change |
|------|--------|
| `README.md` | Fixed `go install` module path to lowercase |
| `docs/release_notes/RELEASE_NOTES_V1.1.0.md` | Fixed `go install` module path to lowercase |
| `docs/release_notes/RELEASE_NOTES_V1.2.0.md` | Fixed `go install` module path to lowercase |
| `docs/release_notes/RELEASE_NOTES_V1.3.0.md` | Fixed `go install` module path to lowercase |

## Verification

**Manual verification:**
- Confirmed `go.mod` declares module as `github.com/arjenschwarz/rune` (lowercase)
- Confirmed all `go install` commands now use matching lowercase path
- Confirmed GitHub URLs and Actions references retain correct mixed-case (these are case-insensitive)

## Prevention

**Recommendations to avoid similar bugs:**
- When adding `go install` commands, always copy the module path from `go.mod` rather than deriving it from the GitHub URL
- Consider adding a CI check that greps for `go install` commands and validates they match the module path

## Related

- Transit ticket: T-826
