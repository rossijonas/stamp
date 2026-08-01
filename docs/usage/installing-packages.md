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
✓ installed htop via apt
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
✓ installed lazygit via brew (note: better git TUI than default)
```

Notes are saved to your manifest so you remember why you installed something.

### Using aliases

```bash
stamp add htop                 # alias for install
stamp reinstall htop           # reinstall and re-track
stamp reinstall -m brew htop   # reinstall with specific manager
```

### DNF package groups

Install a DNF package group with the `--group` / `-g` flag:

```bash
stamp install "Development Tools" -m dnf --group
```

```text
▪ installing group Development Tools via dnf...
✓ installed Development Tools via dnf
```

Group names like "Development Tools" may contain spaces — use quotes.

### Homebrew casks

Stamp auto-detects Homebrew casks (GUI applications) and passes `--cask` automatically:

```bash
stamp install firefox -m brew
```

```text
▪ installing firefox via brew...
✓ installed firefox via brew (cask: true)
```

Casks are recorded in the manifest and restored correctly with `stamp restore`.

### Using show/view aliases

```bash
stamp show htop      # alias for stamp info htop
stamp view htop      # alias for stamp info htop
```

### Reinstall

The `reinstall` command works for both manifest-tracked and pre-existing packages:

```bash
stamp reinstall htop
```

```text
▪ reinstalling htop via apt...
✓ reinstalled htop via apt
```

For pre-existing packages (installed before `stamp init`), reinstall resolves the manager automatically and records the package in the manifest.

### Package name validation

Stamp validates package names to prevent shell injection. Names must start with a letter, number, or underscore, and contain only safe characters (`a-zA-Z0-9_-.+`). Names starting with `-` are rejected.

### Python tools (pipx)

```bash
stamp install black -m pipx
```

Installs Python CLI tools via `pipx install --yes <pkg>`. Requires `pipx` on PATH.

```bash
stamp install ruff -m uv
```

Alternatively, use `uv` (faster) via `uv tool install <pkg>`. Both adapters
are independent — you can have both installed on the same system.

### Go tools

```bash
stamp install github.com/golangci/golangci-lint -m go
```

Go tools require a full module path (e.g., `github.com/example/tool`) and the `-m go` flag.
The go adapter is not in the default precedence, so `-m go` is always required. Short names
like `golangci-lint` are rejected — Stamp cannot derive the module path from a binary name.

Go tools are installed via `go install <module>@latest`. Search, doctor, and repo management
are not supported for the go adapter.

### Aborting an operation

Press Ctrl+C at any point (including the sudo password prompt) to abort cleanly:

- First Ctrl+C — stamp cancels the running command, kills the child process, and restores the terminal. Any in-progress `sudo`/`dnf`/`apt` process is terminated.
- Second Ctrl+C — stamp force-exits with status 130 and kills its entire process group, guaranteeing no orphaned processes are left behind.

### Error handling

If a package is not found, Stamp prints a clear error:

```text
✕ failed to install nonexistent-pkg: exit status 100
Error: install failed
```
