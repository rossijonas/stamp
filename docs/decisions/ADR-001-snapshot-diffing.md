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

---

## Appendix A: Reconcile Behavior Update (2026-07-14)

### Context
After real-world testing, the original interactive prompt design was evaluated against user experience and reliability criteria.

### Decision Updates
1. **Reconcile no longer prompts for user confirmation.** `stamp reconcile` auto-tracks all detected drift without user interaction. The decision is made at snapshot time: if a package exists now that didn't before, it is intentional and should be tracked.
2. **`--dry-run` / `-d` flag added for preview mode.** Users who want to inspect drift before committing use `stamp reconcile --dry-run`. This exits without saving manifest or snapshots.
3. **Pre-existing packages are never detected by reconcile.** Packages installed before `stamp init` are captured in the baseline snapshot. They are invisible to future diffs.

### Rationale
- **Noise pollution from DNF:** `dnf repoquery --userinstalled` flags hundreds of base OS packages (kernel modules, system libraries, etc.) as "user installed". A prompt listing 150 packages is unusable. Snapshot diffing (baseline captures them) followed by auto-tracking (only new packages surface) is the correct solution.
- **Deterministic behavior for scripting:** Non-interactive environments (CI, bootstrapping) must not hang on prompts. Making reconcile fully deterministic eliminates the `-y` requirement for scripts while keeping it for backward compatibility.
- **User intent is captured by action, not by prompt response:** If a user went through the effort of installing a package, the intent is demonstrated by the install itself. The prompt added no value.

### Tracking Pre-Existing Packages
Users who installed packages before `stamp` was initialized can track them explicitly using `stamp reinstall <pkg>`:
1. `stamp reinstall htop` resolves the manager via the resolution engine.
2. Runs the native reinstall command (e.g. `dnf reinstall htop`).
3. Adds the package to the manifest.
4. Saves the snapshot so future reconcile does not re-detect it.

This is the **Explicit Tracking via Reinstall** pattern: instead of reconcile deciding what is intentional, the user consciously declares intent by running `stamp reinstall`.
