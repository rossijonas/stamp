---
---

## stamp info

Show package information across managers

### Synopsis

Query detailed information about a package.
By default, queries all available managers and outputs a summary table.
If -m, --manager is specified, displays the native manager's full raw info block.

```
stamp info <package> [flags]
```

### Examples

```
  # show package info across all managers (summary table)
  stamp info htop

  # show full raw output from a specific manager
  stamp info htop -m dnf

  # machine-readable JSON output
  stamp info htop --json

  # query info about a DNF package group
  stamp info "Development Tools" -m dnf --group
```

### Options

```
  -g, --group            query a DNF package group
  -h, --help             help for info
  -m, --manager string   package manager to query
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one

