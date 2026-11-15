---
---

## stamp untrack

Remove a package from the manifest without touching the system

### Synopsis

Drop a package from the manifest without running the native package manager.

Use this to clear a stale or mis-attributed entry (for example a package
recorded under a name the native manager no longer resolves). The installed
package, if any, is left untouched. Normal uninstalls should use 'stamp remove'.

When the same name is tracked under more than one manager, pass -m to choose
the entry. -y skips the confirmation prompt; without it, non-interactive runs
fail closed.

```
stamp untrack <package> [flags]
```

### Examples

```
  # drop an entry the native manager cannot resolve
  stamp untrack yarn -m dnf

  # skip the confirmation prompt
  stamp untrack yarn -m dnf -y
```

### Options

```
  -h, --help             help for untrack
  -m, --manager string   manager the package is tracked under
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one

