## stamp reconcile

Detect packages installed outside stamp and add them to the manifest

### Synopsis

Compare the current system package state against the last snapshot.
Any new packages found are surfaced as potential intentional installs
and can be added to the manifest.

```
stamp reconcile [flags]
```

### Options

```
  -h, --help             help for reconcile
  -m, --manager string   package manager to reconcile
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.md)	 - A lightweight yet powerful wrapper for your native package managers

