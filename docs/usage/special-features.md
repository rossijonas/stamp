---
---

# Special Features

Cross-manager features that go beyond basic install, search, and remove.

| Feature | Description | Supported Managers | Stamp Command | Status |
|---------|-------------|-------------------|---------------|--------|
| **Cask (GUI Apps)** | macOS GUI application management via `brew install --cask`. Auto-detected on install, stored in manifest. | Brew | `stamp install <pkg> -m brew` | ✅ Complete |
| **File Search (provides)** | Find which package owns a specific file or binary. | DNF, APT, Pacman, Paru, Zypper, MacPorts | `stamp provides <file>` | ✅ Complete |
| **Orphan Cleanup (autoremove)** | Remove unused dependencies that were pulled in automatically. | Brew, DNF, APT, Pacman, Paru, Zypper, Flatpak, MacPorts | `stamp autoremove` | ✅ Complete |
| **Cache Cleanup (clean)** | Clear locally cached package files to free disk space. | Brew, DNF, APT, Pacman, Paru, Zypper, Snap, MacPorts | `stamp clean` | ✅ Complete |
| **Version Pinning (hold)** | Pin packages at specific versions to prevent accidental upgrades. | APT, DNF, Pacman, Paru | `stamp hold <pkg>` | ✅ Complete |
| **Group Install** | Install DNF package groups (e.g. "Development Tools"). | DNF | `stamp install --group` | 🚧 Planned |
| **Flatpak Override** | Manage Flatpak sandbox permissions (filesystem, socket, device access). | Flatpak | `stamp override` | 🚧 Planned |
| **Aliases** | Native command aliases for every supported package manager. | All | See [Aliases Matrix](aliases.html) | 🚧 Planned |

## Related

- [Feature × Manager Support Matrix](../history/feature-per-manager-matrix.html) — per-manager feature coverage
- [Aliases Matrix](aliases.html) — native command aliases
- [Architecture Decision ADR-013](../decisions/ADR-013-feature-expansion.html) — design decisions for these features
