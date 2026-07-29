---
---

## Feature × Manager Support Matrix

Which Stamp features are available for each supported package manager.

### Legend

```
✓ = Full support
✗ = Not supported (returns informative error)
```

### Feature Matrix

| Feature | DNF | APT | Pacman | Paru | Zypper | Snap | Flatpak | Brew | MacPorts | Go | Pipx | Uv | Npm | Cargo |
|---------|:---:|:---:|:------:|:----:|:------:|:----:|:-------:|:----:|:--------:|:--:|:----:|:--:|:---:|:-----:|
| **Install** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **Reinstall** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **Remove** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **Search** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✓ |
| **Info** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **Update** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **CheckUpdate** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **Doctor** | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **Cask** | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **Hold** | ✗ | ✓ | ✓ | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **Add Repo** | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **Remove Repo** | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **List Repos** | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |

### Notes

- **Search:** Cargo is the only toolchain adapter with native search support (`cargo search`). Other toolchains (Go, Npm, Pipx, Uv) do not support search.
- **Update:** All managers support both batch (`stamp update`) and single-package (`stamp update -p <pkg> -m <mgr>`) updates.
- **Doctor:** Only Brew has a native `doctor` command. Other managers print an informational message and continue.
- **Cask:** Brew-specific feature for macOS GUI applications (`brew install --cask`). Flatpak and Snap support GUI apps natively on Linux through their standard install flow — no separate flag needed.
- **Repo management:** DNF supports COPR repos, APT supports PPAs and custom URLs, Brew supports taps, Flatpak supports remotes. Other managers do not support third-party repository management through Stamp.
- **CheckUpdate:** Toolchain managers (Go, Pipx, Uv) cannot preview updates. They print an informational notice during `stamp update --check` and continue.

### See Also

- [OS × Manager Compatibility Matrix](os-manager-matrix.html) — which managers work on which operating systems
- [Feature Matrix](../FEATURE_MATRIX.html) — Stamp CLI feature completion status
