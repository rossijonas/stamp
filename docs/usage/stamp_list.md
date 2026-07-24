---
---

## stamp list

List all intentionally installed packages

### Synopsis

Read the manifest and display all tracked packages.
By default prints a table of package names and their managers.
Use --json for machine-readable output.
Use -m to filter by a specific package manager.

```
stamp list [flags]
```

### Examples

```
  stamp list
  stamp list --json
  stamp list -m brew
```

### Options

```
  -h, --help             help for list
  -m, --manager string   package manager to list
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful wrapper for your native package managers

