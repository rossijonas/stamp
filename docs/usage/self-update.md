---
---

## Self-Update

Update Stamp itself to the latest released version:

```bash
stamp self-update
```

```text
▪ Checking for updates...
▪ Downloading stamp v0.21.0...
✅ Updated to v0.21.0
```

### Check only

```bash
stamp self-update --check
```

```text
▪ Current version: v0.20.0
▪ Latest version:  v0.21.0
  A new version is available. Run stamp self-update to upgrade.
```

When up to date:

```text
▪ stamp is already up to date (v0.20.0)
```

### Alias

```bash
stamp self-upgrade
```

### How it works

1. Fetches the latest release metadata from GitHub API
2. Downloads the tarball + SHA-256 checksums via HTTPS
3. Verifies the checksum of the downloaded archive
4. Extracts the binary from the tarball (with path traversal protection)
5. Checks write permission on the install directory
6. Atomically replaces the binary using a temp file + rename
7. Re-installs shell completions and man pages
