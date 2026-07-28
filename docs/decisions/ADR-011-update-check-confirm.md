---
---

# ADR-011: Update Command Two-Phase Flow (Check + Confirm)

## Status
Accepted

## Date
2026-07-26

## Context

The existing `stamp update` command executes system-wide upgrades across all available package managers concurrently. Because it runs parallelized, stamp passes the equivalent of `-y` or `--noconfirm` to all underlying package managers to prevent prompts from hanging.

While fast, this has several UX drawbacks:
1. Users have no visibility into what packages are about to be upgraded before the process starts.
2. Accidental mass upgrades cannot be easily aborted.
3. It deviates from the native behavior of individual system package managers (e.g., `apt`, `dnf`, `flatpak`, `pacman`) which show a preview and prompt for confirmation by default.

A unified two-phase flow (check for updates, show aggregated list, prompt for confirmation, then upgrade) is desirable but must account for:
- Not all package managers support a "preview/check" feature (e.g., `pipx`, `uv`, `go install`).
- Serialized checks add some latency before the upgrade starts.
- Non-interactive automation/scripts (`-y` flag) should remain extremely fast and skip the check phase entirely to prevent latency overhead.

## Decision

We will implement a two-phase check-and-confirm execution pipeline for the `stamp update` command. 

### Execution Matrix

| Flags | Check Phase | Confirmation Prompt | Run Phase |
|-------|-------------|---------------------|-----------|
| (none) | Serialized (all managers) | Prompt User | Parallelized |
| `--check` | Serialized (all managers) | None (exit 0) | None |
| `-y` | **Skipped** | None | Parallelized |
| `--serial` | Serialized (all managers) | Prompt User | Serialized |
| `--serial -y` | **Skipped** | None | Serialized |
| `-m <mgr>` | Scoped Check | Prompt User | Serialized/Parallelized |
| `-m <mgr> -y` | **Skipped** | None | Serialized/Parallelized |
| `-p <pkg> -m <mgr>`| Scoped Check | Prompt User | Serialized/Parallelized |

### Adapter Interface Changes

We will extend the `Adapter` interface in `internal/manager/manager.go` with a new method:

```go
type UpdateInfo struct {
    Package          string `json:"package"`
    CurrentVersion   string `json:"current_version,omitempty"`
    AvailableVersion string `json:"available_version,omitempty"`
}

type Adapter interface {
    // ...
    CheckUpdate(ctx context.Context, pkg string) ([]UpdateInfo, error)
}
```

### Capability Division

Adapters will handle `CheckUpdate` as follows:

1. **System Managers (Native Support):**
   - `dnf`: `dnf check-update`
   - `apt`: `apt list --upgradable`
   - `brew`: `brew outdated --json`
   - `flatpak`: `flatpak remote-ls --updates`
   - `zypper`: `zypper list-updates`
   - `pacman`: `pacman -Qu`
   - `snap`: `snap refresh --list`
   - `macports`: `port outdated`

2. **Language Toolchain Managers (No Native Support):**
   - `go`, `pipx`, `uv`: Return an unsupported error. The CLI will catch this and display a short informational notice (e.g., `pipx: cannot preview updates`) instead of failing.

## Consequences

- Users get full, clear visibility into available updates across all active package managers before upgrades begin.
- Automation/CI scripts using `stamp update -y` experience **zero latency regression** because they bypass the check phase entirely.
- Added a `--check` flag to provide a clean, non-mutating "dry-run" check-only mode.
- Hand-written and auto-generated command documentations (man pages, website) must be updated to reflect the new workflow and matrix.
