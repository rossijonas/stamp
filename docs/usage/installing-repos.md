---
---

## Installing Repositories

### Add a repository

```bash
stamp repo add ppa:git-core/ppa -m apt
```

```text
▪ adding repo ppa:git-core/ppa via apt...
✅ added ppa:git-core/ppa via apt
```

### Add by URL

```bash
stamp repo add flathub https://dl.flathub.org/repo/flathub.flatpakrepo -m flatpak
```

```text
▪ adding repo flathub via flatpak...
✅ added flathub via flatpak
```

The `--manager` / `-m` flag is **required** for all repo operations.

### Manager-specific repo types

| Manager | Repository type | Example |
|---------|----------------|---------|
| APT | PPA | `ppa:git-core/ppa` |
| APT | Deb URL | `deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main` |
| Brew | Tap | `homebrew/cask` |
| DNF | COPR | `petersen/cava` |
| DNF | RPM URL | `https://rpm.example.com/repo` |
| Flatpak | Remote URL | `https://dl.flathub.org/repo/flathub.flatpakrepo` |

### Using aliases

```bash
stamp repo install ppa:git-core/ppa -m apt
```

### List repositories

```bash
stamp repo list
stamp repo list --json
stamp repo ls -m flatpak
```

```json
[
  {"name": "flathub", "manager": "flatpak", "url": "https://dl.flathub.org/repo/flathub.flatpakrepo"}
]
```
