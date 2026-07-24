---
---

# ADR-004: Use Goreleaser with Auto-Tagging for Release Automation

## Status
Accepted

## Date
2026-07-10

## Context
We need to automate the release process for stamp. Key requirements:
- Build cross-platform binaries (linux + darwin, amd64 + arm64)
- Create GitHub Releases with changelogs
- Automatically version based on conventional commits
- Generate checksums for binary verification
- Support both automated and manual release triggers

## Decision

### Release Pipeline
Use two complementary workflows:

1. **Auto (default):** `auto-tag.yml` triggers on push to `main`. Uses `thenativeweb/get-next-version` to calculate the next version from conventional commits since the last tag. Creates a local git tag, then runs goreleaser to build and publish.

2. **Manual (fallback):** `release.yml` triggers on `v*` tag push. Runs goreleaser directly for hotfix releases from specific commits.

### Version Calculation
`thenativeweb/get-next-version` with `prefix: 'v'`:
- `feat!:` or `BREAKING CHANGE:` → MAJOR
- `feat:` → MINOR
- `fix:` → PATCH
- `chore/docs/style/refactor/perf/test/ci/build` → no bump

### Tag Strategy
- Tags include a `v` prefix (e.g. `v1.2.3`)
- The local tag is created before goreleaser runs so goreleaser can detect the previous tag for changelog generation
- Goreleaser creates the GitHub Release (and remote tag) via API

### Build Matrix
- linux/amd64, linux/arm64
- darwin/amd64, darwin/arm64
- Archives as `.tar.gz` including README.md and LICENSE
- Checksums generated via `checksums.txt`

### Version Injection
Version, commit hash, and build date are injected at build time via ldflags:
```
{% raw %}-X github.com/rossijonas/stamp/internal/cli.Version={{.Version}}
-X github.com/rossijonas/stamp/internal/cli.Commit={{.Commit}}
-X github.com/rossijonas/stamp/internal/cli.Date={{.Date}}{% endraw %}
```

## Alternatives Considered

### Custom Bash Script for Versioning
- **Pros:** Zero external dependencies, full control
- **Cons:** Brittle regex parsing, edge cases with pre-release tags
- **Rejected:** `get-next-version` handles edge cases better

### semantic-release (Node.js)
- **Pros:** Robust semver calculation, rich changelog formatting, npm publishing support
- **Cons:** Adds Node.js dependency to CI, slower builds, overkill for a Go CLI project
- **Rejected:** Go-based action is lighter and more idiomatic for a Go project

### Manual Tagging Only
- **Pros:** Full human control
- **Cons:** Error-prone, easy to forget, no standardization
- **Rejected:** Automation reduces human error

## Consequences
- Goreleaser handles changelog generation between tags
- Auto-tag workflow must create local tag before goreleaser runs (so goreleaser can detect previous tag for diff)
- The `--skip=validate` flag was removed — local tag creation makes validation pass
- Human error in versioning is eliminated for the default path
- Manual hotfix releases are still supported via the fallback workflow
