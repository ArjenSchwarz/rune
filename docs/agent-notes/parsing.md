# Parsing Module Notes

## File Structure

- `internal/task/parse.go` - Main parser: `ParseMarkdown`, `ParseFile`, `ParseFileWithPhases`, `parseContent`, line-level parsing
- `internal/task/references.go` - Front matter: `ParseFrontMatter`, `SerializeWithFrontMatter`, `FrontMatter` struct
- `internal/task/references_test.go` - Unit tests for `ParseFrontMatter` and `SerializeWithFrontMatter`
- `internal/task/parse_frontmatter_test.go` - Integration tests for front matter through `ParseMarkdown`
- `internal/task/parse_basic_test.go` - Basic parsing tests

## CRLF Handling

CRLF normalization happens at two levels:
1. `ParseFrontMatter` normalizes `\r\n` to `\n` at the start, before any delimiter matching
2. `parseContent` trims `\r` from individual lines after splitting on `\n` (line 124 of parse.go)

`ParseFileWithPhases` has its own front matter skipping logic (separate from `ParseFrontMatter`) that uses `strings.TrimSpace` for `---` delimiter checks, which naturally handles `\r`. But it relies on `ParseMarkdown` -> `parseContent` -> `ParseFrontMatter` for the actual parsing.

## Front Matter Delimiter Matching

`ParseFrontMatter` uses hardcoded LF patterns (`"---\n"`, `"\n---\n"`) with fixed byte offsets. The CRLF normalization at the top of the function ensures these patterns work regardless of the input's line ending style.

The closing delimiter matching also handles EOF without a trailing newline: `rest == "---"` for empty front matter at EOF, and `strings.HasSuffix(rest, "\n---")` for content front matter at EOF. In both cases, the remaining content is returned as an empty string.

## CI Notes

The `push.yml` workflow's linter step has pre-existing QF1012 (staticcheck) failures across `cmd/list.go`, `cmd/next.go`, and `internal/task/render.go` for `WriteString(fmt.Sprintf(...))` patterns that should use `fmt.Fprintf(...)`. This affects all branches, not just specific PRs.
