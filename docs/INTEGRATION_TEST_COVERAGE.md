# Integration Test Coverage

Stamp runs Docker-based integration tests across 7 Linux distributions.
Each script tests the native package manager plus all cross-platform adapters.

## Test Matrix

| Distro | Script | Dockerfile | Native Adapter | Brew | Flatpak | Snap |
|--------|--------|------------|----------------|------|---------|------|
| Ubuntu (latest) | `ubuntu.sh` | `Dockerfile.ubuntu` | APT | ✅ search, install, remove | ✅ repo list, search | ⚠️ guarded |
| Debian (latest) | `debian.sh` | `Dockerfile.debian` | APT | ✅ search, install, remove | ✅ repo list, search | ⚠️ guarded |
| Fedora (latest) | `fedora.sh` | `Dockerfile.fedora` | DNF | ✅ search, install, remove | ✅ repo list, search | ⚠️ guarded |
| CentOS Stream 10 | `centos.sh` | `Dockerfile.centos` | DNF | ✅ search, install, remove | ✅ repo list, search | ⚠️ guarded |
| Rocky Linux 9 | `rocky.sh` | `Dockerfile.rocky` | DNF | ✅ search, install, remove | ✅ repo list, search | ⚠️ guarded |
| Arch Linux | `arch.sh` | `Dockerfile.arch` | — (brew only) | ✅ search, install, remove | ✅ repo list, search | ⚠️ guarded |
| openSUSE Tumbleweed | `opensuse.sh` | `Dockerfile.opensuse` | Zypper | ✅ search, install, remove | ✅ repo list, search | ⚠️ guarded |

## Cross-Platform Tests (all scripts)

Each script validates a common set of features:

| Test Suite | What's Covered |
|------------|---------------|
| **doctor** | `stamp doctor` runs, shows managers, `--json` valid |
| **list** | `stamp list`, `stamp list --json` valid |
| **help** | All command `--help` outputs |
| **self-update** | `stamp self-update --check`, `stamp self-upgrade --check` |
| **root command** | `stamp` with no arguments |
| **JSON output** | doctor, list — all parseable via `python3 -m json.tool` |
| **aliases** | `add`, `rm`, `ls`, `uninstall`, `delete`, `del` |
| **shell completions** | `stamp completion --stdout bash`, valid bash syntax |

## Per-Distro Tests

| Distro | Native Adapter Operations |
|--------|--------------------------|
| Ubuntu | APT: search, install, remove, reinstall, repo add/remove/list |
| Debian | APT: search, install, remove, reinstall |
| Fedora | DNF: search, install, remove, reinstall, repo list |
| CentOS | DNF: search, install, remove, repo list |
| Rocky | DNF: search, install, remove, repo list |
| openSUSE | Zypper: search, install, remove |

## Caveats & Known Gaps

### Snap (`snap list -m snap`, `snap search`)
- Snap requires `snapd` which needs `--privileged` + systemd in Docker.
- Our containers are unprivileged — snap tests are guarded by `command -v snap`.
- Tests run only on real Ubuntu machines where `snapd` is installed.
- On systems without `snap`, a message is printed and tests are skipped.

### Flatpak search (`stamp search Calculator -m flatpak`)
- First search downloads Flathub metadata cache — uses `TIMEOUT_LONG`.
- Cached on subsequent runs within the same image.

### Brew install (`stamp install hello -m brew`)
- `hello` is GNU Hello ~100KB — fast to install.
- Requires Homebrew pre-installed (done in all Dockerfiles).

### Distros without native adapter
- Arch Linux has no native adapter yet (Pacman pending).
- Tests on Arch run only cross-platform adapters (brew, flatpak, snap).

### What's NOT tested in integration tests
- **`stamp reconcile` on all distros**: Only tested on Ubuntu, Fedora, CentOS, Debian, Rocky.
- **`stamp restore` on all distros**: Only tested on Ubuntu, Fedora, CentOS, Rocky, Debian.
- **`stamp update` on all distros**: Only tested on Ubuntu, Fedora, CentOS, Debian, Rocky.
- **`stamp repo add/remove`**: Only tested on Ubuntu (PPA), Fedora/CentOS/Rocky (copr).
- **Error paths** (`--invalid` names, nonexistent packages): Only on Debian, Fedora, Ubuntu.

### Test Infrastructure
- All Dockerfiles use `USER linuxbrew` for Homebrew compatibility.
- Flatpak remotes are added per-user as `linuxbrew`.
- `TIMEOUT_EXTRA=120` variable available for slow operations.
- Tests are orchestrated via GitHub Actions (one workflow per distro).
