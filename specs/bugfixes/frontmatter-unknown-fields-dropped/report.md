# Bugfix Report: Front-Matter Unknown Fields Dropped

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1845

## Description of the Issue

Any mutating command (`complete`, `update`, `batch`, `add-frontmatter`, ...) rewrote the file's front matter with only the `references` and `metadata` keys. Every other top-level key was deleted without warning. When a file's front matter held only unsupported keys, the whole front-matter block was removed.

**Reproduction steps:**
1. Create `tasks.md`:
   ```markdown
   ---
   owner: arjen
   tags: [important]
   ---
   # T

   - [ ] 1. Task
   ```
2. Run `rune complete tasks.md 1 --format json`.
3. The command succeeds, but the file no longer has any front matter.

**Impact:** High. Silent data loss for any file whose front matter also holds data from other tools. This breaks the front-matter preservation contract in `specs/next-task-workflow/requirements.md` 3.9 and 3.10.

## Investigation Summary

- **Symptoms examined:** With only unknown keys, the front matter disappeared. With `references` as well, only `references` survived.
- **Code inspected:** `internal/task/references.go` (`FrontMatter`, `ParseFrontMatter`, `SerializeWithFrontMatter`), `internal/task/operations.go` (`WriteFile`, `WriteFileWithPhases`), `internal/task/batch.go` (`deepCopy`).
- **Hypotheses tested:** Phase-aware writes and batch writes go through different code paths. Both were confirmed to drop the fields, for the same reason.

## Discovered Root Cause

`FrontMatter` modelled only `references` and `metadata`. `yaml.Unmarshal` ignores keys that are not in the struct, so the other keys were never kept in memory. The write paths decided whether to emit front matter at all by checking `len(References) > 0 || len(Metadata) > 0`, so front matter holding only unknown keys was treated as empty.

**Defect type:** Data loss (lossy round trip)

**Why it occurred:** The struct was designed only for the fields rune itself uses. It did not account for files that other tools or users also annotate.

**Contributing factors:** The emptiness check was duplicated in three places (`SerializeWithFrontMatter`, `WriteFile`, `WriteFileWithPhases`), and the batch `deepCopy` copied only the known fields.

## Resolution for the Issue

**Changes made:**
- `internal/task/references.go` - Added `Extra []*yaml.Node` (`json:"-"`, so JSON output is unchanged), filled by a custom `UnmarshalYAML` with every top-level key/value pair other than `references`/`metadata`, in source order. A custom `MarshalYAML` writes the known keys, then those nodes. `ParseFrontMatter` does a trial marshal/unmarshal and rejects front matter that would not re-parse. Added `(*FrontMatter).IsEmpty()`, which also counts `Extra`.
- `internal/task/operations.go` - `WriteFile` and `WriteFileWithPhases` now use `IsEmpty()`.
- `internal/task/batch.go` - `deepCopy` also clones `Extra`. Batch operations never mutate the nodes, so a shallow clone is enough.

**Approach rationale:** Preserving the keys meets requirement 3.9 directly and leaves room for extensibility (3.10). Rejecting unknown keys would break existing files that are valid today.

**Alternatives considered:**
- Reject unknown keys at parse time - Not chosen: it turns valid files that other tools have annotated into errors.
- Write back the original front-matter text verbatim - Not chosen: `add-frontmatter` changes the modelled fields, so the YAML has to be regenerated anyway.

**Review follow-up (PR #111):** The first version stored unknown values in an inline `map[string]any`. Decoding into Go values dropped YAML tags, so `owner: !vault value` was written back as `owner: value`, changing its meaning for downstream tools. The inline map also re-sorted keys alphabetically, so an alias could be written before its anchor (`zz: &x ...` / `aa: *x`), producing unparsable YAML. Keeping the original nodes in source order fixes both.

**Known limitations:**
- The YAML is regenerated, not copied: unknown keys move after `references`/`metadata`, comments are lost, and indentation is normalised. Tags, flow style, anchors and aliases are kept.
- Front matter where an unknown key aliases an anchor defined on `references` or `metadata` is now rejected at parse time, for every command, because rune re-encodes those keys without their anchors. Previously the alias was resolved to a plain value.

## Regression Test

**Test file:** `internal/task/frontmatter_unknown_fields_test.go`
**Test names:** `TestWriteFilePreservesUnknownFrontMatterFields`, `TestSerializeWithFrontMatterOnlyUnknownFields`, `TestWriteFilePreservesTaggedAndAliasedFrontMatterValues`, `TestParseFrontMatterRejectsUnpreservableAlias`

**What it verifies:** Unknown keys survive parse → mutate → write → re-parse in four cases: only unknown keys, unknown keys alongside references, a file with phases (the `WriteFileWithPhases` path), and a batch operation (the `deepCopy` path).

**Also verifies (review follow-up):** Custom tags (scalar, nested, and in sequences), an explicit core tag, and an alias whose key sorts before its anchor all survive two consecutive writes unchanged. An unknown key aliasing a `references` anchor is rejected when parsed.

**Run command:** `go test ./internal/task -run 'UnknownFrontMatter|OnlyUnknownFields|TaggedAndAliased|UnpreservableAlias' -v`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/references.go` | `Extra` node list, custom (un)marshalling, write-back check, `IsEmpty()` helper |
| `internal/task/operations.go` | Use `IsEmpty()` in both write paths |
| `internal/task/batch.go` | Clone `Extra` in `deepCopy` |
| `internal/task/frontmatter_unknown_fields_test.go` | Regression tests |

## Verification

**Automated:**
- [x] Regression tests pass (all five subtests failed before the fix)
- [x] `internal/task` and `internal/config` suites pass
- [x] `make lint`: 0 issues
- [ ] `cmd` suite: `TestCreateBackup/error_handling_for_read_failures` and (with INTEGRATION=1) `TestIntegrationRenumber/renumber_write_failure` fail. Both also fail on unmodified main in this environment, because the tests rely on file permissions and the suite runs as root.

**Manual verification:**
- Ran the reproduction with the built binary: `owner` and `tags` are kept after `complete`, and after a later `add-frontmatter --meta k:v`.

## Prevention

**Recommendations to avoid similar bugs:**
- Use `FrontMatter.IsEmpty()` instead of checking individual fields when deciding whether to write front matter.
- When adding a field to `FrontMatter`, update `deepCopy` in `batch.go`.

## Related

- Transit T-1845
- `specs/next-task-workflow/requirements.md` 3.9, 3.10
