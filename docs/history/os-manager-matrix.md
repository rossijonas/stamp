---
---

## OS × Manager Matrix

| Manager | Ubuntu | Debian | Fedora | CentOS | Rocky | Arch | openSUSE | macOS | Windows |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **DNF** | - | - | ✓ | ✓ | ✓ | - | - | - | - |
| **APT** | ✓ | ✓ | - | - | - | - | - | - | - |
| **Zypper** | - | - | - | - | - | - | ✓ | - | - |
| **Snap** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | - | - |
| **Flatpak** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | - | - |
| **Brew** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | - |
| **Paru** | - | - | - | - | - | ✓* | - | - | - |
| **Pacman** | - | - | - | - | - | ✓* | - | - | - |
| **MacPorts** | - | - | - | - | - | - | - | ✓ | |
| **Winget** | - | - | - | - | - | - | - | - | |
| **Chocolatey** | - | - | - | - | - | - | - | - | |
| **Scoop** | - | - | - | - | - | - | - | - | |


### Language Toolchains

| Tool | Ubuntu | Debian | Fedora | CentOS | Rocky | Arch | openSUSE | macOS | Windows |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| `go` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `cargo` | ~ | ~ | ~ | ~ | ~ | ~ | ~ | ~ | ~ |
| `pipx` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `uv` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `npm` | ~ | ~ | ~ | ~ | ~ | ~ | ~ | ~ | ~ |

✓ = Implemented  
✓* = Implemented with conditions  
~ = Planned  
– = Not available on this OS  
*(blank)* = Not implemented or not planned  

### Conditions

1. **Paru** replaces **Pacman** when both are detected on Arch Linux.  
   If Paru is not found, Stamp falls back to Pacman.

### See Also

- [Feature Reference](../project/features.html) — feature implementation status and per-manager support
