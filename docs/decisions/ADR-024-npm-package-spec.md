---
---

# ADR-024: npm Scoped and Versioned Package Specs

## Status
Accepted

Related: [ADR-022](ADR-022-brew-tap-qualified-installs.md) (per-manager
validator precedent); tracked in
[issue #259](https://github.com/rossijonas/stamp/issues/259).

## Date
2026-11-15

## Context

Stamp validated nearly every package name with one strict allowlist
(`^[a-zA-Z0-9_][a-zA-Z0-9_\-\.\+]*$`). npm's native grammar was rejected:

```
> stamp info -m npm @fission-ai/openspec@latest
Error: invalid package name: invalid package name "@fission-ai/openspec@latest": contains invalid characters
```

Three defects compounded:

- `ValidatePackageForManager` special-cased only `go` and `brew`, so npm fell
  through to the strict validator.
- `stamp info` had its own ad-hoc validator (brew-only), so `-m npm` was inert
  and the no-`--manager` query was gated out before any adapter ran.
- The npm adapter re-validated internally with the same generic function, so a
  CLI-only fix would not have worked.

npm accepts `[<@scope>/]<pkg>` with an optional `@<tag>`, `@<version>`, or
`@<version range>`. `npm ls -g --depth=0` already reports scoped names without
versions, so `ListInstalled` and reconcile were never the problem.

## Decision

Support npm's package-spec grammar through a per-manager validator.

- **Validation:** `ValidateNpmPackageName` accepts `name`, `@scope/name`, and
  either form suffixed with `@tag`, `@version`, or a range (no spaces). It rejects a leading
  `-` (flag injection), empty segments, `..`, whitespace, and control
  characters. A bare `scope/name` is rejected with an actionable hint
  (`did you mean "@scope/name"?`). Uppercase legacy names stay valid.
  `ValidatePackageForManager` routes `npm` to it; every other manager's
  acceptance is unchanged.
- **Adapter:** every `Npm` method validates through the npm validator, including
  the batch paths.
- **`stamp info` without `--manager`:** no CLI pre-validation. With no resolved
  manager there is nothing to validate against, and the adapters already report
  a rejected name as not-found. With `--manager`, validation dispatches through
  `ValidatePackageForManager`.
- **Manifest identity:** the manifest records the package name, not the version
  spec. `TrackedName("npm", spec)` drops the `@version` suffix; all other
  managers keep the spec verbatim (brew's `python@3.14` is a real package name).
  Remove, reinstall, note updates, and manifest lookups use the same identity.
- **Errors:** the validator is the single source of the
  `invalid package name` prefix; callers no longer wrap it.

## Alternatives Considered

### Keep the generic validator and require `-m npm` help text
Rejected: the native syntax is documented by npm and used by real packages;
rejecting it makes stamp unable to reach them.

### Reject version specs, telling users to pin with npm directly
Rejected: npm accepts them, and pass-through costs nothing. The manifest
normalization keeps matching intact.

### Store the full spec (`pkg@1.2.3`) in the manifest
Rejected: `ListInstalled` reports unversioned names, so a versioned manifest
entry would never match — the package would look missing and could not be
removed by name.

### Accept a bare `scope/name` by inserting the `@` implicitly
Rejected: silently rewriting user input hides a typo. A hint is actionable and
keeps npm's grammar as the contract.

### Validate without `--manager` by accepting if any manager's validator accepts
Rejected: it runs every validator to reach the same decision the adapters make,
and still needs a rule for names no validator accepts. Skipping pre-validation
is simpler.

## Consequences

- **Positive:** `stamp info`, `install`, `remove`, `reinstall`, and
  `update -p` accept scoped npm packages; `name@version` is passed through and
  tracked by name.
- **Positive:** reconcile-added scoped names can be removed and reinstalled,
  because validation and manifest identity now agree with `ListInstalled`.
- **Residual:** `stamp install @scope/name` without `-m npm` still fails. npm is
  not in the default resolution precedence, so the resolver picks another
  manager. Routing scoped refs to npm is part of the broader package-spec work
  (issue #264).
- **Residual:** other managers keep their current acceptance; their native
  syntaxes remain tracked by issue #264.
- **Tests:** npm validator and identity tables, npm adapter args (including
  batch), CLI info/install/remove/reinstall/update paths, and manifest
  assertions.
