---
---

## Updating Packages

```bash
stamp update
```

Runs the upgrade command for all available package managers concurrently.

```text
▪ Authentication required for system package managers
[apt] Reading package lists... Done
[apt] Upgrading: 3 packages
[brew] Already up-to-date.
[flatpak] Looking for updates... Done
✓ updated packages via apt
✓ updated packages via flatpak
```

### Scoped to a manager

```bash
stamp update -m apt
```

```text
[apt] Reading package lists... Done
[apt] Upgrading: 3 packages
✓ updated packages via apt
```

### Single package

Update only one specific package instead of all packages:

```bash
stamp update -p htop -m apt
```

```text
[apt] Reading package lists... Done
[apt] Upgrading: 1 packages
✓ updated htop via apt
```

The `-m` / `--manager` flag is required when using `-p`.

### Aliases

```bash
stamp upgrade
```

### Serial mode

```bash
stamp update --serial
```

Runs updates one manager at a time (useful for debugging):

```text
▪ Authentication required for system package managers
▪ updating via apt...
[apt] Reading package lists... Done
[apt] Upgrading: 3 packages
✓ updated packages via apt
▪ updating via brew...
Already up-to-date.
✓ updated packages via brew
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

Batch update (`stamp update` without flags) reinstalls all go tools whose module path
is recoverable from the binary metadata. Tools installed before the Go module system,
or with stripped binaries, may be skipped. Use `-p <module> -m go` to update a single
go tool explicitly:

```bash
stamp update -p github.com/golangci/golangci-lint -m go
```

### Error handling

If one manager fails, others continue. The command exits with a non-zero status:

```text
⚠ update failed for apt: exit status 100
✓ updated packages via brew
Error: one or more managers failed to update
```
