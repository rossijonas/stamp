---
---

## stamp outdated

Check for available updates without applying them

### Synopsis

Check across all package managers for outdated packages.
Equivalent to "stamp update --check".

```
stamp outdated [flags]
```

### Examples

```
  # check which packages have newer versions available (read-only)
  stamp outdated
```

### Options

```
  -h, --help   help for outdated
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one

