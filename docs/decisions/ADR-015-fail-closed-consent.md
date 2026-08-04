---
---

# ADR-015: Fail-Closed Consent for Destructive Commands

## Status
Accepted

> **Note:** The preview mechanism in section 3 below (raw `(string, error)` previews, `ErrNoop` string detection, `Info` fallback) is **superseded by [ADR-016](ADR-016-unified-preview-contract.md)**, which introduces the typed, adapter-owned `Preview` contract. This ADR's consent model (`WithYes`, `ErrConfirmationRequired`, the CLI gate, fail-closed non-TTY) remains current.

## Date
2026-08-01

## Context

`stamp install`, `stamp reinstall`, and `stamp remove` executed destructive package-manager operations with no confirmation: the adapters hardcoded assume-yes flags (`-y` for dnf/apt/zypper, `--noconfirm` for pacman, `--yes` for macports) and the CLI never prompted. Any direct caller of `Adapter.Install`/`Remove`/`Reinstall` mutated the system silently. `stamp restore` documented a `-y` flag but had no prompt at all, and `stamp update`'s confirmation defaulted to "yes" and was TTY-gated, so it silently proceeded in CI/pipelines.

The desired behavior (issue #168) mirrors native package-manager UX (e.g. `dnf install`): show what will happen, then prompt, honoring `-y` to skip.

## Decision

Two layers, both required for a coherent fail-closed model:

### 1. Centralized CLI confirmation gate

A shared gate (`internal/cli/confirm.go`) implements refresh → preview → prompt → run:

- `confirmDestructive` renders a native transaction preview (see Previewer below), then prompts with a default of **no** (`[y/N]`). `-y/--yes` skips refresh, preview, and prompt.
- `requireConsent` is a lighter prompt-only gate for `autoremove`, `clean`, `hold`, `unhold`, and `repo add/remove`.
- **Non-interactive input without `-y` fails closed** (returns "aborted"). Pipelines and CI never silently mutate the system. This replaces the previous fail-open defaults (install/remove/reinstall had none; restore had none; update silently proceeded).

### 2. Adapters fail closed via explicit consent

Destructive adapter methods no longer run unconditionally. They require a `manager.WithYes(ctx)` marker, set only by the CLI gate after the user confirmed or passed `-y`:

- `WithYes(ctx)` sets a context value; `requireConsent`/`isYes` read it.
- All mutating methods (`Install`, `Reinstall`, `Remove`, `Update`, `AddRepo`, `RemoveRepo`, `AutoRemove`/`Clean` when not `--dry-run`, `Hold`, `Unhold`) return `ErrConfirmationRequired` when consent is absent.
- The `manager.Mock` test double enforces consent too, so every CLI path is verified end-to-end: a command that forgets to set consent fails its tests loudly.

### 3. Native transaction preview (optional capability)

A new optional interface `manager.Previewer` gives a read-only, side-effect-free preview of what a real operation would do, using the package manager's own dry-run:

| Manager | Preview flags | Privilege |
|---|---|---|
| dnf / yum | `install/remove --assumeno`, `group ... --assumeno` | sudo (side-effect-free) |
| apt | `install/remove --assume-no` (implies `--simulate`) | none |
| pacman | `-S --print` / `-R --print` | none |
| brew | `install/uninstall --dry-run` | none |
| flatpak | `install/uninstall --dry-run` | none |
| zypper | `install/remove --dry-run` | none |
| npm | `install/uninstall --dry-run -g` | none |

Managers without a native dry-run (snap, cargo, go, pipx, uv, paru) and any manager whose dry-run fails fall back to `Info`. Preview output is streamed verbatim; stamp never parses and re-renders it, keeping the vendor as the source of truth.

## Consequences

- `install`, `remove`, `reinstall`, `restore`, `update`, `autoremove`, `clean`, `hold`, `unhold`, and `repo add/remove` now prompt unless `-y` is passed; non-interactive runs without `-y` refuse with a **non-zero exit** (a forgotten `-y` in CI fails loud). Interactive declines and no-op previews stop cleanly with exit 0.
- `stamp update` no longer silently proceeds in CI without `-y` (behavior change from ADR-011's fail-open prompt).
- `stamp restore` gains the prompt it previously only documented.
- Direct adapter calls (future code, third-party) fail closed instead of mutating the system — defense-in-depth at the privileged boundary.
- `flatpak override` (sandbox permission changes) is intentionally outside this consent model: it is reversible, flatpak-scoped, and routed through a separate optional interface. Revisit if it becomes broadly destructive.
- Preview is advisory: TOCTOU between preview and execution is inherent to confirm-then-run UX (dnf itself re-resolves at execution). Single-package operations minimize the delta.
