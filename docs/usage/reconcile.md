---
---

## Tracking External Installations

If you install a package directly with your native package manager (bypassing Stamp), `stamp reconcile` detects it and adds it to your manifest.

```bash
sudo dnf install ripgrep        # bypasses stamp
stamp reconcile                 # detects ripgrep and tracks it
```

```text
▪ Drift detected:
    Added: ripgrep (dnf)
    Added: codehaus-casa (copr)
▪ Tracking 1 new package and 1 new repository...
✓ reconciled — manifest updated
```

### Dry run

Preview what reconcile would track without committing:

```bash
stamp reconcile --dry-run
```

### Missing manifest packages

Reconcile also warns when a tracked package is **no longer installed** —
removed directly via your native package manager (e.g. `sudo dnf remove htop`):

```text
warning: 1 manifest package(s) not installed: htop (dnf)
         run 'stamp ls --type missing' for the full list, or 'stamp restore' to reinstall
```

The warning fires on the no-drift and drift paths and is warning-only: nothing
is reinstalled and the manifest is untouched. The removal is still recorded in
the new snapshot (snapshots reflect reality; the manifest holds intent). Use
`stamp ls --type missing` to list everything missing and `stamp restore` to
reinstall.

```text
▪ Drift detected (dry run — no changes saved):
    Added: ripgrep (dnf)
    Added: codeaus-casa (copr)
  Run stamp reconcile to track these.
```

### No drift

```bash
stamp reconcile
```

```text
▪ No drift detected
```

### Scoped to a manager

```bash
stamp reconcile -m dnf
```

Limits drift detection to a single manager.

### How it works

1. Takes a new snapshot of all packages across every manager
2. Compares against the last saved snapshot
3. Any new packages or repositories are detected as drift
4. Drift is auto-tracked into the manifest (or printed with `--dry-run`)

Reconcile is fully deterministic — no prompts, no decisions. It's the safety net for when you forget to use Stamp.

## Automated Reconcile

Run reconcile automatically on a schedule so you never miss drift:

```bash
stamp auto-reconcile on                    # enable daily timer
stamp auto-reconcile on --period hourly    # check every hour
stamp auto-reconcile on -p weekly          # check once a week
stamp auto-reconcile off                   # disable timer
```

### Platform support

| OS | Timer system | Activation |
|----|-------------|------------|
| Linux | systemd user timer | `systemctl --user enable --now stamp-reconcile.timer` |
| macOS | launchd agent | `launchctl load ~/Library/LaunchAgents/dev.gostamp.stamp-reconcile.plist` |

The timer runs `stamp reconcile` at the configured interval and logs output to `/tmp/stamp-reconcile.log`. If your system doesn't support automatic timers (e.g., containers without systemd), you can install the timer files manually from the `contrib/` directory.
