---
---

## stamp autoremove

Remove orphaned packages and unused dependencies

### Synopsis

Remove orphaned packages and unused dependencies across all
package managers. Use --dry-run to preview what would be removed.

Scoped to a single manager with the --manager flag.

```
stamp autoremove [flags]
```

### Examples

```
  # remove orphaned dependencies
  stamp autoremove

  # preview what would be removed
  stamp autoremove --dry-run

  # scope to a single package manager
  stamp autoremove -m brew
```

### Options

```
  -d, --dry-run          preview orphans without removing
  -h, --help             help for autoremove
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

