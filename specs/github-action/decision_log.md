# Decision Log: GitHub Action Feature

## Decision 001: Feature Name
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Need to establish a feature name for organizing the specification documents.

**Decision:** Use "github-action" as the feature name, matching the current branch name.

**Rationale:** The branch is already named `specs/github-action`, and the feature is specifically about creating a GitHub Action for rune installation.

---

## Decision 002: Supported Architectures
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Need to determine which CPU architectures to support across different platforms.

**Decision:** Support amd64 and arm64 architectures for Linux, macOS, and Windows.

**Rationale:**
- amd64/x86_64 is the most common architecture for GitHub runners
- arm64 is increasingly important, especially for macOS M1/M2 machines
- 32-bit architectures (386/i686) are legacy and not commonly used in CI/CD environments

---

## Decision 003: Version Syntax Support
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Need to determine whether to support semantic version constraints or only exact versions.

**Decision:** Support only exact version specifications (e.g., "1.0.0" or "v1.0.0") and the keyword "latest".

**Rationale:**
- Simpler implementation with fewer edge cases
- Provides explicit version control
- Users know exactly which version will be installed
- Semantic version constraints (~>, >=, ^) add complexity that may not be needed initially
- Can be added in a future version if there's demand

---

## Decision 004: Action Outputs
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Need to determine what information the action should provide to subsequent workflow steps.

**Decision:** Provide two outputs:
1. Installed version (the actual version that was installed)
2. Installation path (the directory where rune was installed)

**Rationale:**
- Installed version is useful when "latest" is specified, allowing users to know exactly what version was installed
- Installation path enables users to reference the binary location if needed
- Cache hit status was considered but deemed less useful for typical workflows

---

## Decision 005: Configuration Options
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Need to determine additional configuration options beyond the basic version parameter.

**Decision:** Support two optional configuration parameters:
1. `github-token`: Custom GitHub token for API requests
2. `install-dir`: Custom installation directory

**Rationale:**
- Custom GitHub token helps avoid API rate limits (60/hour unauthenticated vs 5000/hour authenticated)
- Custom install directory provides flexibility for special deployment scenarios
- Skip checksum verification was rejected as it would compromise integrity checking
- These options match patterns from other setup actions like setup-terraform

---

## Decision 006: "Latest" Version Definition
**Date:** 2025-11-04
**Status:** Accepted

**Context:** The keyword "latest" was ambiguous - should it include pre-releases or only stable releases?

**Decision:** "latest" refers to the most recent stable release, excluding pre-releases.

**Rationale:**
- Most users expect "latest" to mean the latest stable version
- Pre-releases are typically for testing and shouldn't be the default
- Users can still specify exact pre-release versions if needed (e.g., "1.2.0-beta.1")
- This matches behavior of other GitHub Actions like actions/setup-node

---

## Decision 007: Archive Extraction Requirement
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Initial requirements assumed direct binary downloads, but rune releases are packaged in archives.

**Decision:** Add explicit requirement for extracting binaries from compressed archives (.tar.gz for Linux/macOS, .zip for Windows).

**Rationale:**
- Rune's release process (using go-release-action) creates archived releases
- This is standard practice for Go binaries distributed via GitHub releases
- The requirement was missing and would have caused implementation confusion

---

## Decision 008: Checksum Algorithm
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Initial requirements mentioned "checksums" without specifying the algorithm. Rune currently uses MD5.

**Decision:** Use MD5 checksums for integrity verification, not security verification. Rename the requirement section from "Security" to "Integrity Verification".

**Rationale:**
- Rune's current release process generates MD5 checksums
- MD5 is adequate for detecting corruption/incomplete downloads
- MD5 is not suitable for security (protecting against malicious tampering), so we've clarified this is for integrity only
- Changing to SHA256 would require modifying the release process, which is out of scope for this feature
- Future enhancement: consider adding SHA256 or signature verification for stronger security

---

## Decision 009: Cache Key Components
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Initial cache requirements didn't specify what should be included in the cache key beyond version.

**Decision:** Cache key must include version, operating system, and architecture.

**Rationale:**
- Including only version would cause incorrect cache hits (e.g., serving Linux binary on Windows)
- OS and architecture ensure platform-specific binaries are cached separately
- This prevents subtle bugs where the wrong binary is used from cache
- Follows best practices for caching platform-specific artifacts

---

## Decision 010: PATH Handling for Custom Install Directory
**Date:** 2025-11-04
**Status:** Accepted

**Context:** When users specify a custom `install-dir`, it was unclear if the action would still add it to PATH.

**Decision:** The action will add the installation directory to PATH regardless of whether it's the default or a custom directory.

**Rationale:**
- Users expect to use `rune` command immediately after the action runs
- If custom directory isn't added to PATH, the action would be incomplete
- This maintains consistency with the default installation behavior
- Users can always remove from PATH in a subsequent step if needed

---

## Decision 011: Use TypeScript with GitHub Actions Toolkit
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Need to select implementation language and libraries for the GitHub Action.

**Decision:** Implement the action in TypeScript using the GitHub Actions Toolkit packages:
- @actions/core - For inputs, outputs, and logging
- @actions/tool-cache - For downloading, extracting, and caching
- @actions/github - For GitHub API interactions
- @actions/exec - For executing shell commands

**Rationale:**
- TypeScript is the standard for GitHub Actions development
- Provides type safety and better developer experience
- GitHub Actions Toolkit provides battle-tested utilities that handle cross-platform concerns
- @actions/tool-cache specifically handles caching, downloading, and extraction with built-in support for different archive formats
- Matches the implementation approach of setup-terraform and other popular setup actions

---

## Decision 012: Use Tool Cache for All Installation Operations
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Need to determine how to handle downloading, extracting, and caching binaries.

**Decision:** Use @actions/tool-cache for all download, extraction, and caching operations rather than implementing custom logic.

**Rationale:**
- tool-cache provides platform-specific extraction (extractTar, extractZip) that handles cross-platform differences
- Built-in caching mechanism with proper cache key management
- Handles retries and error cases
- Reduces maintenance burden by using GitHub-supported libraries
- Proven reliability in production actions

---

## Decision 013: Default Installation Directory
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Need to determine the default installation location when install-dir is not specified.

**Decision:** Use the tool cache directory managed by @actions/tool-cache. Do not specify a custom default directory.

**Rationale:**
- tool-cache.cacheDir() manages its own directory structure optimized for GitHub Actions
- This directory is automatically cleaned up by the runner
- Provides consistent behavior with other setup actions
- Eliminates concerns about disk space management
- When custom install-dir is specified, the action will use that location and then call cacheDir to register it with the tool cache

---

## Decision 014: Normalize Version Format
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Rune releases use "v1.0.0" format, but users might specify "1.0.0" or "v1.0.0".

**Decision:** Accept both formats from users and normalize internally by stripping the "v" prefix for API calls and comparisons, but preserve it when building download URLs.

**Rationale:**
- Provides better user experience by accepting both formats
- GitHub releases use "v" prefix in tags (v1.0.0)
- Download URLs require the "v" prefix
- Internal version comparisons are easier without prefix
- Matches behavior of other setup actions like setup-node

---

## Decision 015: Error Handling with Custom Error Types
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Need a strategy for error handling that provides clear, actionable error messages.

**Decision:** Create a custom RuneInstallError class with error categories and structured error messages that include what went wrong, why, and how to fix it.

**Rationale:**
- Users need clear guidance when errors occur
- Categorizing errors helps with debugging and analytics
- Structured error messages improve user experience
- Allows for future error telemetry collection
- Follows best practices from other mature actions

---

## Decision 016: Retry Logic for Network Operations
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Network operations can fail temporarily due to transient issues.

**Decision:** Implement exponential backoff retry logic (3 attempts with 2s, 4s, 8s delays) for download operations.

**Rationale:**
- GitHub-hosted runners can experience transient network issues
- Exponential backoff prevents overwhelming servers
- Three retries provides good balance between reliability and speed
- Matches retry patterns in other robust GitHub Actions

---

## Decision 017: Phased Implementation Plan
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Need to sequence implementation to minimize risk and enable incremental testing.

**Decision:** Implement in five phases:
1. Core infrastructure (project setup, platform detection, version resolution)
2. Installation logic (download, checksum, extraction, caching)
3. Configuration options (GitHub token)
4. Error handling and polish (messages, retry logic, documentation)
5. Testing and release (unit tests, integration tests, marketplace)

**Rationale:**
- Each phase delivers working functionality
- Enables testing at each stage
- Reduces risk of large-bang integration
- Allows for early feedback on core functionality
- Matches agile development practices

---

## Decision 018: Remove Custom Install Directory Feature
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Design review revealed that custom install-dir is fundamentally incompatible with @actions/tool-cache, which manages its own directory structure.

**Decision:** Remove the `install-dir` input parameter entirely. The action will use only the tool-cache managed directory.

**Rationale:**
- @actions/tool-cache cannot "register" arbitrary custom directories
- Implementing custom directories would require bypassing caching entirely, negating the main performance benefit
- Copying from cache to custom directory adds unnecessary complexity
- setup-terraform and other popular actions don't provide custom install directories
- Simplifies the design and reduces potential bugs
- Users can still access the installation path via the `path` output if needed

**Impact on Requirements:** Removed requirements 7.3, 7.4, and 7.5 related to custom installation directory.

---

## Decision 019: Version Resolution Must Happen First
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Initial design showed version resolution happening after URL construction, which would create invalid URLs.

**Decision:** Resolve version string to exact version number as the very first operation, before any cache checks, URL construction, or downloads.

**Rationale:**
- Version "latest" must be resolved to actual version (e.g., "1.0.0") before use
- Cache lookup requires exact version number
- Download URLs require exact version number
- Having resolved version upfront simplifies all subsequent operations
- Prevents invalid URLs like "https://.../vlatest/..."

---

## Decision 020: Windows Binary Naming
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Windows executables typically have .exe extension, need to clarify binary naming across platforms.

**Decision:** Binary names are:
- Linux/macOS: `rune`
- Windows: `rune.exe`

**Rationale:**
- Matches Go build conventions
- Required for Windows PATH resolution
- Simplifies verification logic (different binary names per platform)
- Archives already contain the correctly named binaries

---

## Decision 021: Simplify Design - Remove Over-Engineering
**Date:** 2025-11-04
**Status:** Accepted

**Context:** Initial design had formal interfaces, custom error classes, 5 separate files, custom retry logic, and input validation functions. Code-simplifier agent review identified this as over-engineering for a ~350 line action.

**Decision:** Radically simplify the design:
- **Files**: 5 → 2 (main.ts + install.ts)
- **Interfaces**: Remove all 4 formal interfaces, use exported functions
- **Error handling**: Remove custom `RuneInstallError` class, use standard `Error`
- **Retry logic**: Remove custom implementation, trust @actions/tool-cache
- **Input validation**: Remove validation functions, let GitHub API validate
- **Verification**: Remove separate verification step, trust tool-cache

**Rationale:**
- No polymorphism needed - only one implementation per component
- GitHub Actions only displays error messages, categories add no value
- For ~350 lines, 5 files is excessive fragmentation
- @actions/tool-cache already handles retries internally
- Version validation happens naturally when fetching from GitHub API
- Reduces estimated implementation time from 15-20 hours to 7-11 hours
- Easier to understand, maintain, and debug
- Follows principle: "Simple is better than complex"

**Impact on Design:**
- Reduced line count from ~600 to ~350 lines
- Simplified component structure significantly
- Maintained all functional requirements
- Improved maintainability
