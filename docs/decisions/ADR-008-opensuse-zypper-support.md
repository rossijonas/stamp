---
---

# ADR-008: Support openSUSE and Zypper Package Manager

## Status

Accepted

## Context

Stamp has integration tests running on openSUSE Tumbleweed
(`test/Dockerfile.opensuse`, `test/integration/opensuse.sh`) but no native
package manager adapter for Zypper — openSUSE's native package manager. Users
on openSUSE can only use stamp with cross-platform managers (brew, flatpak).

openSUSE is a major Linux distribution with a large user base, especially in
European enterprise and development environments. Zypper is the standard
package manager for openSUSE and SUSE Linux Enterprise.

## Decision

Implement a Zypper adapter matching the `manager.Adapter` interface:

| Method | Command |
|--------|---------|
| ListInstalled | `zypper search --installed-only` |
| Install | `sudo zypper install -y <pkg>` |
| Reinstall | `sudo zypper install --force -y <pkg>` |
| Remove | `sudo zypper remove -y <pkg>` |
| Search | `zypper search <query>` |
| Info | `zypper info <pkg>` |
| Update | `sudo zypper update -y` |
| Doctor | Not supported |
| AddRepo/RemoveRepo/ListRepos | Pending investigation — Zypper has repo management via `zypper repos` and `zypper addrepo` |

- **File:** `internal/manager/zypper.go`
- **NeedsRoot:** true (write operations only)
- **Test file:** `internal/manager/zypper_test.go`

## Consequences

- openSUSE users get full native package manager support via stamp
- Integration tests in `opensuse.sh` will be extended with Zypper-specific
  tests covering install, remove, search, and info operations
- The existing `test/Dockerfile.opensuse` already has the necessary
  dependencies — no infrastructure changes needed
- Zypper uses the same `sudoCmd` helper as APT and DNF for privilege
  escalation
