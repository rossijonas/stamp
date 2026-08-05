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

# Backup retention policy (logrotate-style)
[backup]
max_manifest_backups = 10
min_manifest_backups = 3
min_manifest_backup_age_days = 7
max_manifest_backup_age_days = 30
max_snapshot_backups = 10
min_snapshot_backups = 3
min_snapshot_backup_age_days = 7
max_snapshot_backup_age_days = 30

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

The go adapter is not included in the default precedence because go module paths (which
always contain `/`) do not match standard package name validation. Use the `-m go` flag
explicitly when working with go tools. To auto-route go module paths, add a rule:

```toml
[[rules]]
pattern = "github\\.com/.*"
prefer = "go"
```

#### rules

The `[[rules]]` table allows regex-based routing for specific package name patterns. Each rule has a `pattern` (POSIX regex) and a `prefer` (manager name).

```toml
[[rules]]
pattern = "^com\\..*"
prefer = "flatpak"
```

Rules are evaluated in order. The first match wins. If no rule matches, the global `precedence` is used.

#### backup

Controls timestamped backup retention for the manifest and snapshots. Stamp writes a backup before rewriting either file and prunes old backups per this policy. The semantics mirror logrotate's `rotate`, `minage`, and `maxage` directives; a value of `0` on any key means **unlimited** on that axis.

| Key | Default | Logrotate equivalent | Meaning |
|-----|---------|----------------------|---------|
| `max_manifest_backups` | `10` | `rotate` | Max manifest backup files to keep |
| `min_manifest_backups` | `3` | — | Always keep at least this many manifest backups |
| `min_manifest_backup_age_days` | `7` | `minage` | Backups younger than this are never deleted |
| `max_manifest_backup_age_days` | `30` | `maxage` | Backups older than this are always deleted |
| `max_snapshot_backups` | `10` | `rotate` | Max snapshot backup dirs to keep |
| `min_snapshot_backups` | `3` | — | Always keep at least this many snapshot backups |
| `min_snapshot_backup_age_days` | `7` | `minage` | Snapshot backups younger than this are never deleted |
| `max_snapshot_backup_age_days` | `30` | `maxage` | Snapshot backups older than this are always deleted |

**Precedence (highest to lowest):** (1) min-age floor — backups younger than the min age are protected and never deleted; (2) min-count floor — at least `min_*_backups` backups are always kept (the newest survive, so the max-age ceiling can never wipe the set to zero); (3) max-age ceiling — eligible backups older than the max age are deleted, except those needed to meet the min-count floor; (4) count cap — if the eligible set still exceeds the max count, the oldest surplus are deleted, except those needed to meet the min-count floor. Avoid `min_*_backup_age_days > max_*_backup_age_days` (the floor wins on the overlap; `stamp doctor` reports it as invalid). If `min_*_backups > max_*_backups`, the min-count floor wins.

`stamp reconcile` rotates manifest backups only; `stamp init` re-init rotates both manifest and snapshot backups. Backup files are named `manifest.toml.<YYYYMMDDTHHMMSSZ>.bak`; snapshot backup dirs `snapshots.<YYYYMMDDTHHMMSSZ>.bak/`.

`config.toml` is auto-created by `stamp init` when absent (a commented template with the defaults above). An existing file is never overwritten.

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
origin = "stamped"

[[repositories]]
name = "flathub"
manager = "flatpak"
url = "https://dl.flathub.org/repo/flathub.flatpakrepo"
origin = "stamped"
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
| `origin` | | Provenance: `stamped` (user action) or `reconciled` (auto-tracked); absent = `stamped` |

#### [[repositories]]

Each entry tracks a third-party repository you've added:

| Field | Required | Description |
|-------|----------|-------------|
| `name` | ✓ | Repository alias |
| `manager` | ✓ | Manager that owns the repository |
| `url` | | Repository URL (when applicable) |
| `origin` | | Provenance: `stamped` (user action) or `reconciled` (auto-tracked); absent = `stamped` |

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
