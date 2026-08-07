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
  Missing:
    - htop (dnf)
    - spotify (flatpak)
    run 'stamp restore' to reinstall, or 'stamp ls --type missing' for details

UNIX Compliance:
  NO_COLOR: ✗ Not set
  Version:  stamp v0.31.1
  Man Page: ✓ Up to date (v0.31.1)
  Completions: ✓ Installed (bash, zsh)
```

> Output captured at v0.31.1; manager availability and paths vary by platform.

### JSON output

Machine-readable output for scripting and tooling:

```bash
stamp doctor --json
```

```json
{
  "system": "linux",
  "version": "0.31.1",
  "package_managers": [
    {"name": "apt", "active": true, "path": "/usr/bin/apt", "details": "Default system manager (Debian/Ubuntu)"},
    {"name": "brew", "active": true, "path": "/home/user/.linuxbrew/bin/brew", "details": "User-space manager"},
    ...
  ],
  "manifest": {
    "path": "/home/user/.config/stamp/manifest.toml",
    "valid": true,
    "packages_count": 42,
    "missing": [
      {"Name": "htop", "Manager": "dnf", "Category": "", "Notes": "", "Cask": false, "Group": false, "Origin": ""},
      {"Name": "spotify", "Manager": "flatpak", "Category": "", "Notes": "", "Cask": false, "Group": false, "Origin": ""}
    ]
  },
  "no_color": false,
  "man_page": {"installed": true, "matches": true, "path": "/home/user/.local/share/man/man1/stamp.1", "version": "0.31.1"},
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
| Missing Packages | Manifest entries not installed on the system (per active manager; best-effort) |
| UNIX Compliance | XDG Base Directory, NO_COLOR, version, man page, shell completions |

### Missing packages

Doctor compares the manifest against what is actually installed. If you
removed a tracked package with your native manager (`sudo dnf remove htop`),
it shows up under `Missing:` in "Manifest Integrity" (and as `manifest.missing`
in `--json`). This does **not** change the manifest status — a missing package
is drift, not corruption. `stamp ls --type missing` lists the same set;
`stamp restore` reinstalls them. Managers whose installed state cannot be
queried are skipped.
