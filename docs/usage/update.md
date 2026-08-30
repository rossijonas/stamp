---
---

## Updating Packages

The `update` (alias `upgrade`) command runs system upgrades across all available package managers using a safe two-phase (check + confirm) flow.

```bash
stamp update
```

### Checking for outdated packages

Use the `outdated` or `check-update` commands to see available updates without applying them:

```bash
stamp outdated
stamp check-update
```

```text
▪ Refreshing package metadata...
  refreshing apt...
  refreshing brew...
▪ Checking for updates...
  apt: curl 7.88.1 → 7.88.3
  apt: git 2.43.0 → 2.43.2
  dnf: htop 3.2.1 → 3.2.2
  brew: lazygit 0.40.0 → 0.41.0
  pipx: cannot preview updates
  uv: cannot preview updates
```

Both `outdated` and `check-update` are read-only — they run the check phase and exit without applying anything. Equivalent to `stamp update --check`.

### Default Flow (Check + Confirm)

By default, running `stamp update` performs metadata refresh, then a serialized check across all package managers, aggregates the available updates, and prompts you before applying them:

```text
▪ Refreshing package metadata...
  refreshing apt...
  refreshing brew...
▪ Checking for updates...
  apt: curl 7.88.1 → 7.88.3
  apt: git 2.43.0 → 2.43.2
  dnf: htop 3.2.1 → 3.2.2
  brew: lazygit 0.40.0 → 0.41.0
  pipx: cannot preview updates (unsupported)
  uv: cannot preview updates (unsupported)

Proceed with updates? [Y/n]: y

▪ sudo password:
[apt] Reading package lists... Done
[apt] Upgrading: 2 packages
[dnf] Upgrading: 1 package
[brew] Upgrading: 1 package
updated packages via apt
updated packages via dnf
updated packages via brew
```

Managers that do not support a native "list upgrades" command (like `go`, `pipx`, `uv`) will print a short notice and continue safely.

To ensure accuracy, the check phase runs an explicit metadata refresh step
before querying for updates. Each adapter's `Refresh` method is called
(e.g., `brew update`, `apt update`, `zypper refresh`, `pacman -Sy`)
before the check loop. Managers like `dnf` and `flatpak` skip refresh
since they use local metadata. A single sudo prompt at the start covers
both the refresh and the run phase.

### Check Only (Dry-Run)

Use `--check` or `-c` to check for available updates without applying them:

```bash
stamp update --check
```

```text
▪ Refreshing package metadata...
  refreshing apt...
  refreshing brew...
▪ Checking for updates...
  apt: curl 7.88.1 → 7.88.3
  brew: lazygit 0.40.0 → 0.41.0
```

### Skip Check (Auto-Confirm)

To skip the check phase entirely for speed (useful in scripts or CI), use the `-y` / `--yes` flag:

```bash
stamp update -y
```

This bypasses the check-update latency and proceeds directly to parallelized updates.

### Scoped to a Manager

Scope the check and update to a single package manager with `-m` / `--manager`:

```bash
stamp update -m apt
```

### Single Package

Update only one specific package instead of all packages (requires `-m`):

```bash
stamp update -p htop -m apt
```

### Serial Mode

Run updates one manager at a time instead of concurrently (useful for debugging):

```bash
stamp update --serial
```

### Python tools (pipx / uv)

```bash
stamp update -m pipx
```

Runs `pipx upgrade-all` to upgrade all pipx-installed tools.

```bash
stamp update -m uv
```

Runs `uv tool upgrade --all` to upgrade all uv-managed tools.

### Go tools

Batch update (`stamp update` without flags) reinstalls all go tools whose module path is recoverable from the binary metadata. Tools installed before the Go module system, or with stripped binaries, may be skipped. Use `-p <module> -m go` to update a single go tool explicitly:

```bash
stamp update -p github.com/golangci/golangci-lint -m go
```

### Error Handling

If one manager fails, others continue. The command exits with a non-zero status:

```text
⚠ update failed for apt: exit status 100
updated packages via brew
Error: one or more managers failed to update
```
