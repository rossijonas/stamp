---
---

## Viewing Package Info

### Multi-manager summary

Query package details across all managers:

```bash
stamp info htop
```

```text
htop:
  apt:       v3.2.1
  dnf:       v3.2.2
  brew:      v3.2.1
```

If no manager has the package:

```text
htop: not found in any package manager
```

### Full raw output (scoped)

Use `-m` / `--manager` to see the native manager's full raw output:

```bash
stamp info htop -m dnf
```

```text
htop via dnf:

Name        : htop
Version     : 3.2.2
Release     : 1.fc40
Architecture: x86_64
Size        : 147 kB
Repository  : updates
Summary     : Interactive process viewer
URL         : https://htop.dev/
License     : GPL-2.0-only
```

### JSON output

```bash
stamp info htop --json
```

```json
{
  "Package": "htop",
  "Results": [
    {"Manager": "apt", "Found": true, "Info": "..."},
    {"Manager": "dnf", "Found": false}
  ]
}
```

### DNF group info

Use `--group` / `-g` to get info about a DNF package group:

```bash
stamp info "Development Tools" -m dnf --group
```

```text
Development Tools via dnf:

Group: Development Tools
Description: A basic development environment.
Mandatory Packages: gcc, gcc-c++, make, git
...
```

### Per-manager info support

| Manager | Info command | Raw output | JSON |
|---------|-------------|------------|------|
| APT | `apt show` / `apt-cache show` | ✓ | ✓ |
| DNF | `dnf info` | ✓ | ✓ |
| Brew | `brew info` | ✓ | ✓ |
| Pacman | `pacman -Qi` | ✓ | ✓ |
| All others | Varies | ✓ | ✓ |
