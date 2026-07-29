---
---

## stamp untap

Remove a Homebrew tap (alias for repo remove -m brew)

### Synopsis

Remove a third-party Homebrew tap repository.
Equivalent to "stamp repo remove <name> -m brew".

```
stamp untap <name> [flags]
```

### Examples

```
  # remove a homebrew tap (equivalent to repo remove <name> -m brew)
  stamp untap homebrew/cask
```

### Options

```
  -h, --help   help for untap
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one

