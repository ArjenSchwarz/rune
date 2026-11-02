# Bugfix Report: Front-Matter Flags Split Comma-Containing Values

**Date:** 2026-11-02
**Status:** Investigating
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

_To be filled in after the fix._

## Regression Test

**Test file:** `cmd/frontmatter_flags_test.go`
**Test name:** `TestFrontMatterFlagsPreserveCommas`

**What it verifies:** Through real Cobra flag parsing (`rootCmd.Execute`), `create` and `add-frontmatter` store `--reference 'docs/a,b.md'` as one reference and `--meta 'tags:alpha,beta'` as the metadata value `alpha,beta`, and repeated `--reference` flags still produce multiple references.

**Run command:** `go test ./cmd -run TestFrontMatterFlagsPreserveCommas`
