---
---

## stamp manifest diff

Compare current manifest against a backup

### Synopsis

Show the difference between the current manifest and a specific backup.
Defaults to the most recent backup. The argument may be a backup timestamp
(2026-08-02T09:15:00Z or 20260802T091500Z) or a content-hash prefix shown by
stamp manifest history. Added entries are prefixed with '+', removed with '-'.

```
stamp manifest diff [timestamp|hash] [flags]
```

### Examples

```
  # diff the current manifest against the most recent backup
  stamp manifest diff

  # diff against a specific backup by timestamp or hash prefix
  stamp manifest diff 2026-08-02T09:15:00Z
  stamp manifest diff a1b2c3d4e5f6

  # filter by manager and origin
  stamp manifest diff -m brew --origin stamped
```

### Options

```
  -h, --help             help for diff
  -m, --manager string   filter by package manager
      --origin string    filter by origin: stamped or reconciled
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp manifest](stamp_manifest.html)	 - Inspect manifest backups and changes

