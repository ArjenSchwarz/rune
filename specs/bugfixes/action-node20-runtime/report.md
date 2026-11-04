# Bugfix Report: Setup Action Still Declares the Node20 Runtime

**Date:** 2026-11-04
**Status:** Fixed
**Ticket:** T-1471

## Description of the Issue

`github-action/action.yml` declared `runs.using: 'node20'`. The file's own note said
Node20 reaches EOL in April 2026 and the action should move to `node24` in early 2026;
that date has passed. GitHub has deprecated the Node20 runtime for JavaScript actions
and is moving them to Node24.

**Reproduction steps:**

1. Open `github-action/action.yml`.
2. `runs.using` is `node20`.
3. Workflows that use the action can get Node20 deprecation warnings, or have the
   runner force it onto Node24 even though it was never tested there.

**Impact:** Low. Maintenance for compatibility. The action still installs rune.

## Investigation Summary

- **Code inspected:** `github-action/action.yml`, `package.json`, `tsconfig.json`,
  `RELEASING.md`, `dist/`, `.github/workflows/test.yml`.
- **Runtime compatibility:** `tsconfig.json` targets ES2021, and `@types/node` is
  already `^24`. Nothing in `src/` is specific to Node20.
- **Existing work:** No open PR on `main` makes this change. The only open PR is
  PR 108 (T-1934), which is unrelated.

## Discovered Root Cause

This migration was planned and never done. The runtime value in `action.yml` was never
changed from `node20`.

## Resolution for the Issue

- Set `runs.using` to `node24` and removed the outdated EOL note.
- Updated the current-runtime line in `github-action/RELEASING.md`.
- Ran a clean dependency install, the typecheck, the tests and the build under Node
  v24.22.2. The rebuilt `dist/` is byte-identical to the committed one, so no
  generated files changed.

## Regression Test

`github-action/__tests__/action.test.ts` checks that `action.yml` declares `node24`
and still points at `dist/index.js`. Before the fix it failed with
`Expected: "node24" / Received: "node20"`. It passes after the fix.

## Affected Files

- `github-action/action.yml`
- `github-action/RELEASING.md`
- `github-action/__tests__/action.test.ts`
- `CHANGELOG.md`

## Verification

Under Node v24.22.2:

- `npm ci`, `npm run lint` and `npm test` all pass: 2 suites, 36 tests.
- `npm run build` produces no diff in `dist/`.
- Running `node dist/index.js` with no inputs loads the bundle and exits with the
  expected "GitHub token is required" error.

This change does not touch any Go code. `make lint` passes. `make check` and
`make test-integration` each fail on the same two permission-based tests:
`TestCreateBackup/error_handling_for_read_failures` and
`TestIntegrationRenumber/renumber_write_failure`. These tests rely on chmod denying
access, which does not happen for root, and the cloud container runs as root
(uid 0). They fail the same way on an unmodified `origin/main`. All other Go tests
pass.

The install jobs in `.github/workflows/test.yml` run `./github-action` on Ubuntu,
macOS and Windows, so CI tests the new runtime on all three platforms.

## Prevention

The new metadata test fails if anyone changes the runtime value by accident.

## Related

`RELEASING.md` calls a Node runtime change a breaking change that needs a major
version bump. Choosing the release version is up to the maintainer and is not part
of this fix.
