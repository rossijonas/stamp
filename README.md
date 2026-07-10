# Stamp

```txt

                              
         █▄
        ▄██▄      ▄
   ▄██▀█ ██ ▄▀▀█▄ ███▄███▄ ████▄
   ▀███▄ ██ ▄█▀██ ██ ██ ██ ██ ██
  █▄▄██▀▄██▄▀█▄██▄██ ██ ▀█▄████▀
                           ██
                           ▀

```

*Track your package installation intent across multiple package managers. Rebuild your environment anywhere.*

---

[![CI](https://github.com/rossijonas/stamp/actions/workflows/ci.yml/badge.svg)](https://github.com/rossijonas/stamp/actions/workflows/ci.yml) [![codecov](https://codecov.io/gh/rossijonas/stamp/branch/main/graph/badge.svg)](https://codecov.io/gh/rossijonas/stamp) [![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE) [![Go Reference](https://pkg.go.dev/badge/github.com/rossijonas/stamp.svg)](https://pkg.go.dev/github.com/rossijonas/stamp) [![Code of Conduct](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)

---

## ▪ Intro

> ⚠️ **Active Development:** `stamp` is currently in the MVP phase of active development. Features are being delivered incrementally. For a complete look at our progress, check the [Implementation Plan](docs/IMPLEMENTATION_PLAN.md).

`stamp` is a CLI tool that tracks the software you *intentionally* install across fragmented package managers (`dnf`, `flatpak`, `brew`, etc.). It records your choices into a portable, version-controlled TOML manifest. When you move to a new machine,  run `stamp restore` to recreate your exact environment.

**Current Scope:** The MVP of `stamp` is focused on the Red Hat ecosystem (e.g., Fedora), natively supporting the trio of package managers most commonly used on these systems: `dnf`, `flatpak`, and `brew`.

## ▪ Usage

### ⟲ The Two Workflows

`stamp` is designed to be flexible. You can use it actively as a unified wrapper, or passively as a safety net.

#### ⚙ Workflow A: Active Management (Recommended)

Use `stamp` as your primary package installer. This guarantees 100% traceability of your intent instantly. It will auto-detect the best package manager or allow you to specify one.

**Search across all managers:**
```bash
stamp search ripgrep
```

**Install and track in one step (with aliases like `add`):**
```bash
stamp install htop              # auto-detects dnf on Fedora
stamp install spotify --manager flatpak # or: -m flatpak
stamp add lazygit -m brew       # using the 'add' alias
```

**Remove and untrack (with aliases like `uninstall`, `rm`, `delete`, `del`):**
```bash
stamp remove htop
stamp uninstall lazygit         # using the 'uninstall' alias
stamp rm htop                   # using the 'rm' alias
stamp del htop                  # using the 'del' alias
```

#### ⛨ Workflow B: The Passive Observer (The Safety Net)

If you or a script accidentally bypass `stamp` and use native tools directly, `stamp` acts as a safety net to retroactively capture your intent.

**1. Install Normally (Bypassing stamp):**
```bash
sudo dnf install ripgrep
brew install jq
```

**2. Reconcile:**
Run `reconcile` periodically. `stamp` compares your current system against its snapshot, detects the newly installed `ripgrep` and `jq`, and prompts you to add them to your manifest. You can also pass the `-y` / `--yes` flag to automatically track all newly detected packages without interactive prompts (ideal for automated crontabs).
```bash
stamp reconcile -y
```

### ⚒ Rebuilding Your Environment

When you get a new laptop, clone your dotfiles (containing your `manifest.toml`) and run:

```bash
stamp restore -y
```
`stamp` will read the manifest and execute the appropriate native install commands concurrently. The `-y` / `--yes` flag ensures any safety confirmations are auto-accepted.

To keep everything fresh, run a unified update across all your managers at once:
```bash
stamp update
```

To update `stamp` itself to the latest released version:
```bash
stamp self-update
```

### ✎ Adding Notes to Packages

You can annotate why you installed a specific package directly in the CLI, which saves it to your manifest. This is incredibly useful for remembering why you needed an obscure tool 6 months later.

```bash
stamp install lazygit --note "better git TUI than default"
```
Or add a note to an existing tracked package:
```bash
stamp edit lazygit --note "Required for the backend build script"
```

### ⚙️ Configuration

`stamp` is configured via a simple, human-editable TOML file located at `~/.config/stamp/config.toml`. This file governs how `stamp` resolves ambiguity when a package exists in multiple package managers.

Example configuration:
```toml
# ~/.config/stamp/config.toml

# Global package manager priority (highest to lowest)
precedence = ["dnf", "flatpak", "brew"]

# Fine-grained pattern-based matching rules
[[rules]]
pattern = "^com\\..*|^org\\..*" # Force reverse-DNS names to Flatpak
prefer = "flatpak"

[[rules]]
pattern = "^lib.*|-devel$"     # Force libraries and dev headers to DNF
prefer = "dnf"
```

If you run `stamp install htop`, and `htop` is available in both DNF and Homebrew, `stamp` will automatically select DNF because `dnf` has higher priority than `brew` in your `precedence` config.

## ▪ The Project

### ◩ Roadmap

While `stamp` currently targets Red Hat-based systems, our goal is to become the universal intent tracker across all major Linux distributions and macOS.

### ⚑ Upcoming Milestones

- **[ ] Debian/Ubuntu Ecosystem Support**
  - Implement support for `apt` and `snap`.
- **[ ] Extend MacOS Support**
  - Implement support for MacPorts.
- **[ ] Arch Linux Ecosystem Support**
  - Implement support for `pacman`.
- **[ ] Developer Toolchains**
  - Track language-specific global installs (`cargo`, `pipx`, `go install`).

### ⊞ Compatibility & Support Tracker

`stamp` aims to be the universal intent tracker across all major operating systems and developer toolchains. Below is the current support matrix and architectural context for the package managers we track (or plan to track).

#### ⌨ OS Package Managers

| Status | Package Manager | Target Platforms | Core Binary Format | Scope & Permissions | Key Unique Architectural Feature |
| :---: | :--- | :--- | :--- | :--- | :--- |
| ✅ | **[DNF](https://docs.fedoraproject.org/en-US/quick-docs/dnf/)** | Fedora, RHEL, CentOS | `.rpm` | System-wide, root/sudo | High-performance C-based libsolv engine |
| ✅ | **[Homebrew](https://brew.sh/)** | macOS, Linux | Bottles (tarballs) | User-space, no root/sudo | Avoids duplicating host OS dynamic libraries |
| ✅ | **[Flatpak](https://flatpak.org/)** | Linux (Universal) | OSTree / OCI | User or System | Sandboxed application distribution |
| ⏳ | **[APT](https://ubuntu.com/server/docs/how-to/software/package-management/#)** | Debian, Ubuntu, Mint | `.deb` | System-wide, root/sudo | Robust dependency resolution, stable release focus |
| ⏳ | **[Snap](https://snapcraft.io/)** | Ubuntu, Linux | SquashFS | System-wide, root/sudo | Containerized, auto-updating application bundles |
| ⏳ | **[MacPorts](https://www.macports.org/)** | macOS | Source files, frameworks | System-wide, root/sudo | Fully isolated `/opt/local` directory tree |
| ⏳ | **[Pacman](https://wiki.archlinux.org/title/Pacman)** | Arch Linux | `.pkg.tar.zst` | System-wide, root/sudo | Lightweight rolling-release synchronization |
| | **[Winget](https://learn.microsoft.com/en-us/windows/package-manager/winget/)** | Windows 11 | AppX/MSIX, MSI, EXE | Mixed, user or system | In-place version synchronization with registry |
| | **[Chocolatey](https://chocolatey.org/)** | Windows | `.nupkg` (NuGet wrappers) | System-wide, admin | First-party configuration management integration |
| | **[Scoop](https://scoop.sh/)** | Windows | Portable ZIP extracts | User-space, no admin | Shim-based path management to avoid path pollution |
| | **[APK](https://wiki.alpinelinux.org/wiki/Alpine_Package_Keeper)** | Alpine Linux | `.apk` | System-wide, root/sudo | Designed around musl and BusyBox for minimal size |

#### ⛁ Developer Toolchains (Language Package Managers)

Tracking global CLI tools installed via language package managers is on our roadmap.

| Status | Language | Primary Tool(s) | Manifest Format | Key Architectural Isolation Mechanism |
| :---: | :--- | :--- | :--- | :--- |
| ⏳ | **Go** | `go install` | `go.mod` | Minimal Version Selection (MVS) algorithm |
| ⏳ | **Rust** | `cargo install` | `Cargo.toml` | Highly structured, static build compiler caching |
| ⏳ | **Python** | `pipx` / `uv tool` | `pyproject.toml` | Isolated virtual environments and centralized caches |
| ⏳ | **JS / TS** | `npm` / `bun` | `package.json` | Content-addressable folders linked via symlinks |
| | **Ruby** | Bundler | `Gemfile` | Local execution sandbox isolation |
| | **PHP** | Composer | `composer.json` | Local vendor path isolation |
| | **Java / JVM**| Gradle | `build.gradle` | Highly configurable execution tasks & build graph caching |
| | **.NET** | NuGet | `.csproj` | Multi-targeting framework library extractors |
| | **Swift** | Swift Package Manager | `Package.swift` | Native Xcode compiler static link integration |

## ▪ Architecture & Vision

Read the [Project Vision](docs/VISION.md) to understand the "why" behind the project, or check out the [Technical Specs](docs/SPEC.md) and [Architecture Decisions](docs/decisions/).

## ▪ License

This project is licensed under the Apache License, Version 2.0 (Apache-2.0) - see the [LICENSE](LICENSE) file for details.
