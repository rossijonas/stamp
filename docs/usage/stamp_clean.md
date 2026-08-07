---
---

## stamp clean

Clean package caches and temporary files

### Synopsis

Remove locally cached package files across all package managers.
Use --dry-run to preview what would be cleaned without deleting.

Scoped to a single manager with the --manager flag.

```
stamp clean [flags]
```

### Examples

```
  # clean package caches
  stamp clean

  # preview what would be cleaned
  stamp clean --dry-run

  # scope to a single package manager
  stamp clean -m brew
```

### Options

```
  -d, --dry-run          preview what would be cleaned
  -h, --help             help for clean
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

