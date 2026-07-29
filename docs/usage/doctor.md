---
---

## System Diagnosis

The `doctor` command checks your stamp installation, package managers, manifest, and system compliance.

### Default output

```bash
stamp doctor
```

```text
▪ System Diagnosis (Stamp Doctor)

Package Managers:
  Name       Status     Path                   Details
  apt        ✓ Active   /usr/bin/apt           Default system manager (Debian/Ubuntu)
  dnf        ✓ Active   /usr/bin/dnf           Default system manager (Fedora/RHEL, alias yum)
  brew       ✓ Active   /home/user/.linuxbrew  User-space manager
  flatpak    ✓ Active   /usr/bin/flatpak       Sandboxed application distribution
  snap       ✓ Active   /usr/bin/snap          Universal Linux package manager
  pacman     ✗ Not Found -                     Package manager for Arch Linux
  paru       ✗ Not Found -                     AUR helper for Arch Linux
  zypper     ✗ Not Found -                     Package manager for openSUSE/SLES
  macports   ✗ Not Found -                     Package manager for macOS
  go         ✓ Active   /usr/local/go/bin/go   Language toolchain — go install
  npm        ✓ Active   /usr/bin/npm           Node.js package manager
  cargo      ✗ Not Found -                     Rust package manager — cargo install
  pipx       ✓ Active   /usr/bin/pipx          Python tool installer
  uv         ✗ Not Found -                     Python package manager

Manifest Integrity:
  Path:   /home/user/.config/stamp/manifest.toml
  Status: ✓ Healthy (42 package(s))

UNIX Compliance:
  NO_COLOR: ✗ Not set
  Version:  stamp 0.24.0
  Man Page: ✓ Up to date (0.24.0)
  Completions: ✓ Installed (bash, zsh)
```

### JSON output

Machine-readable output for scripting and tooling:

```bash
stamp doctor --json
```

```json
{
  "system": "linux",
  "package_managers": [
    {"name": "apt", "status": "active", "path": "/usr/bin/apt"},
    {"name": "brew", "status": "active", "path": "/home/user/.linuxbrew"},
    ...
  ],
  "manifest": {
    "path": "/home/user/.config/stamp/manifest.toml",
    "healthy": true,
    "packages_count": 42
  },
  "version": "0.24.0",
  "man_page": {"installed": true, "version": "0.24.0"},
  "completions": {"installed": true, "shells": ["bash", "zsh"]}
}
```

### Scoped to a manager

```bash
stamp doctor -m brew
```

Checks if a specific manager binary is installed and operational.

### What doctor checks

| Section | Checks |
|---------|--------|
| Package Managers | Binary existence on PATH for all 14 supported managers |
| Manifest Integrity | Manifest file exists, parses correctly, lists package count |
| UNIX Compliance | XDG Base Directory, NO_COLOR, version, man page, shell completions |
