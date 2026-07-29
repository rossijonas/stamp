---
---

## stamp restore

Restore all tracked repositories and packages from the manifest

### Synopsis

Read the manifest and restore your system state.
It first adds all tracked repositories sequentially,
then installs all tracked packages concurrently across package managers.

```
stamp restore [flags]
```

### Examples

```
  # restore all repositories and packages from the manifest
  stamp restore

  # skip confirmation and proceed immediately
  stamp restore -y

  # preview what would be restored without making changes
  stamp restore --dry-run

  # restore only packages from a specific manager
  stamp restore -m brew
```

### Options

```
  -d, --dry-run          preview repositories and packages to restore
  -h, --help             help for restore
  -m, --manager string   package manager to restore
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one

