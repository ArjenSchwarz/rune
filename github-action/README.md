# Setup Rune GitHub Action

This GitHub Action installs the [rune CLI](https://github.com/ArjenSchwarz/rune) binary into GitHub Actions runner environments, making it available for use in your workflows.

## Usage

### Basic Usage (Latest Version)

Install the latest stable release of rune:

```yaml
- name: Setup Rune
  uses: ArjenSchwarz/rune/github-action@v1

- name: Use Rune
  run: rune --version
```

### Specific Version

Install a specific version of rune:

```yaml
- name: Setup Rune
  uses: ArjenSchwarz/rune/github-action@v1
  with:
    version: '1.0.0'

- name: Use Rune
  run: rune list tasks.md
```

Version can be specified with or without the `v` prefix:
- `1.0.0` (without prefix)
- `v1.0.0` (with prefix)

### Using Outputs

The action provides outputs that you can use in subsequent steps:

```yaml
- name: Setup Rune
  id: setup-rune
  uses: ArjenSchwarz/rune/github-action@v1

- name: Display version and path
  run: |
    echo "Installed rune version: ${{ steps.setup-rune.outputs.version }}"
    echo "Installation path: ${{ steps.setup-rune.outputs.path }}"
```

### Custom GitHub Token

Provide a custom GitHub token to avoid rate limiting (useful for high-volume workflows):

```yaml
- name: Setup Rune
  uses: ArjenSchwarz/rune/github-action@v1
  with:
    version: 'latest'
    github-token: ${{ secrets.CUSTOM_GITHUB_TOKEN }}
```

By default, the action uses the automatic `GITHUB_TOKEN` provided by GitHub Actions.

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `version` | Version of rune to install (e.g., `1.0.0`, `v1.0.0`, or `latest`) | No | `latest` |
| `github-token` | GitHub token for API requests (helps avoid rate limiting) | No | `${{ github.token }}` |

## Outputs

| Output | Description | Example |
|--------|-------------|---------|
| `version` | The installed version of rune | `1.0.0` |
| `path` | The directory containing the rune binary | `/opt/hostedtoolcache/rune/1.0.0/x64` |

## Platform Support

This action supports the following platforms and architectures:

| Operating System | Architecture | Runner Label |
|-----------------|--------------|--------------|
| Ubuntu Linux | amd64 | `ubuntu-latest`, `ubuntu-22.04`, `ubuntu-20.04` |
| Ubuntu Linux | arm64 | Self-hosted ARM64 runners |
| macOS | amd64 | `macos-13`, `macos-12` |
| macOS | arm64 | `macos-latest`, `macos-14` |
| Windows | amd64 | `windows-latest`, `windows-2022`, `windows-2019` |

## Example Workflows

### Ubuntu Example

```yaml
name: CI
on: [push]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Rune
        uses: ArjenSchwarz/rune/github-action@v1
        with:
          version: 'latest'

      - name: Create task list
        run: |
          rune create tasks.md --title "Build Tasks"
          rune add tasks.md --title "Run tests"
          rune add tasks.md --title "Build artifacts"

      - name: Display tasks
        run: rune list tasks.md
```

### macOS Example

```yaml
name: macOS Build
on: [push]

jobs:
  build:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Rune
        uses: ArjenSchwarz/rune/github-action@v1
        with:
          version: '1.0.0'

      - name: Manage tasks
        run: |
          rune create build-tasks.md --title "macOS Build"
          rune add build-tasks.md --title "Compile for ARM64"
          rune complete build-tasks.md 1
```

### Windows Example

```yaml
name: Windows Build
on: [push]

jobs:
  build:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Rune
        uses: ArjenSchwarz/rune/github-action@v1
        with:
          version: 'latest'

      - name: Use Rune
        run: |
          rune create tasks.md --title "Windows Tasks"
          rune list tasks.md --format json
```

### Matrix Strategy Example

Test across multiple platforms:

```yaml
name: Multi-Platform
on: [push]

jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}

    steps:
      - uses: actions/checkout@v4

      - name: Setup Rune
        uses: ArjenSchwarz/rune/github-action@v1

      - name: Verify installation
        run: rune --version
```

## Caching

The action automatically caches downloaded rune binaries using GitHub Actions' tool cache. Subsequent workflow runs that request the same version will use the cached binary, significantly improving performance.

### Cache Behavior

- **Cache Key**: Includes version, operating system, and architecture
- **Cache Scope**: Per runner, shared across workflow runs
- **Cache Size**: Rune binaries are approximately 2-3 MB compressed
- **Cache Duration**: Managed by GitHub Actions (typically retained for 7 days of inactivity)

When the cache is hit, you'll see a message like:
```
Using cached rune 1.0.0
```

When the cache is missed, the action will:
1. Download the release archive from GitHub
2. Verify the MD5 checksum
3. Extract the binary
4. Cache it for future runs
5. Add it to the PATH

## Integrity Verification

The action verifies the integrity of downloaded binaries using MD5 checksums:

1. Downloads the release archive (`.tar.gz` for Linux/macOS, `.zip` for Windows)
2. Downloads the corresponding `.md5` checksum file
3. Calculates the MD5 hash of the downloaded archive
4. Compares with the expected checksum
5. Fails the workflow if verification fails

All downloads use HTTPS for secure transmission.

## Troubleshooting

### Version Not Found

If you see an error like:
```
Version 1.2.3 not found.
Check available versions at: https://github.com/ArjenSchwarz/rune/releases
```

**Solution**: Verify the version exists by checking the [releases page](https://github.com/ArjenSchwarz/rune/releases). Ensure you're using a valid release version.

### Unsupported Platform

If you see an error like:
```
Unsupported platform: linux-ia32
Supported: linux/darwin/windows on amd64/arm64
```

**Solution**: This action only supports 64-bit platforms (amd64 and arm64). 32-bit architectures are not supported.

### Checksum Verification Failed

If you see an error like:
```
MD5 checksum verification failed!
Expected: abc123...
Actual: def456...
```

**Solution**: This indicates a corrupted download. The action will automatically retry, but if it persists, it may indicate a network issue or a problem with the release assets. Try again or open an issue.

### Rate Limiting

If you see API rate limiting errors:

**Solution**: Provide a custom GitHub token with higher rate limits:
```yaml
- name: Setup Rune
  uses: ArjenSchwarz/rune/github-action@v1
  with:
    github-token: ${{ secrets.CUSTOM_GITHUB_TOKEN }}
```

Unauthenticated requests are limited to 60/hour, while authenticated requests allow 5,000/hour.

## Development

### Prerequisites

- Node.js 20 or later
- npm

### Setup

```bash
cd github-action
npm install
```

### Building

```bash
npm run build
```

This will:
1. Compile TypeScript to JavaScript in `lib/`
2. Bundle the application using `@vercel/ncc` to `dist/index.js`

### Testing

```bash
# Run tests
npm test

# Run tests with coverage
npm run test:coverage
```

### Linting

```bash
npm run lint
```

## Project Structure

```
.
├── action.yml              # Action metadata
├── src/
│   ├── main.ts            # Entry point
│   └── install.ts         # Installation logic
├── __tests__/
│   └── install.test.ts    # Tests
├── dist/
│   └── index.js           # Bundled output (committed)
├── package.json
├── tsconfig.json
└── jest.config.js
```

## License

This action is part of the [rune project](https://github.com/ArjenSchwarz/rune) and is licensed under the same license.

## Contributing

Contributions are welcome! Please see the main [rune repository](https://github.com/ArjenSchwarz/rune) for contribution guidelines.
