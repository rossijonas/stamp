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
  # reinstall a package using the manager recorded in the manifest
  stamp reinstall htop

  # reinstall a pre-existing package from a specific manager
  stamp reinstall lazygit -m brew

  # reinstall multiple packages in one command (per-manager batch, -m required)
  stamp reinstall lazygit jq -m brew

  # annotate a reinstall so you remember why later
  stamp reinstall lazygit -m brew --note "refresh intent"
```

### Options

```
  -h, --help             help for reinstall
  -m, --manager string   package manager to use (pre-existing packages only)
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

