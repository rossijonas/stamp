---
---

## stamp repo list

List all tracked repositories

```
stamp repo list [flags]
```

### Examples

```
  # list all tracked repositories
  stamp repo list

  # filter by package manager
  stamp repo list -m flatpak

  # machine-readable JSON output
  stamp repo list --json
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

* [stamp repo](stamp_repo.html)	 - Manage third-party repositories

