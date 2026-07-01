# ADR-001: Use Local Snapshot Diffing for Intention Tracking

## Status
Accepted

## Date
2026-06-30

## Context
The core value proposition of `stamp` is detecting when a user manually installs a package outside of the `stamp` CLI, so it can be added to the manifest. 

Native package managers offer flags like `dnf repoquery --userinstalled` or `brew leaves`. However, relying solely on these flags is historically brittle. OS version upgrades or dist-upgrades often reset or pollute these flags, causing native package managers to report hundreds of system libraries as "user installed". If `stamp` surfaces 150 false positives during a `reconcile`, the user experience is destroyed.

## Decision
We will implement **Local Snapshot Diffing**. 

`stamp` will maintain a lightweight JSON snapshot of the system's package state at `~/.local/share/stamp/snapshots/`. When `stamp reconcile` runs, it asks the native package managers for their *current* list, and compares it against the *last snapshot*. Only the exact delta (packages added since the last run) is surfaced to the user as potential intentional installs.

## Alternatives Considered

### Rely Solely on Native Flags (`--userinstalled`)
- **Pros:** Zero local state to manage. Always relies on the source of truth.
- **Cons:** Highly vulnerable to distro upgrades destroying the intent history. Massive false positive risk.
- **Rejected:** Unacceptable user experience degradation on Linux.

### Shell Hook Interception
- **Pros:** Intercepts the exact command. 100% accurate intent.
- **Cons:** Highly invasive, difficult to support across shells, fails if GUI is used.
- **Rejected:** Overcomplicates the MVP.

## Consequences
- **Storage:** We must store a JSON file containing an array of package names. Negligible in size.
- **Performance:** `reconcile` must fetch the full list of packages concurrently.
- **Initial Baseline:** `stamp init` must take a baseline snapshot of the entire system so the first `reconcile` doesn't think everything is new.
