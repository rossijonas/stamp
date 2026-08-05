---
---

# ADR-018: sysexits Exit Codes

## Status
Accepted

Supersedes the exit-code note in [ADR-002](ADR-002-cli-io-and-exit-codes.md) (the
"exit status 2 (usage error)" clause). ADR-002's remaining I/O-separation and
flag-constraint decisions stay in force.

## Date
2026-08-05

## Context

[ADR-002](ADR-002-cli-io-and-exit-codes.md) defined strict UNIX/GNU CLI
conventions and stated that the Tier-3 ambiguous-install fallback fails with
"exit status 2 (usage error)". In practice stamp exited **1 for every
failure** — the `Exit*` sysexits constants were defined in `root.go` but never
wired. Consequences:

- Scripts cannot distinguish "bad argument" (should be 64) from "no package
  manager available" (should be 69) from "corrupt manifest" (should be 65).
- The documented "status 2" never matched implementation, and `2` is
  ambiguous in the POSIX model (1–125 are application codes; 126/127 are
  reserved for shell "not found"/"not executable").

## Decision

Adopt BSD `sysexits.h` codes (shipped by glibc on Linux) for error categories,
with `1` as the catchall for unclassified failures.

### Mechanism

- **Category sentinels** (`ErrUsage`, `ErrData`, `ErrNoInput`, `ErrUnavailable`,
  `ErrCanTCreate`, `ErrConfig`) classify errors by kind, decoupled from exit
  numbers. Matches the existing `manager.ErrNotSupported` sentinel pattern.
- **`categorizedError`** attaches a category without altering the message, so
  user-facing text and existing string assertions are unchanged; it satisfies
  `errors.Is` by category identity and unwraps to the original error.
- **`exitCodeFor`** maps an error to a sysexits code via `errors.Is`
  (traverses wrap chains and `errors.Join`). Default is `1`.
- **Single exit boundary:** only `Execute()` calls `os.Exit(exitCodeFor(err))`.
  Commands never exit directly. `os.Exit` skips deferred functions, so all
  cleanup (temp files, signal handlers) stays inside command `RunE`.
- **Flag-parse errors** (unknown flags, invalid values) are classified as
  usage via cobra `SetFlagErrorFunc`, so `stamp list --bogus` exits 64.

### Mapping

| Category | Code | Constant |
|----------|------|----------|
| `ErrUsage` (bad flag/argument) | 64 | EX_USAGE |
| `ErrData` (corrupt input) | 65 | EX_DATAERR |
| `ErrNoInput` (referenced input absent) | 66 | EX_NOINPUT |
| `ErrUnavailable` (manager/resource absent) | 69 | EX_UNAVAILABLE |
| `ErrCanTCreate` (cannot create output) | 73 | EX_CANTCREAT |
| `ErrConfig` (unconfigured/misconfigured) | 78 | EX_CONFIG |
| unclassified | 1 | (POSIX catchall) |

Backup/rotation failures on `reconcile` and `init` are non-fatal (warning to
stderr, exit 0).

## Alternatives Considered

### Keep ADR-002's "exit status 2" for usage errors
- **Pros:** Matches the original ADR.
- **Cons:** Never implemented as 2 (was 1); `2` is a nonstandard usage code
  that collides with shell conventions; the repo already carried sysexits
  constants. Rejected.

### Call `os.Exit(code)` at each error site
- **Pros:** Direct.
- **Cons:** Skips the error-propagation model, untestable without subprocess
  gymnastics, and scatters exit concerns across commands. Rejected.

### Classify by matching error strings at the boundary
- **Pros:** No changes to error production sites.
- **Cons:** Fragile, locale-sensitive, and reintroduces the string-matching
  the `errors` package exists to avoid. Rejected.

## Consequences

- Scripts can distinguish failure modes by exit code (documented in
  `docs/project/spec.md` "Exit Codes").
- Failure exit codes changed from `1` to the mapped sysexits code for the
  categorized errors across **all** commands; success (`0`) and `exit != 0`
  checks are unaffected. Scripts that asserted `== 1` on those categories must
  update.
- Error message text is byte-identical (the category is attached, not spliced
  into the message).
- The sysexits constants live in `internal/cli/exit.go`; `exitCodeFor` is the
  single, unit-tested mapping.
