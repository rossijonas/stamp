---
---

## stamp unhold

Remove a version pin, allowing upgrades

### Synopsis

Remove a version pin from a package, allowing it to be upgraded again.

Scoped to a single manager with the --manager flag.
Supported managers: apt (apt-mark), dnf (dnf versionlock), pacman/paru (IgnorePkg).

```
stamp unhold <package> [flags]
```

### Examples

```
  # unhold with apt (via apt-mark)
  stamp unhold nginx -m apt

  # unhold on arch with pacman (removes from IgnorePkg)
  stamp unhold nginx -m pacman
```

### Options

```
  -h, --help             help for unhold
  -m, --manager string   package manager to use
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one

