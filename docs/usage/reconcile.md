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

## Reconcile is a fallback

Reconcile exists to catch packages installed *outside* Stamp. The primary
workflow is to record intent from day one:

```bash
stamp install <pkg> -m <mgr>   # prefer this — intent is tracked immediately
stamp reconcile                # fallback for packages installed via the native manager
```

Every `stamp reconcile` run prints this reminder:

```text
note: reconcile is a fallback for packages installed outside stamp.
      prefer 'stamp install <pkg> -m <mgr>' so intent is tracked from day one.
```

For packages that already exist on the system, `stamp reinstall <pkg> -m <mgr>`
tracks intent directly without a full reconcile.

## Reliability notes

Not every package manager can reliably report "packages the user installed".
Each manager's `ListInstalled` is annotated, and `stamp reconcile` prints a note
when a manager's output is over-inclusive. These are always shown so unexpected
drift is not mistaken for real package changes:

| Reliability | Meaning | Managers |
|---|---|---|
| Reliable | Lists only user-installed packages | brew, flatpak, go, pipx, uv, npm, cargo, macports, **snap** (filtered) |
| Over-inclusive | Lists all installed packages (base OS + deps included). Output is consistent run-to-run, so baseline diffing stays safe, but reconcile may list system packages | apt, **dnf**, zypper, pacman, paru |

Example output:

```text
note: dnf lists all installed packages; reconcile may detect system packages
note: snap lists all installed packages; reconcile may detect system packages
```

### dnf: dnf4 vs dnf5

- **dnf4/legacy** (RHEL/CentOS/Rocky): uses `dnf history userinstalled` — a
  transaction-based query that is precise.
- **dnf5** (Fedora 41+): `history userinstalled` does not exist; stamp falls
  back to `dnf repoquery --userinstalled`, the only documented dnf5 method.
  This returns base OS packages that Anaconda marks as user-installed (kernel,
  glibc, bash, etc.) — over-inclusive but **consistent run-to-run**, so
  baseline diffing stays safe (see [#176](https://github.com/rossijonas/stamp/issues/176)).
- The command binary is always respected (`m.cmd`) for `dnf`/`dnf5`. `yum`
  (RHEL 7) is special-cased: it invokes the standalone `repoquery` binary,
  since yum itself has neither a `history userinstalled` nor a `repoquery`
  subcommand.

### Snap

System snaps (`core*`, `gnome-*` runtimes, `snapd`, `gtk-common-themes`,
`snap-store`, `firmware-updater`, `bare`) are now filtered out of the listing.
Only user-installed apps surface in reconcile drift.

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
