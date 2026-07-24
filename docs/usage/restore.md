---
---

## Restoring on a New Machine

Clone your dotfiles (containing your `manifest.toml`) and run:

```bash
git clone https://github.com/you/dotfiles.git
cp dotfiles/stamp/manifest.toml ~/.config/stamp/
stamp restore -y
```

```text
▪ Phase 1: Restoring repositories...
  ✅ added flathub via flatpak
  ✅ added homebrew/cask via brew
▪ Phase 2: Restoring packages...
  ✅ installed htop via apt
  ✅ installed lazygit via brew
  ✅ installed spotify via flatpak
✅ Restore complete — 3 packages installed
```

### Dry run

```bash
stamp restore --dry-run
```

```text
▪ Would restore:
    Repositories: flathub, homebrew/cask
    Packages: htop, lazygit, spotify
  Run stamp restore to proceed.
```

### Restoration order

Stamp restores in two phases:

1. **Phase 1 (Sequential):** All repositories are added one by one (order matters for dependencies)
2. **Phase 2 (Concurrent):** All packages are installed in parallel across all managers
