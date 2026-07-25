---
---

# ADR-010: Language Toolchain Support Scope

## Status
Accepted

## Date
2026-07-25

## Context

Language-specific package managers (pip, npm, cargo, go get, etc.) serve two
distinct use cases:

1. **Installing end-user CLI tools** — e.g., `pipx install black`, `go install github.com/golangci/golangci-lint@latest`
2. **Managing project dependencies** — e.g., `pip install requests`, `go get github.com/some/lib`

Stamp is designed for the first use case: installing user-facing tools at the
OS level. The second use case — project dependency management — is out of
scope and belongs to tools like pipenv, npm ci, and Go modules.

Without a clear scope boundary, stamp risks feature creep into dependency
management territory, where it cannot compete with dedicated tools.

## Decision

Stamp supports language toolchains ONLY for installing end-user CLI tools.
This means stamp follows the `pipx` model, not the `pip`/`go get`/`cargo add`
model.

### Supported vs Unsupported Usage

| Ecosystem | Tool installer (stamp) | Dependency manager (NOT stamp) |
|-----------|----------------------|-------------------------------|
| Python | `pipx install <tool>` / `uv tool install <tool>` | `pip install <lib>` |
| Go | `go install <path>@latest` | `go get <path>` |
| Rust | `cargo install <tool>` | `cargo add <dep>` |
| JavaScript | `npm install -g <tool>` / `bun install -g <tool>` | `npm install --save <dep>` |

### Adapter Naming Convention

Each tool installer is registered as its own adapter under its own binary name (e.g.,
`pipx`, `uv`, `go`, `cargo`, `npm`). Adapters are never named after the ecosystem
("python", "rust") — stamp installs via specific package managers, not programming
languages. This is consistent with OS-level adapters (`dnf`, `brew`, `flatpak`) which
are named after the tool, not the OS.

### Adapter Implementation Status

| Priority | Ecosystem | Adapter(s) | Status |
|----------|-----------|------------|--------|
| 1 | **Go** | `go` | ✓ Complete |
| 2 | **Python** | `pipx`, `uv` | ✓ Complete |
| 3 | **Rust** | `cargo` | ⚠ Planned |
| 4 | **JavaScript** | `npm`, `bun` | ⚠ Planned |

## Alternatives Considered

### Full dependency management support
- **Pros:** Caters to developers managing project environments
- **Cons:** Stamp cannot compete with pipenv, npm ci, Go modules in this space.
  Blurs the tool's identity and increases maintenance burden.
- **Rejected:** Out of scope.

### Skip language toolchains entirely
- **Pros:** Simpler codebase, no validation complexity
- **Cons:** Users must use a separate tool (pipx, rustup) to install CLI tools
  from language ecosystems, fragmenting their install workflow
- **Rejected:** Language toolchains are a natural extension of stamp's mission.

## Consequences

- Language toolchain adapters validate package names with `ValidateModulePath`
  which requires a full module path for Go (e.g., `github.com/example/tool`)
- `Search` is unsupported for Go — Go has no built-in search command
- `Doctor` is unsupported for Go — no equivalent of `brew doctor`
- Repo management is unsupported for all language toolchains (no taps, PPAs,
  or remotes in their model)
- Future adapters (pipx, cargo, npm) must document which usage pattern
  they support
