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

*A lightweight yet powerful wrapper for your native package managers. Install, track, and restore without changing your tools.*

---

[![CI](https://github.com/rossijonas/stamp/actions/workflows/ci.yml/badge.svg)](https://github.com/rossijonas/stamp/actions/workflows/ci.yml) [![Release](https://img.shields.io/github/v/release/rossijonas/stamp)](https://github.com/rossijonas/stamp/releases/latest) [![codecov](https://codecov.io/gh/rossijonas/stamp/branch/main/graph/badge.svg)](https://codecov.io/gh/rossijonas/stamp) [![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE) [![Go Reference](https://pkg.go.dev/badge/github.com/rossijonas/stamp.svg)](https://pkg.go.dev/github.com/rossijonas/stamp) [![Code of Conduct](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)

---

## ▪ Intro

> ⚠️ **Active Development:** `stamp` is currently in the MVP phase of active development. Features are being delivered incrementally. For a complete look at our progress, check the [Implementation Plan](docs/IMPLEMENTATION_PLAN.md).

`stamp` is a lightweight yet powerful wrapper for your native package managers. It lets you install, search, get info, and remove packages and repositories across multiple package managers through a single CLI — without conflicts, ecosystem lock-in, or changing your tools.

### Features

#### Core Features

✔️ **Multi-Manager Wrapper** - *Install, search, get info, and remove packages across multiple package managers through a single CLI. No conflicts, no ecosystem lock-in. [See supported package managers →](#-compatibility--support-tracker)*

✔️ **Automatic Intent Tracking** - *Every intentional install is recorded in a portable `manifest.toml`. Dependency packages are not included — you only track what you chose.*

✔️ **One-Command Environment Rebuild** - *`stamp restore` reinstalls all repositories and packages on a new machine. Clone your dotfiles, run one command, done.*

✔️ **Unified Repository Management** - *Add, remove, and list third-party repositories (repos, taps, remotes) across all managers with the same interface.*

✔️ **Safety Net Reconciliation** - *Forgot to use stamp? `stamp reconcile` auto-detects packages installed outside the tool and adds them to your manifest without prompting. Preview with `--dry-run` first.*

✔️ **Self-Contained Documentation** - *Built-in man page generation (`stamp man install`), shell completions (`stamp completion bash|zsh|fish|powershell`), and auto-generated CLI reference docs.*

✔️ **Agnostic & Unopinionated** - *Doesn't dictate how you configure your software (that's the job of `stow` or `chezmoi`). It solely ensures the software exists on your machine.*

✔️ **Context Preservation (Notes)** - *Intent is easily forgotten. The `--note` flag on `stamp install` acts as a memory aid — you aren't just restoring `libfoo`, you're restoring why you needed it.*

#### System & Compliance

✔️ **Built-in System Doctor** - *`stamp doctor` checks manager availability, manifest integrity, and UNIX compliance in a single command. JSON output for scripting.*

✔️ **UNIX Compliant** - *XDG Base Directory, POSIX syntax, NO_COLOR support, strict stdout/stderr separation, and BSD sysexits exit codes.*

✔️ **Predictable & Scriptable** - *Global `--yes` / `-y` flag enables deterministic non-interactive execution in CI, bootstrap scripts, and automation pipelines. Never hangs waiting for user input in headless environments.*

#### Technical

✔️ **Lightweight Yet Powerful** - *Thin wrapper layer, no language lock-in, no philosophical shift. Works with your existing tools, not instead of them.*

✔️ **Built with Go** - *Single static binary, fast startup, no runtime dependencies. Linux and macOS support (amd64 + arm64). Windows on the roadmap.*

✔️ **Extensible Architecture** - *Interface-driven adapter pattern. Adding a new package manager is implementing 7 methods.*

#### Ecosystem

✔️ **Compatible with Popular Package Managers** - *Works with the package managers you already use on Linux and macOS, with more on the way. [See full compatibility table →](#-compatibility--support-tracker)*

✔️ **Developer Toolchain Support (coming soon)** - *`cargo`, `pipx`, `go install`, and `npm`/`bun` for language-specific global tools. [See full compatibility table →](#-compatibility--support-tracker)*

## ▪ Installing

| Method | Command |
| :--- | :--- |
| **Go install** | `go install github.com/rossijonas/stamp/cmd/stamp@latest` |
| **Download binary** | `curl -sSL https://github.com/rossijonas/stamp/releases/latest/download/stamp_{{VERSION}}_{{OS}}_{{ARCH}}.tar.gz \| tar xz && sudo mv stamp /usr/local/bin/` |
| **From source** | `git clone https://github.com/rossijonas/stamp.git && cd stamp && go build -o bin/stamp ./cmd/stamp && sudo cp bin/stamp /usr/local/bin/` |
| **Homebrew** (future) | `brew install rossijonas/tap/stamp` |

*Replace `{{VERSION}}`, `{{OS}}`, and `{{ARCH}}` with the appropriate values for your system (e.g., `v0.1.0`, `linux`, `amd64`). The archive name uses the full tag (e.g. `stamp_v0.1.0_linux_amd64.tar.gz`).*

## ▪ Uninstalling

| Method | Command |
| :--- | :--- |
| **Standard (binary only)** | `rm $(which stamp)` |
| **Hard (remove all data)** | `rm -rf ~/.config/stamp ~/.local/share/stamp && rm -f $(which stamp) && sudo rm -f /usr/local/share/man/man1/stamp.1` |

*Standard uninstall removes just the binary. Hard uninstall also removes configuration, manifest, snapshots, and man pages.*

## ▪ Usage

### ⟲ The Two Workflows

`stamp` is designed to be flexible. You can use it actively as a unified wrapper, or passively as a safety net.

#### 🔰 First-Time Setup

Before using `stamp`, run the setup wizard to initialize your environment:

```bash
stamp setup         # Interactive wizard: completions, man pages, init, doctor
stamp setup -y      # Non-interactive: runs all steps without prompts
```

> **⚠️ Note on Privilege Escalation:** Package managers that require root (e.g., `dnf`) automatically wrap their write operations with `sudo` internally. Always run `stamp install htop` as your normal user — do **not** use `sudo stamp install`. Sudo prompts for your password in a terminal and fails gracefully in non-interactive environments (CI/pipelines).

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
Run `reconcile` periodically. `stamp` compares your current system against its snapshot, detects newly installed `ripgrep` and `jq`, and auto-tracks them to your manifest — no prompts, no decisions. Preview drift with `--dry-run` before committing:
```bash
stamp reconcile --dry-run   # preview only
stamp reconcile             # auto-track detected changes
```

> **Note:** Only packages installed *after* your last snapshot are detected. Pre-existing packages (installed before `stamp init`) are not visible to reconcile. To track a pre-existing package, use `stamp reinstall <pkg>` instead.

### ⚑ Edge Cases

#### Reinstall Gap

`stamp reconcile` uses snapshot diffing: it compares the current system state against the last saved snapshot. This edge case only applies when you **bypass stamp and use native package manager commands (dnf, brew, flatpak) directly**, then rely on reconcile as a safety net. If a package is **removed and then reinstalled** between two reconcile runs, the snapshot shows no net change — the package is present in both old and new snapshots, so reconcile reports no drift.

```
1. Snapshot: [htop, gcc, systemd, …]
2. dnf remove htop           → system: [gcc, systemd, …]
3. dnf install htop          → system: [htop, gcc, systemd, …]
   (reconcile NOT run between remove and install)
4. stamp reconcile           → snapshot [htop,…] vs system [htop,…]
                              → identical → "No drift detected"
```

**Mitigation — Option A: Always Use Stamp (Recommended)**

The edge case never occurs if you manage packages through stamp:

```bash
stamp install htop     # tracks automatically
stamp remove htop      # untracks automatically
```

Use Workflow A (`stamp install`/`stamp remove`) as your primary package manager. Stamp records every install and removal in the manifest instantly — no snapshot diffing needed. Only packages installed outside stamp via native tools are subject to the reinstall gap.

**Mitigation — Option B: Regular Reconciliation**

If you do use native package manager commands directly, remember to run `stamp reconcile` after each uninstall operation to keep snapshots in sync:

```bash
sudo dnf remove htop && stamp reconcile
sudo dnf install htop
```

**Mitigation — Option C: Automated Timer**

Set up a daily timer to run `stamp reconcile` automatically. The `stamp auto-reconcile` command (planned) will handle this setup. In the meantime, timer files are available in `contrib/`:

**Linux (systemd):**
```bash
cp contrib/systemd/stamp-reconcile.* ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now stamp-reconcile.timer
```

**macOS (launchd):**
```bash
cp contrib/launchd/com.rossijonas.stamp.reconcile.plist ~/Library/LaunchAgents/
launchctl load ~/Library/LaunchAgents/com.rossijonas.stamp.reconcile.plist
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
