---
---

# ADR-013: Feature Expansion — Cask, Provides, Autoremove, Clean, Hold, Override, Group Install

## Status
Accepted

## Date
2026-07-28

## Context

Phase 2 feature expansion introduces seven new capabilities beyond stamp's core install/remove/search commands:

1. **Homebrew Cask support** — macOS GUI application management via `brew install --cask`
2. **File search (provides)** — "which package owns this file?" across all system package managers
3. **Autoremove** — orphaned dependency cleanup across all package managers
4. **Clean** — package cache cleanup across all package managers
5. **Hold/Unhold** — version pinning for APT, DNF, Pacman
6. **Flatpak Override** — permission management for sandboxed apps
7. **Group Install** — DNF package group support

Each requires extending the system in different ways: context flags, new interface methods, CLI-only subcommands, and flag extensions.

## Decision

### 1. Cask Support — Context Flag Pattern

Cask support will use a **context flag** (`WithCask`/`isCask`) rather than new interface methods or CLI flags for basic install/remove tracking.

```go
func WithCask(ctx context.Context) context.Context
func isCask(ctx context.Context) bool
```

**Rationale:** Cask is a brew-only concept. Adding it to the Adapter interface would introduce a brew-specific concern into the contract every adapter must implement. The context flag pattern follows the existing `WithStreamIO`/`WithOutputPrefix` precedent in the codebase.

**Manifest change:** A `Cask bool` field is added to the `Package` struct with `toml:"cask,omitempty"` for backward compatibility with existing manifests.

**Detection:** Cask packages are detected automatically after `brew install` via `brew info --cask <pkg>`. No `--cask` CLI flag needed for Phase 1 — `brew install` auto-detects casks.

**ListInstalled:** Merges results from `brew leaves --installed-on-request` (formulas) and `brew list --cask` (casks) so casks appear in reconcile, restore, and list.

Alternatives considered:
- New interface method `InstallCask`: rejected — brew-only concern doesn't belong in the interface
- CLI-only dispatch without adapter changes: rejected — restore and reconcile need adapter-level awareness
- Cask-as-separate-adapter: rejected — redundant with brew adapter

### 2. Provides Command — Interface Method

File search adds a new method to the Adapter interface:

```go
Provides(ctx context.Context, query string) ([]string, error)
```

**Rationale:** Six of 14 supported managers have native `provides` commands (DNF, APT, Pacman, Paru, Zypper, MacPorts). An interface method allows each manager to implement its own native command, while unsupported managers return an informative error.

**Implementation per manager:**

| Manager | Command |
|---------|---------|
| DNF | `dnf provides <query>` |
| APT | `dpkg -S <query>` |
| Pacman | `pacman -F <query>` |
| Paru | `pacman -F <query>` |
| Zypper | `zypper what-provides <query>` |
| MacPorts | `port provides <query>` |
| Others | Return `ErrNotSupported` |

**APT choice:** `dpkg -S` is preferred over `apt-file search` because `dpkg` is always installed on Debian-based systems. `apt-file` requires separate installation.

**CLI:** `stamp provides <file> [-m <manager>]` — displays results from all managers or a single scoped manager.

Alternatives considered:
- Standalone function outside Adapter interface: rejected — breaks the adapter abstraction
- `apt-file` as primary APT provider: rejected — requires separate install
- Normalizing file paths (resolving symlinks): deferred

### 3. Autoremove Command — Interface Method

Orphan cleanup adds a new method to the Adapter interface:

```go
AutoRemove(ctx context.Context, dryRun bool) ([]string, error)
```

**Rationale:** Every major package manager supports orphaned dependency cleanup. An interface method allows each manager to implement its native command with a unified `--dry-run` flag.

**Implementation per manager:**

| Manager | Command (dry-run=false) | Command (dry-run=true) |
|---------|------------------------|------------------------|
| Brew | `brew autoremove` | List orphans (brew's default behavior) |
| DNF | `dnf autoremove` | Same (dnf asks for confirmation) |
| APT | `apt autoremove` | Same (apt asks for confirmation) |
| Pacman | List orphans with `pacman -Qdtq`, then `pacman -Rs --noconfirm` | `pacman -Qdtq` |
| Paru | Same as pacman | Same |
| Zypper | `zypper rm -u` | Same |
| Flatpak | `flatpak uninstall --unused` | Same |
| MacPorts | `port reclaim` | Same |
| Others | Return `ErrNotSupported` | N/A |

**CLI:** `stamp autoremove [--dry-run] [-m <manager>]`

Alternatives considered:
- Single `stamp clean` command combining cache + orphans: deferred — separate concerns, different operations
- Requiring `-y` flag: rejected — `--yes` global flag already exists for non-interactive mode

### 4. Clean Command — Interface Method

Cache cleanup adds a new method to the Adapter interface:

```go
Clean(ctx context.Context, dryRun bool) ([]string, error)
```

**Rationale:** Every major package manager supports cache cleanup. An interface method allows each manager to implement its native command with a unified `--dry-run` flag.

**Implementation per manager:**

| Manager | Command | Notes |
|---------|---------|-------|
| Brew | `brew cleanup [--dry-run]` | |
| DNF | `sudo dnf clean all` | |
| APT | `sudo apt clean` | |
| Pacman | `sudo pacman -Sc --noconfirm` | |
| Paru | `sudo paru -Sc --noconfirm` | |
| Zypper | `sudo zypper clean` | |
| Snap | `snap list --all` + `snap remove --revision` | Removes old revisions |
| MacPorts | `sudo port clean --all installed` | |
| Others | Return `ErrNotSupported` | |

**CLI:** `stamp clean [--dry-run] [-m <manager>]`

Alternatives considered:
- Merging with autoremove: rejected — separate concerns (cache files vs orphaned packages)

### 5. Hold/Unhold — Interface Methods

Version pinning adds three methods to the Adapter interface:

```go
Hold(ctx context.Context, pkg string) error
Unhold(ctx context.Context, pkg string) error
ListHeld(ctx context.Context) ([]string, error)
```

**Rationale:** Production systems need version pinning to prevent accidental upgrades. APT and DNF have native support. Pacman supports via IgnorePkg configuration.

**Implementation per manager:**

| Manager | Hold | Unhold | List |
|---------|------|--------|------|
| APT | `apt-mark hold <pkg>` | `apt-mark unhold <pkg>` | `apt-mark showhold` |
| DNF | `dnf versionlock add <pkg>` | `dnf versionlock delete <pkg>` | `dnf versionlock list` |
| Pacman | Add to IgnorePkg in pacman.conf | Remove from pacman.conf | Parse pacman.conf |
| Paru | Same as pacman | Same | Same |
| Others | Return `ErrNotSupported` | Return `ErrNotSupported` | Return `ErrNotSupported` |

**CLI:** `stamp hold <pkg> -m <mgr>`, `stamp unhold <pkg> -m <mgr>`, `stamp held [-m <mgr>]`

Alternatives considered:
- Config-only approach: rejected — users expect scriptable CLI commands
- Universal pinning system: deferred — each manager has different semantics

### 6. Override Command — Flatpak-Specific Subcommand

Flatpak permission management ships as a CLI-only subcommand, not an interface method.

**Rationale:** Permission management is flatpak-specific. Adding it to the Adapter interface would introduce a flatpak-only concern into every adapter's contract.

**Commands:**
```bash
stamp override firefox -m flatpak --filesystem=host
stamp override firefox -m flatpak --socket=wayland
stamp override firefox -m flatpak --reset
stamp override firefox -m flatpak --list
```

All map to `flatpak override --user [flags] <app-id>`.

Alternatives considered:
- Interface method on Flatpak adapter: rejected — flatpak-only concern doesn't belong in the interface
- Skip entirely: rejected — managing sandbox permissions is a key flatpak differentiator

### 7. Group Install — Flag Extension

DNF group support ships via a `--group` flag on `stamp install` and `stamp remove`. Not an interface method — dnf-only flag behavior.

**Rationale:** Package groups (e.g., "Development Tools") are a DNF/RPM feature that doesn't map to other managers. A `--group` flag is simpler than an interface change.

**Commands:**
```bash
stamp install "Development Tools" -m dnf --group
stamp remove "GNOME Desktop" -m dnf --group
```

Maps to `dnf group install -y <group>` / `dnf group remove -y <group>`.

Alternatives considered:
- Separate `stamp group` subcommand: rejected — adds another command, `--group` flag fits naturally
- Interface method: rejected — DNF-only concern

## Consequences

- Adapter interface gains 4 new methods (`Provides`, `AutoRemove`, `Clean`) and 3 optional methods (`Hold`, `Unhold`, `ListHeld`) — affects mock implementations and all 14 adapters
- Context flag pattern (`WithCask`) sets precedent for future brew-only or manager-specific flags
- Flatpak Override and DNF Group Install ship as CLI-only features (no interface changes)
- All seven features are backward compatible — no existing manifests or workflows break
