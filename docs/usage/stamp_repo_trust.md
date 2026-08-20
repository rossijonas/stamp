---
---

## stamp repo trust

Trust a Homebrew tap

### Synopsis

Mark a Homebrew tap as trusted so Homebrew 6.0.0+ loads its formulae,
casks, and commands. Only brew taps can be trusted.

```
stamp repo trust <name> [flags]
```

### Examples

```
  # trust a tap recorded in the manifest
  stamp repo trust homebrew/cask

  # specify the manager explicitly
  stamp repo trust anomalyco/tap -m brew
```

### Options

```
  -h, --help             help for trust
  -m, --manager string   package manager to use (optional if the repo is tracked in the manifest)
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp repo](stamp_repo.html)	 - Manage third-party repositories

