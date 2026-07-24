---
---

# ADR-006: Documentation Generation Strategy

## Status
Accepted

## Date
2026-07-11

## Context
We need to generate CLI reference documentation in multiple formats:
1. Markdown files for GitHub Pages (`docs/usage/`)
2. Troff man pages for `man stamp` (`docs/man/`)
3. The documentation must stay in sync with the command tree (never drift)
4. Binary users must be able to install man pages without building from source

## Decision

### Dual Approach: Build-Time Tool + Self-Contained Command

#### 1. Build-Time Doc Generation (`tools/docgen/` + `task docs`)
A standalone Go program in `tools/docgen/` imports `github.com/spf13/cobra/doc` and generates:
- Markdown reference docs → `docs/usage/` (one `.md` file per command)
- Man pages → `docs/man/` (one `.1` file per command)

Invoked via `task docs` and enforced in CI:
```yaml
- name: Verify docs are up to date
  run: |
    go run ./tools/docgen/
    git diff --exit-code docs/usage/ || (echo "error: docs out of date" && exit 1)
```

#### 2. Self-Contained Man Page Installation (`stamp man`)
The stamp binary itself can generate and install man pages via `stamp man install`:
- Uses `github.com/spf13/cobra/doc` directly in the binary
- Default install path: `~/.local/share/man/man1/` (no sudo needed)
- Support `--prefix` flag for custom install paths

### Rationale for Two Approaches
- **`tools/docgen/`** keeps `cobra/doc` out of the production binary by default (lighter build)
- **`stamp man`** makes man page installation available to binary users who can't run `task docs`
- Both use the same `cobra/doc` library and generate identical output

### Drift Prevention
- CI runs `task docs` and fails if `docs/usage/` differs from committed versions
- `docs/man/` is gitignored (build artifact)
- `docs/usage/` is committed (required for GitHub Pages)

## Alternatives Considered

### Single Tool Only (Only `tools/docgen`)
- **Pros:** Lighter binary, single responsibility
- **Cons:** Binary users can't install man pages without cloning the repo
- **Rejected:** Does not serve binary users

### Single Binary Only (Only `stamp man`)
- **Pros:** Self-contained, works for everyone
- **Cons:** Adds `go-md2man` dependency to stamp binary (~500KB); GitHub Pages docs can't be auto-generated
- **Rejected:** Loses the CI drift detection and GitHub Pages automation

## Consequences
- CI enforces doc freshness on every PR
- Binary users can run `stamp man install` without cloning the repo
- `docs/man/` is gitignored (regenerated on demand)
- `docs/usage/` must be committed when commands or flags change
- The `man` command uses subcommands (`install`, `check`) — not flags
