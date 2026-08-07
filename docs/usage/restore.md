---
---

## Restoring on a New Machine

Place your `manifest.toml` in the right location and run:

```bash
cp manifest.toml ~/.config/stamp/          # copy your manifest to the stamp config dir
stamp restore -y
```

```text
Phase 1: Restoring Repositories...
  restored repository flathub via flatpak
Phase 2: Restoring Packages...
  installed htop via dnf
  installed lazygit via brew
  installed spotify via flatpak
Restore completed successfully

Restore respects package metadata: Homebrew casks are installed with `--cask`
and DNF groups are installed with `--group` automatically.
```

### Dry run

```bash
stamp restore --dry-run
```

```text
▪ Dry Run (Preview):
Repositories:
  - flathub (flatpak) https://dl.flathub.org/repo/flathub.flatpakrepo
  - homebrew/cask (brew)
Packages:
  - htop (dnf)
  - lazygit (brew)
  - spotify (flatpak)
```

### Restoration order

Stamp restores in two phases:

1. **Phase 1 (Sequential):** All repositories are added one by one (order matters for dependencies)
2. **Phase 2 (Concurrent):** All packages are installed in parallel across all managers
