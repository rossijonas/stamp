## stamp init

Initialize manifest.toml and take baseline snapshot

### Synopsis

Create the stamp configuration directory, an empty manifest.toml,
and take a baseline snapshot of currently installed packages
for each available package manager.

If stamp is already initialized, the existing manifest and snapshots
are always backed up before creating a fresh state. Use -y to skip
the confirmation prompt.

```
stamp init [flags]
```

### Options

```
  -h, --help   help for init
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.md)	 - A lightweight yet powerful wrapper for your native package managers

