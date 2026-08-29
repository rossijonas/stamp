---
---

## stamp remove

Remove a package and untrack it

```
stamp remove <package> [flags]
```

### Examples

```
  # remove using the manager recorded in the manifest
  stamp remove htop

  # specify a manager explicitly
  stamp remove lazygit -m brew

  # remove a DNF package group (by group ID)
  stamp remove development-tools -m dnf --group

  # all these aliases behave the same way
  stamp uninstall htop
  stamp rm htop
  stamp delete htop
  stamp del htop

  # remove multiple packages in one command (per-manager batch, -m required)
  stamp remove htop atop btop -m dnf
```

### Options

```
  -g, --group            remove a DNF package group (by group ID)
  -h, --help             help for remove
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

