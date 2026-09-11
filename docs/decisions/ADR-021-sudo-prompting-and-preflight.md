---
---

# ADR-021: Native Sudo Prompting and Probe-Gated Preflight

## Status
Accepted

Related: [ADR-014](ADR-014-signal-handling-and-clean-abort.md) (shared SIGINT
handler wraps the `sudo -v` preflight too).

## Date
2026-09-10

## Context

Two problems with privileged operations:

1. **NOPASSWD is ignored by `stamp update`.** `update` unconditionally printed
   its own `▪ sudo password:` and cached the password for the parallel run
   phase (`promptSudoPassword` + `manager.SetSudoPassword`). On hosts configured
   with `sudo` `NOPASSWD`, or with a valid credential cache, it prompted anyway —
   while every other command relied on sudo's own prompt and stayed silent.

2. **Parallel `restore` had no pre-auth.** `restorePackages` runs one goroutine
   per manager. On a password-protected host with an empty cache, concurrent
   `sudo` children could prompt at the same time, garbling the terminal.

The stamp-managed password path was also the riskiest part of the code: the
password lived in a package global and was written to each `sudo` child's
stdin via `-S`, where it could starve the child's real stdin or be captured by
an I/O-logging plugin.

## Decision

Stamp never handles the password. `sudo` prompts natively for every command.
Add a probe-gated preflight for the commands with a parallel privileged phase.

### Probe and preflight

- `manager.SudoReady(ctx)` runs `sudo -Nnv` — sudo's documented credential check
  (`-N` no-update, `-n` non-interactive, `-v` validate). It neither executes a
  command nor updates the cache, so it has no side effects.
- `manager.EnsureSudo(ctx)` runs `sudo -v` once with streamed I/O so sudo can
  prompt natively (password, OTP, or askpass).
- `cli.sudoPreflight(cmd, adapters, errOut) bool` decides per command:
  1. no selected manager needs sudo → parallel OK;
  2. `SudoReady` true (NOPASSWD or valid cache) → no-op, no side effects, parallel OK;
  3. not ready, stdin not a terminal → no-op; `sudoCmd` adds `-n` and the real
     command fails fast;
  4. not ready on a terminal → `EnsureSudo` once, then re-probe. Valid cache
     afterward → parallel OK; still not ready (non-caching sudoers, e.g.
     `timestamp_timeout=0`) → return false so the caller **serializes** and
     prompts can never race.

`update` and `restore` call `sudoPreflight` immediately before each privileged
phase (refresh, then again right before the parallel run) to keep the auth
window minimal. `restore` derives the adapter set from the repositories and
packages actually being restored, so a brew-only restore never probes sudo.

### Removed

`manager.SetSudoPassword` / `ClearSudoPassword`, the `sudoPassword` global, the
`-S` branch in `sudoCmd`, and the stdin-piping block in `defaultExecutor`. The
`cli.promptSudoPassword` function is deleted.

## Alternatives Considered

### Keep the stamp prompt, add the probe (Option B)
- **Pros:** Uniform stamp-style prompt.
- **Cons:** Keeps the password global and `-S` stdin path (secret-on-stdin leak
  risk, no askpass/OTP support), expands password capture to more commands, and
  documents a two-path behavior. Rejected.

### Probe, then `-S`-cached fallback only when a password is needed (Option C)
- **Pros:** Robust to non-caching sudoers and long serial runs.
- **Cons:** Retains all `-S` risks above; adds divergent runtime behavior and
  permanent dual-path complexity. Rejected.

### Always run `sudo -v` (no probe)
- **Pros:** Simpler.
- **Cons:** Adds a sudo invocation and, on password hosts, a timestamp update
  even when credentials were already valid; not side-effect-free. Rejected.

## Consequences

- **Positive:** NOPASSWD and cached-credential hosts are always silent; stamp
  holds no password; askpass/OTP/multi-factor sudo works; parallel phases never
  race prompts.
- **Positive:** `update` and every other command now share one sudo model.
- **Behavior change:** `update` no longer prints `▪ sudo password:`; sudo's own
  prompt appears instead.
- **Residual:** On non-caching sudoers (`timestamp_timeout=0`), `update`/
  `restore` fall back to serial execution and every privileged command prompts —
  correct but not parallel. The `sudo -v` preflight runs twice per command
  (before the refresh and before the run phase); the second is a cheap no-op
  when the cache is valid, but on a non-caching policy it prompts again. On fine-grained sudoers (e.g. `NOPASSWD: /usr/bin/dnf`
  only), the generic probe may report not-ready and produce one extra prompt;
  the operation still runs.
- **Residual:** On very old `sudo` without `-N`, the probe fails and NOPASSWD
  hosts degrade to silent-but-serial.
- **Tests:** `manager.SudoReady`/`EnsureSudo` and `cli.sudoPreflight` are covered
  via injectable seams (`sudoProbe`, `sudoValidate`, `sudoReady`, `sudoEnsure`),
  matching the existing `stdIn`/`lookPath`/`isTerminal` override convention.
