# Feature Matrix: Stamp CLI

This document tracks all SPEC.md commands, flags, and compliance items against their current implementation status. Updated after each feature delivery.

## CLI Commands

| Command | Aliases | SPEC.md | Implemented | Wired to Logic | Status |
| :--- | :--- | :---: | :---: | :---: | :---: |
| `stamp install <pkg>` | `add` | ✅ | ✅ | ✅ Resolver → adapter → manifest | ✅ Complete |
| `stamp remove <pkg>` | `uninstall`, `rm`, `delete`, `del` | ✅ | ✅ | ✅ Manifest lookup + adapter | ✅ Complete |
| `stamp reinstall <pkg>` | | ✅ | ✅ | ✅ Manifest-tracked + pre-existing via resolver + `Reinstall` adapter method | ✅ Complete |
| `stamp search <query>` | | ✅ | ✅ | ✅ Queries adapters | ✅ Complete |
| `stamp info <pkg>` | | ✅ | ✅ | ✅ Queries adapter Info() | ✅ Complete |
| `stamp repo add <name> [url]` | `install` | ✅ | ✅ | ✅ Adapter + manifest (--manager required) | ✅ Complete |
| `stamp repo remove <name>` | `uninstall`, `rm`, `delete`, `del` | ✅ | ✅ | ✅ Adapter + manifest (--manager required) | ✅ Complete |
| `stamp repo list` | `ls` | ✅ | ✅ | ✅ Reads manifest | ✅ Complete |
| `stamp reconcile` | | ✅ | ✅ | ✅ Auto-track + `--dry-run` + no prompt + repo drift detection | ✅ Complete |
| `stamp restore` | | ✅ | ✅ | ✅ Sequentially adds repos then concurrently installs packages | ✅ Complete |
| `stamp doctor` | | ✅ | ✅ | ✅ Adapter check + manifest check + compliance report | ✅ Complete |
| `stamp completion [shell]` | | ✅ | ✅ | ✅ Auto-detect, install to path, --stdout flag | ✅ Complete |
| `stamp man` | | ✅ | ✅ | ✅ Shows help for man command group | ✅ Complete |
| `stamp hello` | | ✅ | ✅ | ✅ Prints ASCII logo + suggested next steps | ✅ Complete |
| `stamp setup` | `hello` | ✅ | ❌ | ❌ | ⏳ Pending |
| `stamp init` | | ✅ | ✅ | ✅ Creates dirs + manifest + snapshots | ✅ Complete |
| `stamp update` | `upgrade` | ✅ | ❌ | ❌ | ⏳ Pending |
| `stamp list` | `ls` | ✅ | ✅ | ✅ Reads manifest | ✅ Complete |
| `stamp self-update` | `self-upgrade` | ✅ | ❌ | ❌ | ⏳ Pending |
| `stamp auto-reconcile on\|off` | | ✅ | ❌ | ❌ | ⏳ Pending |

## Repository Commands

| Command | Aliases | SPEC.md | Implemented | Wired to Logic | Status |
| :--- | :--- | :---: | :---: | :---: | :---: |
| `stamp repo add <name> [url]` | `install` | ✅ | ✅ | ✅ Adapter + manifest (--manager required) | ✅ Complete |
| `stamp repo remove <name>` | `uninstall`, `rm`, `delete`, `del` | ✅ | ✅ | ✅ Adapter + manifest (--manager required) | ✅ Complete |
| `stamp repo list` | `ls` | ✅ | ✅ | ✅ Reads manifest | ✅ Complete |

## Man Command (Subcommands)

| Command | Flags | SPEC.md | Implemented | Status |
| :--- | :--- | :---: | :---: | :---: |
| `stamp man` | | ✅ | ✅ Shows help for man command group | ✅ Complete |
| `stamp man install` | `--prefix` | ✅ | ✅ Installs stamp.1 to system path | ✅ Complete |
| `stamp man check` | | ✅ | ✅ Verifies installed version matches binary | ✅ Complete |

## Global Flags

| Flag | Short | SPEC.md | Implemented | Status |
| :--- | :--- | :---: | :---: | :---: |
| `--verbose` | `-v` | ✅ | ✅ Registered in root PersistentFlags | ✅ Complete |
| `--json` | `-j` | ✅ | ✅ Registered in root PersistentFlags | ✅ Complete |
| `--yes` | `-y` | ✅ | ✅ Registered in root PersistentFlags | ✅ Complete |

## Per-Command Flags

| Command | Flag | Short | SPEC.md | Implemented | Status |
| :--- | :--- | :---: | :---: | :---: | :---: |
| `stamp install` | `--manager <name>` | `-m` | ✅ | ✅ | ✅ Complete |
| `stamp install` | `--note <text>` | `-n` | ✅ | ✅ | ✅ Complete |
| `stamp remove` | `--manager <name>` | `-m` | ✅ | ✅ | ✅ Complete |
| `stamp search` | `--manager <name>` | `-m` | ✅ | ✅ | ✅ Complete |
| `stamp info` | `--manager <name>` | `-m` | ✅ | ✅ | ✅ Complete |
| `stamp restore` | `--dry-run` | `-d` | ✅ | ✅ | ✅ Complete |
| `stamp doctor` | `--json` | `-j` | ✅ | ✅ | ✅ Complete |
| `stamp man install` | `--prefix` | | ✅ | ✅ | ✅ Complete |
| `stamp self-update` | `--check` | | ✅ | ❌ | ⏳ Pending |
| `stamp completion` | `--stdout` | `-s` | ✅ | ✅ | ✅ Complete |
| `stamp list` | `--json` | `-j` | ✅ | ✅ | ✅ Complete |
| `stamp repo list` | `--json` | `-j` | ✅ | ✅ | ✅ Complete |
| `stamp reconcile` | `--dry-run` | `-d` | ✅ | ✅ | ✅ Complete |
| `stamp reconcile` | `--manager <name>` | `-m` | ✅ | ✅ | ✅ Complete |
| `stamp restore` | `--manager <name>` | `-m` | ✅ | ✅ | ✅ Complete |
| `stamp repo list` | `--manager <name>` | `-m` | ✅ | ✅ | ✅ Complete |
| `stamp doctor` | `--manager <name>` | `-m` | ✅ | ✅ | ✅ Complete |
| `stamp update` | `--manager <name>` | `-m` | ✅ | ❌ | ⏳ Pending |
| `stamp auto-reconcile` | `--period <interval>` | `-p` | ✅ | ❌ | ⏳ Pending |
| `stamp list` | `--manager <name>` | `-m` | ✅ | ✅ | ✅ Complete |

## UNIX Compliance

| Requirement | SPEC.md | Implemented | Details | Status |
| :--- | :---: | :---: | :--- | :---: |
| POSIX Syntax | ✅ | ✅ | Built-in via spf13/cobra | ✅ Complete |
| XDG Base Directory | ✅ | ✅ | `xdgConfigDir()` in root.go | ✅ Complete |
| Exit Codes (sysexits) | ✅ | ⚠️ Partial | Constants defined (`ExitUsage`, `ExitDataErr`, etc.) but `Execute()` always uses `os.Exit(1)` | ⚠️ Partial |
| I/O Separation | ✅ | ✅ | Confirmations → `stderr`, search results/listings → `stdout` | ✅ Complete |
| NO_COLOR support | ✅ | ✅ | Checks `NO_COLOR` env var, reports in `stamp doctor` | ✅ Complete |
| Auto-Generated Docs | ✅ | ✅ | `task docs` generates markdown + man pages | ✅ Complete |
| UNIX Man Pages | ✅ | ✅ | `stamp man` generates and installs system man page | ✅ Complete |
| Project Landing Page | ✅ | ❌ | Not created (Task 10) | ❌ Missing |

## Phase & Task Progress

| Phase | Task | Description | Status |
| :--- | :--- | :--- | :---: |
| 1 | 1 | Repository Scaffolding & Tooling | ✅ |
| 1 | 2 | Manifest Manager (TOML) | ✅ |
| 1 | 2.5 | Pre-requisite Fixes (Security & CI) | ✅ |
| 2 | 3 | Package Manager Interfaces & Mocks | ✅ |
| 2 | 4 | Native Adapters (Write Operations) | ✅ |
| 2 | 5 | Active CLI Commands | ✅ |
| 3 | 6 | Native Adapters (Read-Only) | ✅ |
| 3 | 7 | State Engine (Snapshotting) | ✅ |
| 3 | 8 | The `reconcile` Command | ✅ |
| 4 | 9 | The `restore` Command | ✅ |
| 4 | 10 | CLI Polish, Manpages, GitHub Pages & Landing Page | ⏳ |
| 4 | 10a | `stamp doctor` command | ✅ |
| 4 | 10b | `stamp completion` shell autocompletion | ✅ |
| 4 | 10c | `stamp man` generation and install | ✅ |
| 4 | 10d | NO_COLOR compliance | ✅ |
| 4 | 10e | Doc generation pipeline (task docs) | ✅ |
| 4 | 10f | Flag standardization (short forms, subcommands) | ✅ |
| 4 | 10h | Uninstall documentation in README.md (standard + hard uninstall) | ✅ |
| 4 | 11 | Self-Update Subcommand | ⏳ |
| 4 | 12 | `stamp hello` welcome command | ✅ |
| 4 | 13 | `stamp info` package info command | ✅ |
| 4 | 14 | `stamp man check` version verification | ✅ |
| 4 | 15 | Per-manager flags for reconcile/restore/doctor/list | ⚠️ Partial |
| 4 | 16 | Multi-platform integration testing (Fedora/Ubuntu/Arch/macOS/Windows) | 📝 |
| 4 | 17 | Package manager feature audit (Homebrew cask, brew services, dnf groupinstall) | 📝 |
| 4 | 18 | `stamp reinstall` command | ✅ |
| 4 | 19 | Generate missing usage & man pages | ✅ |
| 4 | 20 | Create GitHub Pages landing page (`docs/index.html`) | ⏳ |
| 4 | 21 | `stamp init` command | ✅ |
| 4 | 22 | `stamp list` command (alias `ls`) | ✅ |
| 4 | 23 | `stamp update` command (alias `upgrade`) | ⏳ |
| 4 | 24 | Migrate `stamp hello` to `stamp setup` wizard (#59) | ⏳ |
| 4 | 25 | Add shell completion check to `stamp doctor` (#60) | ✅ |
| 4 | 26 | Add `yum` as alias to `dnf` manager (#61) | ⏳ |
| 5 | — | Relicense to Apache-2.0 | ✅ |
| 6 | 27 | Reconcile — Auto-Track and `--dry-run` | ✅ |
| 6 | 28 | Reinstall — Support Pre-Existing Packages | ✅ |
| 6 | 29 | Flag and Compliance Updates | ✅ |
| 6 | 30 | `stamp auto-reconcile` Command | ⏳ Pending |
