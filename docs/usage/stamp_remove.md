---
---

## stamp remove

Remove a package and untrack it

```
stamp remove <package> [flags]
```

### Examples

```
  stamp remove htop
  stamp remove -m brew lazygit
  stamp uninstall htop
  stamp rm htop
  stamp delete htop
```

### Options

```
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

* [stamp](stamp.html)	 - A lightweight yet powerful wrapper for your native package managers

