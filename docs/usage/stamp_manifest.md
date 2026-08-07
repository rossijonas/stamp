---
---

## stamp manifest

Inspect manifest backups and changes

### Synopsis

Command group for manifest management: list backup history and
diff the current manifest against a backup.

```
stamp manifest [flags]
```

### Examples

```
  # list manifest backups (newest first, with content hashes)
  stamp manifest history

  # diff the current manifest against a backup
  stamp manifest diff
```

### Options

```
  -h, --help   help for manifest
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one
* [stamp manifest diff](stamp_manifest_diff.html)	 - Compare current manifest against a backup
* [stamp manifest history](stamp_manifest_history.html)	 - List manifest backups

