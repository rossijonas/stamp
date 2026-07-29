---
---

## stamp search

Search for packages across managers

```
stamp search <query> [flags]
```

### Examples

```
  stamp search htop
  stamp search lazygit -m brew
  stamp search ripgrep
```

### Options

```
  -g, --group            search DNF package groups
  -h, --help             help for search
  -m, --manager string   package manager to search
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one

