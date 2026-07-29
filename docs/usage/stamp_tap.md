---
---

## stamp tap

Add a Homebrew tap (alias for repo add -m brew)

### Synopsis

Add a third-party Homebrew tap repository.
Equivalent to "stamp repo add <name> -m brew".

```
stamp tap <name> [flags]
```

### Examples

```
  # add a homebrew tap (equivalent to repo add <name> -m brew)
  stamp tap homebrew/cask
```

### Options

```
  -h, --help   help for tap
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one

