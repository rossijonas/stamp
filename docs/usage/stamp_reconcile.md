---
---

## stamp reconcile

Detect packages installed outside stamp and add them to the manifest

### Synopsis

Compare the current system package state against the last snapshot.
Any new packages found are auto-tracked to the manifest.
Use --dry-run to preview drift without tracking.

```
stamp reconcile [flags]
```

### Examples

```
  stamp reconcile
  stamp reconcile --dry-run
  stamp reconcile -m dnf
```

### Options

```
  -d, --dry-run          preview drift without tracking
  -h, --help             help for reconcile
  -m, --manager string   package manager to reconcile
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful wrapper for your native package managers

