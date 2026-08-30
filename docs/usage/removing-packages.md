---
---

## Removing Packages

### Basic remove

```bash
stamp remove htop
```

Stamp looks up the package in your manifest and uses the recorded manager.

### Remove multiple packages

Remove several packages in one command. **Batches are per-manager only** —
`-m <manager>` is mandatory, and a single batch never spans managers:

```bash
stamp remove htop atop btop -m dnf
```

Only managers with native multi-package support participate. Single combined
confirmation prompt; the manifest is updated once.

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

Remove a DNF package group with `--group` / `-g`. Like install, groups are
referenced by their **group ID** (see `dnf group list`), not the display name:

```bash
stamp remove development-tools -m dnf --group
```

```text
▪ removing group development-tools via dnf...
✓ removed development-tools via dnf
```

Group removal runs `dnf group remove -y <id>` — it removes the group meta-package but not the individual packages that were installed as part of the group.

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

When the package is not in the manifest and no `-m` is provided, the remove
command falls back to the first available package manager. Use `-m` to target
a specific manager explicitly.
