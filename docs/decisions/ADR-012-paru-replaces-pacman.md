---
---

# ADR-012: Paru Replaces Pacman for Arch Linux (AUR Support)

## Status
Accepted

## Date
2026-07-28

## Context

Arch Linux has two package management scopes:

1. **Official repositories** — handled by `pacman`. Contains ~15,000 binary packages maintained by the Arch Linux team.
2. **AUR (Arch User Repository)** — community-maintained PKGBUILD scripts. Contains ~80,000+ packages built from source.

`paru` is a pacman wrapper that handles both scopes. It delegates all official-repo operations to pacman internally while adding AUR support. Both tools share the same `/var/lib/pacman/` database — `paru -Qq` and `pacman -Qq` produce identical output.

This shared database makes co-existence impossible in stamp: `ListInstalled()` would return every package twice, breaking `reconcile` (duplicate entries), `restore` (double install), and `list` (confusing output).

There is precedent for this replacement pattern: yum→dnf (Task 26) where stamp treats them as interchangeable, registering only one.

## Decision

Replace pacman with paru when paru is available on the system.

### Detection Rule

1. Check `paru` binary first.
2. If found → register Paru adapter, skip Pacman.
3. If not found → register Pacman adapter (current behavior).

### Manager Alias

`ResolveManager("pacman")` returns `"paru"`, allowing backward compatibility:
- Existing manifests with `manager = "pacman""` continue to work.
- Commands like `stamp doctor -m pacman` resolve to the paru adapter.
- New installations on paru-capable systems store `manager = "paru"`.

### Adapter Implementation

The Paru adapter (`internal/manager/paru.go`) mirrors Pacman (`internal/manager/pacman.go`) with binary `"paru"` and uses `sudoCmd` for privilege escalation (same pattern as all system adapters). Key differences from the Pacman adapter:

- `paru -Ss` searches both official repos and AUR (more results).
- `paru -Syu` upgrades both official and AUR packages.
- `paru -Qu` lists updates from both scopes.

Output format is identical to pacman: `parsePacmanSearch` and `parsePacmanQu` parsers are shared.

### CLI Integration

`stamp doctor` includes `paru` in its known managers list. Detection is tested via mock `exec.LookPath` override.

## Alternatives Considered

### Both paru and pacman active
Rejected. Shared database means `ListInstalled()` double-counts every package. The yum→dnf precedent shows replacement is the correct pattern.

### Use paru only for AUR operations, pacman for official
Rejected. Paru already handles this internally. Splitting would require stamp to know which packages are AUR vs official — information not available without separate API calls.

### Support yay instead of paru
Deferred. Yay is an alternative AUR helper with a different CLI. Paru is the most popular AUR helper as of 2026. Yay support can be added as a follow-up if demand warrants.

## Consequences

- Arch users with paru get AUR support automatically on next `stamp` update.
- Arch users without paru see no change (pacman still works).
- Existing manifests with `manager = "pacman"` continue to work via alias.
- No database migration needed — stamp stores whatever manages the package.
- New integration: `docs/history/os-manager-matrix.md` updated with paru.
