---
---

# ADR-014: Signal Handling and Clean Abort on SIGINT

## Status
Accepted

## Date
2026-08-01

## Context

Pressing Ctrl+C at the sudo password prompt during privileged operations (e.g. `stamp reinstall -m dnf drawing`) did not abort cleanly. Only the `update` command installed a SIGINT handler; all other commands relied on Go's default behavior, which exits the process abruptly:

1. **No shared SIGINT handler** — Go's default handler terminates the program without running cleanup, producing confusing interleaved output (`sudo: a password is required` arriving after the shell prompt).
2. **No context cancellation** — children were not killed via `exec.CommandContext`, so `sudo`/`dnf` processes could be orphaned.
3. **No terminal-state restoration** — an interrupted password prompt could leave the terminal echo disabled.

The fix must apply to every command that spawns children (including `restore` and `repo add/remove`, which also invoke sudo), not just the seven commands initially reported.

## Decision

Install a **single shared two-phase SIGINT handler** once at the root command's `PersistentPreRunE`, applied to every runnable command:

- **First SIGINT** — print a newline, restore terminal echo via `stty echo`, and cancel the command's context. `exec.CommandContext` then kills the running child. Children in the same foreground process group also receive the terminal's SIGINT directly.
- **Second SIGINT** — force-abort: send `SIGKILL` to the entire job process group (`syscall.Kill(-getpgrp(), SIGKILL)`) and exit with status 130, guaranteeing no orphaned (possibly privileged) child survives.

The handler is implemented in `internal/cli/signal.go` as `cancelOnInterrupt(ctx, errOut) (context.Context, func())`. The returned cleanup stops the handler and blocks until its goroutine exits; it is wired to the command's `RunE` so it always runs after the command finishes, preventing goroutine leaks across invocations.

Commands without a `RunE` (command groups like `repo`/`man`, help-only output) spawn no children and are skipped.

### Rejected alternative: per-command handlers
Adding the block to each affected command's `RunE` is repetitive and misses commands that spawn sudo indirectly (`restore`, `repo add/remove`).

### Rejected alternative: process-group isolation (`Setpgid`)
Giving children their own process group makes them background relative to the controlling terminal; sudo's password prompt reads `/dev/tty` and would receive `SIGTTIN` and hang. Promoting the child group to the foreground inverts the signal model (the shell stops receiving SIGINT and must emulate a shell). Native tools (`sudo`, `apt`, `dnf`) themselves rely on shared foreground-group SIGINT delivery plus a handler, so the two-phase job-group kill provides the same orphan guarantee at far lower risk.

## Consequences

- **Positive:** Every command aborts cleanly on Ctrl+C; terminal echo is restored; no orphaned privileged processes survive a forced abort.
- **Positive:** The `update` command's inline handler was removed in favor of the shared middleware (single source of truth).
- **Positive:** Testable deterministically via injectable hooks (`notifySigint`, `killJobGroup`, `forceExit`), matching the existing `lookPath`/`isTerminal`/`stdIn` override convention.
- **Residual:** sudo may still emit its own `sudo: a password is required` message before being killed, because it receives the terminal's SIGINT concurrently. The message now appears as the final line before stamp exits, with no shell-prompt interleaving. This cosmetic output is not suppressed.
- **Behavior change:** a single Ctrl+C on a fast read-only command is absorbed (the command completes); a second Ctrl+C force-exits with status 130.
