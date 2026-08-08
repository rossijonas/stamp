---
---

## Installing Repositories

### Add a repository

```bash
stamp repo add ppa:git-core/ppa -m apt
```

```text
▪ adding repo ppa:git-core/ppa via apt...
✓ added ppa:git-core/ppa via apt
```

### Add by URL

```bash
stamp repo add flathub https://dl.flathub.org/repo/flathub.flatpakrepo -m flatpak
```

```text
▪ adding repo flathub via flatpak...
✓ added flathub via flatpak
```

### Add a `.repo` file URL (DNF)

URLs ending in `.repo` (e.g. Brave, Enpass) point at a packaged repository file containing `baseurl`, `gpgkey`, and `gpgcheck=1`. Stamp fetches the file and installs it verbatim, so signature verification is preserved:

```bash
stamp repo add brave https://brave-browser-rpm-release.s3.brave.com/brave-browser.repo -m dnf
```

```text
▪ adding repo brave via dnf...
✓ added brave via dnf
```

### Add by URL without a name

When only a URL is given, the repository name is derived from the URL — the basename of the URL path (with a trailing `.repo` stripped), or the host for pathless URLs:

```bash
stamp repo add https://yum.enpass.io/enpass-yum.repo -m dnf
```

```text
▪ adding repo enpass-yum via dnf...
✓ added enpass-yum via dnf
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
| DNF | `.repo` file URL | `https://yum.enpass.io/enpass-yum.repo` |
| Flatpak | Remote URL | `https://dl.flathub.org/repo/flathub.flatpakrepo` |

### Using aliases

```bash
stamp repo install ppa:git-core/ppa -m apt
stamp repo uninstall ppa:git-core/ppa -m apt
stamp repo ls
```

### Homebrew tap aliases

```bash
stamp tap homebrew/cask        # equivalent to: stamp repo add homebrew/cask -m brew
stamp untap homebrew/cask      # equivalent to: stamp repo remove homebrew/cask -m brew
stamp taps                     # equivalent to: stamp repo list -m brew
```

```text
added tap homebrew/cask via brew
removed tap homebrew/cask via brew
homebrew/cask
homebrew/core
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
