---
---

## stamp held

List all held/pinned packages

### Synopsis

List all packages currently held/pinned across all managers.
Use --manager to scope to a single package manager.

```
stamp held [flags]
```

### Examples

```
  stamp held
  stamp held -m apt
```

### Options

```
  -h, --help             help for held
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

