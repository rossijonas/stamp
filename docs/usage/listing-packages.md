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
  {"name": "htop", "manager": "apt", "note": ""},
  {"name": "lazygit", "manager": "brew", "note": "better git TUI than default"},
  {"name": "spotify", "manager": "flatpak", "note": ""}
]
```

### Filter by manager

```bash
stamp list -m brew
```

```text
lazygit (brew) — better git TUI than default
```

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
