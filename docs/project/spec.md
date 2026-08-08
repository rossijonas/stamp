---
---


# Spec: Stamp (Intent Tracker)

## Objective

Build a lightweight yet powerful wrapper for native package managers. Stamp lets developers install, search, get info, and remove packages and repositories across multiple package managers through a single CLI — tracking every intentional choice into a portable, version-controllable TOML manifest. The primary workflow is using `stamp install` as a unified wrapper to guarantee total traceability from day one. It also acts as a passive safety net, allowing developers to track changes retroactively via local snapshot diffing (`stamp reconcile`) if they bypass the tool. It fully supports tracking custom repositories (taps, remotes) across all supported package managers.

## Tech Stack

- **Language:** Go 1.26+
- **CLI Framework:** `spf13/cobra` (industry standard for Go CLIs)
- **Manifest Parsing:** `pelletier/go-toml/v2`
- **Output/UI:** Standard `fmt` and `log` (keeping it simple for MVP)

## Command Blueprint

The complete surface area of the CLI, including aliases and flags.

**Global Flags:**

*   `--verbose`, `-v`: Enable debug/verbose logging.
*   `--json`, `-j`: Output results in machine-readable JSON format.
*   `--yes`, `-y`: Bypasses all interactive confirmation prompts (Auto-Accept).

### Flag Standardization Rules

1. Every flag SHOULD have a single-character short form (e.g. `--manager`, `-m`).
2. Actions MUST be subcommands, not flags. (e.g. `stamp man install`, not `stamp man --install`).
3. Boolean flags for enabling/disabling behavior are acceptable (e.g. `--dry-run`, `--json`).

### Confirmation & Consent Model

Destructive commands (`install`, `remove`, `reinstall`, `restore`, `update`, `autoremove`, `clean`, `hold`, `unhold`, `repo add/remove`) share one fail-closed gate:

1. `-y/--yes` skips refresh, preview, and prompt entirely.
2. Otherwise the command renders the adapter-owned transaction preview (`manager.Previewer` returns a typed `Preview{Output, Noop}`; combined stdout+stderr, never parsed by the CLI — see [ADR-016](../decisions/ADR-016-unified-preview-contract.md)), then prompts with a default of **no** (`[y/N]`).
3. A `Noop` preview (e.g. package already up to date, or remove of an absent package) fails fast with `nothing to do` — no prompt.
4. A preview that cannot be rendered warns (`⚠ could not render preview`) and still prompts; managers without a `Previewer` do the same.
5. **Non-interactive input without `-y` refuses with a non-zero exit** — a forgotten `-y` in a script or CI pipeline fails loud instead of silently doing nothing. An interactive decline (and a `Noop` preview) stops cleanly with exit 0.
6. After confirmation, the CLI marks the context with `manager.WithYes`; destructive adapter methods refuse to run without that marker (`ErrConfirmationRequired`). This is defense-in-depth at the privileged boundary.

`autoremove`/`clean --dry-run` are read-only and never require consent.

See [ADR-015](../decisions/ADR-015-fail-closed-consent.md) and [ADR-016](../decisions/ADR-016-unified-preview-contract.md) for the full design.

**Core Commands:**

| Command | Aliases | Flags | Description |
| :--- | :--- | :--- | :--- |
| `stamp` | | | Prints welcome message suggesting `stamp hello` or `stamp --help`. |
| `stamp setup` | `hello` | | Runs first-time setup wizard: completions, man pages, init, doctor. |
| `stamp init` | | | Initializes `manifest.toml` and takes baseline snapshot. |
| `stamp install <pkg>...` | `add` | `--manager, -m <name>`, `--note, -n <text>` | Installs natively and records intent. Multiple packages require `-m` (native batch support). |
| `stamp remove <pkg>...` | `uninstall`, `rm`, `delete`, `del` | `--manager, -m <name>` | Removes natively and untracks. Multiple packages require `-m` (native batch support). |
| `stamp reinstall <pkg>...` | | | Reinstalls natively and records intent. Works for both manifest-tracked and pre-existing packages. Multiple packages require `-m` (native batch support). |
| `stamp search <query>` | | `--manager, -m <name>` | Searches across managers. |
| `stamp info <pkg>` | | `--manager, -m <name>` | Shows package information across managers, including raw outputs. |
| `stamp reconcile` | | `--dry-run, -d`, `--manager, -m <name>` | Detects drift since last snapshot and auto-tracks discovered packages and repositories. Warns when tracked packages are no longer installed. |
| `stamp restore` | | `--dry-run, -d`, `--manager, -m <name>` | Reinstalls repos and packages on a new machine. |
| `stamp update` | `upgrade` | `--manager, -m <name>`, `--package, -p <pkg>`, `--serial, -s` | Runs system upgrades across all managers. Parallel by default. Use `-s` for serial, `-p` for single-package. |
| `stamp list` | `ls` | `--json, -j`, `--manager, -m <name>`, `--type, -t <type>` | Lists tracked packages and repos. Filter by entity type and origin (stamped/reconciled); `--type missing` lists manifest packages not installed. |
| `stamp manifest` | | `--json, -j` | Manifest management. Subcommands: `history` (list backups), `diff [ts\|hash]` (compare current with a backup). |
| `stamp doctor` | | `--json, -j`, `--manager, -m <name>` | Checks manager availability, manifest integrity, manifest-vs-system drift, and UNIX compliance. |
| `stamp self-update` | `self-upgrade` | `--check, -c` | Checks for and installs the latest version of `stamp`. |
| `stamp completion [shell]` | | `--stdout, -s` | Generates and installs shell completion scripts. Auto-detects shell if not specified. |
| `stamp man` | | | Command group for system reference page management. |
| `stamp auto-reconcile on\|off` | | `--period, -p hourly\|daily(default)\|weekly` | Installs or removes automated reconcile timer (systemd/launchd). |

**Man Subcommands:**

| Command | Flags | Description |
| :--- | :--- | :--- |
| `stamp man` | | Shows help for `stamp man` command group. Same as `stamp man help`. |
| `stamp man install` | `--prefix <path>` | Installs man page to system or user path. Default: `~/.local/share/man/man1/`. |
| `stamp man check` | | Verifies installed man page version matches stamp version. |

**Repository Commands:**

| Command | Aliases | Flags | Description |
| :--- | :--- | :--- | :--- |
| `stamp repo add <name> [url]` | `install` | `--manager, -m <name> (Required)` | Adds custom repository and records it. DNF: URL ending in `.repo` is fetched and installed verbatim, preserving gpg settings. A single URL argument derives the name from the URL (basename with `.repo` stripped, or host). |
| `stamp repo remove <name>` | `uninstall`, `rm`, `delete`, `del` | `--manager, -m <name>` | Removes a repository and untracks it. `-m` is optional when the repo is tracked in the manifest. |
| `stamp repo list` | `ls` | `--json, -j`, `--manager, -m <name>` | Lists all tracked repositories. |

---

### Supported Package Managers

#### System Managers

| Manager | OS | Notes |
| :--- | :--- | :--- |
| **DNF / YUM** | Fedora/RHEL/CentOS | DNF preferred, YUM fallback when DNF unavailable |
| **APT / apt-get** | Debian/Ubuntu | |
| **Paru** | Arch Linux | Preferred when both Paru and Pacman are installed |
| **Pacman** | Arch Linux | Fallback when Paru not installed |
| **Zypper** | openSUSE/SLES | |
| **Snap** | Linux (universal) | |
| **Flatpak** | Linux (sandboxed) | |
| **Brew** | macOS, Linux | User-space, no sudo for most operations |
| **MacPorts** | macOS | |

#### Language Toolchain Managers

| Manager | Scope | Notes |
| :--- | :--- | :--- |
| `go install` | Go binaries | Full module paths required |
| `pipx` | Python CLI tools | |
| `uv` | Python CLI tools | Faster alternative to pipx |

## Package Manager Resolution Engine

When a user runs a package or repository command (e.g., `stamp install htop`) without specifying `--manager`, the tool resolves ambiguity using a three-tier engine:

1. **Tier 1: Explicit Override:** If `--manager <name>` or `-m <name>` is provided, `stamp` directly executes that manager's command.

2. **Tier 2: User Preference (Declarative):** If the package exists in multiple managers, `stamp` checks the user's `config.toml` precedence list:
   ```toml
   precedence = ["dnf", "flatpak", "brew"]
   ```
   If a match is found, `stamp` automatically selects the manager with the highest configured precedence.

3. **Tier 3: Interactive Choice (Fallback):** If no precedence is defined (or there's a tie) and the process runs in an interactive terminal (TTY), `stamp` prompts the user to select the manager. In non-interactive environments (scripts/pipelines), the command fails with a clean error prompting the user to specify `--manager`.

---

## Configuration

The `stamp` configuration is stored securely at `~/.config/stamp/config.toml`. It allows users to define global precedence and regex-based routing rules.

### TOML Schema:

```toml
# ~/.config/stamp/config.toml

# The global order of preference when a package exists in multiple managers.
# Checked from left to right.
precedence = ["dnf", "flatpak", "brew"]

# Pattern-based rules override the global precedence list.
# Useful for routing specific patterns (like reverse-DNS or development libs).
[[rules]]
pattern = "^com\\..*|^org\\..*" # Matches reverse-DNS App IDs
prefer = "flatpak"

[[rules]]
pattern = "^lib.*|-devel$"     # Matches libraries and dev headers
prefer = "dnf"
```

### Precedence Matching Logic:

1.  **Rule Match:** The resolution engine iterates through the `[[rules]]` slice. If the package name matches a defined regular expression `pattern`, the engine immediately selects the associated `prefer` manager.
2.  **Global Precedence:** If no pattern rules match, the engine scans the global `precedence` array from left to right. The first manager in the list that reports the package as "available" is selected.
3.  **Tie-Breaker:** If the package is not found in the precedence list (or the list is empty), the engine falls back to prompting the user (in an interactive TTY) or failing cleanly (in scripts).

---

## Commands Specs

Detailed specifications, execution behaviors, and business rules for every subcommand.

### `stamp` (root)

- **Usage:** Suggests running `stamp hello` or `stamp --help` when executed with no arguments.
- **Output:** Help reference to stderr.

### `stamp setup` (alias `hello`) — Setup Wizard (C1)

- **Usage:** Interactive first-time setup wizard. Runs completion installation, man page setup, initialization, and diagnostics in sequence.
- **Flags:** Accepts global `-y` flag to skip all prompts.
- **Behavior:**
  - Step 1: Shell completions (prompt, default Yes)
  - Step 2: Man pages (prompt, default Yes)
  - Step 3: Initialize manifest and baseline snapshot
    - First-time: prompt "Create manifest and baseline snapshot? [Y/n]" (default Yes)
    - Already initialized: shows warning, prompt "Re-initialize (backup old configuration)? [y/N]" (default No)
  - Step 4: System diagnosis (no prompt)
  - `-y` skips all prompts, runs everything
- **TTY Output Example (interactive):**
  ```text
         █▄
        ▄██▄      ▄
   ▄██▀█ ██ ▄▀▀█▄ ███▄███▄ ████▄
   ▀███▄ ██ ▄█▀██ ██ ██ ██ ██ ██
  █▄▄██▀▄██▄▀█▄██▄██ ██ ▀█▄████▀
                           ██
                           ▀

  ▪ Stamp Setup Wizard

  Step 1 of 4: Shell Completions
    Install shell completions? [Y/n]:
  ```
- **TTY Output Example (auto-accept):**
  ```text
         █▄
        ▄██▄      ▄
   ▄██▀█ ██ ▄▀▀█▄ ███▄███▄ ████▄
   ▀███▄ ██ ▄█▀██ ██ ██ ██ ██ ██
  █▄▄██▀▄██▄▀█▄██▄██ ██ ▀█▄████▀
                           ██
                           ▀

  ▪ Stamp Setup Wizard (auto-accept)

    Step 1: Shell Completions...  ✓
    Step 2: Man Pages...          ✓
    Step 3: Initialize...         ✓
    Step 4: System Diagnosis...   ✓

  ▪ Setup complete!
  ```

### `stamp init`

- **Usage:** Initializes `manifest.toml` and takes a baseline snapshot of current system packages.
- **Flags:** Accepts global `-y` flag.
- **Behavior:** Creates `~/.config/stamp` and `~/.local/share/stamp/snapshots` directories. Writes a default `config.toml` (commented `[backup]` template) if absent — never overwrites an existing config, and failure is non-fatal. Generates empty manifest.toml. Takes baseline snapshot for each available manager and saves them.
- **Re-init guard:** If `manifest.toml` already exists (system is initialized), the user is prompted for confirmation (default No) before overwriting. On confirmation, **backup is mandatory** — the existing manifest and snapshots are always timestamp-backed up (`<path>.<YYYYMMDD>THHMMSSZ.bak`) before creating fresh state, and backup rotation then prunes old backups per the `[backup]` policy (both manifest and snapshot policies). The `-y` flag bypasses the prompt for scripting.
- **Backup is NOT optional:** When re-init is confirmed, backup always runs before rewriting.
- **Dry-run:** No config write, no backup, no rotation, no manifest rewrite.
- **Output:** `manifest initialized and system baseline snapshot taken` to stderr.
- **Re-init messages:** `existing manifest backed up to <path>`, `existing snapshots backed up to <path>`, `re-init aborted` to stderr.

### `stamp install <pkg>` (alias `add`)

- **Usage:** Installs a package natively and records it in the manifest.
- **Flags:** `--manager`, `-m`, `--note`, `-n`
- **Behavior:** Validates name, resolves manager, refreshes metadata and shows a native dry-run preview, prompts for confirmation (`Install <pkg> via <mgr>? [y/N]`, default No, `-y` to skip), then runs native install, appends package to manifest, saves manifest. For managers requiring root (e.g., DNF), write operations automatically wrap with `sudo` — TTY-aware, prompts for password when needed. On systems where `dnf` is unavailable, the adapter falls back to `yum` automatically.
- **Multiple packages (issue #185):** `stamp install <pkg1> <pkg2> ... -m <mgr>` installs a whole batch in one native invocation. **Batch constraint:** multi-package batches are **per-manager only** — `-m <manager>` is mandatory and a single batch never spans managers; to operate on packages across managers, run one command per manager. Only managers with native multi-package support participate (`go`, `pipx`, `uv` reject with a capability error; brew falls back to per-package installs when a batch mixes casks and formulae). Confirmation is a single combined prompt (`Install N package(s) via <mgr>? [y/N]`); the manifest is saved once; output is `installed N package(s) via <mgr>`.

### `stamp remove <pkg>` (aliases `uninstall`, `rm`, `delete`, `del`)

- **Usage:** Removes a package natively and untracks it.
- **Flags:** `--manager`, `-m`
- **Behavior:** Looks up recorded manager from manifest if not overridden by `-m`. Prompts for confirmation (`Remove <pkg> via <mgr>? [y/N]`, `-y` to skip). Runs native remove, deletes package from manifest, saves manifest.
- **Multiple packages (issue #185):** `stamp remove <pkg1> <pkg2> ... -m <mgr>` removes a whole batch in one native invocation. **Batch constraint:** multi-package batches are **per-manager only** — `-m <manager>` is mandatory and a single batch never spans managers; to operate on packages across managers, run one command per manager. Only managers with native multi-package support participate. Single combined prompt; one manifest save; output `removed N package(s) via <mgr>`.

### `stamp reinstall <pkg>` (C4)

- **Usage:** Reinstalls a package natively and records it in the manifest. Works as the primary mechanism for tracking pre-existing packages that were installed before `stamp init`.
- **Flags:** None (accepts global `-y` flag).
- **Behavior:**
  1. Looks up `<pkg>` in the manifest.toml.
  2. **If found:** Resolves its recorded manager (e.g. `brew`). Calls `adapter.Install()` on the active manager.
  3. **If NOT found (pre-existing package):** Resolves manager via the 3-tier resolution engine. Runs native reinstall command (e.g. `dnf reinstall htop`). Falls back to native install if reinstall not supported. Appends package to manifest.
  4. Before executing, prompts for confirmation (`Reinstall <pkg> via <mgr>? [y/N]`, default No, `-y` to skip).
  5. Saves new system snapshots and saves manifest (updates `updated_at`).
- **Output:** `reinstalled htop via brew` to stderr.
- **Multiple packages (issue #185):** `stamp reinstall <pkg1> <pkg2> ... -m <mgr>` reinstalls a whole batch in one native invocation. **Batch constraint:** multi-package batches are **per-manager only** — `-m <manager>` is mandatory and a single batch never spans managers; to operate on packages across managers, run one command per manager. A package already tracked under a different manager fails fast (`package X is tracked under <mgr>, not <other>; reinstall it with -m <mgr>`) before anything runs. Only managers with native multi-package reinstall support participate (`snap` is excluded — reinstall is remove + install there). Single combined prompt; snapshot + manifest saved once; output `reinstalled N package(s) via <mgr>`.

### `stamp search <query>`

- **Usage:** Searches for matching packages across all available managers.
- **Flags:** `--manager`, `-m`
- **Behavior:** Queries all adapters or the scoped manager and prints matching packages.

### `stamp info <pkg>` (C2)

- **Usage:** Queries detailed package information.
- **Flags:** `--manager`, `-m`
- **Behavior:**
  - **No `-m`:** Queries all managers, prints a summary table of matching versions.
  - **With `-m`:** Displays the raw info block from the specific package manager (e.g., `dnf info htop`, `brew info htop`).
- **Raw TTY Output Example:**
  ```text
  $ stamp info htop -m dnf
  htop via dnf:

  Name           : htop
  Version        : 3.4.1
  Release        : 3.fc44
  Architecture   : x86_64
  Download size  : 203.6 KiB
  Installed size : 464.3 KiB
  Summary        : Interactive process viewer
  URL            : https://htop.dev/
  License        : GPL-2.0-or-later
  Description    : htop is an interactive text-mode process viewer...
  ```

### `stamp reconcile`

- **Usage:** Detects drift between the system state and the last snapshot, and auto-tracks discovered packages and repositories into the manifest.
- **Flags:** `--manager`, `-m`, `--dry-run`, `-d`
- **Behavior:**
  - Fetches current state (packages and repositories) from all adapters.
  - Diffs against the last snapshot.
  - If no drift: exits with "No drift detected".
  - If drift found AND `--dry-run`: shows all discovered packages/repos and exits without tracking.
  - If drift found (not `--dry-run`): backs up the current manifest (copy-based, original kept), adds all discovered packages to manifest with `origin = "reconciled"`, saves new snapshots, then rotates manifest backups per the `[backup]` policy. Snapshot rotation is NOT triggered by reconcile — it lives in `stamp init` re-init.
  - Missing-package warning (issue #182): manifest entries that disappeared from the system since the last snapshot (e.g. removed via `dnf remove`) are reported to stderr as a warning — `N manifest package(s) not installed: ... — run 'stamp ls --type missing' for the full list, or 'stamp restore' to reinstall`. The warning fires on both the no-drift and drift paths, is warning-only (no manifest mutation, no reinstall), and the removal is still recorded in the new snapshot (the snapshot reflects reality; the manifest holds intent). Group and cask entries are excluded.
  - No interactive prompt. Reconcile is fully deterministic:
    - `stamp reconcile` — auto-tracks, no questions.
    - `stamp reconcile --dry-run` — preview only, no tracking, no backups, no rotation.
    - `stamp reconcile -y` — identical to `stamp reconcile` (kept for scripting consistency).
- **Design Rationale:** Reconcile is the safety net. There is no user decision to make: if a package was installed intentionally, it should be tracked. Users who want to inspect potential drift before committing use `--dry-run`. Pre-existing packages (installed before `stamp init`) are never detected — they are captured in the baseline snapshot. To track a pre-existing package, use `stamp reinstall <pkg>` instead.

### `stamp restore`

- **Usage:** Restores environment on a new machine from the manifest.
- **Flags:** `--dry-run`, `-d`, `--manager`, `-m` (Proposed)
- **Behavior:** Adds repositories sequentially in Phase 1, then installs packages concurrently in Phase 2. Prompts for confirmation before running (`Restore tracked repositories and packages? [y/N]`, default No, `-y` to skip); `--dry-run` previews without prompting.

### `stamp doctor`

- **Usage:** Checks manager availability, manifest health, UNIX compliance, and shell completion installation status.
- **Flags:** `--json`, `-j`, `--manager`, `-m` (Proposed)
- **Behavior:** Audits managers, parses manifest, checks `NO_COLOR`, `stamp man check` statuses, and shell completion installation status.
- **Manifest vs system check (issue #182):** when the manifest is valid, doctor queries each active manager's installed packages (concurrently, best-effort — a manager whose query fails is skipped) and reports manifest entries absent from the system under a `Missing:` section in "Manifest Integrity" (TTY) and as `manifest.missing` (`--json`). This does not flip the manifest status — `✓ Healthy` refers to manifest integrity, and a missing package is drift, not corruption. Group and cask entries are excluded.
- **UNIX Compliance TTY section:**
  ```text
  UNIX Compliance:
  NO_COLOR: ✓ Set
  Man Page: ⚠ Outdated (man v1.1.0, binary v1.2.3) — run 'stamp man install'
  Completions: ✗ Not installed — run 'stamp completion'
  ```
- **UNIX Compliance TTY section:**
  ```text
  UNIX Compliance:
    NO_COLOR: ✓ Set
    Man Page: ⚠ Outdated (man v1.1.0, binary v1.2.3) — run 'stamp man install'
  ```

### `stamp update` (alias `upgrade`)

- **Usage:** Runs system upgrades across all available package managers using a safe two-phase (check + confirm) flow.
- **Flags:** `--manager, -m`, `--package, -p`, `--serial, -s`, `--check, -c`
- **Behavior:**
  1. **Check Phase:**
     a. If `-y` is set: the check phase is **skipped** entirely for performance.
     b. Otherwise: runs a serialized, non-mutating check across all target package managers by calling `CheckUpdate`.
     c. If an adapter natively supports update-checking (e.g. `dnf`, `apt`, `brew`, `flatpak`), it queries the native package manager and lists the available version deltas.
     d. If an adapter does not support update-checking (e.g. `pipx`, `uv`, `go`), it returns an unsupported error. The CLI catches this and prints a short info notice (e.g. `pipx: cannot preview updates`) instead of aborting.
     e. If `--check` / `-c` is set: the command displays the check results/notices and exits 0 immediately without performing any upgrades.
  2. **Confirm Phase:**
     a. If `-y` is set: the confirmation prompt is skipped.
     b. Otherwise: the CLI displays the aggregated list of available updates/warnings and prompts the user to confirm. The prompt defaults to **no** (`[y/N]`); non-interactive input without `-y` aborts (fail closed). If rejected, exits 0.
  3. **Run Phase:**
     a. If `-m` is set: runs that single manager's native upgrade command.
     b. If `-p <pkg>` is set (requires `-m`): updates only the specified package instead of all packages.
     c. If `--serial` / `-s` is set: runs upgrades one manager at a time (sequential).
     d. Otherwise: runs upgrades concurrently using `sync.WaitGroup` for speed.
  4. Each manager streams its native output to stderr during execution.
  5. Errors from one manager do NOT block others. If any manager fails, the command exits non-zero.
  6. No manifest or snapshot interaction — `update` only touches the system.
- **Output:** `updated packages via <manager>` or `updated <pkg> via <manager>` to stderr per manager. `⚠ update failed for <manager>: <error>` on failure.
- **Exit code:** 0 if all managers succeed, 1 if any manager fails.

### `stamp self-update` (alias `self-upgrade`)

- **Usage:** Upgrades the stamp binary from the GitHub releases API.
- **Flags:** `--check`, `-c`
- **Behavior:**
  1. Fetches the latest release metadata from `https://api.github.com/repos/rossijonas/stamp/releases/latest`.
  2. If `--check`: prints current vs latest version. Exits 0 if up to date, 1 if update available.
  3. If a newer version is available:
     a. Downloads the tarball + `checksums.txt` via HTTPS with 30s timeout.
     b. Verifies SHA-256 checksum of the downloaded tarball against `checksums.txt`.
     c. Extracts the binary from the tarball (tar slip protection via `filepath.Base` sanitization).
     d. Checks write permission on the install directory before writing — prompts for `sudo` if needed.
     e. Atomically replaces the current binary via `os.Rename` (same-filesystem temp file).
     f. Preserves original binary permissions (execute bits, group/world).
     g. Re-installs shell completions (auto-detected shell).
     h. Re-installs man pages.
- **Output:** `Updated to vX.Y.Z` to stderr on success. `integrity check failed` on checksum mismatch.
- **Exit code:** 0 if already up to date or update succeeded, 1 if check found update or error.

### `stamp completion [shell]`

- **Usage:** Generates and installs completion scripts. Auto-detects the current shell if not specified. Uses `--stdout` / `-s` to print the script to stdout without installing.
- **Flags:** `--stdout`, `-s`
- **Behavior:** Without args, detects shell via `$SHELL` and installs to the correct path:
  - Bash: `~/.local/share/bash-completion/completions/stamp`
  - Zsh: `~/.local/share/zsh/site-functions/_stamp` or `~/.zfunc/_stamp`
  - Fish: `~/.config/fish/completions/stamp.fish`
  - PowerShell: not auto-installable, falls back to `--stdout`
- **Output:** `completion installed to /path` to stderr on success.

### `stamp man`

- **Usage:** Displays help output for man page subcommands.
- **Subcommands:** `install` (install man pages to path), `check` (verify man page version vs binary version).

### `stamp repo`

- **Usage:** Command group managing custom package repositories.
- **Subcommands:** `add` (install repo), `remove` (untrack repo), `list` (ls tracked repos).

---

## Data Model

The TOML manifest supports `notes` for user context, an `origin` field for provenance, and a `repositories` block.

```toml
[[repositories]]
name = "flathub"
manager = "flatpak"
url = "https://dl.flathub.org/repo/flathub.flatpakrepo"
origin = "stamped"

[[packages]]
name = "lazygit"
manager = "brew"
notes = "better git TUI than default"
origin = "stamped"
```

### Origin Provenance

Every `[[packages]]` and `[[repositories]]` entry may carry an `origin` field
recording how it entered the manifest. It is optional (`omitempty`) — an
absent field is treated as `stamped`, so pre-existing manifests load without
migration.

| Value | Meaning |
|-------|---------|
| `stamped` | Recorded by a direct user action (`stamp install`, `stamp repo add`, `stamp reinstall`) |
| `reconciled` | Auto-tracked by `stamp reconcile` after drift detection |

The `origin` field powers `stamp list --type` (issue #178) and `stamp manifest
history` / `stamp manifest diff` (issue #179).

### List Command (`stamp list`)

`stamp list` (alias `ls`) lists tracked packages by default. The `--type, -t`
flag filters by entity type and origin. All flags compose with AND logic
(`--type` × `--manager` × `--json`).

Valid `--type` values:

| Value | Description |
|-------|-------------|
| `packages` | All packages (default, backward compatible) |
| `repos` | All repositories |
| `stamped` | All entries installed via stamp (packages + repos) |
| `reconciled` | All entries discovered by reconcile (packages + repos) |
| `stamped-packages` | Packages with `origin = "stamped"` |
| `stamped-repos` | Repos with `origin = "stamped"` |
| `reconciled-packages` | Packages with `origin = "reconciled"` |
| `reconciled-repos` | Repos with `origin = "reconciled"` |
| `missing` | Manifest packages not installed on this system |

`--type packages` and `--type repos` ignore origin and show all entries of that
entity type. An unknown value returns `unknown type "<value>"; valid types:
packages, repos, stamped, reconciled, stamped-packages, stamped-repos,
reconciled-packages, reconciled-repos, missing`. On a pre-origin manifest (entries
without an `origin` field) the origin defaults to `stamped`, so
`--type stamped-packages` shows everything and `--type reconciled-packages`
shows nothing.

`--type missing` (issue #182) is the only system-aware view: it queries each
active manager's installed packages and lists manifest entries absent from the
system, i.e. packages removed via the native manager. It composes with
`--manager, -m` and `--json`, but not with an origin filter. Group and cask
entries are excluded (they never appear in the installed list). A manager whose
installed list cannot be queried is skipped. No matches prints
`no missing packages`. It never mutates the manifest — `stamp restore` is the
convergence tool.

### Manifest Management (`stamp manifest`)

`stamp manifest history` lists the current manifest and every timestamped
backup (`manifest.toml.<TS>.bak`), newest first, with package/repo counts and a
short SHA-256 content-hash prefix. `*` marks the current entry; backups whose
content equals the current manifest are marked unchanged. Corrupted backups are
skipped with a warning. No backups yet prints a hint pointing at re-init and
reconcile.

`stamp manifest diff [ts|hash]` compares the current manifest against a backup
(default: most recent), for both packages and repositories using `name+manager`
as the identity key. The argument is either a timestamp (`2026-08-02T09:15:00Z`
or `20260802T091500Z`) or a content-hash prefix (pure hex, ≥ 6 chars) from
`history`. Added entries render with `+`, removed with `-`. `--manager, -m` and
`--origin` (stamped/reconciled) filter both sets after diffing. An unknown or
ambiguous reference errors with `no backup found for <arg>` / `ambiguous hash`.
Diffing against a corrupted backup errors with `failed to parse backup at
<path>`. If no backup exists, `diff` errors with `no backup to compare against`.

### Exit Codes

`stamp` follows BSD `sysexits.h` conventions (shipped by glibc on Linux) for
error categories. Success is always `0`. Unclassified failures exit `1`, the
POSIX catchall. Scripts can distinguish failure modes by code:

| Code | Constant | Category | Examples |
|------|----------|----------|----------|
| `64` | EX_USAGE | Bad command line / flag / argument | invalid `--type`, `--origin`, diff timestamp, repo name/URL; flag-parse errors |
| `65` | EX_DATAERR | Input data incorrect | corrupt manifest, corrupt backup, ambiguous hash |
| `66` | EX_NOINPUT | Referenced input absent | `diff` with no backup, no matching timestamp/hash |
| `69` | EX_UNAVAILABLE | Required resource absent | no package manager available, `-m` manager not installed |
| `73` | EX_CANTCREAT | Cannot create output | manifest/snapshot save or backup failure |
| `78` | EX_CONFIG | Unconfigured/misconfigured | invalid `config.toml`, `manifest not found; run stamp init first` |

Backup/rotation failures on reconcile and init are non-fatal (warning to
stderr, exit `0`).

The mapping and its rationale are recorded in
[ADR-018](../decisions/ADR-018-sysexits-exit-codes.md) (supersedes the
exit-code note in ADR-002).

### Backup Retention Policy

Stamp keeps timestamped backups before rewriting the manifest or snapshots.
Backup naming is lexicographically sortable: `manifest.toml.<YYYYMMDDTHHMMSSZ>.bak`
(files) and `snapshots.<YYYYMMDDTHHMMSSZ>.bak/` (directories).

Retention is controlled by the `[backup]` section of `config.toml` and mirrors
logrotate's `rotate`, `minage`, and `maxage` directives:

| Config key | Default | Logrotate equivalent | Meaning |
|------------|---------|----------------------|---------|
| `max_manifest_backups` | `10` | `rotate` | Max manifest backups to keep (count cap) |
| `min_manifest_backups` | `3` | — | Always keep at least this many manifest backups (count floor) |
| `min_manifest_backup_age_days` | `7` | `minage` | Backups younger than this are never deleted (floor) |
| `max_manifest_backup_age_days` | `30` | `maxage` | Backups older than this are always deleted (ceiling) |
| `max_snapshot_backups` | `10` | `rotate` | Max snapshot backup dirs to keep |
| `min_snapshot_backups` | `3` | — | Always keep at least this many snapshot backups (count floor) |
| `min_snapshot_backup_age_days` | `7` | `minage` | Snapshot backups younger than this are never deleted |
| `max_snapshot_backup_age_days` | `30` | `maxage` | Snapshot backups older than this are always deleted |

A value of `0` on any axis means **unlimited** on that axis. The manifest and
snapshot policies are independent; `stamp reconcile` only rotates manifest
backups, while `stamp init` re-init rotates both manifest and snapshot backups.

**Precedence (highest to lowest):**

1. **Min-age floor** — backups younger than `min_*_backup_age_days` are
   **protected**: never deleted, even when the count cap is exceeded.
2. **Min-count floor** — at least `min_*_backups` backups are always kept.
   The newest backups survive, so the max-age ceiling can never wipe the
   backup set to zero.
3. **Max-age ceiling** — among eligible backups (age ≥ min-age), any backup
   older than `max_*_backup_age_days` is deleted, except those needed to meet
   the min-count floor (count does not protect ancient backups beyond the floor).
4. **Count cap** — if the eligible set still exceeds `max_*_backups`, the
   oldest surplus backups are deleted, except those needed to meet the
   min-count floor.

**Worked example (ceiling vs min-count):** `min_manifest_backups=3`,
`max_manifest_backup_age_days=30`, 5 backups all older than 30 days → the
ceiling wants to delete all 5, but the min-count floor keeps the newest 3,
so only 2 are deleted.

**Misconfiguration:** if `min_*_backup_age_days > max_*_backup_age_days`, the
floor wins on the overlapping window (protective), but the configuration is
reported as invalid in `stamp doctor` and the docs warn against it. If
`min_*_backups > max_*_backups`, the min-count floor wins (protective).

## Project Structure

```text
stamp/
├── cmd/stamp/         → Main application entrypoint
├── internal/
│   ├── cli/           → Cobra commands (init, reconcile, restore, install, etc.)
│   ├── manager/       → Package manager adapters (dnf/yum, brew, flatpak)
│   ├── state/         → Local JSON snapshotting and delta calculation
│   ├── manifest/      → TOML parsing and writing
│   └── config/        → XDG path resolution and user config
├── tools/docgen/      → Build-time doc generation tool
├── docs/              → ADRs, specifications, generated docs
└── README.md
```

## Code Style

Idiomatic Go with strict error wrapping and interface-driven design for testability.

## Testing Strategy

- **Framework:** standard `testing` package + `stretchr/testify` for assertions.
- **Test Locations:** Co-located with source (`state_test.go` next to `state.go`).
- **Core Coverage:** 100% on `internal/state/` and `internal/manifest/`.
- **Mocks:** Mock the `PackageManager` interface.
- **Minimum:** 90% overall project coverage.

## Go Adapter

The Go adapter supports `go install <path>@latest` for installing end-user CLI tools. It
follows the `pipx` model — installing user-facing tools, not project dependencies.

### Package Names
- Requires full module paths (e.g., `github.com/golangci/golangci-lint`).
- Short names (e.g., `golangci-lint`) are rejected.
- Module paths are validated via `ValidateModulePath` — blocked: empty, missing `/`,
  or containing shell metacharacters.

### ListInstalled (Name-space Contract)
- Returns module paths when recoverable from binary metadata via `go version -m <bin>`.
- Falls back to the binary name for stripped or old binaries.
- Module paths are stored in the manifest and snapshot — enabling round-trip remove/info.

### Search / Doctor / Repos
- Search and Doctor return errors (`not supported`). Search errors are printed as warnings
  to stderr; valid results from other managers remain on stdout.
- `ListRepos` returns an empty list (no error) — adapters without repo support signal
  "no repos available" rather than failing the snapshot pipeline.
- `AddRepo`/`RemoveRepo` return errors (`not supported`) — these are user-facing write ops.

### Update
- Single package (`-p <module> -m go`): runs `go install <module>@latest`.
- Batch (`stamp update`): lists installed binaries, recovers module paths, reinstalls
  each one with `@latest`. Binaries without recoverable module paths are skipped.

### GOBIN / GOPATH Resolution
- Checks `go env GOBIN` first (returns the bin directory directly).
- Falls back to `go env GOPATH` (uses the first entry of the colon-separated list
  joined with `/bin`).
- Falls back to `$HOME/go/bin`.

## Pipx Adapter

The Pipx adapter supports `pipx install` for CLI tools from the Python ecosystem.

### Package Names
- Uses standard `ValidatePackageName` (simple names like `black`, `httpie`).
- Names starting with `-` are rejected. Shell metacharacters rejected.

### ListInstalled
- Tries `pipx list --json` first (parses the `venvs` JSON object for package names).
- Falls back to `pipx list` text parsing for older pipx installations.

### Search / Doctor / Repos
- Search and Doctor return errors. Repo operations: `AddRepo`/`RemoveRepo` error; `ListRepos` returns empty list.

### Update
- Single: `pipx upgrade <pkg>`. Batch: `pipx upgrade-all`.

## Uv Adapter

The Uv adapter supports `uv tool install` for CLI tools from the Python ecosystem.

### Package Names
- Uses standard `ValidatePackageName`. Simple names like `black`, `ruff`.

### ListInstalled
- Parses `uv tool list` output line-by-line. Package names are the first token on each non-indented line.

### Search / Doctor / Repos
- All return errors or empty (same pattern as pipx).

### Update
- Single: `uv tool upgrade <pkg>`. Batch: `uv tool upgrade --all`.

## Boundaries

- **Always:** Use `context.Context` for all shell executions (`os/exec`).
- **Always:** Return meaningful delta states (added, removed, unchanged).
- **Always:** Every flag MUST have a single-character short form.
- **Always:** Actions MUST be subcommands, not flags.
- **Always:** Snapshot diffing is the default mechanism for drift detection.
- **Always:** Packages and repositories installed before `stamp init` are never tracked or detected by `stamp reconcile`. They are captured in the baseline snapshot.
- **Always:** To track a pre-existing package, use `stamp reinstall <pkg>`.
- **Ask first:** Before adding any third-party dependencies beyond `cobra` and `go-toml`.
- **Ask first:** Before changing the structure of the `manifest.toml`.
- **Never:** Mutate the actual system state (run native installs) during a `reconcile` or `list` command.
- **Never:** Use flags to represent actions (e.g. `--install`). Use subcommands instead.
- **Never:** Present interactive prompts during `stamp reconcile`. The command is fully deterministic.
- **Never:** Execute destructive adapter operations without explicit consent (`manager.WithYes`). The CLI gate is the only source of consent.

## Edge Cases

### Reinstall Gap

**Scenario:** A package is removed and reinstalled between two `stamp reconcile` runs. Snapshot diffing sees no net change and reports no drift. This edge case only applies when the user **bypasses stamp and uses native package manager commands (dnf, brew, flatpak) directly**, then relies on reconcile as a safety net.

**Root Cause:** Snapshot diffing is a point-in-time comparison between two snapshots. If the removed package is reinstalled before the next reconcile, the baseline and current snapshots are identical. Stamp has no event monitoring — it cannot observe intermediate states.

**Mitigation:**

- **Always use stamp (recommended):** The edge case never occurs if packages are managed through stamp (`stamp install`/`stamp remove`). Stamp records every install and removal in the manifest instantly — no snapshot diffing involved.
- **Regular reconciliation:** If using native commands directly, remember to run `stamp reconcile` after each uninstall operation to keep snapshots in sync.
- **Automated timer:** `stamp auto-reconcile on` installs a daily systemd/launchd timer.
- **Manual timer files:** Pre-configured service/timer files available in `contrib/`.

## UNIX Compliance & Documentation Strategy

To be a "good UNIX citizen", `stamp` must adhere to:
- **POSIX Syntax:** Handled natively by `spf13/cobra`.
- **XDG Base Directory:** Config in `~/.config/stamp`, state in `~/.local/share/stamp`.
- **Exit Codes:** `0` for success, `>0` for failures (e.g., standard `sysexits`).
- **I/O Separation:** Informational output/UI to `stdout`, errors to `stderr`.
- **NO_COLOR:** Respect the `NO_COLOR=1` environment variable.
- **Auto-Generated Docs:** Usage documentation for GitHub Pages must be auto-generated from the codebase using `github.com/spf13/cobra/doc` to ensure docs and code never drift.
- **UNIX Man Pages:** System reference pages (Section 1) must be self-contained via `stamp man` so users can run `man stamp` locally.
- **Project Landing Page:** A custom landing page at `docs/index.html` served via GitHub Pages (`/docs` folder on main branch, `https://rossijonas.github.io/stamp/`).

## Success Criteria

1. **Init:** Running `stamp init` creates the correct XDG directories and an empty `manifest.toml`, and takes baseline snapshots for each available manager.
2. **Reconcile (No Drift):** If system state matches the last snapshot, `reconcile` exits cleanly with `"No drift detected"`.
3. **Reconcile (Drift):** If `flatpak install com.spotify.Client` is run externally, `stamp reconcile` detects this one new package and auto-tracks it to `manifest.toml` without prompting.
4. **Reconcile (Dry Run):** `stamp reconcile --dry-run` shows all discovered drift but does NOT save manifest or snapshots.
5. **Reconcile (Pre-existing):** Packages installed before `stamp init` are never detected by `stamp reconcile`. To track them, use `stamp reinstall <pkg>`.
6. **Reinstall (Manifest-tracked):** `stamp reinstall htop` reinstalls a manifest-tracked package using its recorded manager.
7. **Reinstall (Pre-existing):** `stamp reinstall htop` installs a pre-existing package not in the manifest, resolves its manager, runs native reinstall, and records it in the manifest.
8. **Restore:** Running `stamp restore` successfully adds repositories *before* executing the respective package manager install commands concurrently.
9. **Notes:** A user can pass `--note "reason"` to `stamp install` or `stamp edit`, which will be correctly saved in the TOML manifest.
10. **Doctor:** `stamp doctor` reports manager status, manifest health, and UNIX compliance in both TTY and JSON.
11. **Man Pages:** `stamp man` displays help; `stamp man install` installs man pages; `stamp man check` verifies version matches binary.
12. **Completions:** `stamp completion bash|zsh|fish|powershell` generates valid shell completion scripts.
13. **Info:** `stamp info htop -m dnf` prints raw dnf info metadata directly.
14. **Install:** `stamp install htop` installs the package natively via the resolved manager and records it in `manifest.toml`.
15. **Remove:** `stamp remove htop` removes the package natively and removes it from the manifest.
16. **Search:** `stamp search ripgrep` returns matching packages from all available managers.
17. **Repo Add:** `stamp repo add myrepo -m brew` adds the repository via the specified manager and records it.
18. **Repo Remove:** `stamp repo remove myrepo` removes the repository and untracks it. When tracked, the manager is looked up from the manifest; `-m` overrides it. For DNF, URL-added repos are removed by deleting their `.repo` file and COPR repos via `dnf copr disable`.
19. **Repo List:** `stamp repo list` prints all tracked repositories; `--json` outputs machine-readable.
20. **Setup:** `stamp setup` runs the setup wizard with completion, man pages, init, and doctor. `stamp hello` works as an alias.
21. **Completion:** `stamp completion bash|zsh|fish|powershell` generates valid shell completion scripts for each shell.
22. **Reconcile (Repo Drift):** If a new flatpak remote or brew tap is added externally, `stamp reconcile` detects and auto-tracks the repository alongside packages.
23. **Reconcile (Manager Scope):** `stamp reconcile -m dnf` scopes drift detection to a single manager only.
24. **Reinstall (Manager Flag):** `stamp reinstall htop -m brew` overrides manager resolution via the `--manager` flag for pre-existing packages.
25. **Reinstall (Adapters):** `adapter.Reinstall()` executes the native reinstall command for each manager (brew reinstall, dnf reinstall, flatpak install).
26. **Reconcile (Snapshot Save on No Drift):** If reconcile detects no drift, the current snapshot is saved to disk so future package removals are tracked correctly.
27. **Update:** `stamp update` runs native upgrade commands for all available managers concurrently. Errors from one manager don't block others. `--manager` flag scopes to a single manager. Non-zero exit if any manager fails.
27. **Auto-Reconcile:** `stamp auto-reconcile on --period daily` installs a systemd or launchd timer to run `stamp reconcile` automatically at the configured interval.
