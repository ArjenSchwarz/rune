# Bugfix Report: Front Matter Comments Lost on Mutation

**Date:** 2026-11-04
**Status:** Fixed
**Ticket:** T-2079

## Description of the Issue

Any mutating command rewrote YAML front matter from decoded values, so comments were dropped, and quoted or folded scalars and metadata key order were normalised, even when only a task's status changed.

**Reproduction steps:**
1. Create a file whose front matter has a `# project note` head comment, a reference with a line comment, and `metadata` with `z: 'last'` before `a: >-` (folded).
2. Run `rune progress tasks.md 1`.
3. Observe that both comments are gone, `'last'` and the folded value are plain scalars, and `a` now sorts before `z`.

**Impact:** Low. Comments in hand-edited front matter were lost silently.

## Investigation Summary

- **Symptoms examined:** Rewritten front matter, which `yaml.Marshal` regenerates from `FrontMatter` values.
- **Code inspected:** `FrontMatter.UnmarshalYAML`/`MarshalYAML`, `ParseFrontMatter` and `SerializeWithFrontMatter` in `internal/task/references.go`; `TaskList.deepCopy` in `internal/task/batch.go`.
- **Hypotheses tested:** yaml.v3 keeps comments, scalar style and key order on the parsed `yaml.Node`, and marshalling an unmodified mapping node reproduces them. So the data was available at parse time and simply not kept.

## Discovered Root Cause

`UnmarshalYAML` decoded `references` and `metadata` into Go values and dropped their nodes. `MarshalYAML` encoded a new node from those values, which has no comments, uses default scalar styles, and sorts map keys. The batch deep copy also rebuilt `FrontMatter` field by field, so it would have dropped any extra state.

**Defect type:** Data loss on round trip.

## Resolution for the Issue

**Changes made:**
- `internal/task/references.go`: `FrontMatter` keeps the parsed top-level mapping node (`source`) and the values it decoded to (`parsed`). `MarshalYAML` writes from the source mapping and replaces the `references` or `metadata` value with a fresh encoding only when that value has changed. A modelled key that is newly added goes at the end, and one that is now empty is dropped. Unknown keys keep their source nodes as before. Comments that yaml.v3 attaches to the document (separated by a blank line) are moved onto the mapping at parse time so they are written back too.
- `ParseFrontMatter`'s check that unknown keys can be written back still validates the fresh encoding, so files rejected before (aliases of anchors on a modelled key) are still rejected.
- `internal/task/batch.go`: `deepCopy` uses the new `FrontMatter.clone()`, so batch writes and dry-run previews keep the source. The source node is never modified, so copies can share it.

**Approach rationale:** Smallest change that keeps meaningful comments without a YAML formatting rewrite. Merged behaviour (unknown-field preservation, alias rejection, 4-space indentation of regenerated YAML) is unchanged.

**Limitations:** When `references` or `metadata` itself changes (for example `add-frontmatter --meta`), that key's value is re-encoded, so comments inside that value are lost. Comments elsewhere remain. Blank lines between a detached comment and the first key are collapsed. Indentation of the whole block is still normalised to 4 spaces by the YAML encoder.

## Regression Test

**Test file:** `internal/task/frontmatter_comments_test.go`
**Test name:** `TestMutationsPreserveFrontMatterComments`

**What it verifies:** Head, line and unknown-key comments survive `WriteFile`, the phase-aware write, batch execution, batch dry-run previews, and adding metadata or a reference. Blank-line-separated document comments survive too, and the output parses back to the same values.

`TestFrontMatterPreservation` now expects `tags: "important,test"` exactly as written in its source, rather than the previously normalised unquoted form.

**Run command:** `go test ./internal/task -run TestMutationsPreserveFrontMatterComments`

## Verification

**Automated:**
- [x] Regression test passes (all 7 cases fail without the fix)
- [x] Full test suite passes, except two pre-existing permission tests (`TestCreateBackup`, `TestIntegrationRenumber` write-failure case) that fail only when run as root, identically on the base branch
- [x] `make lint`: 0 issues

**Manual verification:**
- `rune progress tasks.md 1` on the ticket's file keeps both comments, `'last'`, the folded scalar and the `z`/`a` order.
