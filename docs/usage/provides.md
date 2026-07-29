---
---

## Finding File Owners

Find which package owns a specific file or binary (useful when a command is missing).

### Query all managers

```bash
stamp provides /usr/bin/htop
```

```text
htop-3.2.2-1.fc40.x86_64 (dnf)
htop: /usr/bin/htop (apt)
```

Results are tagged with the manager name in parentheses.

### Scoped to a manager

```bash
stamp provides /usr/bin/htop -m dnf
```

```text
htop-3.2.2-1.fc40.x86_64 (dnf)
```

### No results

```bash
stamp provides /usr/bin/nonexistent
```

```text
no packages provide /usr/bin/nonexistent
```

### Per-manager support

| Manager | Command | Notes |
|---------|---------|-------|
| DNF | `dnf provides <file>` | |
| APT | `dpkg -S <file>` | Always available (not `apt-file`) |
| Pacman | `pacman -F <file>` | Requires `pacman -Fy` to sync file database first |
| Paru | `pacman -F <file>` | Same as pacman |
| Zypper | `zypper what-provides <file>` | |
| MacPorts | `port provides <file>` | |
| Others | ✗ Not supported | — |

Managers without provides support are silently skipped.
