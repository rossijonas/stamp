# Contributing to Stamp

We welcome contributions to `stamp`! This document outlines our development standards.

## Prerequisites

- [Go 1.26+](https://go.dev/doc/install)
- [Taskfile](https://taskfile.dev/installation/) (`go install github.com/go-task/task/v3/cmd/task@latest` or `brew install go-task`)
- [golangci-lint](https://golangci-lint.run/)
- **Ruby 3+** — required for local docs preview (see [Documentation](#documentation) section)
- **Bundler** — `gem install bundler` (after installing Ruby)

## Development Workflow

After cloning the repository, ensure you download the required dependencies:

```bash
go mod tidy
```

We use `task` instead of `make`. Here are the essential commands:

- `task check` - Runs all quality gates: module verification, static analysis, unit tests, and vulnerability scanning. **Must pass before opening a PR.**
- `task build` - Builds the binary into the `bin/` directory.
- `task test` - Runs all unit tests with the race detector enabled. Enforces 90% minimum coverage.
- `task lint` - Runs static analysis (Go).
- `task lint:sh` - Lints shell scripts with ShellCheck. Requires `shellcheck` (`sudo dnf install -y shellcheck`).
- `task verify` - Ensures `go.mod` and `go.sum` are clean and cryptographically verified.
- `task security` - Runs `govulncheck` to scan for known CVEs.
- `task test:integration` - Builds the binary, runs integration smoke tests in Docker containers (Ubuntu, Debian, Fedora, CentOS Stream 10, Rocky Linux 9, Arch Linux, openSUSE Tumbleweed). Requires Docker or Podman (`podman-docker` on Fedora).
- `task clean` - Removes build artifacts.
- `task docs` - Regenerates CLI reference docs from cobra into `docs/usage/` and `docs/man/`.
- `task docs:serve` - Serves the Jekyll docs site locally at `http://localhost:4000` (requires `bundle install` first).

## Documentation

The project website is served at [https://gostamp.dev](https://gostamp.dev) via GitHub Pages, built with Jekyll from the `docs/` directory.

### Prerequisites

```bash
# Fedora
sudo dnf install ruby-devel gcc-c++

# Ubuntu/Debian
sudo apt install ruby-dev g++
```

### Local preview

```bash
gem install bundler
bundle install
task docs:serve
```

### Regenerate CLI reference docs

After adding examples or changing command structure:

```bash
task docs
```

This runs `go run ./tools/docgen/` which generates Markdown + man pages from the cobra command tree into `docs/usage/` and `docs/man/`.

### File structure

| Path | Purpose |
|------|---------|
| `docs/index.html` | Landing page |
| `docs/_layouts/` | Page templates |
| `docs/_includes/` | Header, nav, footer partials |
| `docs/assets/` | CSS and JS |
| `docs/usage/` | CLI reference (auto-generated + editorial) |
| `docs/getting-started/` | Intro and installation guides |
| `docs/history/` | Vision, spec, feature matrix |
| `docs/community/` | Community links |
| `docs/CNAME` | Custom domain (`gostamp.dev`) |

### CI/CD

`.github/workflows/docs.yml` automatically regenerates CLI docs, builds the Jekyll site, and deploys to GitHub Pages on every push to `main`.

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
2. **Every PR must include corresponding documentation updates.** Changes to CLI commands, features, workflows, or architecture require updates to the relevant docs pages (`docs/usage/`, `docs/getting-started/`, `docs/history/`, README, etc.). Documentation changes must be atomic with the code they describe — not deferred.
3. CI automatically runs: lint, tests (with race detector and ≥90% coverage), govulncheck, and validates that at least one commit follows the Conventional Commits format
4. Once merged to `main`, a new version is automatically tagged and released

## Release Process

Releases are fully automated — no manual version bumping required.

### Auto (default)

Merging a PR to `main` triggers `.github/workflows/auto-tag.yml`:

1. `thenativeweb/get-next-version` calculates the next version from conventional commits since the last tag
2. goreleaser builds binaries for linux + darwin (amd64 + arm64)
3. A GitHub Release is created with auto-generated changelog and release artifacts

### Manual (fallback)

For hotfix releases from a specific commit, push a `v*` tag to trigger `.github/workflows/release.yml`:

```bash
# Tag HEAD
git tag v1.2.3
git push origin v1.2.3

# Or tag a specific commit (hotfix)
git tag v1.2.3 <commit-sha>
git push origin v1.2.3
```

The manual workflow runs tests, then goreleaser builds and publishes the release just like the auto path.

> **Note:** The auto and manual workflows are independent. Auto-tag pushes (`GITHUB_TOKEN`) don't trigger the manual workflow, so there's no risk of duplicate releases.

## Branching Strategy

We use [Trunk-Based Development with Short-Lived Feature Branches](https://trunkbaseddevelopment.com/short-lived-feature-branches/). Please branch off `main`, keep your branches short-lived (1-3 days), and submit Pull Requests frequently.

## Code of Conduct

We are committed to making participation in this project a harassment-free experience for everyone. We expect all contributors to adhere to our [Code of Conduct](CODE_OF_CONDUCT.md) in all community interactions.
