---
---

## Removing Repositories

```bash
stamp repo remove ppa:git-core/ppa -m apt
```

```text
▪ removing repo ppa:git-core/ppa via apt...
✓ removed ppa:git-core/ppa via apt
```

The `--manager` / `-m` flag is optional: when the repository is tracked in the
manifest, the recorded manager is used automatically.

```bash
stamp repo remove enpass
```

### Using aliases

```bash
stamp repo rm homebrew/tap -m brew
stamp repo delete copr:user/repo -m dnf
stamp repo uninstall ppa:git-core/ppa -m apt
```

### DNF removal by repo type

For DNF, removal follows how the repository was added:

| Added as | Removed by |
|---------|-----------|
| COPR (`stamp repo add user/repo -m dnf`) | `dnf copr disable` |
| URL (baseurl or `.repo` file, e.g. Brave, Enpass) | deleting `/etc/yum.repos.d/<name>.repo` |

Repos added by URL are removed by deleting their `.repo` file; name-only COPR
repos are disabled via `dnf copr disable`.

### Homebrew tap removal

`stamp repo remove <tap> -m brew` untaps the tap **and** best-effort untrusts it
(the prompt reads "Remove and untrust repo X via brew"), so a later re-tap
starts clean under Homebrew 6.0.0+ tap-trust.

