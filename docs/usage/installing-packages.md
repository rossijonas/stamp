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

_Icons (`▪`, `✓`) are shown on interactive terminals; piped output is plain.

### Specify a manager

```bash
stamp install spotify --manager flatpak
stamp install lazygit -m brew
```

### Install multiple packages

Install several packages in one command. **Batches are per-manager only** —
`-m <manager>` is mandatory, and a single batch never spans managers; to
install packages from different managers, run one command per manager:

```bash
stamp install htop atop btop -m dnf
```

```text
Install 3 package(s) via dnf? [y/N]: y
▪ installing 3 package(s) via dnf...
✓ installed 3 package(s) via dnf
```

Only managers with native multi-package support participate (`go`, `pipx`, and
`uv` reject multi-install; Homebrew falls back to per-package installs when a
batch mixes casks and formulae). All names are validated before anything runs.

### Add a note

```bash
stamp install lazygit -m brew --note "better git TUI than default"
```

```text
▪ installing lazygit via brew...
✓ installed lazygit via brew (note: better git TUI than default)
```

Notes are saved to your manifest so you remember why you installed something.

### Reinstall multiple packages

Reinstall several packages in one command. **Batches are per-manager only** —
`-m <manager>` is mandatory, and a single batch never spans managers:

```bash
stamp reinstall lazygit jq -m brew
```

Only managers with native multi-package reinstall support participate (`snap`
is excluded — its reinstall is remove + install). Single combined confirmation
prompt; snapshots and the manifest are updated once.

If a package in the batch is already tracked under a different manager, the
batch fails fast before anything runs — reinstall that package with its
recorded manager (`-m <mgr>`).

### Using aliases

```bash
stamp add htop                 # alias for install
stamp reinstall htop           # reinstall and re-track
stamp reinstall -m brew htop   # reinstall with specific manager
```

### DNF package groups

Install a DNF package group with the `--group` / `-g` flag. Groups are
referenced by their **group ID** (the first column of `dnf group list`), not
the human-readable display name:

```bash
stamp install development-tools -m dnf --group
```

```text
▪ installing group development-tools via dnf...
✓ installed development-tools via dnf
```

Group IDs contain only lowercase letters, digits, `-` and `_`. Display names
like `"Development Tools"` are rejected — find the ID first with
`stamp search -m dnf --group <query>`.

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
