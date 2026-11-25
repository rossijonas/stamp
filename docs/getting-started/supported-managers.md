---
---

## Supported Package Managers

Stamp registers a manager only when its executable is on `$PATH`, and some
managers are limited to one operating system. The table below lists every
manager stamp can register. Run `stamp doctor` to see which ones are active on
this machine.

| Manager | OS | Notes |
| :--- | :--- | :--- |
| apt | Debian, Ubuntu | Falls back to `apt-get` when `apt` is not installed |
| brew | macOS, Linux | Homebrew, user-space installs; casks for GUI apps |
| cargo | macOS, Linux | Rust crates via `cargo install` |
| dnf | RHEL family (Fedora, RHEL, CentOS, Rocky, AlmaLinux) | Prefers `dnf`; `yum` is an alias, not a separate manager |
| flatpak | Linux | Sandboxed applications |
| go | macOS, Linux | Module paths via `go install` |
| macports | macOS | Detected through the `port` binary |
| npm | macOS, Linux | Globally installed CLI tools |
| pacman | Arch Linux | Fallback when `paru` is not installed |
| paru | Arch Linux | Preferred AUR helper; AUR support |
| pipx | macOS, Linux | Python CLI tools |
| snap | Linux | Universal Linux packages |
| uv | macOS, Linux | Python CLI tools via `uv tool` |
| yay | Arch Linux | AUR helper used when `paru` is not installed; runs as the user |
| zypper | openSUSE, SLES | |

### Detection rules

- A manager is registered only when its binary is found in `$PATH`. Run
  `stamp doctor` for the current list and resolved paths.
- On Arch Linux, exactly one helper is registered: `paru`, then `yay`, then
  plain `pacman`. Stamp warns when more than one helper is installed.
- `dnf` registers once, under the `dnf` name, whether the system provides `dnf`
  or `yum`.
- `macports` is macOS-only and is detected through the `port` binary.

### Windows

Windows is not supported yet. The Winget, Chocolatey, and Scoop adapters are
tracked in [#54](https://github.com/rossijonas/stamp/issues/54),
[#55](https://github.com/rossijonas/stamp/issues/55), and
[#56](https://github.com/rossijonas/stamp/issues/56).

### See also

- [Package Specification Support](/project/package-spec-audit.html) — accepted native install grammar per manager
- [Feature Reference](/project/features.html) — feature support per manager
- [OS × Manager Matrix](/history/os-manager-matrix.html) — platform compatibility
