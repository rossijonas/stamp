---
---

## stamp repo untrust

Stop trusting a Homebrew tap

### Synopsis

Stop trusting a Homebrew tap. Only brew taps can be untrusted.

```
stamp repo untrust <name> [flags]
```

### Examples

```
  # untrust a tap recorded in the manifest
  stamp repo untrust homebrew/cask

  # specify the manager explicitly
  stamp repo untrust anomalyco/tap -m brew
```

### Options

```
  -h, --help             help for untrust
  -m, --manager string   package manager to use (optional if the repo is tracked in the manifest)
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp repo](stamp_repo.html)	 - Manage third-party repositories

