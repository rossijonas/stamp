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
✅ reconciled — manifest updated
```

### Dry run

Preview what reconcile would track without committing:

```bash
stamp reconcile --dry-run
```

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
