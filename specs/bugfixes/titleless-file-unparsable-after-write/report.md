# Bugfix Report: Titleless File Unparsable After Write

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-2062

## Description of the Issue

`ParseMarkdown` accepts task files that have no H1 title. However, `RenderMarkdown` and `RenderMarkdownWithPhases` wrote a bare `# ` line when the title was empty. The parser deliberately rejects that line (see the `bare_hash_space_not_consumed_as_title` case in `parse_basic_test.go`), so the first successful write made a valid file unparsable.

**Reproduction steps:**
1. Create `t.md` containing `- [ ] 1. Untitled task`.
2. Run `rune progress t.md 1`: it exits 0 and writes `# \n\n- [-] 1. Untitled task`.
3. Run `rune list t.md`: it exits 1 with `line 1: unexpected content at this indentation level`.

**Impact:** High. Any write path (add, update, status changes, remove, claim, batch, add-frontmatter, with or without phases) could make a titleless file unusable.

## Discovered Root Cause

Both renderers had an `else` branch that wrote `"# \n\n"` as a placeholder heading. That output is outside the format the parser accepts.

**Defect type:** Renderer/parser contract mismatch

## Resolution for the Issue

**Changes made:**
- `internal/task/render.go`: both renderers omit the H1 when `Title` is empty.
- `cmd/create_test.go`: the `empty title` case expected the old `# ` output. It now expects an empty file.

**Approach rationale:** The parser already rejects a bare `# ` on purpose (it has a dedicated test), so the renderer is the side to fix.

**Alternatives considered:**
- Make the parser accept a bare `#`/`# ` as an empty title: Not chosen. It changes a deliberate parser contract, and the renderer fix alone stops the corruption.

**Limitation:** Files that this bug has already rewritten still start with `# ` and still fail to parse. Deleting that first line repairs them.

## Regression Test

**Test file:** `internal/task/roundtrip_regression_test.go`
**Test names:** `TestTitlelessFileRoundTrip` (parse → mutate → `WriteFile` → re-parse, without and with phases), `TestRenderEmptyTitleOmitsHeading` (both renderers directly)

**Run command:** `go test ./internal/task -run 'TitlelessFileRoundTrip|RenderEmptyTitleOmitsHeading' -v`

## Verification

- [x] Regression tests failed before the fix (re-parse error `line 1: unexpected content at this indentation level`) and pass after it.
- [x] `go test ./internal/...` passes. `make lint` reports 0 issues.
- [x] Manual check: `rune progress t.md 1` writes `- [-] 1. Untitled task`, and a following `rune list` exits 0.

## Related

- T-1500 (validating empty titles for `create`)
