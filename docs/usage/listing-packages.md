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
| Package name | The name of the installed package |
| Manager | The package manager used to install it (in parentheses) |
| Note | Any user-provided note (shown after em dash if present) |

Only intentionally installed packages appear in the list — no dependency noise.
