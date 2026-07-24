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
  stamp doctor
  stamp doctor --json
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

* [stamp](stamp.html)	 - A lightweight yet powerful wrapper for your native package managers

