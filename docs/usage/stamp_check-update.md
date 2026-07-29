---
---

## stamp check-update

Check for available updates without applying them

### Synopsis

Check across all package managers for available updates.
Equivalent to "stamp update --check".

```
stamp check-update [flags]
```

### Examples

```
  # check for available updates (read-only, same as stamp outdated)
  stamp check-update
```

### Options

```
  -h, --help   help for check-update
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one

