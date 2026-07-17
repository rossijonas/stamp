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

**Core Commands:**
| Command | Aliases | Flags | Description |
| :--- | :--- | :--- | :--- |
| `stamp` | | | Prints welcome message suggesting `stamp hello` or `stamp --help`. |
| `stamp setup` | `hello` | | Runs first-time setup wizard: completions, man pages, init, doctor. |
| `stamp init` | | | Initializes `manifest.toml` and takes baseline snapshot. |
| `stamp install <pkg>` | `add` | `--manager, -m <name>`, `--note, -n <text>` | Installs natively and records intent. |
| `stamp remove <pkg>` | `uninstall`, `rm`, `delete`, `del` | `--manager, -m <name>` | Removes natively and untracks. |
| `stamp reinstall <pkg>` | | | Reinstalls natively and records intent. Works for both manifest-tracked and pre-existing packages. |
| `stamp search <query>` | | `--manager, -m <name>` | Searches across managers. |
| `stamp info <pkg>` | | `--manager, -m <name>` | Shows package information across managers, including raw outputs. |
| `stamp reconcile` | | `--dry-run, -d`, `--manager, -m <name>` | Detects drift since last snapshot and auto-tracks discovered packages and repositories. |
| `stamp restore` | | `--dry-run, -d`, `--manager, -m <name>` | Reinstalls repos and packages on a new machine. |
| `stamp update` | `upgrade` | `--manager, -m <name>` | Runs system upgrades across all managers in parallel. |
| `stamp list` | `ls` | `--json, -j`, `--manager, -m <name>` | Lists all intentionally installed packages. |
| `stamp doctor` | | `--json, -j`, `--manager, -m <name>` | Checks manager availability, manifest integrity, and UNIX compliance. |
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
| `stamp repo add <name> [url]` | `install` | `--manager, -m <name> (Required)` | Adds custom repository and records it. |
| `stamp repo remove <name>` | `uninstall`, `rm`, `delete`, `del` | `--manager, -m <name> (Required)` | Removes a repository and untracks it. |
| `stamp repo list` | `ls` | `--json, -j`, `--manager, -m <name>` | Lists all tracked repositories. |

---

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
  - Step 3: Initialize manifest and baseline snapshot (prompt, default Yes)
  - Step 4: System diagnosis (no prompt)
  - `-y` skips all prompts, runs everything
- **TTY Output Example (interactive):**
  ```text
  ▪ Stamp Setup Wizard

  Step 1 of 4: Shell Completions
    Install shell completions? [Y/n]:
  ```
- **TTY Output Example (auto-accept):**
  ```text
  ▪ Stamp Setup Wizard (auto-accept)

    Step 1: Shell Completions...  ✅
    Step 2: Man Pages...          ✅
    Step 3: Initialize...         ✅
    Step 4: System Diagnosis...   ✅

  ▪ Setup complete!
  ```

### `stamp init`
- **Usage:** Initializes `manifest.toml` and takes a baseline snapshot of current system packages.
- **Flags:** None.
- **Behavior:** Creates `~/.config/stamp` and `~/.local/share/stamp/snapshots` directories. Generates empty manifest.toml. Takes baseline snapshot for each available manager and saves them.
- **Output:** `manifest initialized and system baseline snapshot taken` to stderr.

### `stamp install <pkg>` (alias `add`)
- **Usage:** Installs a package natively and records it in the manifest.
- **Flags:** `--manager`, `-m`, `--note`, `-n`
- **Behavior:** Validates name, resolves manager, runs native install, appends package to manifest, saves manifest. For managers requiring root (e.g., DNF), write operations automatically wrap with `sudo` — TTY-aware, prompts for password when needed. On systems where `dnf` is unavailable, the adapter falls back to `yum` automatically.

### `stamp remove <pkg>` (aliases `uninstall`, `rm`, `delete`, `del`)
- **Usage:** Removes a package natively and untracks it.
- **Flags:** `--manager`, `-m`
- **Behavior:** Looks up recorded manager from manifest if not overridden by `-m`. Runs native remove, deletes package from manifest, saves manifest.

### `stamp reinstall <pkg>` (C4)
- **Usage:** Reinstalls a package natively and records it in the manifest. Works as the primary mechanism for tracking pre-existing packages that were installed before `stamp init`.
- **Flags:** None (accepts global `-y` flag).
- **Behavior:**
  1. Looks up `<pkg>` in the manifest.toml.
  2. **If found:** Resolves its recorded manager (e.g. `brew`). Calls `adapter.Install()` on the active manager.
  3. **If NOT found (pre-existing package):** Resolves manager via the 3-tier resolution engine. Runs native reinstall command (e.g. `dnf reinstall htop`). Falls back to native install if reinstall not supported. Appends package to manifest.
  4. Saves new system snapshots and saves manifest (updates `updated_at`).
- **Output:** `reinstalled htop via brew` to stderr.

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
  - If drift found (not `--dry-run`): adds all discovered packages to manifest, saves new snapshots.
  - No interactive prompt. Reconcile is fully deterministic:
    - `stamp reconcile` — auto-tracks, no questions.
    - `stamp reconcile --dry-run` — preview only, no tracking.
    - `stamp reconcile -y` — identical to `stamp reconcile` (kept for scripting consistency).
- **Design Rationale:** Reconcile is the safety net. There is no user decision to make: if a package was installed intentionally, it should be tracked. Users who want to inspect potential drift before committing use `--dry-run`. Pre-existing packages (installed before `stamp init`) are never detected — they are captured in the baseline snapshot. To track a pre-existing package, use `stamp reinstall <pkg>` instead.

### `stamp restore`
- **Usage:** Restores environment on a new machine from the manifest.
- **Flags:** `--dry-run`, `-d`, `--manager`, `-m` (Proposed)
- **Behavior:** Adds repositories sequentially in Phase 1, then installs packages concurrently in Phase 2.

### `stamp doctor`
- **Usage:** Checks manager availability, manifest health, UNIX compliance, and shell completion installation status.
- **Flags:** `--json`, `-j`, `--manager`, `-m` (Proposed)
- **Behavior:** Audits managers, parses manifest, checks `NO_COLOR`, `stamp man check` statuses, and shell completion installation status.
- **UNIX Compliance TTY section:**
  ```text
  UNIX Compliance:
  NO_COLOR: ✅ Set
  Man Page: ⚠️ Outdated (man v1.1.0, binary v1.2.3) — run 'stamp man install'
  Completions: ❌ Not installed — run 'stamp completion'
  ```
- **UNIX Compliance TTY section:**
  ```text
  UNIX Compliance:
    NO_COLOR: ✅ Set
    Man Page: ⚠️ Outdated (man v1.1.0, binary v1.2.3) — run 'stamp man install'
  ```

### `stamp self-update` (alias `self-upgrade`)
- **Usage:** Upgrades the stamp binary from the GitHub releases API.
- **Flags:** `--check`, `-c`

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
The TOML manifest supports `notes` for user context and a `repositories` block.
```toml
[[repositories]]
name = "flathub"
manager = "flatpak"
url = "https://dl.flathub.org/repo/flathub.flatpakrepo"

[[packages]]
name = "lazygit"
manager = "brew"
notes = "better git TUI than default"
```

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

## Edge Cases

### Reinstall Gap

**Scenario:** A package is removed and reinstalled between two `stamp reconcile` runs. Snapshot diffing sees no net change and reports no drift. This edge case only applies when the user **bypasses stamp and uses native package manager commands (dnf, brew, flatpak) directly**, then relies on reconcile as a safety net.

**Root Cause:** Snapshot diffing is a point-in-time comparison between two snapshots. If the removed package is reinstalled before the next reconcile, the baseline and current snapshots are identical. Stamp has no event monitoring — it cannot observe intermediate states.

**Mitigation:**
- **Always use stamp (recommended):** The edge case never occurs if packages are managed through stamp (`stamp install`/`stamp remove`). Stamp records every install and removal in the manifest instantly — no snapshot diffing involved.
- **Regular reconciliation:** If using native commands directly, remember to run `stamp reconcile` after each uninstall operation to keep snapshots in sync.
- **Automated timer:** `stamp auto-reconcile on` (planned) installs a daily systemd/launchd timer.
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
18. **Repo Remove:** `stamp repo remove myrepo -m brew` removes the repository and untracks it.
19. **Repo List:** `stamp repo list` prints all tracked repositories; `--json` outputs machine-readable.
20. **Setup:** `stamp setup` runs the setup wizard with completion, man pages, init, and doctor. `stamp hello` works as an alias.
21. **Completion:** `stamp completion bash|zsh|fish|powershell` generates valid shell completion scripts for each shell.
22. **Reconcile (Repo Drift):** If a new flatpak remote or brew tap is added externally, `stamp reconcile` detects and auto-tracks the repository alongside packages.
23. **Reconcile (Manager Scope):** `stamp reconcile -m dnf` scopes drift detection to a single manager only.
24. **Reinstall (Manager Flag):** `stamp reinstall htop -m brew` overrides manager resolution via the `--manager` flag for pre-existing packages.
25. **Reinstall (Adapters):** `adapter.Reinstall()` executes the native reinstall command for each manager (brew reinstall, dnf reinstall, flatpak install).
26. **Reconcile (Snapshot Save on No Drift):** If reconcile detects no drift, the current snapshot is saved to disk so future package removals are tracked correctly.
27. **Auto-Reconcile (Planned):** `stamp auto-reconcile on --period daily` installs a systemd or launchd timer to run `stamp reconcile` automatically at the configured interval.
