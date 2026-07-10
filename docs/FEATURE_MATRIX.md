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
| `stamp init` | | ✅ | ❌ | ❌ | ⏳ Pending (Phase 4) |
| `stamp reconcile` | | ✅ | ❌ | ❌ | ⏳ Pending (Phase 3) |
| `stamp restore` | | ✅ | ❌ | ❌ | ⏳ Pending (Phase 4) |
| `stamp update` | `upgrade` | ✅ | ❌ | ❌ | ⏳ Pending |
| `stamp list` | `ls` | ✅ | ❌ | ❌ | ⏳ Pending |
| `stamp doctor` | | ✅ | ❌ | ❌ | ⏳ Pending |
| `stamp self-update` | `self-upgrade` | ✅ | ❌ | ❌ | ⏳ Pending |

## Global Flags

| Flag | Short | SPEC.md | Implemented | Status |
| :--- | :--- | :---: | :---: | :---: |
| `--verbose` | `-v` | ✅ | ✅ Registered in root PersistentFlags | ✅ Complete |
| `--json` | | ✅ | ✅ Registered in root PersistentFlags | ✅ Complete |
| `--yes` | `-y` | ✅ | ✅ Registered in root PersistentFlags | ✅ Complete |

## Per-Command Flags

| Command | Flag | SPEC.md | Implemented | Status |
| :--- | :--- | :---: | :---: | :---: |
| `stamp install` | `--manager, -m <name>` | ✅ | ✅ | ✅ Complete |
| `stamp install` | `--note <text>` | ✅ | ✅ | ✅ Complete |
| `stamp remove` | `--manager, -m <name>` | ✅ | ✅ | ✅ Complete |
| `stamp search` | `--manager, -m <name>` | ✅ | ✅ | ✅ Complete |
| `stamp restore` | `--dry-run` | ✅ | ❌ | ⏳ Pending |
| `stamp repo add` | `--manager, -m <name>` | ✅ Required | ✅ MarkFlagRequired | ✅ Complete |
| `stamp repo remove` | `--manager, -m <name>` | ✅ Required | ✅ MarkFlagRequired | ✅ Complete |
| `stamp repo add` | `[url]` (positional) | ✅ Optional | ✅ Parsed from args | ✅ Complete |
| `stamp self-update` | `--check` | ✅ | ❌ | ⏳ Pending |
| `stamp doctor` | `--json` | ✅ | ❌ | ⏳ Pending |
| `stamp list` | `--json` | ✅ | ❌ | ⏳ Pending |
| `stamp repo list` | `--json` | ✅ | ❌ | ⏳ Pending |

## UNIX Compliance

| Requirement | SPEC.md | Implemented | Details | Status |
| :--- | :---: | :---: | :--- | :---: |
| POSIX Syntax | ✅ | ✅ | Built-in via spf13/cobra | ✅ Complete |
| XDG Base Directory | ✅ | ✅ | `xdgConfigDir()` in root.go | ✅ Complete |
| Exit Codes (sysexits) | ✅ | ⚠️ Partial | Constants defined (`ExitUsage`, `ExitDataErr`, etc.) but `Execute()` always uses `os.Exit(1)` | ⚠️ Partial |
| I/O Separation | ✅ | ✅ | Confirmations → `stderr`, search results/listings → `stdout` | ✅ Complete |
| NO_COLOR support | ✅ | ❌ | Not implemented | ❌ Missing |
| Auto-Generated Docs | ✅ | ❌ | cobra/doc pipeline not wired (Task 10) | ❌ Missing |
| UNIX Man Pages | ✅ | ❌ | Not generated (Task 10) | ❌ Missing |
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
| 3 | 8 | The `reconcile` Command | ⏳ |
| 4 | 9 | The `restore` Command | ⏳ |
| 4 | 10 | CLI Polish, Manpages, GitHub Pages & Landing Page | ⏳ |
| 4 | 11 | Self-Update Subcommand | ⏳ |
| 5 | 11 | Relicense to Apache-2.0 | ✅ |
