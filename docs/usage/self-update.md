---
---

## Self-Update

Update Stamp itself to the latest released version:

```bash
stamp self-update
```

```text
▪ Self-Update
  Downloading stamp-v0.31.1-linux-amd64.tar.gz...
  ✓ Updated to v0.31.1
  Reinstalling shell completions...
  ✓ Completions updated
  Reinstalling man pages...
  ✓ Man pages updated
```

> Version numbers shown for illustration; actual values reflect the current release.

### Check only

```bash
stamp self-update --check
```

```text
  Current version: v0.31.1
  Latest version:  v0.32.0
  A new version is available.
```

When up to date:

```text
  Current version: v0.31.1
  Latest version:  v0.31.1
  Already up to date.
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
