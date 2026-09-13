---
---

# ADR-022: Tap-Qualified Homebrew Formula Installs

## Status
Accepted

Related: [ADR-016](ADR-016-unified-preview-contract.md) (tap-qualified refs
deliberately opt out of the dry-run preview contract);
tracked in [issue #251](https://github.com/rossijonas/stamp/issues/251).

## Date
2026-09-13

## Context

`stamp install nklmilojevic/sofka/sofka -m brew` was rejected before brew ran:
`validPkgNameRegex` disallows `/`, and brew's package validation routed
through it, even though brew's own tap validation already tolerated slashes.

Homebrew 6.0 introduced **tap trust**: non-official taps must be trusted before
Homebrew evaluates their Ruby. Crucially, the documented model is that
**installing a fully qualified name (`brew install owner/tap/formula`) trusts
only that item** and auto-taps the tap — no separate `brew trust` is needed.
Whole-tap trust (`brew trust owner/tap`) is the broader alternative.

Two further findings: `stamp tap` did not record the tap in the manifest (unlike
`stamp repo add`), and its error was double-wrapped.

## Decision

Support tap-qualified **formulae** for brew only, mirroring brew's behavior.

- **Validation:** `ValidateBrewPackageName` accepts a bare name or
  `owner/tap/name` (1-3 slash-separated segments; rejects leading `-`, empty
  segments, spaces). `ValidatePackageForManager` routes `brew` to it; every
  other manager keeps rejecting `/`.
- **Routing:** `owner/tap/formula` auto-routes to brew (a two-segment
  `owner/tap` in `install` returns a usage error pointing at `stamp tap`).
- **Execution:** stamp passes the qualified name straight to
  `brew install`/`reinstall`/`uninstall`/`upgrade`/`info`; brew handles
  per-item trust. stamp does **not** tap or trust the whole tap.
- **Tracking:** the manifest stores the qualified name as `Package.Name`. Since
  `brew leaves --installed-on-request` reports the same qualified string,
  presence/drift matching is exact — no basename normalization.
- **No pre-consent side effects:** for a qualified name, stamp skips cask
  detection (`brew info --cask`, which could load the tap) and the dry-run
  previews — `brew install --dry-run` (which could auto-tap) and
  `brew uninstall --dry-run` (unsupported on some brews, and able to load a
  trust-gated tap formula). The confirmation prompt stands alone.
- **Whole-tap intent stays explicit:** `stamp tap owner/tap` is the deliberate
  whole-tap action and now records a `Repository` in the manifest (parity with
  `stamp repo add`, and `stamp untap` removes it).

## Alternatives Considered

### Track the short name and record the tap
- **Pros:** reconcile-friendly, matches the existing repo+package model.
- **Cons:** restore would re-tap and whole-tap-trust, broadening trust beyond
  what brew's qualified install granted. Rejected.

### Normalize qualified↔basename in presence checks
- **Pros:** works even if brew reported short names.
- **Cons:** unnecessary — brew reports the qualified name — and it introduces
  cross-tap basename-collision ambiguity. Rejected.

### Reorder `AddRepo` to trust before/independently of `brew tap`
- **Cons:** speculative; the qualified-install path removes the need for
  `stamp tap` in the reported case. Deferred.

### Support tap-qualified casks in the same change
- **Cons:** cask detection under tap trust is unverified. Deferred to a
  follow-up; `brew install --cask owner/tap/cask` remains the workaround.

## Consequences

- **Positive:** `owner/tap/formula` works for `install`, `reinstall`, `remove`,
  `info`, `update -p`, and `restore`, with behavior matching brew and no
  whole-tap trust, no implicit tap, and no implicit untap.
- **Positive:** `stamp tap`/`untap` now persist and remove their manifest entry.
- **Residual:** qualified **casks** are not supported yet.
- **Residual:** `stamp tap owner/tap` can still fail for a tap whose formula
  brew refuses to validate (`brew tap` aborts before stamp's `brew trust`);
  that is a brew-side behavior. Users needing a single formula should prefer
  the qualified install.
- **Residual (preview oracle):** tap-qualified refs opt out of ADR-016's
  dry-run no-op oracle — an absent qualified formula prompts, then surfaces
  brew's own `No such keg` error, and an already-installed one is a brew no-op
  after consent. The trade-off buys zero pre-consent tap loads.
- **Tests:** brew validator table, brew preview-skip (install and remove),
  install/remove routing, and reinstall tracking are covered.
