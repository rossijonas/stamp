---
---

## stamp manifest history

List manifest backups

### Synopsis

List the current manifest and all timestamped backups, newest first.
Each row shows the backup timestamp, a short content hash, and package/repo
counts. The current manifest is marked with '*'. Backups whose content is
identical to the current manifest are marked as unchanged.

```
stamp manifest history [flags]
```

### Examples

```
  # list manifest backups, newest first
  stamp manifest history

  # machine-readable history
  stamp manifest history -j
```

### Options

```
  -h, --help   help for history
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp manifest](stamp_manifest.html)	 - Inspect manifest backups and changes

