# Bugfix Report: Setup Action Checksum and Authorization

**Date:** 2026-11-02
**Status:** Fixed
**Tickets:** T-1143, T-2241

## Description of the Issue

Two defects in the Setup Rune GitHub Action's download path (`github-action/src/install.ts`, cache miss only):

1. **T-1143 – MD5 sidecar instead of SHA-256.** The action downloaded `${assetName}.md5` and verified it with `crypto.createHash('md5')`. The release workflow explicitly requests SHA-256 sidecars (`sha256sum: true`) and the Homebrew job consumes `.sha256`. The `.md5` sidecar only exists because `wangyoucao577/go-release-action` defaults `md5sum` to `true`; the workflow never asks for it.
2. **T-2241 – malformed Authorization header.** The raw token was passed as the third argument of `tc.downloadTool`. That argument is the whole `Authorization` header value (`@actions/tool-cache` assigns it unchanged to `headers.authorization`), so requests went out as `Authorization: ghs_...` with no scheme.

**Reproduction steps:**
1. Call `installRune("1.0.0", "ghs_example")` with `tc.find` returning `''` (cache miss).
2. Inspect the mocked `tc.downloadTool` calls.
3. Observe the third argument is `ghs_example` (no `Bearer`/`token` scheme) and the checksum URL ends in `.md5`.

**Impact:** Setup action installs on cache miss. For T-1143, the action depended on an upstream default that the release workflow doesn't control, and used a weaker hash than the one the project publishes deliberately. For T-2241, every authenticated download sent an invalid credential.

## Investigation Summary

- **Symptoms examined:** `installRune` download calls, `verifyChecksum` hashing, release workflow config, assets on the live v1.4.0 release.
- **Code inspected:** `github-action/src/install.ts`, `github-action/__tests__/install.test.ts`, `.github/workflows/release.yml`, `node_modules/@actions/tool-cache/lib/tool-cache.js` (`downloadTool`/`downloadToolAttempt`).
- **Hypotheses tested:**
  - *"No `.md5` sidecar exists, so installs fail today"* (T-1143's stated symptom): **not reproduced**. The v1.4.0 release publishes `.md5` (33 bytes), `.sha256` (65 bytes) and the archives. MD5 verification currently works only because of the go-release-action default.
  - *`.sha256` sidecar format*: confirmed against v1.4.0 `rune-v1.4.0-linux-amd64.tar.gz.sha256`. It is the bare 64-char hex digest plus a newline, and matches `sha256sum` of the archive.
  - *Raw-token header rejected*: an unauthenticated request with `Authorization: ghs_invalidexample` to a public release download URL still returned 302, so github.com doesn't visibly reject this for public assets. The contract defect is clear from the tool-cache source; the end-to-end rejection described in T-2241 couldn't be shown without a real token.

## Discovered Root Cause

**Defect type:** API contract misuse (T-2241); configuration drift between the release workflow and its consumer (T-1143).

**Why it occurred:** The action was written against go-release-action's default MD5 output. Later the release workflow added `sha256sum: true` for Homebrew, but the action wasn't updated. The tool-cache `auth` parameter name reads like "token", but it's documented as the "authorization header". Existing tests asserted the raw value, so the mocks locked in the misuse.

## Resolution for the Issue

**Changes made:**
- `github-action/src/install.ts` `verifyChecksum` – hash with SHA-256. Parse the first whitespace-separated field of the sidecar (accepts both a bare digest and `sha256sum` `<digest>  <file>` format), lower-cased. The error message now says `SHA-256 checksum verification failed!`.
- `github-action/src/install.ts` `installRune` – download `${assetName}.sha256`. If that download fails with `tc.HTTPError` status 404 (releases before v1.4.0 have no `.sha256`), download `${assetName}.md5` and verify with MD5 instead. Any other error, including 401/403/5xx, network errors and a SHA-256 mismatch, stays fatal. `verifyChecksum` takes the algorithm as a parameter (default `sha256`). Pass `Bearer ${token}` as the Authorization header to both `downloadTool` calls (or `undefined` when there's no token).
- `github-action/dist/index.js`, `dist/index.js.map` – rebuilt with `npm run build`.
- `github-action/README.md` – integrity section and troubleshooting message now say SHA-256 / `.sha256`.

**Legacy fallback (added after review):** `.sha256` sidecars exist only from v1.4.0. v1.0.0–v1.3.0 return 404 for `.sha256` but serve `.md5`, and `test.yml` pins `1.0.0`. Without a fallback, the SHA-256 switch broke those supported pins. The fallback triggers only on a confirmed-missing sidecar (404, which tool-cache doesn't retry), so a corrupt or tampered `.sha256`, auth failures and server errors can't downgrade verification to MD5.

**Approach rationale:** Each change follows its ticket directly. Keeping authentication (instead of omitting it) matches the ticket's preferred option and keeps the same behaviour if the repo or assets are ever private.

**Alternatives considered:**
- Omit auth for public downloads – simpler, but changes behaviour beyond the fix and would break private-asset access.
- `token ${token}` scheme – also valid. `Bearer` works for both `ghs_` installation tokens and PATs.

## Regression Test

**Test file:** `github-action/__tests__/install.test.ts`
**Tests:**
- `verifyChecksum › passes when SHA-256 checksums match`
- `verifyChecksum › accepts sha256sum format with a trailing filename`
- `verifyChecksum › rejects an MD5 digest even when it matches the archive`
- `verifyChecksum › throws error when checksums do not match`
- `installRune › sends a valid Authorization header for both downloads` (asserts the full value `Bearer ghs_example`)
- `installRune › checksum sidecar selection`: SHA-256 is preferred when present (no `.md5` request); 404 falls back to `.md5` with Bearer auth on all three downloads; a legacy MD5 mismatch is fatal; a SHA-256 mismatch is fatal with no fallback; 401/403/500/network errors on the `.sha256` download are fatal with no fallback. These use the real `HTTPError` class.
- The existing `installRune` cache-miss tests now assert the `.sha256` URL and the `Bearer test-token` header.

**Run command:** `cd github-action && npx jest`

## Affected Files

| File | Change |
|------|--------|
| `github-action/src/install.ts` | SHA-256 sidecar and hash; `Bearer` Authorization header |
| `github-action/__tests__/install.test.ts` | Regression tests; updated expectations |
| `github-action/dist/index.js`, `dist/index.js.map` | Rebuilt bundle |
| `github-action/README.md` | Docs: SHA-256 instead of MD5 |

## Verification

**Automated:**
- [x] Regression tests fail before the fix (9 failed / 17 passed) and pass after (26/26)
- [x] Fallback tests fail against the SHA-256-only version (2 failed / 32 passed) and pass after (34/34)
- [x] Full action test suite passes (`npx jest`)
- [x] Type check passes (`npm run lint`, i.e. `tsc --noEmit`)
- [x] Bundle rebuilt (`npm run build`)

**Manual verification:**
- Ran the compiled `verifyChecksum` against the real v1.4.0 `linux-amd64` archive and sidecars: the `.sha256` sidecar verifies, and the `.md5` sidecar is rejected under SHA-256.
- Against the live releases: `tc.downloadTool` on v1.0.0's `.sha256` throws `tc.HTTPError` with `httpStatusCode` 404. v1.0.0 `linux-amd64` verifies via its `.md5`, and v1.4.0 via its `.sha256`.

## Prevention

- Download-path tests should assert the exact HTTP contract values (full header, exact URL) and not just pass-through arguments.

## Related

- `.github/workflows/release.yml` (`sha256sum: true`, Homebrew job parsing `.sha256`)
