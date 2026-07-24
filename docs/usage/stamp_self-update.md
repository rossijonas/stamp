---
---

## stamp self-update

Update stamp to the latest version

### Synopsis

Check for and apply updates to the stamp binary.

Downloads the latest release from GitHub, verifies its SHA-256 checksum,
replaces the current binary atomically, and re-installs shell completions
and man pages automatically. Use --check to query without downloading.

```
stamp self-update [flags]
```

### Examples

```
  stamp self-update
  stamp self-update --check
  stamp self-upgrade
```

### Options

```
  -c, --check   check for update without downloading
  -h, --help    help for self-update
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful wrapper for your native package managers

