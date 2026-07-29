---
---

## stamp repo remove

Remove a third-party repository

```
stamp repo remove <name> [flags]
```

### Examples

```
  # remove a PPA or repository
  stamp repo remove ppa:git-core/ppa -m apt

  # aliases behave the same way
  stamp repo rm ppa:git-core/ppa -m apt
```

### Options

```
  -h, --help             help for remove
  -m, --manager string   package manager (required)
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp repo](stamp_repo.html)	 - Manage third-party repositories

