---
---

## stamp auto-reconcile

Manage automated reconcile timer

### Synopsis

Install or remove a system timer that runs stamp reconcile automatically.
On Linux: uses systemd user timer.
On macOS: uses launchd agent.

### Options

```
  -h, --help   help for auto-reconcile
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp](stamp.html)	 - A lightweight yet powerful tool that wraps many package managers into one
* [stamp auto-reconcile off](stamp_auto-reconcile_off.html)	 - Remove automated reconcile timer
* [stamp auto-reconcile on](stamp_auto-reconcile_on.html)	 - Install automated reconcile timer

