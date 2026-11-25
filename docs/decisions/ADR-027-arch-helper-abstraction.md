---
---

# ADR-027: Shared Arch Helper Abstraction and yay Support

## Status
Accepted

## Date
2026-11-24

## Context

Stamp supported two pacman-compatible managers: `pacman` for official repositories and `paru` as the AUR helper. `paru` was a near-copy of `pacman` with a different binary, and its write operations were wrapped in `sudo`, which cannot work because AUR helpers refuse to run as root.

`yay` is now the most widely used AUR helper. Adding it as a third copy would triple the duplicated logic and leave the privilege bug in place. ADR-012 also documented a `pacman → paru` alias that was never implemented.

## Decision

### One shared adapter

`internal/manager/arch.go` implements the shared pacman behavior once. `pacman`, `paru`, and `yay` are thin configurations:

| Field | pacman | paru | yay |
| :--- | :--- | :--- | :--- |
| `bin` | `pacman` | `paru` | `yay` |
| `escalate` (stamp-level sudo) | yes | no | no |
| `strictQu` (reject unknown `-Qu` output) | yes | no | no |
| extra flags | none | `--noconfirm` only | `--answerclean None --answerdiff None --answeredit None`; `--answerupgrade None` on full update |

`pacman` keeps its `Previewer` implementation; helpers do not preview.

### Detection and precedence

Exactly one Arch manager is registered, in the order `paru > yay > pacman`. When more than one helper is installed, stamp warns once and names the ignored binaries. No configuration knob is added; users who want a different helper can remove the preferred binary or use `-m`.

### Privilege model

AUR helpers run as the invoking user and escalate internally. Stamp never wraps `paru`/`yay` in `sudo`, and helper write operations refuse to run when `euid == 0` with an actionable error. Pacman keeps `sudo` because it performs the package transaction. Helpers remain in the sudo preflight set: their internal escalation still benefits from a cached credential.

Orphan removal always uses `sudo pacman -Rs`, because helpers cannot remove packages themselves. `Clean` and `Refresh` use the helper binary as the user; pacman uses `sudo pacman`.

### Portability

Manifests keep the actual helper name. `manager.Equivalents` groups `pacman`, `paru`, and `yay`; CLI resolution and restore prefer an exact match, then any installed member of the group. A manifest recorded as `paru` restores on a `yay` host, and `-m pacman` selects the installed helper.

Spec validation follows each adapter's grammar: `yay` accepts pacman's grammar (including `repo/pkg`), while `paru` keeps its stricter bare-name rule. A `repo/pkg` entry recorded by `yay` therefore cannot be restored on a `paru` host; this is a documented limitation.

## Consequences

- One implementation covers three managers; a new helper is a configuration, not a copy.
- `paru` write operations work as the user instead of failing under `sudo`.
- `sudo stamp update` fails cleanly for helpers instead of breaking an AUR build.
- The manifest stays truthful while restore remains portable across helpers.
- The Arch integration job runs as a non-root user, so `yay` is exercised there: a shim checks argv/flags deterministically, then a best-effort `yay-bin` install covers real search/install/remove, root refusal, the multiple-helper warning, and a paru→yay restore. AUR failures skip the real-helper checks.

## Alternatives Considered

- **Copy `paru.go` into `yay.go`.** Rejected: triples duplication and preserves the privilege bug.
- **Route official packages to pacman and AUR packages to yay.** Rejected: requires origin metadata and extra network calls, and does not improve correctness.
- **Drop privileges to `$SUDO_USER` when stamp runs as root.** Rejected: re-executing the whole process and its context is more fragile than refusing root with guidance.
- **Config-selected helper.** Deferred: precedence plus a warning covers the common case; add configuration only if users need it.
