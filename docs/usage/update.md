---
---

## Updating Packages

```bash
stamp update
```

Runs the upgrade command for all available package managers concurrently.

```text
▪ Authentication required for system package managers
[apt] Reading package lists... Done
[apt] Upgrading: 3 packages
[brew] Already up-to-date.
[flatpak] Looking for updates... Done
✅ updated packages via apt
✅ updated packages via flatpak
```

### Scoped to a manager

```bash
stamp update -m apt
```

```text
[apt] Reading package lists... Done
[apt] Upgrading: 3 packages
✅ updated packages via apt
```

### Aliases

```bash
stamp upgrade
```

### Serial mode

```bash
stamp update --serial
```

Runs updates one manager at a time (useful for debugging):

```text
▪ Authentication required for system package managers
▪ updating via apt...
[apt] Reading package lists... Done
[apt] Upgrading: 3 packages
✓ updated packages via apt
▪ updating via brew...
Already up-to-date.
✓ updated packages via brew
```

### Error handling

If one manager fails, others continue. The command exits with a non-zero status:

```text
⚠ update failed for apt: exit status 100
✅ updated packages via brew
Error: one or more managers failed to update
```
