# Bugfix Report: ParseFrontMatter Ignores CRLF Delimiters

**Date:** 2026-03-04
**Status:** Fixed

## Description of the Issue

`ParseFrontMatter` failed to detect front matter in files with Windows-style CRLF (`\r\n`) line endings. The function used hardcoded LF-only patterns (`"---\n"`, `"\n---\n"`) for detecting the opening and closing front matter delimiters, so CRLF content bypassed all front matter parsing.

**Reproduction steps:**
1. Create a task file with CRLF line endings containing a `---` front matter block
2. Parse it with `ParseMarkdown` or `ParseFrontMatter`
3. Observe that front matter is not detected -- the entire content (including the `---` lines) is treated as task content, causing parse errors

**Impact:** Any task file authored or edited on Windows (or transferred through systems that convert line endings) would fail to parse front matter. The front matter block would be misinterpreted as task content, producing parse errors like "unexpected content at this indentation level".

## Investigation Summary

- **Symptoms examined:** `ParseFrontMatter` returned the original content unchanged when given CRLF input, with no front matter extracted
- **Code inspected:** `internal/task/references.go` (ParseFrontMatter), `internal/task/parse.go` (parseContent, ParseFileWithPhases)
- **Hypotheses tested:** The root cause was immediately apparent from code inspection -- all delimiter patterns were LF-only

## Discovered Root Cause

`ParseFrontMatter` used four hardcoded LF-only string patterns:
1. `strings.HasPrefix(content, "---\n")` -- opening delimiter check
2. `searchStart := 4` -- assumes 4-byte opening delimiter
3. `strings.HasPrefix(content[searchStart:], "---\n")` -- empty front matter check
4. `endPattern := "\n---\n"` -- closing delimiter search

None of these match `\r\n` line endings, so CRLF content never enters the front matter parsing path.

**Defect type:** Missing input normalization

**Why it occurred:** The function was written assuming LF-only input. While `parseContent` (the downstream consumer) already handles CRLF by trimming `\r` from lines after splitting, `ParseFrontMatter` runs before that normalization step.

**Contributing factors:** The existing test suite only used Go raw string literals (backtick strings), which always produce LF-only content.

## Resolution for the Issue

**Changes made:**
- `internal/task/references.go:21` -- Added `strings.ReplaceAll(content, "\r\n", "\n")` at the start of `ParseFrontMatter`

**Approach rationale:** Normalizing CRLF to LF at the entry point of `ParseFrontMatter` is the simplest fix. It avoids modifying every pattern and offset calculation in the function. This is consistent with how `parseContent` already handles CRLF (trimming `\r` from split lines). After normalization, all existing LF-based logic works correctly.

**Alternatives considered:**
- Modifying each pattern to match both `\n` and `\r\n` -- rejected because it would require changing four separate locations and adjusting offset arithmetic, making the code harder to read and more error-prone
- Normalizing in `ParseMarkdown` instead -- rejected because `ParseFrontMatter` is a public function that can be called independently; the fix should be at the point of use

## Regression Test

**Test file:** `internal/task/references_test.go`
**Test names:** `TestParseFrontMatter/CRLF_front_matter_with_references`, `TestParseFrontMatter/CRLF_empty_front_matter`, `TestParseFrontMatter/CRLF_unclosed_front_matter`, `TestParseFrontMatter/CRLF_front_matter_with_metadata`

**Additional integration tests:** `internal/task/parse_frontmatter_test.go`
**Test names:** `TestParseMarkdownWithFrontMatter/CRLF_with_front_matter_and_references`, `TestParseMarkdownWithFrontMatter/CRLF_without_front_matter`, `TestParseMarkdownWithFrontMatter/CRLF_empty_front_matter`

**What it verifies:** CRLF line endings are handled correctly in all front matter scenarios: populated front matter, empty front matter, unclosed front matter (error case), and the full ParseMarkdown pipeline.

**Run command:** `go test -run "CRLF" -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/references.go` | Added CRLF-to-LF normalization at start of ParseFrontMatter |
| `internal/task/references_test.go` | Added 4 CRLF test cases to TestParseFrontMatter |
| `internal/task/parse_frontmatter_test.go` | Added 3 CRLF test cases to TestParseMarkdownWithFrontMatter |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed CRLF tests fail before the fix and pass after

## Prevention

**Recommendations to avoid similar bugs:**
- When writing string-matching code that processes user-provided text files, always consider CRLF line endings
- Normalize line endings early in the parsing pipeline, before any pattern matching
- Include CRLF test cases when testing file-parsing functions

## Related

- Transit ticket: T-265
