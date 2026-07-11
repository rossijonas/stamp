## stamp restore

Restore all tracked repositories and packages from the manifest

### Synopsis

Read the manifest and restore your system state.
It first adds all tracked repositories sequentially,
then installs all tracked packages concurrently across package managers.

```
stamp restore [flags]
```

### Options

```
      --dry-run   preview repositories and packages to restore
  -h, --help      help for restore
```

### Options inherited from parent commands

```
      --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.md)	 - Track package installation intent across multiple package managers

