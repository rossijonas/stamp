---
---

## stamp provides

Find which package provides a given file

### Synopsis

Search across all system package managers to find which package
owns the specified file. Supports both absolute and relative paths.

Scoped to a single manager with the --manager flag.

```
stamp provides <file> [flags]
```

### Examples

```
  # find which package owns a binary across all managers
  stamp provides /usr/bin/htop

  # scope to a single manager for faster results
  stamp provides libssl.so -m dnf

  # no results returns a clear message
  stamp provides /usr/bin/nonexistent
```

### Options

```
  -h, --help             help for provides
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

