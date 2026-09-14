# Security Policy

## Supported Versions

stamp is pre-1.0; only the latest release line is supported.

| Version | Supported |
| ------- | --------- |
| 0.38.x  | ✅        |
| < 0.38  | ❌        |

## Reporting a Vulnerability

Do **not** open a public issue for a security vulnerability. Report it privately through GitHub Security Advisories:

https://github.com/rossijonas/stamp/security/advisories/new

Include, where possible:

- the affected `stamp --version`
- your OS and distribution
- the package manager involved
- reproduction steps or a proof of concept
- the impact you believe it has

## Scope

In scope:

- privilege escalation or unsafe `sudo` / privileged command handling
- command or argument injection via package names, managers, or manifest values
- destructive operations performed without the expected confirmation or beyond the requested action (`remove`, `reconcile`, `restore`)
- corruption of the manifest, completions, or snapshots
- path traversal or symlink attacks against stamp-managed files
- leakage of secrets or sensitive data

Out of scope:

- vulnerabilities in the underlying package managers themselves (report those upstream)
- issues that require an already-compromised host, or an attacker who already has `sudo` access
- social engineering

## Disclosure

This is a volunteer-run project, so response times are best-effort. We will acknowledge your report, keep you updated on a fix, and credit you in the advisory unless you prefer otherwise. Please allow a reasonable window to release a fix before public disclosure.
