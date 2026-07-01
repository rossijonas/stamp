# Contributing to Stamp

We welcome contributions to `stamp`! This document outlines our development standards.

## Prerequisites
- [Go 1.22+](https://go.dev/doc/install)
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
We follow [Conventional Commits](https://www.conventionalcommits.org/), example:
- `feat:` for new features.
- `fix:` for bug fixes.
- `chore:` for maintenance, tooling, or dependency updates.
- `docs:` for documentation changes.
- `test:` for adding or updating tests.

## Branching Strategy

We use [Trunk-Based Development with Short-Lived Feature Branches](https://trunkbaseddevelopment.com/short-lived-feature-branches/). Please branch off `main`, keep your branches short-lived (1-3 days), and submit Pull Requests frequently.
