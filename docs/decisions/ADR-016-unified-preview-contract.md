---
---

# ADR-016: Unified Preview Contract for Destructive Commands

## Status
Accepted

## Date
2026-08-03

## Context

`install`, `remove`, and `reinstall` gained confirmation gates (ADR-015) whose preview layer relied on CLI-side heuristics: the gate streamed raw dry-run output, string-matched vendor messages for no-op detection (`Nothing to do.`, `is already installed`, `up to date`), and fell back to `Info` when the preview errored. This was fragile and confusing in practice:

- **dnf5 renders its transaction UI to stderr**; `cmd.Output()` captured stdout only, so real remove/reinstall previews surfaced as empty and the gate fell back to showing `dnf info` (misleading).
- No-op detection matched unstable vendor strings in the CLI.
- An `Info` dump was shown as if it were "what will happen".

Meanwhile `update` (ADR-011) already used the robust pattern: the **adapter** parses its own output into typed data (`UpdateInfo`) and the CLI renders it. This ADR extends that pattern to install/remove/reinstall and hardens `update`'s own gaps.

## Decision

### 1. Typed, adapter-owned previews

Replace the raw `(string, error)` preview return with a structured `Preview` the adapter fully owns:

```go
type Preview struct {
    Output string // verbatim combined stdout+stderr of the native dry-run
    Noop   bool   // adapter asserts no transaction would occur
}

type Previewer interface {
    PreviewInstall(ctx, pkg) (Preview, error)
    PreviewRemove(ctx, pkg) (Preview, error)
    PreviewReinstall(ctx, pkg) (Preview, error)
}
```

- The **adapter** runs its dry-run with combined output and decides `Noop` from its own vendor knowledge (same file as its other parsers) — the single place a vendor change is handled.
- The **CLI** renders `Output` verbatim, never parses it.

### 2. Combined-output dry-runs

A new `WithCombinedOutput(ctx)` executor mode uses `cmd.CombinedOutput()`, which captures stderr and returns output even on a non-zero exit. This fixes the dnf5 stderr/`dnf info` regression: a `dnf --assumeno` dry-run that displays the transaction then aborts non-zero now surfaces the transaction.

### 3. No-op is dry-run-owned

All no-op decisions come from the **dry-run output** — the authoritative "what would happen" oracle. No-op detection does NOT consult `ListInstalled`: managers like dnf (`repoquery --userinstalled`) and brew (`leaves --installed-on-request`) return only a *subset* of installed packages, which caused remove/reinstall previews to falsely report `nothing to do` for installed packages outside that subset. The dry-run signals are per-adapter: an absent package surfaces as `No match for argument` (dnf), `is not installed` / `0 to remove` (apt), `was not found` (pacman), `No such keg` (brew), `Nothing to uninstall` / `No such ref` (flatpak), `Nothing to do.` / `not installed` (zypper). Reinstall of an *installed* package is never a no-op (it is always a real operation).

### 4. CLI gate behavior

- `Noop` preview → print output + `nothing to do`, skip the prompt (fail fast).
- Preview error → **warn-and-prompt** (`⚠ could not render preview`), no `Info` fallback — the prompt is the consent gate, so a preview failure degrades to confirm-without-preview.
- Managers without a `Previewer` (snap, cargo, go, pipx, uv, paru) warn-and-prompt too.
- SIGINT cancellation during refresh/preview aborts before prompting (ADR-014).

### 5. Update hardening

- dnf/yum `Refresh` now runs `makecache` (previously a no-op), making ADR-011's "refresh metadata" step real for dnf.
- `CheckUpdate` (dnf, apt, pacman) surfaces unrecognized output as a `parser may be outdated` error instead of silently reporting "All up to date". apt treats a bare `Listing…` header as "no updates". brew already errors on invalid JSON.

Consequence: because dnf `Refresh` runs `makecache`, `stamp update --check` (and update's check phase) performs a real metadata refresh on dnf systems. Interactively it reuses the pre-auth'd sudo password and downloads metadata; in CI/non-TTY it emits a best-effort refresh warning and proceeds on cached metadata. This is an accepted trade-off of fresh checks for network/time — the `--check` dry-run is no longer free on dnf.

## Consequences

- Install/remove/reinstall now follow ADR-011's typed, adapter-owned pattern; vendor knowledge lives in exactly one place per manager.
- The fragile CLI heuristics from ADR-015 (`ErrNoop` string-matching, Info fallback, non-zero-exit guessing) are removed.
- UX changes: non-Previewer managers no longer show an `Info` block (warn-and-prompt instead); dnf install/reinstall/update now run `makecache` (sudo + network) during refresh, matching apt.
- `-y` still skips refresh/preview/prompt entirely.
- Remaining fragility is bounded and documented: `Noop` detection is adapter-owned output knowledge, updated in the same file as the vendor parser when a manager changes format.
