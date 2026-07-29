---
---

## Cleaning Caches

Remove locally cached package files across all managers to free disk space.

### Clean all managers

```bash
stamp clean
```

```text
  brew: cleaned 3 item(s)
  dnf: cleaned
  apt: cleaned
```

Managers without a native cache clean command are silently skipped.

### Scoped to a manager

```bash
stamp clean -m brew
```

```text
  brew: cleaned 3 item(s)
```

### Preview without deleting

Use `--dry-run` / `-d` to see what would be cleaned:

```bash
stamp clean --dry-run
```

```text
  brew: would clean 2 item(s)
```

### Per-manager details

| Manager | Command | Dry-run support |
|---------|---------|----------------|
| Brew | `brew cleanup` | ✓ `--dry-run` |
| DNF | `sudo dnf clean all` | ✗ |
| APT | `sudo apt clean` | ✗ |
| Pacman | `sudo pacman -Sc --noconfirm` | ✗ |
| Paru | `sudo paru -Sc --noconfirm` | ✗ |
| Zypper | `sudo zypper clean --non-interactive` | ✗ |
| Snap | Old revisions removal via `snap remove --revision` | ✓ (shows old revisions) |
| MacPorts | `sudo port clean --all installed` | ✗ |
| Others | ✗ Not supported | — |
