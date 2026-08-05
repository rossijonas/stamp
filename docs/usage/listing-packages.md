---
---

## Listing Packages

### List all tracked packages

```bash
stamp list
```

Shows every package recorded in your manifest.

```text
htop (apt)
lazygit (brew) — better git TUI than default
spotify (flatpak)
```

### JSON output

```bash
stamp list --json
```

```json
[
  {"Name": "htop", "Manager": "apt", "Notes": "", "Origin": "stamped"},
  {"Name": "lazygit", "Manager": "brew", "Notes": "better git TUI than default", "Origin": "stamped"},
  {"Name": "spotify", "Manager": "flatpak", "Notes": "", "Origin": "stamped"}
]
```

### Filter by manager

```bash
stamp list -m brew
```

```text
lazygit (brew) — better git TUI than default
```

### Filter by entity type and origin

Use `--type, -t` to filter by entity type (packages vs repos) and origin
(stamped vs reconciled):

```bash
stamp list -t repos                        # all repositories
stamp list -t stamped                      # packages + repos installed via stamp
stamp list -t reconciled                   # everything discovered by reconcile
stamp list -t stamped-packages             # packages installed via stamp only
stamp list -t reconciled-repos -m dnf      # repos reconcile discovered, dnf only
stamp list -t stamped-repos -j             # stamp-added repos, as JSON
```

Valid values: `packages` (default), `repos`, `stamped`, `reconciled`,
`stamped-packages`, `stamped-repos`, `reconciled-packages`, `reconciled-repos`.

```text
# stamp list -t stamped
htop (dnf) — system monitor
lazygit (brew) — better git TUI than default
my-tap (brew)

# stamp list -t reconciled
NetworkManager (dnf)
brave-browser (dnf) https://example.com/brave
```

### Understanding Origins

Every package and repository in the manifest has an origin that records how
stamp learned about it:

- **stamped** — The user installed this via `stamp install` or
  `stamp reinstall`. Stamp recorded intent at install time. This is the
  recommended way to track packages.
- **reconciled** — The user installed this directly via their package manager
  (e.g. `dnf install`) and stamp later discovered it through `stamp reconcile`.
  These packages are also user-installed, but stamp learned about them after
  the fact.

Both origins represent packages the user installed — the difference is how
stamp learned about them.

Use `stamp list -t stamped` to see everything you explicitly tracked through
stamp. Use `stamp list -t reconciled` to see everything reconcile
auto-discovered. Use `stamp list -t stamped-packages` to see only packages
(excluding repos) you installed via stamp.

### Alias

```bash
stamp ls
```

### What you see

| Column | Description |
|--------|-------------|
| Package name | The name of the installed package or module path |
| Manager | The package manager used to install it (in parentheses) |
| Note | Any user-provided note (shown after em dash if present) |

Go tools appear as module paths (e.g., `github.com/golangci/golangci-lint`) when the path
is recoverable from the binary metadata. If the module path is not recoverable, the binary
name is shown instead.

Only intentionally installed packages appear in the list — no dependency noise.
