---
---

## stamp list

List all intentionally installed packages

### Synopsis

Read the manifest and display all tracked packages.
By default prints a table of package names and their managers.
Use --json for machine-readable output.
Use -m to filter by a specific package manager.
Use --type to filter by entity type (packages/repos) and origin
(stamped/reconciled). --type missing lists manifest packages not
currently installed (removed via the native manager).

```
stamp list [flags]
```

### Examples

```
  # list all tracked packages
  stamp list

  # machine-readable output
  stamp list --json

  # filter by package manager
  stamp list -m brew

  # filter by entity type and origin (stamped/reconciled)
  stamp list -t stamped-packages

  # packages in the manifest not installed on this system
  stamp list -t missing
```

### Options

```
  -h, --help             help for list
  -m, --manager string   package manager to list
  -t, --type string      Filter by entity type and origin. Valid types:
                           packages           All packages (default)
                           repos              All repositories
                           stamped            Everything installed via stamp (packages + repos)
                           reconciled         Everything discovered by reconcile (packages + repos)
                           stamped-packages   Packages installed via stamp
                           stamped-repos      Repos added via stamp
                           reconciled-packages Packages discovered by reconcile
                           reconciled-repos    Repos discovered by reconcile
                           missing            Manifest packages not installed on this system
                         
                         Origin meanings:
                           "stamped"    = installed explicitly via stamp install/reinstall
                           "reconciled" = installed outside stamp, auto-discovered by reconcile
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one

