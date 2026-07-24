---
---

## Configuration

Stamp is configured via two files in the XDG config directory (`~/.config/stamp/`):

- `config.toml` — Package manager precedence and routing rules
- `manifest.toml` — Your tracked packages and repositories

### config.toml

Controls how Stamp resolves which package manager to use when none is specified with `-m`.

```toml
# ~/.config/stamp/config.toml

# Global priority order (highest to lowest)
precedence = ["dnf", "flatpak", "brew"]

# Pattern-based routing rules override the global precedence
[[rules]]
pattern = "^com\\..*|^org\\..*"
prefer = "flatpak"

[[rules]]
pattern = "^lib.*|-devel$"
prefer = "dnf"
```

#### precedence

The `precedence` array defines the priority order. When a package exists in multiple managers, Stamp selects the first match in this list.

```toml
precedence = ["dnf", "flatpak", "brew"]
```

#### rules

The `[[rules]]` table allows regex-based routing for specific package name patterns. Each rule has a `pattern` (POSIX regex) and a `prefer` (manager name).

```toml
[[rules]]
pattern = "^com\\..*"
prefer = "flatpak"
```

Rules are evaluated in order. The first match wins. If no rule matches, the global `precedence` is used.

### Resolution order

When running `stamp install <pkg>` without `-m`:

1. **Rules check** — If the package name matches any `[[rules]]` pattern, use that manager
2. **Precedence scan** — Scan the `precedence` list left to right, use the first manager that has the package available
3. **Fallback** — In interactive mode: prompt the user. In non-interactive mode: error

### manifest.toml

The manifest records every package and repository you intentionally install through Stamp.

```toml
# ~/.config/stamp/manifest.toml
version = 1
system = "linux"
updated_at = "2026-07-21T12:00:00Z"

[[packages]]
name = "htop"
manager = "apt"

[[packages]]
name = "lazygit"
manager = "brew"
notes = "better git TUI than default"

[[repositories]]
name = "flathub"
manager = "flatpak"
url = "https://dl.flathub.org/repo/flathub.flatpakrepo"
```

#### version

Manifest schema version. Currently `1`.

#### system

The operating system the manifest was created on (`linux` or `darwin`).

#### [[packages]]

Each entry tracks a package you've installed:

| Field | Required | Description |
|-------|----------|-------------|
| `name` | ✓ | Package name |
| `manager` | ✓ | Manager used to install it |
| `notes` | | Optional description of why you installed it |

#### [[repositories]]

Each entry tracks a third-party repository you've added:

| Field | Required | Description |
|-------|----------|-------------|
| `name` | ✓ | Repository alias |
| `manager` | ✓ | Manager that owns the repository |
| `url` | | Repository URL (when applicable) |

### Storage locations

Stamp follows the XDG Base Directory specification:

| Data | Path |
|------|------|
| Config file | `~/.config/stamp/config.toml` |
| Manifest | `~/.config/stamp/manifest.toml` |
| Snapshots | `~/.local/share/stamp/snapshots/` |
| Man pages | `~/.local/share/man/man1/stamp.1` |
| Completions | `~/.local/share/bash-completion/completions/stamp` (bash) |
| Completions | `~/.local/share/zsh/site-functions/_stamp` (zsh) |
| Completions | `~/.config/fish/completions/stamp.fish` (fish) |
