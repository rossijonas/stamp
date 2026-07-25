---
---

## stamp update

Run system upgrades across all package managers

### Synopsis

Run system upgrade commands for each available package manager.
Updates and upgrades all packages to their latest versions.
Use -m to scope to a single package manager.
Use -p to update a single package (requires -m).
Use --serial to run updates one manager at a time (default: parallel).

```
stamp update [flags]
```

### Examples

```
  stamp update
  stamp update -m apt
  stamp update -p htop -m brew
  stamp update --serial
  stamp upgrade
```

### Options

```
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

