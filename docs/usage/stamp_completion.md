---
---

## stamp completion

Generate and install shell completion script

### Synopsis

Generate and install shell completion scripts for stamp.

Without arguments, auto-detects the current shell and installs to the
correct system path. Use --stdout to print the script instead.

```
stamp completion [bash|zsh|fish|powershell]
```

### Examples

```
  stamp completion
  stamp completion --stdout bash
  stamp completion fish
```

### Options

```
  -h, --help     help for completion
  -s, --stdout   print completion script to stdout instead of installing
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful wrapper for your native package managers

