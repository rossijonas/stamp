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
| `stamp hello` | | | Prints ASCII logo, project about, and suggests next steps. |
| `stamp init` | | | Initializes `manifest.toml` and takes baseline snapshot. |
| `stamp install <pkg>` | `add` | `--manager, -m <name>`, `--note, -n <text>` | Installs natively and records intent. |
| `stamp remove <pkg>` | `uninstall`, `rm`, `delete`, `del` | `--manager, -m <name>` | Removes natively and untracks. |
| `stamp reinstall <pkg>` | | | Reinstalls a package currently tracked in the manifest using its recorded manager. |
| `stamp search <query>` | | `--manager, -m <name>` | Searches across managers. |
| `stamp info <pkg>` | | `--manager, -m <name>` | Shows package information across managers, including raw outputs. |
| `stamp reconcile` | | `--manager, -m <name>` | Detects drift, prompts user, records intent. |
| `stamp restore` | | `--dry-run, -d`, `--manager, -m <name>` | Reinstalls repos and packages on a new machine. |
| `stamp update` | `upgrade` | `--manager, -m <name>` | Runs system upgrades across all managers in parallel. |
| `stamp list` | `ls` | `--json, -j`, `--manager, -m <name>` | Lists all intentionally installed packages. |
| `stamp doctor` | | `--json, -j`, `--manager, -m <name>` | Checks manager availability, manifest integrity, and UNIX compliance. |
| `stamp self-update` | `self-upgrade` | `--check, -c` | Checks for and installs the latest version of `stamp`. |
| `stamp completion <shell>` | | | Generates shell completion scripts (bash, zsh, fish, powershell). |
| `stamp man` | | | Command group for system reference page management. |

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

### `stamp hello` — Welcome Command (C1)
- **Usage:** Prints welcome message containing ASCII logo, brief "about", and suggested next steps (init, doctor, man install).
- **Flags:** None.
- **TTY Output Example:**
  ```text
    stamp — A lightweight yet powerful wrapper for your native package managers.

    For a fresh installation, try:
      stamp init          — Create manifest and take initial snapshot
      stamp doctor        — Verify system configuration
      stamp man install   — Install offline documentation

    Need help? Run:  stamp --help
  ```

### `stamp init`
- **Usage:** Initializes `manifest.toml` and takes a baseline snapshot of current system packages.
- **Flags:** None.
- **Behavior:** Creates `~/.config/stamp` and `~/.local/share/stamp/snapshots` directories. Generates empty manifest.toml. Takes baseline snapshot for each available manager and saves them.
- **Output:** `manifest initialized and system baseline snapshot taken` to stderr.

### `stamp install <pkg>` (alias `add`)
- **Usage:** Installs a package natively and records it in the manifest.
- **Flags:** `--manager`, `-m`, `--note`, `-n`
- **Behavior:** Validates name, resolves manager, runs native install, appends package to manifest, saves manifest. For managers requiring root (e.g., DNF), write operations automatically wrap with `sudo` — TTY-aware, prompts for password when needed.

### `stamp remove <pkg>` (aliases `uninstall`, `rm`, `delete`, `del`)
- **Usage:** Removes a package natively and untracks it.
- **Flags:** `--manager`, `-m`
- **Behavior:** Looks up recorded manager from manifest if not overridden by `-m`. Runs native remove, deletes package from manifest, saves manifest.

### `stamp reinstall <pkg>` (C4)
- **Usage:** Reinstalls a package currently tracked in the manifest using its recorded manager.
- **Flags:** None (accepts global `-y` flag).
- **Behavior:**
  1. Looks up `<pkg>` in the manifest.toml. If not found, aborts with: `package "<package>" is not tracked in the manifest`.
  2. Resolves its recorded manager (e.g. `brew`).
  3. Calls `adapter.Install()` on the active manager.
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
- **Usage:** Detects drift between the system state and the last snapshot, and prompts to track.
- **Flags:** `--manager`, `-m` (Proposed)
- **Behavior:** Fetches current state, diffs against snapshots, reports added packages, auto-tracks on `--yes` or prompts. Saves manifest and new snapshots.

### `stamp restore`
- **Usage:** Restores environment on a new machine from the manifest.
- **Flags:** `--dry-run`, `-d`, `--manager`, `-m` (Proposed)
- **Behavior:** Adds repositories sequentially in Phase 1, then installs packages concurrently in Phase 2.

### `stamp doctor`
- **Usage:** Checks manager availability, manifest health, and UNIX compliance.
- **Flags:** `--json`, `-j`, `--manager`, `-m` (Proposed)
- **Behavior:** Audits managers, parses manifest, checks `NO_COLOR` and `stamp man check` statuses.
- **UNIX Compliance TTY section:**
  ```text
  UNIX Compliance:
    NO_COLOR: ✅ Set
    Man Page: ⚠️ Outdated (man v1.1.0, binary v1.2.3) — run 'stamp man install'
  ```

### `stamp self-update` (alias `self-upgrade`)
- **Usage:** Upgrades the stamp binary from the GitHub releases API.
- **Flags:** `--check`, `-c`

### `stamp completion <shell>`
- **Usage:** Generates completion scripts for `bash`, `zsh`, `fish`, or `powershell`.

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
│   ├── manager/       → Package manager adapters (dnf, brew, flatpak)
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
- **Ask first:** Before adding any third-party dependencies beyond `cobra` and `go-toml`.
- **Ask first:** Before changing the structure of the `manifest.toml`.
- **Never:** Mutate the actual system state (run native installs) during a `reconcile` or `list` command.
- **Never:** Use flags to represent actions (e.g. `--install`). Use subcommands instead.

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
1. **Init:** Running `stamp init` creates the correct XDG directories and an empty `manifest.toml`.
2. **Reconcile (No Drift):** If system state matches the last snapshot, `reconcile` exits cleanly.
3. **Reconcile (Drift):** If `flatpak install com.spotify.Client` is run externally, `stamp reconcile` detects this one new package, prompts, and adds it to `manifest.toml`.
4. **Restore:** Running `stamp restore` successfully adds repositories *before* executing the respective package manager install commands concurrently.
5. **Notes:** A user can pass `--note "reason"` to `stamp install` or `stamp edit`, which will be correctly saved in the TOML manifest.
6. **Doctor:** `stamp doctor` reports manager status, manifest health, and UNIX compliance in both TTY and JSON.
7. **Man Pages:** `stamp man` displays help; `stamp man install` installs man pages; `stamp man check` verifies version matches binary.
8. **Completions:** `stamp completion bash|zsh|fish|powershell` generates valid shell completion scripts.
9. **Reinstall:** `stamp reinstall htop` successfully reinstalls a manifest-tracked package using its recorded manager.
10. **Info:** `stamp info htop -m dnf` prints raw dnf info metadata directly.
11. **Install:** `stamp install htop` installs the package natively via the resolved manager and records it in `manifest.toml`.
12. **Remove:** `stamp remove htop` removes the package natively and removes it from the manifest.
13. **Search:** `stamp search ripgrep` returns matching packages from all available managers.
14. **Repo Add:** `stamp repo add myrepo -m brew` adds the repository via the specified manager and records it.
15. **Repo Remove:** `stamp repo remove myrepo -m brew` removes the repository and untracks it.
16. **Repo List:** `stamp repo list` prints all tracked repositories; `--json` outputs machine-readable.
17. **Hello:** `stamp hello` displays ASCII logo, project description, and suggested next steps.
18. **Completion:** `stamp completion bash|zsh|fish|powershell` generates valid shell completion scripts for each shell.
