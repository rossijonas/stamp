---
---

## What is Stamp?

Stamp is a lightweight yet powerful CLI tool that wraps **many package managers into one**. It lets you install, track, and restore packages across DNF, APT, Brew, Flatpak, Snap, Zypper, Pacman, Paru, and MacPorts — plus `cargo`, `npm`, `pipx`, `uv`, and `go` language-specific package managers — all through a single command.

### How it works

1. **Install** — `stamp install htop` auto-detects the best manager, runs the native install, and records your intent in a portable `manifest.toml`.
2. **Track** — Every intentional install is saved. No dependency noise, only your choices.
3. **Restore** — `stamp restore` rebuilds your entire environment on a new machine from your manifest.
4. **Reconcile** — Forgot to use stamp? `stamp reconcile` detects packages installed outside the tool and adds them retroactively.

Stamp doesn't replace your package managers — it unifies them into a single workflow.

### Origins: stamped vs reconciled

Every entry in your manifest records how stamp learned about it:

- **stamped** — you installed it via `stamp install` (or `stamp reinstall`). Stamp recorded your intent at install time.
- **reconciled** — you installed it directly via your package manager (e.g. `dnf install`) and `stamp reconcile` discovered it afterwards.

Both are user-installed packages — the difference is how stamp learned about them.

- `stamp list -t stamped` shows everything you explicitly tracked through stamp.
- `stamp list -t reconciled` shows everything reconcile auto-discovered.
- `stamp list -t stamped-packages` narrows to packages only, excluding repos.

### Vision

Stamp is built for developers who want reproducible environments without the overhead of Nix or Ansible. Read the full [About / Vision](/project/about.html) to understand the project's goals.

### See Also

- [OS × Manager Compatibility Matrix](/history/os-manager-matrix.html) — which managers work on which OS
- [Feature Reference](/project/features.html) — feature implementation status and per-manager support
