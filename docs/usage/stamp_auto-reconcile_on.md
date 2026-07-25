---
---

## stamp auto-reconcile on

Install automated reconcile timer

### Synopsis

Install a system timer to run stamp reconcile automatically.
Use --period to set the interval (daily, hourly, weekly).
On Linux: creates systemd user service + timer.
On macOS: creates a launchd agent.

```
stamp auto-reconcile on [flags]
```

### Options

```
  -h, --help            help for on
  -p, --period string   timer period: hourly, daily, weekly (default "daily")
```

### Options inherited from parent commands

```
  -j, --json      output results in JSON format
  -v, --verbose   enable debug logging
  -y, --yes       auto-accept all prompts
```

### SEE ALSO

* [stamp auto-reconcile](stamp_auto-reconcile.html)	 - Manage automated reconcile timer

