---
---

## Setup

The setup wizard (`stamp setup`) runs first-time configuration in four steps. It's the recommended way to initialize Stamp on a new machine.

```bash
stamp setup        # interactive wizard (prompts before each step)
stamp setup -y     # non-interactive, accept all defaults
stamp hello        # alias for setup
```

### Step 1: Shell Completions

Installs shell completion scripts for auto-completion of commands and flags.

| Shell | Install path |
|-------|-------------|
| Bash | `~/.local/share/bash-completion/completions/stamp` |
| Zsh | `~/.local/share/zsh/site-functions/_stamp` |
| Fish | `~/.config/fish/completions/stamp.fish` |
| PowerShell | Falls back to `--stdout` (no auto-install) |

If completions are already installed, this step detects them and skips re-installation.

### Step 2: Man Pages

Generates and installs the system man page for Stamp.

```bash
man stamp
```

Man pages are installed to `~/.local/share/man/man1/stamp.1`. To install to a custom location, use `stamp man install --prefix /usr/local` instead.

### Step 3: Initialize Manifest + Snapshot

Creates the manifest and takes a **baseline snapshot** of all currently installed packages across every available package manager.

```text
~/.config/stamp/
  manifest.toml          # Tracked packages and repositories
~/.local/share/stamp/
  snapshots/             # Baseline snapshots per manager
```

#### First-time init

On a fresh system, the wizard creates an empty manifest and takes a full system snapshot. Everything installed *before* `stamp init` is captured in the baseline and will never be detected by `stamp reconcile`.

#### Re-init (already initialized)

If `manifest.toml` already exists, the wizard warns you:

```text
⚠ Already initialized. Re-initialize? Existing manifest and snapshots will be backed up. [y/N]:
```

On confirmation (default: No), Stamp:

1. **Backs up** the existing manifest to `manifest.toml.<timestamp>.bak`
2. **Backs up** the existing snapshots directory to `snapshots.<timestamp>.bak/`
3. **Creates** a fresh empty manifest
4. **Takes** a new baseline snapshot of current system state

The backup is mandatory — it always runs before rewriting. The timestamp uses ISO 8601 format (`20260721T120000Z`) for easy sorting.

#### Skip prompt with `-y`

In non-interactive mode (`-y`), no prompt is shown. If already initialized, re-init proceeds with automatic backup.

### Step 4: System Diagnosis

Runs `stamp doctor` to verify the setup:

- Checks each package manager is available
- Verifies manifest integrity
- Reports UNIX compliance (NO_COLOR, man page, completions)

### Full output example (interactive)

```text
▪ Stamp Setup Wizard

Step 1 of 4: Shell Completions
  Install shell completions? [Y/n]: y
  ✅ completion installed to ~/.local/share/bash-completion/completions/stamp

Step 2 of 4: Man Pages
  Install man page? [Y/n]: y
  ✅ installed man page(s) to ~/.local/share/man

Step 3 of 4: Initialize
  Create manifest and baseline snapshot? [Y/n]: y
  ✅ manifest initialized and system baseline snapshot taken

Step 4 of 4: System Diagnosis
  ✅ 3 managers active
  ✅ Manifest healthy
  ✅ UNIX compliance verified
```

### Full output example (auto-accept)

```text
▪ Stamp Setup Wizard (auto-accept)

  Step 1: Shell Completions...  ✅
  Step 2: Man Pages...          ✅
  Step 3: Initialize...         ✅
  Step 4: System Diagnosis...   ✅

▪ Setup complete!
```
