<p align="center">
  <picture>
    <source srcset="docs/media/stamp-logo-white-nobkg-exp_2.svg" media="(prefers-color-scheme: dark)">
    <source srcset="docs/media/stamp-logo-black-nobkg-exp_2.svg" media="(prefers-color-scheme: light)">
    <img src="docs/media/stamp-logo-black-nobkg-exp_2.svg" alt="Stamp logo" width="320">
  </picture>
</p>

<p align="center"><em>A lightweight yet powerful tool that wraps many package managers into one. Install, track, and restore without changing your tools.</em></p>

<p align="center">
  <a href="https://github.com/rossijonas/stamp/actions/workflows/ci.yml"><img src="https://github.com/rossijonas/stamp/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/rossijonas/stamp/actions/workflows/test-integration-ubuntu.yml"><img src="https://github.com/rossijonas/stamp/actions/workflows/test-integration-ubuntu.yml/badge.svg" alt="Ubuntu"></a>
  <a href="https://github.com/rossijonas/stamp/actions/workflows/test-integration-debian.yml"><img src="https://github.com/rossijonas/stamp/actions/workflows/test-integration-debian.yml/badge.svg" alt="Debian"></a>
  <a href="https://github.com/rossijonas/stamp/actions/workflows/test-integration-fedora.yml"><img src="https://github.com/rossijonas/stamp/actions/workflows/test-integration-fedora.yml/badge.svg" alt="Fedora"></a>
  <a href="https://github.com/rossijonas/stamp/actions/workflows/test-integration-centos.yml"><img src="https://github.com/rossijonas/stamp/actions/workflows/test-integration-centos.yml/badge.svg" alt="CentOS"></a>
  <a href="https://github.com/rossijonas/stamp/actions/workflows/test-integration-rocky.yml"><img src="https://github.com/rossijonas/stamp/actions/workflows/test-integration-rocky.yml/badge.svg" alt="Rocky"></a>
  <a href="https://github.com/rossijonas/stamp/actions/workflows/test-integration-arch.yml"><img src="https://github.com/rossijonas/stamp/actions/workflows/test-integration-arch.yml/badge.svg" alt="Arch"></a>
  <a href="https://github.com/rossijonas/stamp/actions/workflows/test-integration-opensuse.yml"><img src="https://github.com/rossijonas/stamp/actions/workflows/test-integration-opensuse.yml/badge.svg" alt="openSUSE"></a>
  <a href="https://github.com/rossijonas/stamp/releases/latest"><img src="https://img.shields.io/github/v/release/rossijonas/stamp" alt="Release"></a>
  <a href="https://codecov.io/gh/rossijonas/stamp"><img src="https://codecov.io/gh/rossijonas/stamp/branch/main/graph/badge.svg" alt="codecov"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache_2.0-blue.svg" alt="License"></a>
  <a href="https://pkg.go.dev/github.com/rossijonas/stamp"><img src="https://pkg.go.dev/badge/github.com/rossijonas/stamp.svg" alt="Go Reference"></a>
  <a href="CODE_OF_CONDUCT.md"><img src="https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg" alt="Code of Conduct"></a>
</p>

---

## ▪ Intro

> ⚠️ **Active Development:** `stamp` is currently in the MVP phase of active development. Features are being delivered incrementally. For a complete look at our progress, check the [Implementation Plan](docs/IMPLEMENTATION_PLAN.md).

Stamp is a lightweight yet powerful tool that wraps many package managers into one. Install, track, and restore without changing your tools. One manifest. One command to restore it all.

Full documentation at **[https://gostamp.dev](https://gostamp.dev)**.

### Features

**`[>]` Multi-Manager Wrapper** — Install, search, and remove packages across DNF, APT, Brew, Flatpak, Snap, Zypper, Pacman, and MacPorts &mdash; plus language toolchains (`go`, `cargo`, `pipx`, `npm`) &mdash; through a single CLI.

**`[+]` Cross-Platform** — Works on **Linux** &amp; **macOS** today. **Windows** support is planned.

**`[*]` Automatic Intent Tracking** — Every install is recorded in a portable `manifest.toml`. Only your intentional choices, not dependency noise.

**`[<]` One-Command Restore** — Clone your dotfiles and run `stamp restore` to rebuild your entire environment on a new machine in minutes.

**`[/]` Safety Net Reconciliation** — Forgot to use stamp? `stamp reconcile` detects packages installed outside the tool and tracks them automatically.

**`[#]` Unified Repository Management** — Add, remove, and list third-party repositories &mdash; PPAs, taps, remotes &mdash; across all managers with the same interface.

**`[?]` Self-Contained Docs** — Built-in man pages (`stamp man install`), shell completions, and auto-generated CLI reference docs.

**`[!]` Context Preservation** — Add `--note` to any install so you remember *why* you needed a package six months later.

**`[$]` UNIX Compliant** — XDG Base Directory, NO_COLOR support, strict stdout/stderr separation, and TTY-aware sudo.

See **[Installation](https://gostamp.dev/getting-started/installation.html)**, **[Usage](https://gostamp.dev/usage/)**, **[CLI Reference](https://gostamp.dev/usage/stamp.html)**, and more at  **[https://gostamp.dev](https://gostamp.dev)**.

## ▪ Compatibility

See the full [OS × Manager compatibility matrix](docs/history/os-manager-matrix.md). Integration test coverage is documented [here](docs/INTEGRATION_TEST_COVERAGE.md).

## ▪ Architecture & Vision

Read the [Project Vision](docs/VISION.md) to understand the "why" behind the project, or check out the [Technical Specs](docs/SPEC.md) and [Architecture Decisions](docs/decisions/).

## ▪ License

This project is licensed under the Apache License, Version 2.0 (Apache-2.0) - see the [LICENSE](LICENSE) file for details.
