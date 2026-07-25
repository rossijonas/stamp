---
---

# ADR-009: User-Level Installation (No Sudo Required)

## Status
Accepted

## Date
2026-07-25

## Context

Stamp's install script and documentation defaulted to `/usr/local/bin/` as the install directory, which is owned by root. This meant:

- Installing stamp required `sudo` for most users
- `stamp self-update` needed `sudo` to replace the binary
- The `curl [...] | bash` pattern escalated to root from an untrusted pipe — a security anti-pattern
- Stamp's data and configuration already followed XDG conventions at the user level (`~/.config/stamp/`, `~/.local/share/stamp/`), but the binary did not

The discrepancy between user-level data and system-level binary was inconsistent. All major toolchains (rustup, nvm, pipx, go install) install to user-writable directories by default.

## Decision

Change the default install directory from `/usr/local/bin/` to `$HOME/.local/bin/` for all installation methods. The `STAMP_INSTALL_DIR` environment variable is retained as an override for users who want a system-wide install.

### Implementation

1. **Install script (`docs/install`):** Default `INSTALL_DIR` changed from `/usr/local/bin/` to `$HOME/.local/bin/`
2. **Landing page (`docs/index.html`):** Release and source tabs updated to use `~/.local/bin/` without `sudo`
3. **Install docs (`docs/getting-started/installation.md`):** Updated commands and added `$PATH` setup section

### PATH Handling

The install script checks whether `$INSTALL_DIR` is in `$PATH` after installation. If not, it prints a warning with per-shell instructions:

```bash
▪ Stamp installed to /home/user/.local/bin/stamp
  Warning: /home/user/.local/bin is not in your PATH.
  To add it:
    Bash: echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
    Zsh:  echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
    Fish: fish_add_path "$HOME/.local/bin"
```

### Legacy Detection

If stamp exists at `/usr/local/bin/stamp` when the install script runs, a warning is printed:

```bash
  ⚠ stamp also found at /usr/local/bin/stamp
    Remove it to avoid confusion: sudo rm /usr/local/bin/stamp
```

## Alternatives Considered

### Keep `/usr/local/bin/` (status quo)
- **Pros:** No change needed, existing users unaffected
- **Cons:** Requires sudo for install and self-update, security anti-pattern with `curl | sudo bash`, inconsistent with XDG data locations
- **Rejected:** The cons outweighed the convenience of no change

### Use `$HOME/bin/` instead of `$HOME/.local/bin/`
- **Pros:** Simpler path, some distros include it in PATH
- **Cons:** Not the XDG-specified path, `.local/bin` is the standard per freedesktop.org
- **Rejected:** XDG compliance is more important than minor PATH convenience

### Auto-detect and add to PATH automatically
- **Pros:** Zero user friction
- **Cons:** Modifying shell config files without consent is dangerous and fragile
- **Rejected:** Print warning with instructions instead — user chooses to add

## Consequences

- Stamp installs to `$HOME/.local/bin/` by default — no sudo needed
- `stamp self-update` works without sudo
- `curl [...] | bash` no longer escalates to root
- Users must have `~/.local/bin` in their `$PATH` — script warns if missing
- Users with stamp at `/usr/local/bin/` are warned but not automatically migrated
- `STAMP_INSTALL_DIR` env var preserved for system-wide installs
- Test infrastructure (Dockerfiles) unaffected — they use `COPY` not the install script
