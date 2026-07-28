---
---

# Aliases Matrix

Every stamp command has aliases matching your favorite package manager's native syntax.

## Forward Map (alias → canonical)

| Alias | Canonical | Origin |
|-------|-----------|--------|
| `stamp show <pkg>` | `stamp info <pkg>` | APT, DNF, Snap |
| `stamp outdated` | `stamp update --check` | Brew, npm, Bun |
| `stamp check-update` | `stamp update --check` | DNF |
| `stamp refresh` | `stamp update` | Snap |
| `stamp view <pkg>` | `stamp info <pkg>` | npm |
| `stamp tap <name>` | `stamp repo add <name> -m brew` | Brew |
| `stamp untap <name>` | `stamp repo remove <name> -m brew` | Brew |
| `stamp taps` | `stamp repo list -m brew` | Brew |
| `stamp list` / `stamp ls` | `stamp list` | npm, Pipx, Uv |
| `stamp add <pkg>` | `stamp install <pkg>` | Bun |
| `stamp uninstall <pkg>` | `stamp remove <pkg>` | npm, Pipx, Cargo |
| `stamp rm <pkg>` | `stamp remove <pkg>` | Common |
| `stamp upgrade` | `stamp update` | Common |
| `stamp self-upgrade` | `stamp self-update` | Common |

## Reverse Map (by manager)

| Tool | Native Command | Stamp Equivalent |
|------|---------------|-----------------|
| APT / DNF | `apt show` / `dnf info` | `stamp show <pkg>` |
| APT | `apt list --upgradable` | `stamp outdated` |
| DNF | `dnf check-update` | `stamp check-update` |
| Brew | `brew outdated` | `stamp outdated` |
| Brew | `brew tap <name>` | `stamp tap <name>` |
| Brew | `brew untap <name>` | `stamp untap <name>` |
| Brew | `brew tap` (list) | `stamp taps` |
| Snap | `snap refresh` | `stamp refresh` |
| npm | `npm view <pkg>` | `stamp view <pkg>` |
| npm / Bun | `npm outdated` / `bun outdated` | `stamp outdated` |
| npm / Bun | `npm update` / `bun update` | `stamp update` |
| npm | `npm uninstall` | `stamp uninstall` |
| Bun | `bun add` | `stamp add` |
| Cargo | `cargo uninstall` | `stamp uninstall` |
| Pipx | `pipx uninstall` | `stamp uninstall` |
| Pipx / Uv | `pipx list` / `uv tool list` | `stamp list` |

## Reverse Map (by stamp command)

| Canonical | Aliases |
|-----------|---------|
| `stamp info <pkg>` | `show`, `view` |
| `stamp update` | `upgrade`, `refresh` |
| `stamp update --check` | `outdated`, `check-update` |
| `stamp install <pkg>` | `add` |
| `stamp remove <pkg>` | `uninstall`, `rm`, `delete`, `del` |
| `stamp repo add <name>` | `tap` |
| `stamp repo remove <name>` | `untap` |
| `stamp repo list` | `taps` |
| `stamp list` | `ls` |
| `stamp self-update` | `self-upgrade` |
