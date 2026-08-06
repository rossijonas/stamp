---
---

# ADR-017: Manifest Origin Segmentation and Backup Rotation

## Status
Accepted

## Date
2026-08-05

## Context

Three gaps motivated issue #177:

1. **No provenance.** `stamp reconcile` silently appends discovered packages to the same manifest that `stamp install` writes. Users cannot tell which entries are intentional (`stamped`) versus auto-tracked (`reconciled`), which blocks `stamp list --type` (#178) and `stamp manifest history`/`diff` (#179).
2. **No retention on backups.** `stamp init` re-init already takes timestamped backups (rename-based), but backups accumulate forever — no pruning, no configurable policy.
3. **No generated config.** Fresh `stamp init` creates the manifest and snapshots but never writes `config.toml`, so users never see the config surface and retention options are undiscoverable.

Existing helpers (`manifest.Backup`, `state.BackupSnapshots`) rename the live file/dir into a `.bak`; they are only called from `init` re-init. `stamp reconcile` never calls `saveManifest` on the no-drift path, so a rename-based backup there would orphan the manifest.

## Decision

### 1. Origin field, absence = `stamped`

Add optional `origin` (`omitempty`) to `manifest.Package` and `manifest.Repository`. Two values: `stamped` (direct user action) and `reconciled` (auto-tracked). Because the field is optional, pre-existing manifests load unchanged — `OriginEffective()` returns `stamped` for an absent field. No migration required.

### 2. Copy-based backup for reconcile, rename retained for init

- `manifest.CopyBackup(path)` copies the manifest (atomic temp + rename pattern) and keeps the original — safe for the reconcile no-drift path that never saves the manifest.
- `manifest.Backup()` (rename) and `state.BackupSnapshots()` (rename) are **retained for `stamp init` re-init only**, where the live file is being destroyed anyway and rename is cheap and atomic.
- `state.RotateSnapshotBackups` prunes snapshot backup dirs (`snapshots.*.bak/`) and `manifest.RotateBackups` prunes manifest backup files (`manifest.toml.<TS>.bak`); both delegate to `backup.Rotate`, which removes entries with `os.RemoveAll` (works for files and directories). Deletion is scoped strictly to stamp-owned `.bak` globs.

### 3. Logrotate-aligned 3-axis retention (plus a count floor)

Retention mirrors logrotate's `rotate`, `minage`, and `maxage` directives, exposed as eight `[backup]` config keys (separate manifest/snapshot policies), each defaulting per the spec table; `0` = unlimited on that axis.

A **min-count floor** (`min_manifest_backups`, `min_snapshot_backups`, default `3`) closes a logrotate-model gap: `maxage` deletes regardless of count, so a backup set whose members are all ancient would otherwise be wiped to zero — the ceiling is self-defeating precisely when the data is oldest and most valuable. The floor guarantees at least `min_*_backups` backups survive, keeping the newest ones. The floor applies to the *total* backup set, not just the eligible subset, and both the max-age ceiling and the count cap respect it via a shared deletion budget of `len(entries) - min_*_backups`.

Precedence (highest to lowest): **min-age floor** (protects recent) > **min-count floor** (keeps at least N, newest) > **max-age ceiling** (deletes ancient, except to meet the floor) > **count cap** (trims surplus, except to meet the floor).

**Floor trade-offs:** when `min_*_backup_age_days > max_*_backup_age_days`, the min-age floor wins on the overlapping window (protective), the policy is flagged as invalid by `stamp doctor`, and the docs warn against it. When `min_*_backups > max_*_backups`, the min-count floor wins (protective). A floor that is "too high" can permanently pin old backups — the protective defaults (`min 7 < max 30` days, `min 3 < max 10` count) avoid this.

## Consequences

- The manifest grows one optional field per entry; file size impact is negligible and gated by `omitempty`.
- Reconcile now performs backup + rotation on every non-dry-run drift, adding I/O proportional to the manifest size.
- `stamp init` gains config generation (non-fatal, never overwrites) and rotation on re-init; fresh-init behavior otherwise unchanged.
- Dry-run paths (`init`, `reconcile`) write nothing — no backups, no rotation, no config.
- Users can rely on logrotate semantics (`rotate`/`minage`/`maxage`) instead of learning a new policy model.
- Follow-ups #178 (`stamp list --type`) and #179 (`manifest history`/`diff`) consume `OriginEffective()` as their data source.
