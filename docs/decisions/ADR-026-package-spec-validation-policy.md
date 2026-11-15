---
---

# ADR-026: Package Specification Validation Policy

## Status
Accepted

Related: [ADR-010](ADR-010-language-toolchain-scope.md) (go pinning),
[ADR-022](ADR-022-brew-tap-qualified-installs.md) (brew tap refs),
[ADR-024](ADR-024-npm-package-spec.md) (npm grammar),
[ADR-025](ADR-025-alias-install-manifest-identity.md) (dnf native identity);
tracked in [issue #264](https://github.com/rossijonas/stamp/issues/264).

## Date
2026-11-15

## Context

Stamp validated nearly every package with one allowlist. Native syntaxes such as
`pkg=version`, `repo/pkg`, `name@version`, and `pkg[extra]` were rejected as
“invalid characters”, while adapters re-validated independently. Three rounds of
bugs (#227, #259, #270) showed the cost of patching one syntax at a time.

The [Package Specification Support](../project/package-spec-audit.html) page is
the audit and reference matrix. This ADR records the policy behind it.

## Decision

One registry owns validation and manifest identity.

- **Entry point:** `ParseSpecForManager(manager, spec)` returns
  `Spec{Raw, Name}`. `Raw` is passed to the native command; `Name` is recorded in
  the manifest. `ValidateSpecForManager` is the validation-only wrapper.
  Adapters call the registry instead of their own validators.
- **Security guard:** reject empty, leading `-`, control characters, shell
  metacharacters, and overlong input first. Per-manager allowlists follow. Args
  are passed to `exec` without a shell. Whitespace is rejected except in dnf
  rich-dependency expressions.
- **Supported forms:** apt `=version`, `/release`, `:arch`; dnf
  NEVRA/provides/file-provides/binaries/globs/rich deps; pacman `repo/pkg` and
  version constraints; zypper `repo:pkg` and capability syntax; snap instances;
  brew tap/versioned formulae; macports `@version`; cargo `@version`; pipx/uv
  `==version`, `@version`, extras; npm per ADR-024.
- **Identity:** lexical normalization per manager (`pkg=ver` → `pkg`,
  `repo/pkg` → `pkg`, `crate@ver` → `crate`, …). Exceptions: brew keeps
  `@version` because brew reports versioned formula names; snap keeps instance
  names; dnf complex specs keep the raw string and are canonicalized by
  ADR-025 after install, or resolved before remove/reinstall.
- **Operation arguments:** `install` passes the typed spec (versions and
  qualifiers honored). `remove`, `reinstall`, and `update -p` pass the tracked
  identity (`Spec.Name`), because native remove/update commands do not reliably
  accept install-only qualifiers. Maintenance identity is resolved as manifest
  name, then installed-name resolver (dnf aliases/NEVRA), then lexical identity.
  Batches apply the same rule per package.
- **Rejected with hints:** apt `pkg+`/`pkg-` modifiers; `@group` and other
  category selectors (`pattern:`, flatpak remotes); flags inside specs
  (`--git`, `--suffix`, `--from`, `--channel`, `--repo`); file paths and URLs;
  empty versions/qualifiers; go `@version` (ADR-010).
- **Categories:** `--group` remains the only category surface. No new flags in
  this change.
- **No automatic manifest rewrite:** legacy entries are cleared with
  `stamp untrack` and reinstalled (ADR-019/025).

## Alternatives Considered

### Keep the generic allowlist and add more special cases
Rejected: that is the failure mode this issue tracks. Each special case was a
different path with its own drift.

### Support everything the native managers accept
Rejected for this change: category selectors need new flags, VCS/URL specs need
their own validation and trust model, and dnf complex specs already need
operation-time resolution. The registry rejects them with actionable messages.

### Keep adapter-level validation and validate again at the CLI
Rejected: duplicated rules drift. One registry is the source of truth.

### Reject every versioned/qualified spec
Rejected: blocks common apt, cargo, pipx, and uv usage that works today in the
native managers.

## Consequences

- **Positive:** accepted syntax is documented once; invalid forms fail with a
  hint, not a character-class error.
- **Positive:** manifest identity is consistent across install, remove,
  reinstall, restore, and reconcile for lexical forms.
- **Positive:** security surface stays narrow: no shell, no paths/URLs, no
  category inference from names.
- **Residual:** dnf complex specs need resolver-backed lookup on
  remove/reinstall; unresolved specs fail with a hint.
- **Residual:** category flags and VCS/URL specs are follow-ups.
- **Residual:** legacy entries are not migrated automatically.
- **Tests:** per-manager parser table, adapter arg assertions, CLI manifest
  identity, injection cases.
