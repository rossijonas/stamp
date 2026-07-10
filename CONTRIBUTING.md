# Contributing to Stamp

We welcome contributions to `stamp`! This document outlines our development standards.

## Prerequisites
- [Go 1.26+](https://go.dev/doc/install)
- [Taskfile](https://taskfile.dev/installation/) (`go install github.com/go-task/task/v3/cmd/task@latest` or `brew install go-task`)
- [golangci-lint](https://golangci-lint.run/)

## Development Workflow

After cloning the repository, ensure you download the required dependencies:

```bash
go mod tidy
```

We use `task` instead of `make`. Here are the essential commands:
- `task check` - Runs all quality gates: module verification, static analysis, unit tests, and vulnerability scanning. **Must pass before opening a PR.**
- `task build` - Builds the binary into the `bin/` directory.
- `task test` - Runs all unit tests with the race detector enabled. Enforces 90% minimum coverage.
- `task lint` - Runs static analysis.
- `task verify` - Ensures `go.mod` and `go.sum` are clean and cryptographically verified.
- `task security` - Runs `govulncheck` to scan for known CVEs.
- `task clean` - Removes build artifacts.

## Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

| Type | Description | Version bump |
|:---|:---|---:|
| `feat` | New feature | MINOR |
| `fix` | Bug fix | PATCH |
| `feat!:` or `fix!:` | Breaking change | MAJOR |
| `chore`, `docs`, `test`, `refactor`, `perf`, `ci`, `build`, `style` | Maintenance | None |

## Pull Request Process

1. Ensure `task check` passes locally before opening a PR
2. CI automatically runs: lint, tests (with race detector and ≥90% coverage), govulncheck, and validates that at least one commit follows the Conventional Commits format
3. Once merged to `main`, a new version is automatically tagged and released

## Release Process

Releases are fully automated:
- **Auto:** Merging to `main` triggers `thenativeweb/get-next-version` to calculate the next version from conventional commits, then goreleaser builds binaries and creates a GitHub Release with changelog
- **Manual:** Push a `v*` tag to trigger `release.yml` directly

No manual version tagging or changelog editing is required.

## Branching Strategy

We use [Trunk-Based Development with Short-Lived Feature Branches](https://trunkbaseddevelopment.com/short-lived-feature-branches/). Please branch off `main`, keep your branches short-lived (1-3 days), and submit Pull Requests frequently.

## Code of Conduct

We are committed to making participation in this project a harassment-free experience for everyone. We expect all contributors to adhere to our [Code of Conduct](CODE_OF_CONDUCT.md) in all community interactions.
