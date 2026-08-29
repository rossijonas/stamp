---
---

## Searching Packages

### Basic search

Search for a package across all available package managers:

```bash
stamp search htop
```

```text
htop (apt)
htop (dnf)
htop (brew)
```

### Scoped to a manager

Limit search to a single package manager with `-m` / `--manager`:

```bash
stamp search lazygit -m brew
```

```text
lazygit (brew)
```

### Search DNF package groups

Use `--group` / `-g` to search DNF package groups instead of individual
packages. Results are **group IDs**, ready to copy into
`stamp install <id> -m dnf --group`:

```bash
stamp search development -m dnf --group
```

```text
c-development (dnf)
development-tools (dnf)
```

### No results

```bash
stamp search xyznonexistent
```

```text
no results found
```

### Using aliases

```bash
stamp search htop -m apt
stamp search htop -m dnf
```

### Per-manager search notes

| Manager | Supported | Notes |
|---------|-----------|-------|
| APT | ✓ | Uses `apt-cache search` |
| DNF | ✓ | Uses `dnf search -q` |
| Brew | ✓ | Uses `brew search` |
| Pacman | ✓ | Uses `pacman -Ss` |
| Paru | ✓ | Uses `paru -Ss` (includes AUR) |
| Flatpak | ✓ | Uses `flatpak search --columns=application` |
| Zypper | ✓ | Uses `zypper search` |
| Snap | ✓ | Uses `snap find` |
| MacPorts | ✓ | Uses `port search` |
| Go | ✗ | Not supported |
| Npm | ✗ | Not supported |
| Pipx | ✗ | Not supported |
| Uv | ✗ | Not supported |
| Cargo | ✓ | Uses `cargo search` |

Managers without search support print a warning and are skipped when searching without `-m`.
