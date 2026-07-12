# Feature Matrix: Stamp CLI

This document tracks all SPEC.md commands, flags, and compliance items against their current implementation status. Updated after each feature delivery.

## CLI Commands

| Command | Aliases | SPEC.md | Implemented | Wired to Logic | Status |
| :--- | :--- | :---: | :---: | :---: | :---: |
| `stamp install <pkg>` | `add` | ✅ | ✅ | ✅ Resolver → adapter → manifest | ✅ Complete |
| `stamp remove <pkg>` | `uninstall`, `rm`, `delete`, `del` | ✅ | ✅ | ✅ Manifest lookup + adapter | ✅ Complete |
| `stamp search <query>` | | ✅ | ✅ | ✅ Queries adapters | ✅ Complete |
| `stamp repo add <name> [url]` | `install` | ✅ | ✅ | ✅ Adapter + manifest (--manager required) | ✅ Complete |
| `stamp repo remove <name>` | `uninstall`, `rm`, `delete`, `del` | ✅ | ✅ | ✅ Adapter + manifest (--manager required) | ✅ Complete |
| `stamp repo list` | `ls` | ✅ | ✅ | ✅ Reads manifest | ✅ Complete |
| `stamp reconcile` | | ✅ | ✅ | ✅ State diff + manifest update | ✅ Complete |
| `stamp restore` | | ✅ | ✅ | ✅ Sequentially adds repos then concurrently installs packages | ✅ Complete |
| `stamp doctor` | | ✅ | ✅ | ✅ Adapter check + manifest check + compliance report | ✅ Complete |
| `stamp completion [shell]` | | ✅ | ✅ | ✅ Cobra Gen*Completion | ✅ Complete |
| `stamp man` | | ✅ | ✅ | ✅ Cobra doc.GenMan | ✅ Complete |
| `stamp init` | | ✅ | ❌ | ❌ | ⏳ Pending |
| `stamp update` | `upgrade` | ✅ | ❌ | ❌ | ⏳ Pending |
| `stamp list` | `ls` | ✅ | ❌ | ❌ | ⏳ Pending |
| `stamp self-update` | `self-upgrade` | ✅ | ❌ | ❌ | ⏳ Pending |
| `stamp hello` | | ✅ | ❌ | ❌ | 📝 Spec ready (C1) |
| `stamp info <pkg>` | | ✅ | ❌ | ❌ | 📝 Spec ready (C2) |

## Repository Commands

| Command | Aliases | SPEC.md | Implemented | Wired to Logic | Status |
| :--- | :--- | :---: | :---: | :---: | :---: |
| `stamp repo add <name> [url]` | `install` | ✅ | ✅ | ✅ Adapter + manifest (--manager required) | ✅ Complete |
| `stamp repo remove <name>` | `uninstall`, `rm`, `delete`, `del` | ✅ | ✅ | ✅ Adapter + manifest (--manager required) | ✅ Complete |
| `stamp repo list` | `ls` | ✅ | ✅ | ✅ Reads manifest | ✅ Complete |

## Man Command (Subcommands)

| Command | Flags | SPEC.md | Implemented | Status |
| :--- | :--- | :---: | :---: | :---: |
| `stamp man` | | ✅ | ✅ Prints man page to stdout | ✅ Complete |
| `stamp man install` | `--prefix` | ✅ | ❌ Currently `stamp man --install` | 🔄 Planned refactor |
| `stamp man check` | | ✅ | ❌ | 📝 Spec ready (C3) |

## Global Flags

| Flag | Short | SPEC.md | Implemented | Status |
| :--- | :--- | :---: | :---: | :---: |
| `--verbose` | `-v` | ✅ | ✅ Registered in root PersistentFlags | ✅ Complete |
| `--json` | | ✅ | ✅ Registered in root PersistentFlags | ✅ Complete |
| `--yes` | `-y` | ✅ | ✅ Registered in root PersistentFlags | ✅ Complete |

## Per-Command Flags

| Command | Flag | Short | SPEC.md | Implemented | Status |
| :--- | :--- | :---: | :---: | :---: | :---: |
| `stamp install` | `--manager <name>` | `-m` | ✅ | ✅ | ✅ Complete |
| `stamp install` | `--note <text>` | | ✅ | ✅ | ✅ Complete |
| `stamp remove` | `--manager <name>` | `-m` | ✅ | ✅ | ✅ Complete |
| `stamp search` | `--manager <name>` | `-m` | ✅ | ✅ | ✅ Complete |
| `stamp restore` | `--dry-run` | | ✅ | ✅ | ✅ Complete |
| `stamp doctor` | `--json` | | ✅ | ✅ | ✅ Complete |
| `stamp man` | `--install` | | ✅ | ❌ | 🔄 Planned refactor |
| `stamp man` | `--prefix` | | ✅ | ❌ | 🔄 Planned refactor |
| `stamp man install` | `--prefix` | | ❌ | ❌ | 📝 Proposed |
| `stamp self-update` | `--check` | | ✅ | ❌ | ⏳ Pending |
| `stamp list` | `--json` | | ✅ | ❌ | ⏳ Pending |
| `stamp repo list` | `--json` | | ✅ | ✅ | ✅ Complete |
| `stamp reconcile` | `--manager <name>` | `-m` | ❌ | ❌ | 📝 Proposed |
| `stamp restore` | `--manager <name>` | `-m` | ❌ | ❌ | 📝 Proposed |
| `stamp doctor` | `--manager <name>` | `-m` | ❌ | ❌ | 📝 Proposed |
| `stamp update` | `--manager <name>` | `-m` | ✅ | ❌ | ⏳ Pending |
| `stamp list` | `--manager <name>` | `-m` | ❌ | ❌ | 📝 Proposed |
| `stamp repo list` | `--manager <name>` | `-m` | ❌ | ❌ | 📝 Proposed |

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
| 4 | 10f | Flag standardization (short forms, subcommands) | ⏳ |
| 4 | 10g | GitHub Pages landing page | ⏳ |
| 4 | 11 | Self-Update Subcommand | ⏳ |
| 4 | 12 | `stamp hello` welcome command | 📝 Spec ready |
| 4 | 13 | `stamp info` package info command | 📝 Spec ready |
| 4 | 14 | `stamp man check` version verification | 📝 Spec ready |
| 4 | 15 | Per-manager flags for reconcile/restore/doctor/list | 📝 |
| 4 | 16 | Multi-platform integration testing (Fedora/Ubuntu/Arch/macOS/Windows) | 📝 |
| 4 | 17 | Package manager feature audit (Homebrew cask, brew services, dnf groupinstall) | 📝 |
| 5 | 11 | Relicense to Apache-2.0 | ✅ |
