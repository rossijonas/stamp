---
---

## stamp setup

Run first-time setup wizard

### Synopsis

Guided setup for new stamp installations.
Runs completion installation, man page setup, initialization, and diagnostics.
Use -y to skip all prompts for scripting.

```
stamp setup [flags]
```

### Examples

```
  # run the interactive first-time setup wizard
  stamp setup

  # non-interactive setup for scripting
  stamp setup -y
```

### Options

```
  -h, --help   help for setup
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one

