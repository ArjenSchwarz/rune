# GitHub Action for Rune Installation

## Introduction

This feature provides a GitHub Action that installs the rune binary into the GitHub Actions runner environment, similar to how the HashiCorp setup-terraform action works. The action simplifies the process of using rune in CI/CD workflows by automatically downloading and installing the appropriate binary version.

## Requirements

### 1. Binary Installation

**User Story:** As a GitHub Actions user, I want to install rune in my workflow, so that I can use it for task management in my CI/CD pipeline.

**Acceptance Criteria:**

1. <a name="1.1"></a>The action SHALL download the rune release archive for the runner's operating system and architecture
2. <a name="1.2"></a>The action SHALL extract the rune binary from compressed archives (.tar.gz for Linux/macOS, .zip for Windows)
3. <a name="1.3"></a>The action SHALL make the rune binary executable
4. <a name="1.4"></a>The action SHALL add the rune binary directory to the system PATH, whether using the default or custom installation directory
5. <a name="1.5"></a>The action SHALL verify successful installation by checking the binary is accessible via PATH
6. <a name="1.6"></a>The action SHALL fail gracefully with clear error messages if installation fails

### 2. Version Selection

**User Story:** As a GitHub Actions user, I want to specify which version of rune to install, so that I can ensure consistency and compatibility with my workflows.

**Acceptance Criteria:**

1. <a name="2.1"></a>The action SHALL accept an optional `version` input parameter
2. <a name="2.2"></a>The action SHALL default to `latest` when no version is specified, where `latest` refers to the most recent stable release excluding pre-releases
3. <a name="2.3"></a>The action SHALL support exact version specifications (e.g., "1.0.0" or "v1.0.0")
4. <a name="2.4"></a>The action SHALL only support exact versions and the keyword `latest` (no semantic version constraints)
5. <a name="2.5"></a>The action SHALL download release archives from GitHub releases
6. <a name="2.6"></a>The action SHALL provide a clear error message when a specified version does not exist

### 3. Platform Support

**User Story:** As a GitHub Actions user, I want the action to work on different runner types, so that I can use rune across various CI/CD environments.

**Acceptance Criteria:**

1. <a name="3.1"></a>The action SHALL support Linux runners with amd64 and arm64 architectures
2. <a name="3.2"></a>The action SHALL support macOS runners with amd64 and arm64 architectures
3. <a name="3.3"></a>The action SHALL support Windows runners with amd64 and arm64 architectures
4. <a name="3.4"></a>The action SHALL automatically detect the runner's operating system and architecture
5. <a name="3.5"></a>The action SHALL fail with a clear error message on unsupported platforms or architectures

### 4. Performance and Caching

**User Story:** As a GitHub Actions user, I want subsequent runs to be fast, so that my CI/CD pipeline remains efficient.

**Acceptance Criteria:**

1. <a name="4.1"></a>The action SHALL cache downloaded binaries using GitHub Actions cache
2. <a name="4.2"></a>The action SHALL use cached binaries when the same version is requested again on the same platform
3. <a name="4.3"></a>The action SHALL include version, operating system, and architecture information in the cache key

### 5. Integrity Verification

**User Story:** As a GitHub Actions user, I want to ensure the downloaded archive is not corrupted, so that my CI/CD pipeline uses a valid binary.

**Acceptance Criteria:**

1. <a name="5.1"></a>The action SHALL verify MD5 checksums of downloaded release archives for integrity checking
2. <a name="5.2"></a>The action SHALL download MD5 checksum files from the official GitHub release
3. <a name="5.3"></a>The action SHALL fail if checksum verification fails
4. <a name="5.4"></a>The action SHALL use HTTPS for all downloads

### 6. Action Outputs

**User Story:** As a GitHub Actions user, I want to know which version was installed and where, so that I can use this information in subsequent workflow steps.

**Acceptance Criteria:**

1. <a name="6.1"></a>The action SHALL output the installed version of rune
2. <a name="6.2"></a>The action SHALL output the installation path where the binary is located
3. <a name="6.3"></a>The action SHALL make outputs available to subsequent workflow steps using the standard GitHub Actions output mechanism

### 7. Configuration Options

**User Story:** As a GitHub Actions user, I want to customize the installation process, so that I can handle special requirements like avoiding API rate limits.

**Acceptance Criteria:**

1. <a name="7.1"></a>The action SHALL accept an optional `github-token` input parameter
2. <a name="7.2"></a>The action SHALL use the provided github-token for GitHub API requests when specified, falling back to the default GITHUB_TOKEN otherwise
