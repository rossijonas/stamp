---
---

## stamp reconcile

Detect packages installed outside stamp and add them to the manifest

### Synopsis

Compare the current system package state against the last snapshot.
Any new packages found are auto-tracked to the manifest.
Use --dry-run to preview drift without tracking.

Before tracking, the current manifest is timestamp-backed up, and old
manifest backups are pruned per the [backup] policy in config.toml.
Dry-run performs no writes, no backups, and no rotation.

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

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one

