---
---

## stamp reinstall

Reinstall a package and record it in the manifest

### Synopsis

Look up the package in the manifest to find its recorded package manager,
then execute the native reinstallation command. If the package is not
tracked in the manifest, resolve the manager and track it.

```
stamp reinstall <package> [flags]
```

### Examples

```
  stamp reinstall htop
  stamp reinstall -m brew lazygit
```

### Options

```
  -h, --help             help for reinstall
  -m, --manager string   package manager to use (pre-existing packages only)
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful wrapper for your native package managers

