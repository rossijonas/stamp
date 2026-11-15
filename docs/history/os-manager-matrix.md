---
---

## OS × Manager Matrix

| Manager | Ubuntu | Debian | Fedora | CentOS | Rocky | Arch | openSUSE | macOS |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **DNF** | - | - | ✓ | ✓ | ✓ | - | - | - |
| **APT** | ✓ | ✓ | - | - | - | - | - | - |
| **Zypper** | - | - | - | - | - | - | ✓ | - |
| **Snap** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | - |
| **Flatpak** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | - |
| **Brew** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **Paru** | - | - | - | - | - | ✓* | - | - |
| **Pacman** | - | - | - | - | - | ✓* | - | - |
| **MacPorts** | - | - | - | - | - | - | - | ✓ |


### Language Toolchains

| Toolchain | Ubuntu | Debian | Fedora | CentOS | Rocky | Arch | openSUSE | macOS |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| `go` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `cargo` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `pipx` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `uv` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `npm` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |

✓ = Implemented  
✓* = Implemented with conditions  
- = Not available on this OS  

### Conditions

1. **Paru** replaces **Pacman** when both are installed on Arch Linux. Both rows
   are marked `✓*` because either can be active; Stamp falls back to Pacman when
   Paru is absent.

### Windows

Windows is not supported yet. See
[Supported Managers](../getting-started/supported-managers.html) for the
planned adapters.

### See Also

- [Supported Managers](../getting-started/supported-managers.html) — full manager list and detection rules
- [Feature Reference](../project/features.html) — feature implementation status and per-manager support
