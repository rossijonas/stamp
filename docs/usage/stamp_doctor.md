---
---

## stamp doctor

Diagnose system configuration and manifest health

### Synopsis

Check package manager availability and manifest integrity.
Reports which managers are installed and whether the manifest is valid.

```
stamp doctor [flags]
```

### Examples

```
  # check the whole system
  stamp doctor

  # machine-readable output for scripting
  stamp doctor --json

  # check a single manager's native diagnostics
  stamp doctor -m dnf
```

### Options

```
  -h, --help             help for doctor
  -m, --manager string   package manager to check
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one

