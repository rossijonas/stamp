---
---

## stamp hold

Pin a package at its current version to prevent upgrades

### Synopsis

Pin a package at its current version to prevent accidental upgrades.

Scoped to a single manager with the --manager flag.
Supported managers: apt (apt-mark), dnf (dnf versionlock), pacman/paru (IgnorePkg).

```
stamp hold <package> [flags]
```

### Examples

```
  # hold with apt (via apt-mark)
  stamp hold nginx -m apt

  # hold with dnf (via dnf versionlock)
  stamp hold nginx -m dnf

  # hold on arch with pacman (adds to IgnorePkg in pacman.conf)
  stamp hold nginx -m pacman
```

### Options

```
  -h, --help             help for hold
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

