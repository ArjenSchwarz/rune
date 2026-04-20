# Homebrew Install

## Overview

Enable installing rune via Homebrew by publishing a formula to a companion tap repo (`ArjenSchwarz/homebrew-rune`) and automating formula updates on every GitHub release. Pattern mirrors the sibling project `fog`/`homebrew-fog`, but replaces fog's manual formula bumps with a job appended to rune's existing `release.yml` that commits the updated formula automatically once the binary matrix finishes.

## Requirements

- The system MUST provide a Homebrew formula that installs the `rune` binary on darwin-arm64, darwin-amd64, linux-arm64, and linux-amd64 from the GitHub release tarballs produced by `release.yml`.
- The system MUST, on every `release: published` event, automatically commit an updated `Formula/rune.rb` in `ArjenSchwarz/homebrew-rune` with the new version and per-platform `sha256` values, only after all four tarball assets are available.
- The formula MUST pass `brew audit --strict --online Formula/rune.rb` and `brew test Formula/rune.rb` on a `macos-latest` runner before being pushed to the tap repo.
- The rune `README.md` MUST document `brew install arjenschwarz/rune/rune` as an install option alongside the existing `go install` instructions.
- The workflow MUST fail loudly (non-zero exit, visible in Actions UI) if any platform tarball is missing, a checksum cannot be computed, `brew audit` or `brew test` fails, or the push to `homebrew-rune` is rejected.
- The workflow MUST be idempotent: re-running for the same tag MUST NOT fail and MUST NOT create empty commits.
- The workflow MUST use a repository secret `HOMEBREW_TAP_TOKEN` — a fine-grained PAT with `contents:write` on `homebrew-rune` only — to push cross-repo.
- The workflow SHOULD expose a `workflow_dispatch` trigger with a `tag` input so it can be re-run manually for an existing release (for recovery or retesting).
- The workflow SHOULD use a `concurrency` group keyed on the repo to prevent two concurrent releases racing for the same tap commit.

## Implementation Approach

### Files in this repo (`rune`)

- **`.github/workflows/release.yml`** (modified):
  - **Existing `releases-matrix` job**: add `sha256sum: true` to the `wangyoucao577/go-release-action` step so each matrix leg publishes a `.sha256` sidecar (e.g. `rune-v1.3.0-darwin-amd64.tar.gz.sha256`) alongside the tarball. The action already supports this (confirmed via action README); default is `FALSE`. This avoids a re-download of tarballs downstream.
  - **New `homebrew` job** with `needs: releases-matrix`, `runs-on: macos-latest`. Steps:
    1. `actions/checkout@v4` (rune repo).
    2. Resolve tag: `TAG="${GITHUB_REF_NAME}"` (e.g. `v1.3.0`); `VERSION="${TAG#v}"` (e.g. `1.3.0`). Used in the formula's `version "..."` line; the literal `v` stays in the URL template so filenames match rune's actual asset names (confirmed: `rune-v1.3.0-darwin-amd64.tar.gz`).
    3. Fetch the four `.sha256` sidecars (~80 bytes each) — not the tarballs — with `gh release download "$TAG" -R "$GITHUB_REPOSITORY" -p 'rune-*-darwin-*.tar.gz.sha256' -p 'rune-*-linux-*.tar.gz.sha256'` (uses built-in `GITHUB_TOKEN`).
    4. Parse each sidecar: `awk '{print $1}' rune-v${VERSION}-<os>-<arch>.tar.gz.sha256` to extract the hex digest.
    5. Render `Formula/rune.rb` via an inline heredoc in the workflow step (no separate template file) — structure copied from `../homebrew-fog/Formula/fog.rb` with substitutions for `version`, URLs, and the 4 sha256 values.
    6. Validate: `brew audit --strict --online Formula/rune.rb` then `brew test Formula/rune.rb`. (`brew test` itself downloads and installs the macOS tarball — this is the only unavoidable tarball fetch in the pipeline and is genuine install validation, not redundant work.)
    7. Clone the tap repo using `actions/checkout@v4` with `repository: ArjenSchwarz/homebrew-rune`, `token: ${{ secrets.HOMEBREW_TAP_TOKEN }}`, `path: homebrew-rune`.
    8. Copy rendered formula into `homebrew-rune/Formula/rune.rb`; `cd homebrew-rune`; set `git config user.name "rune-release-bot"` and `user.email "rune-release-bot@users.noreply.github.com"`; `git add Formula/rune.rb`; `git diff --cached --quiet || git commit -m "rune ${TAG}"`; `git push origin main`.
    9. Job-level: `concurrency: group: homebrew-${{ github.repository }}, cancel-in-progress: false`.
- **`.github/workflows/release.yml`** also grows a `workflow_dispatch` trigger with input `tag` (string, required). When dispatched, the new job resolves the tag from the input instead of `GITHUB_REF_NAME`. The original binary-matrix job must be gated so it does not rebuild on manual dispatch — simplest: skip `releases-matrix` and `needs:` when `github.event_name == 'workflow_dispatch'`, allowing the `homebrew` job to run against the existing release assets for recovery.
- **`README.md`**: add a Homebrew subsection under `## Installation`, before the `go install` block:
  ```markdown
  ### Homebrew (macOS/Linux)

  ```bash
  brew install arjenschwarz/rune/rune
  ```
  ```

### Files in the tap repo (`ArjenSchwarz/homebrew-rune`, manual prerequisites)

- **`Formula/rune.rb`**: commit a placeholder formula — the automation will overwrite it. The user will cut a `v1.3.1` release immediately after the tap is created, which fires the new workflow and populates the real formula. No manual sha256 computation is required.

  Placeholder content (non-functional, only exists so the repo is a valid tap until the first automated commit):

  ```ruby
  class Rune < Formula
    desc "CLI for managing hierarchical markdown task lists"
    homepage "https://github.com/ArjenSchwarz/rune"
    version "0.0.0"
    url "https://github.com/ArjenSchwarz/rune/releases/download/v0.0.0/placeholder.tar.gz"
    sha256 "0000000000000000000000000000000000000000000000000000000000000000"

    def install
      bin.install "rune"
    end

    test do
      system "#{bin}/rune", "--version"
    end
  end
  ```

- **`README.md`**: one-liner explaining the tap and the install command (may also note that the formula is auto-updated from the rune release pipeline and who owns `HOMEBREW_TAP_TOKEN`).

### Existing patterns leveraged

- Release tarballs are already produced by `.github/workflows/release.yml` via `wangyoucao577/go-release-action` with `binary_name: rune` and `extra_files: "LICENSE README.md"`. Verified asset names: `rune-v1.3.0-darwin-amd64.tar.gz` and peers.
- The release action's built-in `sha256sum: true` parameter (confirmed in the action's README) publishes `.sha256` sidecars with zero extra scripting. The current release (v1.3.0) publishes only `.md5` sidecars; enabling `sha256sum: true` adds `.sha256` sidecars going forward and is what the new homebrew job consumes.
- Formula shape copied from `/Users/arjen/projects/personal/homebrew-fog/Formula/fog.rb`; the only structural differences are the binary name, the tap name (`arjenschwarz/rune/rune`), and the literal `v` in the URL template (fog tags without `v`, rune tags with `v`).
- `rune --version` is verified to exit 0 on the current codebase (prints `rune version dev` for unstamped builds, stamped `Version` otherwise via ldflags in `cmd/version.go` and `cmd/root.go`), so the formula's `test do` block works.

### Out of scope

- Homebrew core tap submission — this stays in a personal tap.
- Windows builds via Homebrew (Homebrew does not target Windows).
- Binary signing/notarisation — tracked separately under T-872 (Rune) and T-873 (Fog), deferred to a follow-up now that an Apple Developer account exists.
- Changes to how tarballs are built by `wangyoucao577/go-release-action`.
- Adding a `--version` CLI — already exists.

### Version handling (summary)

| Item | Value for v1.3.0 example |
| --- | --- |
| Git tag | `v1.3.0` |
| `$GITHUB_REF_NAME` | `v1.3.0` |
| Asset filename | `rune-v1.3.0-darwin-amd64.tar.gz` |
| Formula `version` line | `"1.3.0"` |
| Formula URL template | `https://github.com/ArjenSchwarz/rune/releases/download/v#{version}/rune-v#{version}-<os>-<arch>.tar.gz` |

## Risks and Assumptions

- **Risk:** `HOMEBREW_TAP_TOKEN` (fine-grained PAT) has a maximum lifetime of 1 year and will eventually expire, silently breaking releases. **Mitigation:** tap README documents required scope and owner; workflow fails loudly on `git push` auth error; owner sets a calendar reminder for rotation.
- **Risk:** `brew audit --strict --online` on `macos-latest` is slow (~2 minutes) and occasionally flaky (upstream Homebrew changes can introduce new warnings). **Mitigation:** acceptable cost for catching broken releases before users hit them; if audit flakiness becomes a problem, downgrade to `brew audit` without `--strict` in a follow-up.
- **Cost:** `macos-latest` minutes are free on public repositories (rune is public); the 10× macOS multiplier only applies to private repos. No billable cost from this job.
- **Assumption:** The owner will manually create the `ArjenSchwarz/homebrew-rune` repository with the placeholder formula above, add the `HOMEBREW_TAP_TOKEN` secret to the rune repo, and then cut `v1.3.1` to trigger the first automated update. Until this sequence completes, `brew install arjenschwarz/rune/rune` will fail.
- **Assumption:** Release tarball naming (`rune-v<version>-<os>-<arch>.tar.gz`) and binary-at-archive-root layout remain stable — verified against the current v1.3.0 release assets and the `wangyoucao577/go-release-action@v1` configuration in `release.yml`.
- **Prerequisite:** `HOMEBREW_TAP_TOKEN` secret must exist on the rune repo before the first release fires the `homebrew` job.
- **Prerequisite:** `ArjenSchwarz/homebrew-rune` must exist with `Formula/rune.rb` present (placeholder) so the `git push` has a valid target branch.
