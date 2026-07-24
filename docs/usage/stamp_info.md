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
  stamp info htop
  stamp info -m brew lazygit
  stamp info htop --json
```

### Options

```
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

* [stamp](stamp.html)	 - A lightweight yet powerful wrapper for your native package managers

