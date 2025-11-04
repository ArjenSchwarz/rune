---
references:
    - specs/github-action/requirements.md
    - specs/github-action/design.md
    - specs/github-action/decision_log.md
---
# GitHub Action Implementation Tasks

## Phase 1: Project Setup

- [ ] 1. Initialize TypeScript project with dependencies
  - Install @actions/core, @actions/tool-cache, @actions/github, @actions/exec
  - Configure tsconfig.json for Node 20 target
  - Set up @vercel/ncc for bundling to dist/index.js

- [ ] 2. Create action.yml metadata file
  - Define name, description, author, and branding
  - Specify inputs: version (default: latest), github-token (default: github.token)
  - Specify outputs: version, path
  - Set runs.using to node20 and runs.main to dist/index.js
  - Requirements: [2.1](requirements.md#2.1), [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [6.3](requirements.md#6.3), [7.1](requirements.md#7.1)
  - References: specs/github-action/design.md

- [ ] 3. Set up Jest test infrastructure
  - Install Jest and ts-jest
  - Configure jest.config.js for TypeScript
  - Add test scripts to package.json
  - References: specs/github-action/design.md

- [ ] 4. Configure build scripts in package.json
  - Add build script using ncc
  - Add test script
  - Add lint script if using eslint
  - Set up pre-commit checks
  - References: specs/github-action/design.md

## Phase 2: Core Implementation

- [ ] 5. Write unit tests for resolveVersion()
  - Test latest version resolution
  - Test exact version with v prefix (v1.0.0)
  - Test exact version without v prefix (1.0.0)
  - Test 404 error for non-existent version
  - Mock GitHub API responses using jest.mock
  - Requirements: [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.6](requirements.md#2.6)
  - References: specs/github-action/design.md

- [ ] 6. Implement resolveVersion() function
  - Use getOctokit() to create GitHub API client
  - Handle latest by calling repos.getLatestRelease()
  - Strip v prefix from tag_name
  - Handle exact versions by calling repos.getReleaseByTag()
  - Throw clear error for 404 responses
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [2.5](requirements.md#2.5), [2.6](requirements.md#2.6)
  - References: specs/github-action/design.md

- [ ] 7. Write unit tests for getPlatformAsset()
  - Test asset name generation for all 6 platform combinations (linux/darwin/windows × amd64/arm64)
  - Test correct extensions (.tar.gz for Unix, .zip for Windows)
  - Test error for unsupported platform (e.g., freebsd)
  - Test error for unsupported architecture
  - Mock process.platform and process.arch
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.3](requirements.md#3.3), [3.4](requirements.md#3.4), [3.5](requirements.md#3.5)
  - References: specs/github-action/design.md

- [ ] 8. Implement getPlatformAsset() function
  - Map process.platform to OS names (linux/darwin/windows)
  - Map process.arch to architecture names (amd64/arm64)
  - Throw clear error for unsupported combinations
  - Build asset name: rune-v{version}-{os}-{arch}.{ext}
  - Return {assetName, isWindows} object
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.3](requirements.md#3.3), [3.4](requirements.md#3.4), [3.5](requirements.md#3.5), [1.1](requirements.md#1.1)
  - References: specs/github-action/design.md

- [ ] 9. Write unit tests for verifyChecksum()
  - Test successful checksum verification
  - Test checksum mismatch throws error
  - Test file read errors
  - Mock fs.promises.readFile and fs.createReadStream
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3)
  - References: specs/github-action/design.md

- [ ] 10. Implement verifyChecksum() function
  - Read expected checksum from file
  - Calculate MD5 using crypto.createHash()
  - Use fs.createReadStream for efficient file reading
  - Compare expected vs actual checksums
  - Throw error with both values on mismatch
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3)
  - References: specs/github-action/design.md

- [ ] 11. Write unit tests for installRune()
  - Test cache hit scenario (tc.find returns path)
  - Test cache miss with successful download
  - Test extraction for .tar.gz (Linux/macOS)
  - Test extraction for .zip (Windows)
  - Test chmod execution on non-Windows
  - Mock @actions/tool-cache functions
  - Mock @actions/exec functions
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.3](requirements.md#1.3), [1.4](requirements.md#1.4), [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3)
  - References: specs/github-action/design.md

- [ ] 12. Implement installRune() orchestration function
  - Call resolveVersion() first to get exact version
  - Call getPlatformAsset() to get asset info
  - Check cache using tc.find() with version and arch
  - If cached, add to PATH and return early
  - Download archive and checksum using tc.downloadTool()
  - Call verifyChecksum()
  - Extract using tc.extractZip() or tc.extractTar() based on platform
  - Run chmod +x on non-Windows
  - Cache extracted directory using tc.cacheDir()
  - Add cached path to PATH using core.addPath()
  - Return {version, path}
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.3](requirements.md#1.3), [1.4](requirements.md#1.4), [1.5](requirements.md#1.5), [2.5](requirements.md#2.5), [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3), [5.4](requirements.md#5.4)
  - References: specs/github-action/design.md

- [ ] 13. Implement main.ts entry point
  - Read version input using core.getInput(), default to latest
  - Read github-token input, fallback to process.env.GITHUB_TOKEN
  - Call installRune() with version and token
  - Set version output using core.setOutput()
  - Set path output using core.setOutput()
  - Wrap in try-catch and call core.setFailed() on errors
  - Include type guard for Error instanceof
  - Requirements: [1.6](requirements.md#1.6), [2.1](requirements.md#2.1), [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [6.3](requirements.md#6.3), [7.1](requirements.md#7.1), [7.2](requirements.md#7.2)
  - References: specs/github-action/design.md

## Phase 3: Testing

- [ ] 14. Create integration test workflow file
  - Create .github/workflows/test.yml
  - Set up matrix for ubuntu-latest, macos-latest, windows-latest
  - Test with version: 1.0.0
  - Test with version: latest
  - Test cache behavior (run twice with same version)
  - Verify rune --version output
  - Verify outputs.version and outputs.path are set
  - Requirements: [1.5](requirements.md#1.5), [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.3](requirements.md#3.3), [4.2](requirements.md#4.2), [6.1](requirements.md#6.1), [6.2](requirements.md#6.2)
  - References: specs/github-action/design.md

- [ ] 15. Add integration test for error handling
  - Test non-existent version (should fail gracefully)
  - Use continue-on-error: true
  - Verify failure message is clear
  - Check that step.outcome is failure
  - Requirements: [1.6](requirements.md#1.6), [2.6](requirements.md#2.6)
  - References: specs/github-action/design.md

- [ ] 16. Run tests and verify >80% coverage
  - Run npm test
  - Check coverage report
  - Ensure all functions have tests
  - Verify error paths are tested
  - References: specs/github-action/design.md

- [ ] 17. Test manually on all platforms
  - Trigger integration workflow
  - Verify successful runs on Ubuntu, macOS, Windows
  - Check cache behavior works correctly
  - Verify binary is accessible in subsequent steps
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.3](requirements.md#3.3), [4.2](requirements.md#4.2)
  - References: specs/github-action/design.md

## Phase 4: Documentation & Release

- [ ] 18. Write README.md with usage examples
  - Add basic usage example with default latest version
  - Add example with specific version
  - Add example showing how to use outputs
  - Document all inputs and outputs
  - Add examples for Ubuntu, macOS, and Windows
  - Include cache behavior explanation
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [7.1](requirements.md#7.1)
  - References: specs/github-action/design.md

- [ ] 19. Build production bundle
  - Run npm run build
  - Verify dist/index.js is created
  - Verify bundle includes all dependencies
  - Test that action works with bundled dist/
  - References: specs/github-action/design.md

- [ ] 20. Create v1.0.0 release
  - Commit dist/index.js to repository
  - Create git tag v1.0.0
  - Create GitHub release with changelog
  - Move v1 tag to point to v1.0.0 (for users who want latest v1.x)
  - Verify action can be used as username/repo@v1
  - References: specs/github-action/design.md

- [ ] 21. Publish to GitHub Marketplace (optional)
  - Add marketplace categories to action.yml
  - Create marketplace listing
  - Add action icon and color branding
  - Submit for marketplace publication
  - References: specs/github-action/design.md
