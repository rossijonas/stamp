---
---

## stamp install

Install a package and record intent

```
stamp install <package> [flags]
```

### Examples

```
  stamp install htop
  stamp install spotify --manager flatpak
  stamp add lazygit -m brew --note "better git TUI"
```

### Options

```
  -g, --group            install a DNF package group
  -h, --help             help for install
  -m, --manager string   package manager to use
  -n, --note string      annotation for this package
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one

