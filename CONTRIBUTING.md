# Contributing to rune

Thanks for your interest in contributing to rune. This guide covers what you need to get started and what we expect from contributions.

## Getting Started

**Prerequisites:**

- Go 1.25 or later
- [golangci-lint](https://golangci-lint.run/welcome/install/)

**Setup:**

```bash
git clone https://github.com/<your-fork>/rune.git
cd rune
make build
```

## Development Workflow

1. Create a branch from `main`
2. Make your changes
3. Run `make check` — this formats, lints, and tests in one step
4. Commit and push
5. Open a pull request against `main`

## Code Standards

All Go code must pass formatting and linting before merge. CI runs these automatically, but catch issues early by running them locally:

```bash
make fmt         # Format code
make lint        # Run golangci-lint
make modernize   # Apply Go modernization fixes
```

## Testing

```bash
make test              # Unit tests
make test-integration  # Integration tests (sets INTEGRATION=1)
make test-all          # Both unit and integration tests
make test-coverage     # Generate HTML coverage report
```

**Conventions:**

- Test files live alongside their source (`foo.go` / `foo_test.go`)
- Use map-based test tables with named cases
- Integration tests are gated behind `INTEGRATION=1` and live in `cmd/integration_test.go`
- Add or update tests for any changed behaviour

## Documentation Checklist

When adding or changing features, check whether these need updating:

- [ ] `README.md` — user-facing command docs and examples
- [ ] `CHANGELOG.md` — follows [Keep a Changelog](https://keepachangelog.com/) format, add entry under `[Unreleased]`
- [ ] `skill/SKILL.md` — skill definition used by AI agents
- [ ] `examples/` — example task files if relevant

## Commit Messages

Use a prefix that describes the type of change:

| Prefix | Use for |
|--------|---------|
| `[feat]:` | New features |
| `[fix]:` | Bug fixes |
| `[doc]:` | Documentation changes |
| `chore:` | Dependencies, CI, tooling |

Examples from the project history:

```
[feat]: Allow batch command to accept target file as positional argument (#29)
[fix]: Use last slash for branch name stripping in discovery (#24)
[doc]: Release notes for v1.2.0
chore: update dependencies and modernize Go patterns
```

## Pull Requests

A good PR:

- Passes `make check` (CI will verify this)
- Includes tests for new or changed behaviour
- Updates documentation (see checklist above)
- Has a clear description of what changed and why
- Keeps changes focused — one logical change per PR

## Architecture Overview

- `cmd/` — CLI commands (Cobra framework), one file per command
- `internal/task/` — Core business logic: parsing, operations, rendering, search, batch

See `CLAUDE.md` for detailed architecture notes, design principles, and data flow.

## License

This project is licensed under the MIT License. By contributing, you agree that your contributions will be licensed under the same terms.
