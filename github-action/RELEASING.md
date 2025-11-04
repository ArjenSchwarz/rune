# Releasing the Setup Rune GitHub Action

This document describes the process for releasing new versions of the Setup Rune GitHub Action.

## Release Process

### 1. Ensure All Changes Are Tested

Before releasing, verify that:
- All unit tests pass: `npm test`
- Test coverage is maintained: `npm run test:coverage`
- Linting passes: `npm run lint`
- The bundle is up to date: `npm run build`

### 2. Update Version Number

Update the version in `package.json`:

```bash
# For a patch release (bug fixes)
npm version patch

# For a minor release (new features, backward compatible)
npm version minor

# For a major release (breaking changes)
npm version major
```

This will automatically create a git tag.

### 3. Rebuild the Distribution Bundle

**CRITICAL**: The `dist/` directory must be rebuilt and committed before releasing:

```bash
npm run build
git add dist/
git commit -m "chore: rebuild distribution bundle for v$(node -p 'require("./package.json").version')"
```

The action runs from the bundled `dist/index.js`, not the source files. Forgetting this step will cause the action to run old code.

### 4. Push Changes and Tags

```bash
# Push the version commit
git push

# Push the version tag
git push --tags
```

### 5. Create GitHub Release

Create a GitHub release from the new tag:

```bash
# Get the version from package.json
VERSION=$(node -p 'require("./package.json").version')

# Create release using gh CLI
gh release create "v${VERSION}" \
  --title "v${VERSION}" \
  --notes-file RELEASE_NOTES.md
```

Or create the release manually via the GitHub web interface:
1. Go to https://github.com/ArjenSchwarz/rune/releases/new
2. Select the tag (e.g., `v1.0.0`)
3. Add release notes describing the changes
4. Click "Publish release"

### 6. Update Major Version Tag

For users to reference `@v1` (recommended), maintain a major version tag that points to the latest release in that major version:

```bash
# For version 1.2.3, update v1 to point to v1.2.3
git tag -fa v1 -m "Update v1 tag to v1.2.3"
git push origin v1 --force
```

This allows users to use:
```yaml
uses: ArjenSchwarz/rune/github-action@v1  # Latest v1.x.x
```

Instead of:
```yaml
uses: ArjenSchwarz/rune/github-action@v1.2.3  # Pinned to specific version
```

## Version Numbering Strategy

Follow [Semantic Versioning](https://semver.org/):

- **Major version (x.0.0)**: Breaking changes to inputs, outputs, or behavior
  - Changing required inputs
  - Removing or renaming inputs/outputs
  - Changing default behavior in breaking ways

- **Minor version (1.x.0)**: New features, backward compatible
  - Adding new optional inputs
  - Adding new outputs
  - Enhancing existing functionality without breaking changes

- **Patch version (1.0.x)**: Bug fixes, backward compatible
  - Fixing bugs
  - Updating dependencies
  - Documentation improvements

## Release Checklist

Use this checklist for each release:

- [ ] All tests passing (`npm test`)
- [ ] Test coverage maintained (`npm run test:coverage`)
- [ ] Linting passes (`npm run lint`)
- [ ] Version updated in `package.json`
- [ ] Distribution bundle rebuilt (`npm run build`)
- [ ] `dist/` directory committed
- [ ] Changes pushed to GitHub
- [ ] Tag pushed to GitHub
- [ ] GitHub Release created with release notes
- [ ] Major version tag updated (if applicable)
- [ ] Release announcement (if major/minor release)

## Testing a Release

Before creating an official release, test the action in a real workflow:

1. Push your changes to a branch
2. Create a test workflow that uses the action from that branch:
   ```yaml
   uses: ArjenSchwarz/rune/github-action@your-branch-name
   ```
3. Verify the action works as expected
4. Once verified, proceed with the release process

## Rollback Procedure

If a release has issues:

1. **Immediate rollback of major version tag**:
   ```bash
   # Point v1 back to the previous good version
   git tag -fa v1 -m "Rollback v1 to v1.2.2" v1.2.2
   git push origin v1 --force
   ```

2. **Create a patch release** with the fix as soon as possible

3. **Document the issue** in the release notes of the fix

## Node.js Version Migration

The action currently uses Node.js 20 (`node20` in `action.yml`). When migrating to a new Node.js version:

1. Update `action.yml`: `using: 'node24'` (or relevant version)
2. Test thoroughly across all platforms (Ubuntu, macOS, Windows)
3. This is a **breaking change** - bump major version
4. Document the change in release notes
5. Update the comment in `action.yml` about EOL dates

## Support and Maintenance

- **Active support**: Latest major version (currently v1.x)
- **Security fixes**: Latest two major versions
- **EOL policy**: Major versions are supported for 1 year after the next major version release

## Questions?

For questions about the release process, open an issue in the rune repository.
