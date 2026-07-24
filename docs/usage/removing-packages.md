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
✅ removed htop via apt
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

### What happens

1. Stamp finds the package in the manifest (or uses `-m` override)
2. Runs the native remove command
3. Removes the package from the manifest
4. Saves the updated manifest

### Error handling

If the package is not in the manifest and no `-m` is provided:

```text
✕ package htop is not tracked in the manifest
  Use --manager / -m to specify a manager, or stamp reconcile to track it first.
```
