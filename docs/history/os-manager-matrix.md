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
| **Yay** | - | - | - | - | - | ✓* | - | - |
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

1. Exactly one AUR helper is registered on Arch Linux, in the order
   **Paru** > **Yay** > **Pacman**. All three rows are marked `✓*` because any of
   them can be active; Stamp warns when more than one helper is installed.

### Windows

Windows is not supported yet. See
[Supported Managers](../getting-started/supported-managers.html) for the
planned adapters.

### See Also

- [Supported Managers](../getting-started/supported-managers.html) — full manager list and detection rules
- [Feature Reference](../project/features.html) — feature implementation status and per-manager support
