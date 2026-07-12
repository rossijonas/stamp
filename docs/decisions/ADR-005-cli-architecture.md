# ADR-005: CLI Architecture — Manifest Error Guards and Output Streams

## Status
Accepted

## Date
2026-07-11

## Context
As the stamp CLI grew beyond simple install/remove commands, several architectural decisions needed to be made consistently:
1. How to handle TOML manifest parsing errors without crashing diagnostic commands
2. Where to direct human-readable vs. machine-readable output
3. How to structure flag vs. subcommand boundaries for actions
4. How to detect and respond to TTY vs. non-TTY environments

## Decision

### Manifest Error Guards
When the manifest TOML file is corrupted on disk:
- `AppContext.manifestErr` stores the error and provides a fallback empty manifest
- **Write commands** (install, remove, repo add/remove, reconcile, restore) check `app.manifestErr` and fail early to prevent overwriting a corrupted manifest with incomplete data
- **Read-only commands** (doctor, list, search) proceed with the fallback empty manifest, allowing users to diagnose the corruption
- This prevents data loss where running `reconcile` on a corrupted manifest would silently replace it

### Output Streams
- **Human-readable output** (TTY tables, progress messages, confirmations): Always goes to `cmd.OutOrStdout()` to support piping
- **Errors and diagnostics** (warnings, failure messages): go to `cmd.ErrOrStderr()`
- **JSON output** (`--json` flag): Always goes to `cmd.OutOrStdout()` for machine parsing
- All output uses `cmd.InOrStdin()` / `cmd.OutOrStdout()` / `cmd.ErrOrStderr()` (never raw `os.Stdin`/`os.Stdout`/`os.Stderr`) to support test capture buffers

### Actions as Subcommands (Not Flags)
Actions that change system state or perform operations MUST be subcommands, not flags:
- ✅ `stamp man install` (not `stamp man --install`)
- ❌ `stamp man --prefix /path` → ✅ `stamp man install --prefix /path`
- This ensures consistent CLI design and discoverability
- Boolean flags for enabling/disabling behavior are acceptable (e.g. `--dry-run`, `--yes`, `--json`)

### TTY Detection
Use `isTerminal() bool` helper that checks `os.ModeCharDevice` on the input reader:
- In non-TTY environments (CI, pipelines), prompts are skipped and sensible defaults are used
- For reconcile/restore, non-TTY without `--yes` defaults to auto-accept
- Declared as a package-level variable for test overrides

### Flag Short Forms
Every flag SHOULD have a single-character short form:
- `--verbose`, `-v`
- `--yes`, `-y`
- `--manager`, `-m`

## Alternatives Considered

### Returning Errors from newAppContext (Previous Behavior)
- **Pros:** Simple, fail-fast on any error
- **Cons:** Doctor can't diagnose corrupt manifests; user gets raw parse error instead of formatted diagnostic
- **Rejected:** The doctor's purpose is to help diagnose problems — it needs to run when things are broken

### Sending TTY Output to Stderr (Previous Behavior)
- **Pros:** Follows Unix convention of "stdout for data, stderr for diagnostics"
- **Cons:** `stamp doctor > report.txt` redirects nothing — user expects output on stdout
- **Rejected:** Doctor output IS the program output, not diagnostic noise

## Consequences
- All new commands must follow the subcommand pattern for actions
- All new commands must use `cmd.InOrStdin()` / `cmd.OutOrStdout()` / `cmd.ErrOrStderr()`
- The existing `stamp man --install` flag must be refactored to `stamp man install` subcommand
- New flags should include a short form where possible
- `isTerminal` is testable via the variable override pattern
