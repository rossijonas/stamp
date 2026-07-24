---
---

## Removing Repositories

```bash
stamp repo remove ppa:git-core/ppa -m apt
```

```text
▪ removing repo ppa:git-core/ppa via apt...
✅ removed ppa:git-core/ppa via apt
```

### Using aliases

```bash
stamp repo rm homebrew/tap -m brew
stamp repo delete copr:user/repo -m dnf
stamp repo uninstall ppa:git-core/ppa -m apt
```

The `--manager` / `-m` flag is required.
