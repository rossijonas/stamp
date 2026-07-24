---
---

## Installing Packages

### Basic install

```bash
stamp install htop
```

Stamp auto-detects the best package manager for your system.

```text
▪ installing htop via apt...
✅ installed htop via apt
```

### Specify a manager

```bash
stamp install spotify --manager flatpak
stamp install lazygit -m brew
```

### Add a note

```bash
stamp install lazygit -m brew --note "better git TUI than default"
```

```text
▪ installing lazygit via brew...
✅ installed lazygit via brew (note: better git TUI than default)
```

Notes are saved to your manifest so you remember why you installed something.

### Using aliases

```bash
stamp add htop                 # alias for install
stamp reinstall htop           # reinstall and re-track
stamp reinstall -m brew htop   # reinstall with specific manager
```

### Reinstall

The `reinstall` command works for both manifest-tracked and pre-existing packages:

```bash
stamp reinstall htop
```

```text
▪ reinstalling htop via apt...
✅ reinstalled htop via apt
```

For pre-existing packages (installed before `stamp init`), reinstall resolves the manager automatically and records the package in the manifest.

### Package name validation

Stamp validates package names to prevent shell injection. Names must start with a letter, number, or underscore, and contain only safe characters (`a-zA-Z0-9_-.+`). Names starting with `-` are rejected.

### Error handling

If a package is not found, Stamp prints a clear error:

```text
✕ failed to install nonexistent-pkg: exit status 100
Error: install failed
```
