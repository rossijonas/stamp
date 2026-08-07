---
---

# ADR-018: Missing-Package Drift Visibility

## Status
Accepted

## Date
2026-08-07

## Context

Issue #182: a manifest-tracked package removed via the native package manager
(e.g. `sudo dnf remove foo`) was invisible to every command.

- `stamp reconcile` only ever inspects `Delta.Added`/`AddedRepos`
  (`collectDiscovered`); `Delta.Removed` was ignored. On the no-drift path the
  new snapshot is still saved, so the removal is silently recorded as reality
  with zero signal while the manifest still lists the package.
- `stamp doctor` parses the manifest and checks the file exists, but never
  compares it against what is installed, so dangling entries go unreported.
- `stamp ls` is a pure manifest read and `stamp manifest diff` compares the
  manifest against *backups*, not against the system.

`stamp restore` remains the convergence tool (manifest → system), and reconcile
is documented as one-way tracking (system → manifest). The gap was *signal*:
an accidental removal goes unnoticed until a later restore.

## Decision

Surface missing packages as **diagnostics only**. Three touchpoints, all
warning-only — no manifest mutation, no auto-reinstall, no auto-remove:

1. **`stamp reconcile`** prints a stderr warning listing manifest entries that
   disappeared from the system since the last snapshot (`N manifest
   package(s) not installed: ...`), with a pointer to `stamp ls --type missing`
   and `stamp restore`. Fires on the no-drift and drift paths, including
   `--dry-run`. The removal is still recorded in the new snapshot — the
   snapshot reflects reality, the manifest holds intent.
2. **`stamp doctor`** reports the same set under a `Missing:` section in
   "Manifest Integrity" (TTY) and as `manifest.missing` (`--json`). The
   `✓ Healthy` status is unchanged: a missing package is drift, not manifest
   corruption.
3. **`stamp ls --type missing`** is the queryable listing surface (TTY +
   `--json`, composes with `-m`). It queries active managers' installed
   packages directly (not snapshots), so it always reflects the live system.

Shared computation lives in `internal/cli/missing.go`:
`missingFromSystem` (doctor, `ls --type missing`) and `missingFromDeltas`
(reconcile) both return `[]manifest.Package`, excluding `Group` and `Cask`
entries (they never appear in an installed list, so including them would be a
false positive). A manager whose installed list fails to load is skipped —
the check is best-effort diagnostics, not a hard error.

### Grouping rules

- Missing = manifest entry `(name, manager)` whose manager is active/available
  and whose `ListInstalled` lacks `name`.
- Same-manager match only. A manifest entry satisfied via another manager
  (e.g. flatpak) still counts as missing for its declared manager — acceptable
  because nothing is auto-actioned.
- Deterministic output: deduplicated and sorted by `(manager, name)`.

## Alternatives Considered

### Auto-reinstall missing packages from `stamp reconcile`

Rejected. Reconcile is documented as a one-way tracking safety net; making it
destructive (installing on drift) overlaps `stamp restore`, adds a privilege/
consent question to a deliberately deterministic, non-interactive command, and
contradicts the snapshot-is-reality / manifest-is-intent split.

### Auto-remove manifest entries that vanish from the system

Rejected. The manifest is the user's declared intent. A package removed via
`dnf remove` may be a deliberate uninstall; silently dropping it would erase
intent, and is the opposite of `stamp restore`'s job.

### `stamp restore --dry-run --drift-only`

Deferred (follow-up). Restore is not system-aware today and listing only drift
would change its contract (adds a `ListInstalled` dependency and a second
flag). A dedicated `stamp ls --type missing` covers the listing need now.

### Status-view precedent

The warning + dedicated listing split follows established declarative-tooling
UX: `brew bundle check` reports missing dependencies separately from
`bundle install`; `chezmoi status`/`diff` list drift while `apply` mutates;
`git status` is the signal that points to details. Warnings stay one-line and
point at the queryable view rather than embedding long lists.

## Consequences

- Accidental `dnf remove` is now visible in three places.
- No new mutation surface; reconcile remains non-interactive and deterministic.
- `stamp ls --type missing` adds a system query to `stamp ls` (first
  system-aware view). Latency is bounded by concurrent `ListInstalled` calls;
  failed managers are skipped, never fatal.
- `stamp doctor` gains up to N concurrent `ListInstalled` calls (active
  managers only).
- Follow-up tracked: `stamp restore --drift-only` (or drift-aware dry-run) as
  the natural companion.
