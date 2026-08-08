---
---

# Feature Reference

This document tracks all features, commands, and manager support across Stamp.

See also: [Technical Spec](spec.html), [OS × Manager Compatibility Matrix](../history/os-manager-matrix.html).

---

## Adapters

| Adapter | Status | Commands | Notes |
| :--- | :---: | :--- | :--- |
| DNF / YUM | ✓ Complete | All | Fedora/RHEL, sudo for write ops, yum alias |
| APT / apt-get | ✓ Complete | All | Debian/Ubuntu, sudo for write ops, dpkg-query fallback, add-apt-repository for PPAs |
| Brew | ✓ Complete | All | macOS, user-space, two-phase update, cask support for GUI apps |
| Flatpak | ✓ Complete | All | Linux sandboxed, -y flag |
| Snap | ✓ Complete | All except repo mgmt | Ubuntu, Linux (universal), sudo for write ops |
| Zypper | ✓ Complete | All except repo mgmt | openSUSE/SLE, sudo for write ops |
| Paru | ✓ Complete | All except repo mgmt | Arch Linux, AUR + official repos, replaces pacman when available, sudo for write ops |
| MacPorts | ✓ Complete | All except repo mgmt | macOS, sudo for write ops |
| Pipx | ✓ Complete | Install, Remove, Info, ListInstalled, Update (single + batch) | pip-installable CLI tools, --yes for non-interactive, JSON listing with text fallback, no search/doctor/repo support |
| Uv | ✓ Complete | Install, Remove, Info, ListInstalled, Update (single + batch) | Rust-based Python tool manager (uv tool subcommand), --force for reinstall, no search/doctor/repo support |
| Go | ✓ Complete | Install, Remove, Info, ListInstalled, Update (single + batch) | go install with module path validation, no search/doctor/repo support, ListInstalled recovers module paths via binary metadata, batch update reinstalls recoverable paths, binary-name fallback for unrecoverable |
| Npm | ✓ Complete | Install, Remove, Info, ListInstalled, Update (single + batch) | Node.js package manager for globally installed CLI tools, no search/doctor/repo support, user-space |
| Cargo | ✓ Complete | Install, Remove, Search, Info, ListInstalled, Update (single + batch) | Rust crate installer via cargo install, native search/info via crates.io, --force for reinstall, no doctor/repo support, user-space |

---

## CLI Commands

| Command | Aliases | SPEC.md | Implemented | Wired to Logic | Status |
| :--- | :--- | :---: | :---: | :---: | :---: |
| `stamp install <pkg>...` | `add` | ✓ | ✓ | ✓ Resolver → adapter → manifest; multi-package batch with `-m` (per-manager only) | ✓ Complete |
| `stamp remove <pkg>...` | `uninstall`, `rm`, `delete`, `del` | ✓ | ✓ | ✓ Manifest lookup + adapter; multi-package batch with `-m` (per-manager only) | ✓ Complete |
| `stamp reinstall <pkg>...` | | ✓ | ✓ | ✓ Manifest-tracked + pre-existing via resolver + `Reinstall` adapter method; multi-package batch with `-m` (per-manager only, snap excluded) | ✓ Complete |
| `stamp search <query>` | | ✓ | ✓ | ✓ Queries adapters | ✓ Complete |
| `stamp info <pkg>` | | ✓ | ✓ | ✓ Queries adapter Info() | ✓ Complete |
| `stamp repo add <name> [url]` | `install` | ✓ | ✓ | ✓ Adapter + manifest (--manager required) | ✓ Complete |
| `stamp repo remove <name>` | `uninstall`, `rm`, `delete`, `del` | ✓ | ✓ | ✓ Adapter + manifest (--manager optional when tracked) | ✓ Complete |
| `stamp repo list` | `ls` | ✓ | ✓ | ✓ Reads manifest | ✓ Complete |
| `stamp reconcile` | | ✓ | ✓ | ✓ Auto-track + `--dry-run` + no prompt + repo drift detection + missing-package warning | ✓ Complete |
| `stamp restore` | | ✓ | ✓ | ✓ Sequentially adds repos then concurrently installs packages | ✓ Complete |
| `stamp doctor` | | ✓ | ✓ | ✓ Adapter check + manifest check + manifest-vs-system check + `[backup]` config validation + compliance report | ✓ Complete |
| `stamp completion [shell]` | | ✓ | ✓ | ✓ Auto-detect, install to path, --stdout flag | ✓ Complete |
| `stamp man` | | ✓ | ✓ | ✓ Shows help for man command group | ✓ Complete |
| `stamp hello` | | ✓ | ✓ | ✓ Prints ASCII logo + suggested next steps | ✓ Complete |
| `stamp setup` | `hello` | ✓ | ✓ | ✓ Interactive wizard for completions, man, init, doctor | ✓ Complete |
| `stamp init` | | ✓ | ✓ | ✓ Creates dirs + manifest + snapshots | ✓ Complete |
| `stamp update` | `upgrade` | ✓ | ✓ | ✓ Two-phase check + confirm, --check flag, -y skips check, parallel execution, --serial flag | ✓ Complete |
| `stamp list` | `ls` | ✓ | ✓ | ✓ Reads manifest, filter by type/origin/missing | ✓ Complete |
| `stamp manifest` | | ✓ | ✓ | ✓ Backup history + diff against backups | ✓ Complete |
| `stamp manifest history` | `backups` | ✓ | ✓ | ✓ List backups with counts + hashes | ✓ Complete |
| `stamp manifest diff` | | ✓ | ✓ | ✓ Compare current vs backup, ts/hash refs, filters | ✓ Complete |
| `stamp self-update` | `self-upgrade` | ✓ | ✓ | ✓ Atomic binary replacement + SHA-256 verification + post-update hooks | ✓ Complete |
| `stamp auto-reconcile on\|off` | | ✓ | ✓ | ✓ systemd/launchd timer, --period flag | ✓ Complete |
| `stamp autoremove` | | ✓ | ✓ | ✓ Iterates adapters, handles ErrNotSupported | ✓ Complete |
| `stamp clean` | | ✓ | ✓ | ✓ Iterates adapters, handles ErrNotSupported | ✓ Complete |
| `stamp provides <file>` | | ✓ | ✓ | ✓ Per-adapter Provides(), appends manager name | ✓ Complete |
| `stamp hold <pkg>` | | ✓ | ✓ | ✓ Per-adapter Hold(), --manager required | ✓ Complete |
| `stamp unhold <pkg>` | | ✓ | ✓ | ✓ Per-adapter Unhold(), --manager required | ✓ Complete |
| `stamp held` | | ✓ | ✓ | ✓ Aggregates ListHeld across adapters | ✓ Complete |
| `stamp override <app-id>` | | ✓ | ✓ | ✓ Flatpak-only, CLI command via type assertion | ✓ Complete |
| `stamp outdated` | | ✓ | ✓ | ✓ Delegates to runCheck (same as update --check) | ✓ Complete |
| `stamp check-update` | | ✓ | ✓ | ✓ Delegates to runCheck (same as update --check) | ✓ Complete |
| `stamp tap <name>` | | ✓ | ✓ | ✓ Delegates to brew AddRepo | ✓ Complete |
| `stamp untap <name>` | | ✓ | ✓ | ✓ Delegates to brew RemoveRepo | ✓ Complete |
| `stamp taps` | | ✓ | ✓ | ✓ Delegates to brew ListRepos | ✓ Complete |

### Repository Commands

| Command | Aliases | SPEC.md | Implemented | Wired to Logic | Status |
| :--- | :--- | :---: | :---: | :---: | :---: |
| `stamp repo add <name> [url]` | `install` | ✓ | ✓ | ✓ Adapter + manifest (--manager required) | ✓ Complete |
| `stamp repo remove <name>` | `uninstall`, `rm`, `delete`, `del` | ✓ | ✓ | ✓ Adapter + manifest (--manager optional when tracked) | ✓ Complete |
| `stamp repo list` | `ls` | ✓ | ✓ | ✓ Reads manifest | ✓ Complete |

### Man Command (Subcommands)

| Command | Flags | SPEC.md | Implemented | Status |
| :--- | :--- | :---: | :---: | :---: |
| `stamp man` | | ✓ | ✓ Shows help for man command group | ✓ Complete |
| `stamp man install` | `--prefix` | ✓ | ✓ Installs stamp.1 to system path | ✓ Complete |
| `stamp man check` | | ✓ | ✓ Verifies installed version matches binary | ✓ Complete |

### Global Flags

| Flag | Short | SPEC.md | Implemented | Status |
| :--- | :--- | :---: | :---: | :---: |
| `--verbose` | `-v` | ✓ | ✓ Registered in root PersistentFlags | ✓ Complete |
| `--json` | `-j` | ✓ | ✓ Registered in root PersistentFlags | ✓ Complete |
| `--yes` | `-y` | ✓ | ✓ Registered in root PersistentFlags | ✓ Complete |

### Per-Command Flags

| Command | Flag | Short | SPEC.md | Implemented | Status |
| :--- | :--- | :---: | :---: | :---: | :---: |
| `stamp install` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp install` | `--note <text>` | `-n` | ✓ | ✓ | ✓ Complete |
| `stamp remove` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp search` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp info` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp restore` | `--dry-run` | `-d` | ✓ | ✓ | ✓ Complete |
| `stamp doctor` | `--json` | `-j` | ✓ | ✓ | ✓ Complete |
| `stamp man install` | `--prefix` | | ✓ | ✓ | ✓ Complete |
| `stamp self-update` | `--check` | | ✓ | ✓ | ✓ Complete |
| `stamp completion` | `--stdout` | `-s` | ✓ | ✓ | ✓ Complete |
| `stamp list` | `--json` | `-j` | ✓ | ✓ | ✓ Complete |
| `stamp repo list` | `--json` | `-j` | ✓ | ✓ | ✓ Complete |
| `stamp reconcile` | `--dry-run` | `-d` | ✓ | ✓ | ✓ Complete |
| `stamp reconcile` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp restore` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp repo list` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp doctor` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp update` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp update` | `--package <pkg>` | `-p` | ✓ | ✓ | ✓ Complete |
| `stamp update` | `--serial` | `-s` | ✓ | ✓ | ✓ Complete |
| `stamp auto-reconcile` | `--period <interval>` | `-p` | ✓ | ✓ | ✓ Complete |
| `stamp list` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp list` | `--type <type>` | `-t` | ✓ | ✓ | ✓ Complete |
| `stamp manifest history` | `--json` | `-j` | ✓ | ✓ | ✓ Complete |
| `stamp manifest diff` | `--json` | `-j` | ✓ | ✓ | ✓ Complete |
| `stamp manifest diff` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp manifest diff` | `--origin` | | ✓ | ✓ | ✓ Complete |
| `stamp autoremove` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp autoremove` | `--dry-run` | `-d` | ✓ | ✓ | ✓ Complete |
| `stamp clean` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp clean` | `--dry-run` | `-d` | ✓ | ✓ | ✓ Complete |
| `stamp provides` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp hold` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp unhold` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp held` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp override` | `--manager <name>` | `-m` | ✓ | ✓ | ✓ Complete |
| `stamp override` | `--filesystem <value>` | | ✓ | ✓ | ✓ Complete |
| `stamp override` | `--socket <value>` | | ✓ | ✓ | ✓ Complete |
| `stamp override` | `--device <value>` | | ✓ | ✓ | ✓ Complete |
| `stamp override` | `--env <key=value>` | | ✓ | ✓ | ✓ Complete |
| `stamp override` | `--reset` | | ✓ | ✓ | ✓ Complete |
| `stamp override` | `--show` | | ✓ | ✓ | ✓ Complete |
| `stamp override` | `--system` | | ✓ | ✓ | ✓ Complete |
| `stamp install` | `--group` | `-g` | ✓ | ✓ | ✓ Complete |
| `stamp remove` | `--group` | `-g` | ✓ | ✓ | ✓ Complete |
| `stamp search` | `--group` | `-g` | ✓ | ✓ | ✓ Complete |
| `stamp info` | `--group` | `-g` | ✓ | ✓ | ✓ Complete |

---

## Per-Manager Feature Support

| Feature | DNF | APT | Pacman | Paru | Zypper | Snap | Flatpak | Brew | MacPorts | Go | Pipx | Uv | Npm | Cargo |
|---------|:---:|:---:|:------:|:----:|:------:|:----:|:-------:|:----:|:--------:|:--:|:----:|:--:|:---:|:-----:|
| **Install** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **Reinstall** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **Remove** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **Search** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✓ |
| **Info** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **Update** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **CheckUpdate** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **Doctor** | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **Cask** | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **Hold** | ✗ | ✓ | ✓ | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **Add Repo** | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **Remove Repo** | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **List Repos** | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |

### Notes

- **Search:** Cargo is the only toolchain adapter with native search support (`cargo search`). Other toolchains (Go, Npm, Pipx, Uv) do not support search.
- **Update:** All managers support both batch (`stamp update`) and single-package (`stamp update -p <pkg> -m <mgr>`) updates.
- **Doctor:** Only Brew has a native `doctor` command. Other managers print an informational message and continue.
- **Cask:** Brew-specific feature for macOS GUI applications (`brew install --cask`). Flatpak and Snap support GUI apps natively on Linux through their standard install flow — no separate flag needed.
- **Hold:** APT (apt-mark), DNF (dnf versionlock), Pacman/Paru (IgnorePkg in pacman.conf). Other managers do not support version pinning.
- **Repo management:** DNF supports COPR repos and `.repo` file URLs (fetched verbatim, gpg settings preserved; URL-added repos are removed by deleting the `.repo` file, COPR repos via `dnf copr disable`), APT supports PPAs and custom URLs, Brew supports taps, Flatpak supports remotes. Other managers do not support third-party repository management through Stamp.
- **CheckUpdate:** Toolchain managers (Go, Pipx, Uv) cannot preview updates. They print an informational notice during `stamp update --check` and continue.
- **Backup retention:** The `[backup]` section of `config.toml` controls timestamped backup retention (logrotate-style: max/min count + min/max age axes, `0` = unlimited). `stamp reconcile` rotates manifest backups; `stamp init` re-init rotates manifest + snapshot backups. Misconfigurations are reported by `stamp doctor`.
- **Batch package operations:** `stamp install`/`remove`/`reinstall` accept multiple packages in one command with `-m <manager>` (per-manager only; `go`, `pipx`, `uv` excluded; `snap` excluded from batch reinstall). Homebrew falls back to per-package operations when a batch mixes casks and formulae.

---

## Cross-Manager Special Features

| Feature | Description | Supported Managers | Stamp Command | Status |
|---------|-------------|-------------------|---------------|--------|
| **Cask (GUI Apps)** | macOS GUI application management via `brew install --cask`. Auto-detected on install, stored in manifest. | Brew | `stamp install <pkg> -m brew` | ✓ Complete |
| **File Search (provides)** | Find which package owns a specific file or binary. | DNF, APT, Pacman, Paru, Zypper, MacPorts | `stamp provides <file>` | ✓ Complete |
| **Orphan Cleanup (autoremove)** | Remove unused dependencies that were pulled in automatically. | Brew, DNF, APT, Pacman, Paru, Zypper, Flatpak, MacPorts | `stamp autoremove` | ✓ Complete |
| **Cache Cleanup (clean)** | Clear locally cached package files to free disk space. | Brew, DNF, APT, Pacman, Paru, Zypper, Snap, MacPorts | `stamp clean` | ✓ Complete |
| **Version Pinning (hold)** | Pin packages at specific versions to prevent accidental upgrades. | APT, DNF, Pacman, Paru | `stamp hold <pkg>` | ✓ Complete |
| **Group Install** | Install DNF package groups (e.g. "Development Tools"). | DNF | `stamp install --group` | ✓ Complete |
| **Flatpak Override** | Manage Flatpak sandbox permissions (filesystem, socket, device access). | Flatpak | `stamp override` | ✓ Complete |
| **Aliases** | Native command aliases for every supported package manager. | All | See [Aliases Matrix](../usage/aliases.html) | ✓ Complete |

---

## Phase & Task Progress

| Phase | Task | Description | Status |
| :--- | :--- | :--- | :---: |
| 1 | 1 | Repository Scaffolding & Tooling | ✓ |
| 1 | 2 | Manifest Manager (TOML) | ✓ |
| 1 | 2.5 | Pre-requisite Fixes (Security & CI) | ✓ |
| 2 | 3 | Package Manager Interfaces & Mocks | ✓ |
| 2 | 4 | Native Adapters (Write Operations) | ✓ |
| 2 | 5 | Active CLI Commands | ✓ |
| 3 | 6 | Native Adapters (Read-Only) | ✓ |
| 3 | 7 | State Engine (Snapshotting) | ✓ |
| 3 | 8 | The `reconcile` Command | ✓ |
| 4 | 9 | The `restore` Command | ✓ |
| 4 | 10 | CLI Polish, Manpages, GitHub Pages & Landing Page | ~ |
| 4 | 10a | `stamp doctor` command | ✓ |
| 4 | 10b | `stamp completion` shell autocompletion | ✓ |
| 4 | 10c | `stamp man` generation and install | ✓ |
| 4 | 10d | NO_COLOR compliance | ✓ |
| 4 | 10e | Doc generation pipeline (task docs) | ✓ |
| 4 | 10f | Flag standardization (short forms, subcommands) | ✓ |
| 4 | 10h | Uninstall documentation in README.md (standard + hard uninstall) | ✓ |
| 4 | 11 | Self-Update Subcommand | ✓ |
| 4 | 12 | `stamp hello` welcome command | ✓ |
| 4 | 13 | `stamp info` package info command | ✓ |
| 4 | 14 | `stamp man check` version verification | ✓ |
| 4 | 15 | Per-manager flags for reconcile/restore/doctor/list | ⚠ Partial |
| 4 | 16 | Multi-platform integration testing | ✓ Complete |
| 4 | 17 | Package manager feature audit | * |
| 4 | 18 | `stamp reinstall` command | ✓ |
| 4 | 19 | Generate missing usage & man pages | ✓ |
| 4 | 20 | Create GitHub Pages landing page | ~ |
| 4 | 21 | `stamp init` command | ✓ |
| 4 | 22 | `stamp list` command | ✓ |
| 4 | 23 | `stamp update` command | ✓ |
| 4 | 24 | Migrate `stamp hello` to `stamp setup` wizard | ✓ |
| 4 | 25 | Add shell completion check to `stamp doctor` | ✓ |
| 4 | 25b | Re-init guard for `stamp init` | ✓ |
| 4 | 26 | Add `yum` as alias to `dnf` manager | ✓ |
| 4 | 32 | APT package manager adapter | ✓ |
| 4 | 33 | Docker-based integration testing | ✓ |
| 4 | 34 | Post-release integration CI pipelines | ✓ |
| 5 | — | Relicense to Apache-2.0 | ✓ |
| 6 | 27 | Reconcile — Auto-Track and `--dry-run` | ✓ |
| 6 | 28 | Reinstall — Support Pre-Existing Packages | ✓ |
| 6 | 29 | Flag and Compliance Updates | ✓ |
| 6 | 30 | `stamp auto-reconcile` Command | ✓ |
| 6 | 31 | Go toolchain adapter | ✓ |
| 7 | 32 | npm toolchain adapter (#52) | ✓ |
| 7 | 33 | Cargo toolchain adapter (#53) | ✓ |
| 7 | 34 | cmd_test.go split (#105) | ✓ |
| 7 | 35 | Homebrew cask support (#152) | ✓ |
| 7 | 36 | Provides command (#154) | ✓ |
| 7 | 37 | Autoremove command (#155) | ✓ |
| 7 | 38 | Clean command (#158) | ✓ |
| 7 | 39 | Hold/unhold commands (#156) | ✓ |
| 7 | 40 | Flatpak override command (#157) | ✓ |
| 7 | 41 | DNF group install (#159) | ✓ |
| 7 | 42 | CLI aliases (#160) | ✓ |

## UNIX Compliance

| Requirement | SPEC.md | Implemented | Details | Status |
| :--- | :---: | :---: | :--- | :---: |
| POSIX Syntax | ✓ | ✓ | Built-in via spf13/cobra | ✓ Complete |
| XDG Base Directory | ✓ | ✓ | `xdgConfigDir()` in root.go | ✓ Complete |
| Exit Codes (sysexits) | ✓ | ⚠ Partial | Constants defined (`ExitUsage`, `ExitDataErr`, etc.) but `Execute()` always uses `os.Exit(1)` | ⚠ Partial |
| I/O Separation | ✓ | ✓ | Confirmations → `stderr`, search results/listings → `stdout` | ✓ Complete |
| NO_COLOR support | ✓ | ✓ | Checks `NO_COLOR` env var, reports in `stamp doctor` | ✓ Complete |
| Auto-Generated Docs | ✓ | ✓ | `task docs` generates markdown + man pages | ✓ Complete |
| UNIX Man Pages | ✓ | ✓ | `stamp man` generates and installs system man page | ✓ Complete |
| Project Landing Page | ✓ | ✗ | Not created (Task 10) | ✗ Missing |
