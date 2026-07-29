---
---

## Removing Packages

### Basic remove

```bash
stamp remove htop
```

Stamp looks up the package in your manifest and uses the recorded manager.

```text
▪ removing htop via apt...
✓ removed htop via apt
```

### Specify a manager

```bash
stamp remove lazygit -m brew
```

### Using aliases

```bash
stamp uninstall htop
stamp rm htop
stamp delete htop
stamp del htop
```

All aliases behave identically.

### DNF package groups

Remove a DNF package group with `--group` / `-g`:

```bash
stamp remove "Development Tools" -m dnf --group
```

```text
▪ removing group Development Tools via dnf...
✓ removed Development Tools via dnf
```

Group removal runs `dnf group remove -y <name>` — it removes the group meta-package but not the individual packages that were installed as part of the group.

### What happens

1. Stamp finds the package in the manifest (or uses `-m` override)
2. Runs the native remove command
3. Removes the package from the manifest
4. Saves the updated manifest

### Go tools

```bash
stamp remove github.com/golangci/golangci-lint -m go
```

Go tools require the full module path and `-m go`. If the module was auto-tracked by
`stamp reconcile`, use the same module path shown in `stamp list`.

### Error handling

If the package is not in the manifest and no `-m` is provided:

```text
✕ package htop is not tracked in the manifest
  Use --manager / -m to specify a manager, or stamp reconcile to track it first.
```
