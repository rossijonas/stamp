---
---

## stamp man install

Install the stamp man page to system or user path

```
stamp man install [flags]
```

### Examples

```
  # install to the default user path (~/.local/share/man)
  stamp man install

  # install under a custom prefix
  stamp man install --prefix /usr/local
```

### Options

```
  -h, --help            help for install
      --prefix string   install prefix (default: ~/.local)
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp man](stamp_man.html)	 - Manage stamp troff man pages

