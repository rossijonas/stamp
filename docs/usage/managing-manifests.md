---
---

## Manifest Management

Stamp stores your intended package state in `manifest.toml`. Before rewriting it
on re-init or reconcile, stamp keeps a timestamped backup
(`manifest.toml.<YYYYMMDDTHHMMSSZ>.bak`). Because the manifest is a local file —
not tracked by git — `stamp manifest history` and `stamp manifest diff` are the
only way to see what changed and which backup to restore from.

### History

`stamp manifest history` lists the current manifest and every backup, newest
first, with package/repo counts and a short content hash.

```bash
stamp manifest history
```

```text
Available manifest backups:
  * 2026-08-04T12:00:00Z  3fa9c21e4b12  145 packages, 14 repos  (current)
    2026-08-03T18:30:00Z  8d2f00a1c9e3  142 packages, 12 repos
    2026-08-02T09:15:00Z  6b1e44c90aa7  38 packages, 8 repos
    2026-07-28T14:00:00Z  12ba0ef341dd  35 packages, 6 repos  (unchanged)
```

- `*` marks the current manifest
- The hash is the first 12 hex characters of each file's SHA-256 — copy it to
  use with `stamp manifest diff`
- `(unchanged)` marks a backup whose content is identical to the current
  manifest — a good candidate for pruning

```bash
stamp manifest history -j
```

```json
[
  {"timestamp": "2026-08-04T12:00:00Z", "hash": "3fa9c21e4b12", "current": true, "packages": 145, "repos": 14},
  {"timestamp": "2026-08-03T18:30:00Z", "hash": "8d2f00a1c9e3", "current": false, "packages": 142, "repos": 12}
]
```

### Diff

`stamp manifest diff` shows what changed between the current manifest and a
backup. It defaults to the most recent backup. The argument may be a timestamp
or a content-hash prefix from `history`.

```bash
stamp manifest diff
stamp manifest diff 2026-08-02T09:15:00Z
stamp manifest diff a1b2c3d4e5f6
```

```text
Comparing: current vs 2026-08-03T18:30:00Z

+ htop (dnf)
+ lazygit (brew)
- NetworkManager (dnf)
- gnome-shell (dnf)

  2 added, 2 removed
```

`+` means present in the current manifest but not the backup; `-` means present
in the backup but removed since. Both packages and repositories are compared,
using `name + manager` as the identity key.

Filters compose with `--manager, -m` and `--origin`:

```bash
stamp manifest diff --origin stamped
stamp manifest diff -m brew
stamp manifest diff -m brew --origin reconciled -j
```

```json
{
  "baseline": "2026-08-03T18:30:00Z",
  "added": [
    {"name": "htop", "manager": "dnf", "origin": "stamped", "kind": "package"}
  ],
  "removed": [
    {"name": "old-repo", "manager": "dnf", "origin": "stamped", "kind": "repo"}
  ]
}
```

### Restoring from a backup

`history` and `diff` help you pick the right backup. To roll the manifest back,
copy a backup over the live manifest (or use `stamp restore` after editing).
Backup filenames use the compact timestamp form, e.g. for the backup shown as
`2026-08-02T09:15:00Z` in `history`:

```bash
cp ~/.config/stamp/manifest.toml.20260802T091500Z.bak ~/.config/stamp/manifest.toml
```

The next `stamp restore` rebuilds your environment from that state.

### Understanding origins

Every entry's `origin` records how stamp learned about it — `stamped`
(installed via `stamp install`) or `reconciled` (discovered by
`stamp reconcile`). See [Listing Packages](listing-packages.html) for details.
