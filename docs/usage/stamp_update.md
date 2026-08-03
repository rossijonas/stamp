---
---

## stamp update

Run system upgrades across all package managers

### Synopsis

Run system upgrade commands for each available package manager using a safe two-phase (check + confirm) flow.

By default, checks for available updates, displays them, and prompts for confirmation before upgrading.
Use --check to only run the check phase (dry-run).
Note: the check phase refreshes package metadata first, which may require sudo and network access.
Use -y to skip the check phase and auto-confirm for maximum speed.
Use -m to scope to a single package manager.
Use -p to update a single package (requires -m).
Use --serial to run updates one manager at a time (default: parallel).

```
stamp update [flags]
```

### Examples

```
  # default two-phase flow (check + confirm, then update)
  stamp update

  # dry-run: check for updates without applying them
  # (refreshes package metadata first — may require sudo and network access)
  stamp update --check

  # skip check phase, auto-confirm (useful in scripts)
  stamp update -y

  # update a specific package (requires --manager)
  stamp update -p htop -m brew

  # run updates one manager at a time instead of parallel
  stamp update --serial

  # alias
  stamp upgrade
```

### Options

```
  -c, --check            check for available updates without applying them
  -h, --help             help for update
  -m, --manager string   package manager to update
  -p, --package string   update a single package (requires --manager)
  -s, --serial           run updates one at a time (sequential)
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one

