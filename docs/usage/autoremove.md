---
---

## Removing Orphaned Packages

Remove orphaned dependencies that were pulled in automatically but are no longer needed.

### Remove orphans from all managers

```bash
stamp autoremove
```

```text
  brew: removed 2 package(s)
  dnf: cleaned
  flatpak: cleaned
```

Managers without a native autoremove command are silently skipped.

### Preview orphans

Use `--dry-run` / `-d` to see which packages would be removed:

```bash
stamp autoremove --dry-run
```

```text
  pacman: would remove 3 package(s)
    - libfoo
    - libbar
    - libbaz
```

Only managers that can list orphans independently (Pacman, Paru) show specific package names in dry-run mode.

### Scoped to a manager

```bash
stamp autoremove -m brew
```

```text
  brew: removed 2 package(s)
```

### Per-manager details

| Manager | Command | Dry-run |
|---------|---------|---------|
| Brew | `brew autoremove` | ✗ (no preview) |
| DNF | `sudo dnf autoremove -y` | ✗ |
| APT | `sudo apt autoremove -y` | ✗ |
| Pacman | List: `pacman -Qdtq`, Remove: `sudo pacman -Rs` | ✓ lists orphans |
| Paru | Same as pacman | ✓ lists orphans |
| Zypper | `sudo zypper rm -u --non-interactive` | ✗ |
| Flatpak | `flatpak uninstall --unused -y` | ✗ |
| MacPorts | `sudo port reclaim --yes` | ✗ |
| Others | ✗ Not supported | — |
