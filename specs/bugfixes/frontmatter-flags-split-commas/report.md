# Bugfix Report: Front-Matter Flags Split Comma-Containing Values

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1804

## Description of the Issue

The repeatable `--reference` and `--meta` flags on `create` and `add-frontmatter` split each value on commas, so a single value containing a comma could not be stored.

**Reproduction steps:**
1. `rune create c.md --title T --reference 'docs/a,b.md' --format json` → exits 0, reports `"references": 2`, and writes `docs/a` and `b.md` as two YAML entries.
2. `rune create m.md --title T --meta 'tags:alpha,beta'` → fails with `invalid metadata format: beta (expected key:value)`.
3. `rune add-frontmatter f.md --reference 'docs/a,b.md'` → reports `Added 2 reference(s)`.

**Impact:** Medium. Reference paths and metadata values with commas are silently corrupted or rejected.

## Investigation Summary

- **Symptoms examined:** The split happens before `runCreate`/`runAddFrontMatter` see the values.
- **Code inspected:** Flag registration in `cmd/create.go` and `cmd/add_frontmatter.go`; `task.ParseMetadataFlags` (splits on the first colon only, so it is not the cause).
- **Hypotheses tested:** Confirmed the values arrive pre-split from pflag's `StringSlice` type, which parses each occurrence as CSV.

## Discovered Root Cause

Both commands register the flags with `StringSliceVar`. pflag's `stringSlice` type reads each flag occurrence as a CSV record, so one occurrence becomes several values. The flags are documented as "can be used multiple times", which is what `StringArrayVar` provides without splitting.

**Defect type:** Wrong flag type.

**Why it occurred:** `StringSliceVar` is the commonly used repeatable-string flag in Cobra; its comma-splitting is easy to miss.

## Resolution for the Issue

**Changes made:**
- `cmd/create.go:50-51` - `--reference` and `--meta` registered with `StringArrayVar` instead of `StringSliceVar`.
- `cmd/add_frontmatter.go:52-53` - same change.

**Approach rationale:** `StringArrayVar` keeps one value per flag occurrence and never splits, which matches the documented "can be used multiple times" behaviour and the original design (`specs/front-matter-references/design.md`: "Single flag with commas stays as string"). Help output changes from `strings` to `stringArray`.

**Alternatives considered:**
- Re-joining or custom parsing of split values - cannot recover the original strings reliably (CSV quoting) and adds code for no benefit.

**Behaviour change:** `--reference a.md,b.md` previously produced two references and now produces one reference `a.md,b.md`. Multiple values must be passed as repeated flags, as documented. No repository docs or examples used the comma form.

## Regression Test

**Test file:** `cmd/frontmatter_flags_test.go`
**Test name:** `TestFrontMatterFlagsPreserveCommas`

**What it verifies:** Through real Cobra flag parsing (`rootCmd.Execute`), `create` and `add-frontmatter` store `--reference 'docs/a,b.md'` as one reference and `--meta 'tags:alpha,beta'` as the metadata value `alpha,beta`, and repeated `--reference` flags still produce multiple references.

**Run command:** `go test ./cmd -run TestFrontMatterFlagsPreserveCommas`

## Verification

**Automated:**
- [x] Regression test passes (`go test ./cmd -run TestFrontMatterFlagsPreserveCommas`)
- [x] Existing `TestCreateCommand*` and `TestAddFrontMatter*` tests pass
- [x] `make lint`: 0 issues

**Manual verification:**
- Rebuilt CLI: `create --reference 'docs/a,b.md' --format json` reports `"references": 1`; `create --meta 'tags:alpha,beta'` exits 0 and writes `tags: alpha,beta`; `add-frontmatter --reference 'docs/a,b.md'` reports `Added 1 reference(s)`.

## Prevention

- Use `StringArrayVar` for repeatable flags whose values are arbitrary strings; reserve `StringSliceVar` for genuinely comma-separated lists.
