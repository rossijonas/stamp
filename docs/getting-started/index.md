---
---

## What is Stamp?

Stamp is a lightweight yet powerful CLI tool that wraps **many package managers into one**. It lets you install, track, and restore packages across DNF, APT, Brew, Flatpak, Snap, Zypper, Pacman, and MacPorts — plus language toolchains like `go`, `cargo`, `pipx`, and `npm` — all through a single command.

### How it works

1. **Install** — `stamp install htop` auto-detects the best manager, runs the native install, and records your intent in a portable `manifest.toml`.
2. **Track** — Every intentional install is saved. No dependency noise, only your choices.
3. **Restore** — `stamp restore` rebuilds your entire environment on a new machine from your manifest.
4. **Reconcile** — Forgot to use stamp? `stamp reconcile` detects packages installed outside the tool and adds them retroactively.

Stamp doesn't replace your package managers — it unifies them into a single workflow.

### Vision

Stamp is built for developers who want reproducible environments without the overhead of Nix or Ansible. Read the full [Vision](/VISION.html) to understand the project's goals.
