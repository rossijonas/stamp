---
---

# Package Specification Support

Use this page to check how stamp handles the package spec you want to type. For
each manager it lists the native install forms, what stamp accepts today, why a
form is rejected, and whether the gap is tracked.

Start with your manager's row before filing a syntax bug. The gap tables tell you
what is planned, what is intentionally rejected, and which workaround to use
meanwhile. The identity section explains what stamp records in the manifest, so
remove, reinstall, and restore keep working.

This is the audit behind [issue #264](https://github.com/rossijonas/stamp/issues/264).
Implementation status lives in the gap tables; the matrix is the source of truth
for current acceptance. The validation policy is recorded in
[ADR-026](../decisions/ADR-026-package-spec-validation-policy.html). The list of
managers covered here is in
[Supported Managers](../getting-started/supported-managers.html).

## Method

- Current acceptance read from `internal/manager/manager.go` and each adapter's
  validation call sites.
- Native grammar taken from upstream man pages or project docs; linked when published, man page named otherwise.
- `Native install forms` includes argument forms and install flags users type; every flag is marked supported or unsupported in `Rejected because`.
- `Verified` records how the claim was checked: **local** (command/man page on the
  audit machine), **upstream** (published documentation only), **issue** (reproduced
  in a linked issue), or **unverified** when no source was found.
- Claims that could not be checked are marked **unverified** rather than guessed.

## Current validators

| Validator | Pattern / rule | Used by |
|---|---|---|
| `ValidatePackageName` | `^[a-zA-Z0-9_][a-zA-Z0-9_.+-]*$`; leading `-` rejected | apt, cargo, dnf, flatpak, macports, pacman, paru, pipx, snap, uv, zypper |
| `ValidateModulePath` | `^[a-zA-Z0-9][a-zA-Z0-9_.+/-]*$`; requires `/` | go |
| `ValidateBrewPackageName` | 1-3 `/`-separated segments; no `@` | brew |
| `ValidateNpmPackageName` | optional `@scope/`, optional `@tag\|version\|range` | npm |

`ValidatePackageForManager` routes `go`, `brew`, and `npm`; everything else falls to the
generic validator. The CLI pre-validates in install/remove/reinstall/update/info, and each
adapter re-validates before running its native command. Args are passed to `exec` without a
shell, so the real injection risk is a leading `-` plus control characters, not `@ / = :`.

## Matrix

| Manager | Native install forms | Accepted today | Rejected because | Class | Verified | Related |
|---|---|---|---|---|---|---|
| apt | `pkg`, `pkg=ver`, `pkg/release`, `pkg:arch`, regex/glob | `pkg`, `pkg=ver`, `pkg/release`, `pkg:arch` | trailing `+`/`-` modifiers are apt-specific; regex/wildcards rejected | spec, category | upstream ([apt(8)](https://manpages.debian.org/bookworm/apt/apt.8.en.html), [Multiarch](https://wiki.debian.org/Multiarch/HOWTO), [APT arch-qualified docs](https://github.com/debian/apt/blob/main/doc/external-installation-planner-protocol.md)) | #264 |
| dnf (yum alias) | name, NEVRA, provides, file provides, binaries, globs, rich deps, group via `@group` or `group install` | name, NEVRA, provides, file provides, binaries, globs, rich deps; groups via `--group` | `@group` as a package name, URLs/`.rpm` paths, non-path `/`; groups need `--group` | spec, category, identity | local (`man dnf5-specs`, `man dnf5-group`; `dnf5 install --assumeno @core`) | #202, #264, #270 |
| pacman | `pkg`, `repo/pkg`, `pkg>=ver`, groups, provides; `-U` file/URL | `pkg`, `repo/pkg`, `pkg>=ver` | groups not routed; `-U` file/URL paths rejected | spec, category, identity | upstream ([pacman(8)](https://man.archlinux.org/man/pacman.8.en)) | #264 |
| paru | target name, `--repo`/`--aur` flags, groups (via pacman, unverified) | target name | repo-qualified targets, selector flags, paths rejected | spec, category | upstream ([paru man](https://github.com/Morganamilo/paru/blob/master/man/paru.8)) | #262, #264 |
| yay | pacman grammar (`pkg`, `repo/pkg`, version constraints), AUR names | pacman grammar | paths/URLs rejected | spec, category | local (yay is a pacman wrapper; [yay(8)](https://jguer.github.io/yay/man.html)) | #262 |
| zypper | name, `repo:pkg`, capability `NAME[.ARCH][op EDITION]`, `KIND:NAME` (`patch:`, `pattern:`), RPM path/URI, `-name` removal modifier | name, `repo:pkg`, capability and arch forms | `patch:`/`pattern:` selectors, paths, `-`/`!` modifiers rejected | spec, category | upstream ([zypper(8)](https://manpages.opensuse.org/Tumbleweed/zypper/zypper.8.en.html)) | #264 |
| snap | name(s), instance `name_instance`, channel/revision flags, local snap file | name, `name_instance` | `=`/paths rejected; channel and revision flags unsupported | spec, category | local (`man snap`, `snap install --help`) | #264 |
| flatpak | `[REMOTE] REF...`, REF `(app\|runtime)/ID/ARCH/BRANCH`, `--from/--bundle/--image LOCATION` | app ID (dots allowed); explicit multi-arg remote form rejected with guidance | `/`, spaces rejected; remote/ref split and bundle/ref flags unsupported | category, spec | local (`man flatpak-install`) | #227, #264 |
| brew | formula, `owner/tap/formula`, versioned `python@3.14` ([Versions](https://github.com/Homebrew/brew/blob/main/docs/Versions.md)), cask token | formula, tap-qualified, versioned formula names; casks by name | paths/URLs are not documented formula specs | spec | local (`man brew`, ADR-022) | #264 |
| macports | `port install name@version_revision+variants`, separate `+variant`/`-variant` args | `name`, `name@version` | variant tokens, spaces rejected | spec | upstream ([port(1)](https://github.com/macports/macports-base/blob/master/doc/port.1.txt), [Variants](https://guide.macports.org/chunked/using.variants.html)) | #264 |
| go | module path, `module@version`, `/cmd/tool` | module path; stamp always appends `@latest` | `@` rejected by `ValidateModulePath` (ADR-010) | spec, identity | local (`go help install`) | ADR-010, #264 |
| cargo | `crate[@version]`, `--version`, `--git/--path/--branch/--tag/--rev` | `crate`, `crate@version` | version and source flags unsupported | spec, category | upstream ([cargo-install](https://doc.rust-lang.org/cargo/commands/cargo-install.html)) | #264 |
| npm | `[scope/]name[@tag\|version\|range]` | full grammar from ADR-024 | alias/git/local specs rejected | spec, identity | local | #259, ADR-024 |
| pipx | `pkg`, `pkg==ver`, `pkg[extra]`, VCS/local spec, `--suffix` | `pkg`, `pkg==ver`, `pkg[extra]` | VCS/URL specs, flags unsupported | spec, category | local (`pipx install --help`) | #264 |
| uv | `uv tool install pkg`, `pkg==ver`, `pkg@version`, `pkg@latest`, `pkg[extra]`, VCS/local spec, `--from` | `pkg`, `pkg==ver`, `pkg@version`, `pkg@latest`, `pkg[extra]` | VCS/URL specs, flags unsupported | spec, category | local (`uv tool install --help`) | #264 |

### Implementation status

The registry implements the supported forms in the matrix above. Rejected forms
return a targeted hint instead of a character-class error.

`install` passes the typed spec, so versions and qualifiers are honored.
`remove`, `reinstall`, and `update -p` pass the tracked identity instead;
native maintenance commands do not reliably accept install-only qualifiers.
Maintenance identity is resolved as manifest name, then the installed-name
resolver (dnf aliases/NEVRA), then lexical identity. Batch operations apply the
same rule per package; batch installs resolve names after success.

Remaining follow-ups: category flags (`--pattern`, flatpak `--remote`), flags
inside specs (`--git`, `--suffix`, `--from`, `--channel`, `--repo`), VCS/URL
specs, and legacy manifest migration (use `stamp untrack`).

## Confirmed gaps (Task 1 audit)

The tables below are the original audit. Implementation status is summarized
above; a row that says “Support” is implemented only where the matrix’s
“Accepted today” column lists the form.

## Confirmed gaps

### Spec syntax

| Gap | Managers | Disposition |
|---|---|---|
| Version pinning in the spec (`pkg=ver`, `pkg==ver`, `crate@ver`, `module@ver`, `name@ver`) | apt, dnf, pacman, zypper, cargo, go, macports, pipx, uv | Support where the manager accepts it without extra flags; normalize identity to the package name. Go pinning conflicts with ADR-010, so reject with a hint until that policy changes. |
| Architecture / release qualifiers (apt `pkg:arch`/`pkg/release`; dnf NEVRA `.arch`; zypper capability arch) | apt, dnf, zypper | Support as pass-through; identity drops the qualifier. Release may select a downgrade — document it. |
| Capability / provides specs | dnf, pacman, zypper | Support; requires an identity resolver like the dnf one in ADR-025. |
| Version constraints / boolean rich deps (pacman/zypper `pkg>=ver`; dnf `(a or b)`) | pacman, zypper, dnf | Support; stamp does not interpret the operators, but the user's shell still splits spaces and parentheses, so quote them. |
| Language-tool specs (`pkg[extra]`, `pkg==ver`) | pipx, uv | Support extras and `==`; normalize identity to the package name. |

### Category mismatch

| Gap | Managers | Disposition |
|---|---|---|
| Groups (`@group`, `group install`, pacman group, zypper `pattern:`) | dnf, pacman, zypper | dnf already has `--group`. Add equivalent flags for pacman/paru/zypper, or reject with a targeted hint. Do not overload package names. |
| AUR/repo selector flags (`--aur`, `--repo`) | paru | Add explicit flags or reject with a hint; do not infer the source from package names. |
| Repository-qualified (`repo/pkg`, `repo:pkg`) | pacman, zypper | Support; keep `repo` out of the manifest identity. |
| Remote-qualified Flatpak refs | flatpak | Add an explicit remote flag; keep the current multi-arg rejection until then. |
| Local files, bundles, URLs | apt (`.deb`), dnf/pacman/zypper RPM paths, snap `.snap`, flatpak bundles/refs, cargo/pipx/uv VCS paths | Reject as package names. Route through explicit flags if support is added; paths/URLs need their own validation. |

### Manifest identity

| Gap | Managers | Disposition |
|---|---|---|
| NEVRA / qualified spec identity | dnf (batch, unresolved aliases) | Single installs already canonicalize through ADR-025. Extend resolution to batch installs and unresolved aliases; never record NEVRA/qualifier strings. |
| Group name recorded as a package | dnf/pacman/zypper | `Group: true` entries keep the group ID; presence and restore must treat them as groups. |
| Version suffix recorded | apt, pacman, zypper, cargo, macports, pipx, uv | Strip before `AddPackage`/matching; keep pass-through only for the native command. npm and dnf single installs already normalize (ADR-024/025). Go pinning is rejected by ADR-010, so no suffix is recorded. |
| Alias resolves to a different package | dnf | Already handled by ADR-025 for single installs; extend to batch when batch canonicalization lands. |

## Open questions for Task 2

1. **Version policy:** pass through and normalize, or reject with “use the native manager”?
   Supporting versions increases manifest/matching code; rejecting is simpler but blocks
   common apt/cargo/pipx usage.
2. **Category scope:** add flags (`--group`, `--remote`, `--kind`), or keep categories out of
   scope and reject with hints?
3. **Security policy:** keep the leading-`-` rejection and widen the allowlist, or keep a
   per-manager registry? Paths/URLs must stay behind explicit flags.
4. **Registry shape:** one `ValidateSpecForManager` plus per-manager normalizers, replacing
   the `go`/`brew`/`npm` special cases and duplicated adapter validation.
5. **Manifest identity:** define a single normalizer (`(name, manager)` from a spec) and
   reuse it in install, remove, reinstall, restore, and reconcile.
6. **Documentation:** a user-facing “Package Specification Grammar” page and an ADR recording
   the policy, linked from the supported-manager page (#261).

## Out of scope

- Windows adapters (#54–#56).
- Full command-by-command grammar for search/info/remove/update; only divergences are noted.
- Implementation. Task 2 owns validation changes, registry, ADR, and tests.
