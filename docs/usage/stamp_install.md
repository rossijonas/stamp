---
---

## stamp install

Install a package and record intent

```
stamp install <package> [flags]
```

### Examples

```
  # install htop using the default system manager
  stamp install htop

  # install from a specific manager
  stamp install spotify -m flatpak

  # install a DNF package group (name may contain spaces)
  stamp install "Development Tools" -m dnf --group

  # add a note so you remember why later
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

