---
---

# ADR-025: Native Package Identity for Alias Installs

## Status
Accepted

Related: [ADR-019](ADR-019-missing-package-drift-warning.md) (manifest is
intent; explicit `stamp untrack` is the escape hatch), [ADR-024](ADR-024-npm-package-spec.md)
(npm version suffixes are not identity); tracked in
[issue #270](https://github.com/rossijonas/stamp/issues/270).

## Date
2026-11-15

## Context

Stamp recorded the requested spec verbatim in the manifest. Native managers do
not always install a package with that name. On Fedora:

```
> sudo dnf install -y yarn
...
Install nodejs-corepack-0:0.36.0-1.fc44.noarch User updates
```

dnf resolved `yarn` to `nodejs-corepack` (via provides / file-provides), exited
0, and stamp wrote `yarn (dnf)`. `remove`, `reinstall`, and `restore` then target
a name the native manager cannot uninstall or reinstall, and the entry only
surfaces as a false "missing" or a no-op removal.

Reconcile could not produce this: `DNF.ListInstalled` reports RPM names.

Part 1 (`stamp untrack`, issue #270) added a supported way to clear such
entries. This ADR covers the source fix: record what the manager installed.

## Decision

Resolve native identity **after** a successful install or reinstall, through an
optional adapter capability.

- **Interface:** `InstalledNameResolver` with
  `ResolveInstalledName(ctx, spec) (string, error)`. Adapters that do not
  implement it keep current behavior; `Adapter` is unchanged.
- **dnf implementation:** read-only queries in dnf's own resolution order:
  1. `repoquery --installed <spec>` — exact name;
  2. `repoquery --installed --whatprovides <spec>` — provides;
  3. `repoquery --installed --whatprovides '*/<spec>'` — file-provides basename.
  Distinct matches are deduplicated. Exactly one → that package name. Zero or
  several → `ErrInstalledNameUnresolved`. `yum` uses the standalone
  `repoquery` binary, mirroring `CheckInstalled`.
- **CLI:** after `Install`/`Reinstall` succeeds, single-package paths call the
  resolver. A different name is recorded and a note is printed
  (`note: yarn is provided by nodejs-corepack; tracking nodejs-corepack`).
  Resolution failure prints a warning and records the requested spec: the
  system already changed, so the command still succeeds and still records
  intent.
- **Scope:** single-package `install` and `reinstall`. Batches keep the
  requested names, and `--group` installs keep the group ID (groups are not
  packages, so they are never resolved). Canonicalizing a batch costs one
  resolution per package and is a follow-up.
- **Existing entries are not rewritten.** Users clear legacy alias entries with
  `stamp untrack` and reinstall. Manifest intent is not mutated automatically
  (ADR-019).

## Alternatives Considered

### Reject alias installs
Rejected: dnf accepts these specs and users rely on them; refusing removes a
working capability to avoid a bookkeeping bug.

### Pre-resolve the spec before installing
Rejected: the installed name cannot be known until the transaction runs, and
dnf may resolve through provides that only become visible after the package is
present.

### Snapshot `rpm -qa` before and after and diff
Rejected: heavier, needs pre-install state, and the diff includes dependency
packages; separating the requested package requires history parsing.

### Parse `dnf history info` reason column
Rejected: locale-fragile table parsing.

### Change `Adapter.Install` to return a name
Rejected: breaking change across every adapter for a capability only dnf needs
today. The optional interface matches `InstalledChecker`/`BatchInstaller`.

## Consequences

- **Positive:** new dnf alias installs record `nodejs-corepack`, so `remove`,
  `reinstall`, `restore`, doctor, and reconcile agree with the native manager.
- **Positive:** no change for npm, brew, apt, or any adapter without the
  resolver.
- **Residual:** legacy entries still need `stamp untrack` + reinstall.
- **Residual:** batch installs are resolved after success (ADR-026), so aliases
  canonicalize there too.
- **Residual:** multiple providers or missing metadata leave the spec in place
  with a warning; stamp never guesses.
- **Cost:** one to three read-only `repoquery` calls per single install.
- **Tests:** resolver table (exact/provides/file-provides/unresolved/ambiguous/
  exec failure/yum binary) and CLI manifest assertions for install and
  reinstall.
