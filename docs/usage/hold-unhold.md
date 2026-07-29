---
---

## Version Pinning

Pin packages at specific versions to prevent accidental upgrades.

### Hold a package

```bash
stamp hold nginx -m apt
```

```text
held nginx via apt
```

### Remove a version pin

```bash
stamp unhold nginx -m apt
```

```text
unheld nginx via apt
```

### List all held packages

```bash
stamp held
```

```text
nginx (apt)
redis (apt)
```

### List held packages for a specific manager

```bash
stamp held -m dnf
```

```text
nginx (dnf)
```

### No packages held

```bash
stamp held
```

```text
no packages held
```

### Per-manager details

| Manager | Hold command | Unhold command | List command |
|---------|-------------|---------------|-------------|
| APT | `sudo apt-mark hold <pkg>` | `sudo apt-mark unhold <pkg>` | `apt-mark showhold` |
| DNF | `sudo dnf versionlock add <pkg>` | `sudo dnf versionlock delete <pkg>` | `dnf versionlock list` |
| Pacman | Adds to `IgnorePkg` in `/etc/pacman.conf` | Removes from `IgnorePkg` | Parses `IgnorePkg` |
| Paru | Same as pacman | Same as pacman | Same as pacman |
| Others | ✗ Not supported | ✗ Not supported | ✗ Not supported |

Hold/unhold requires `--manager` flag. Listing `stamp held` aggregates across all supported managers.
