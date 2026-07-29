---
---

## stamp override

Manage Flatpak sandbox permissions

### Synopsis

Set, show, or reset Flatpak sandbox permissions for an application.

Requires --manager flatpak. Use repeatable flags for filesystem, socket,
device, and environment variables. At least one action flag is required.

```
stamp override <app-id> [flags]
```

### Examples

```
  stamp override firefox -m flatpak --filesystem=host
  stamp override firefox -m flatpak --socket=wayland
  stamp override firefox -m flatpak --device=dri
  stamp override firefox -m flatpak --env=MY_VAR=value
  stamp override firefox -m flatpak --reset
  stamp override firefox -m flatpak --show
```

### Options

```
      --device stringArray       grant device access (repeatable)
      --env stringArray          set environment variable KEY=VALUE (repeatable)
      --filesystem stringArray   grant filesystem access (repeatable)
  -h, --help                     help for override
  -m, --manager string           package manager to use (required)
      --reset                    reset all overrides to defaults
      --show                     show current overrides
      --socket stringArray       grant socket access (repeatable)
      --system                   apply system-wide (requires sudo)
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one

