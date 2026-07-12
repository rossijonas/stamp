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
*   `--json`: Output results in machine-readable JSON format.
*   `--yes`, `-y`: Bypasses all interactive confirmation prompts (Auto-Accept).

### Flag Standardization Rules
1. Every flag SHOULD have a single-character short form (`--manager`, `-m`).
2. Actions MUST be subcommands, not flags. (e.g. `stamp man install`, not `stamp man --install`).
3. Boolean flags for enabling/disabling features are acceptable (e.g. `--dry-run`, `--json`).

**Core Commands:**
| Command | Aliases | Flags | Description |
| :--- | :--- | :--- | :--- |
| `stamp` | | | Prints welcome message suggesting `stamp hello` or `stamp --help`. |
| `stamp hello` | | | Prints ASCII logo, project about, and suggests next steps. |
| `stamp init` | | | Initializes `manifest.toml` and takes baseline snapshot. |
| `stamp install <pkg>` | `add` | `--manager, -m <name>`, `--note <text>` | Installs natively and records intent. |
| `stamp remove <pkg>` | `uninstall`, `rm`, `delete`, `del` | `--manager, -m <name>` | Removes natively and untracks. |
| `stamp search <query>` | | `--manager, -m <name>` | Searches across managers. |
| `stamp info <pkg>` | | `--manager, -m <name>` | Shows package information across managers. |
| `stamp reconcile` | | `--manager, -m <name>` | Detects drift, prompts user, records intent. |
| `stamp restore` | | `--dry-run`, `--manager, -m <name>` | Reinstalls repos and packages on a new machine. |
| `stamp update` | `upgrade` | `--manager, -m <name>` | Runs system upgrades across all managers in parallel. |
| `stamp list` | `ls` | `--json`, `--manager, -m <name>` | Lists all intentionally installed packages. |
| `stamp doctor` | | `--json`, `--manager, -m <name>` | Checks manager availability, manifest integrity, and UNIX compliance. |
| `stamp self-update` | `self-upgrade` | `--check` | Checks for and installs the latest version of `stamp`. |
| `stamp completion <shell>` | | | Generates shell completion scripts (bash, zsh, fish, powershell). |
| `stamp man` | | | Prints the stamp man page to stdout. |

**Man Subcommands:**
| Command | Flags | Description |
| :--- | :--- | :--- |
| `stamp man` | | Prints the stamp man page to stdout. |
| `stamp man install` | `--prefix <path>` | Installs man page to system or user path. Default: `~/.local/share/man/man1/`. |
| `stamp man check` | | Verifies installed man page version matches stamp version. |

**Repository Commands:**
| Command | Aliases | Flags | Description |
| :--- | :--- | :--- | :--- |
| `stamp repo add <name> [url]` | `install` | `--manager, -m <name> (Required)` | Adds custom repository and records it. |
| `stamp repo remove <name>` | `uninstall`, `rm`, `delete`, `del` | `--manager, -m <name> (Required)` | Removes a repository and untracks it. |
| `stamp repo list` | `ls` | `--json`, `--manager, -m <name>` | Lists all tracked repositories. |

## Package Manager Resolution Engine
When a user runs a package or repository command (e.g., `stamp install htop`) without specifying `--manager`, the tool resolves ambiguity using a three-tier engine:

1. **Tier 1: Explicit Override:** If `--manager <name>` or `-m <name>` is provided, `stamp` directly executes that manager's command.
2. **Tier 2: User Preference (Declarative):** If the package exists in multiple managers, `stamp` checks the user's `config.toml` precedence list:
   ```toml
   precedence = ["dnf", "flatpak", "brew"]
   ```
   If a match is found, `stamp` automatically selects the manager with the highest configured precedence.
3. **Tier 3: Interactive Choice (Fallback):** If no precedence is defined (or there's a tie) and the process runs in an interactive terminal (TTY), `stamp` prompts the user to select the manager. In non-interactive environments (scripts/pipelines), the command fails with a clean error prompting the user to specify `--manager`.

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

## System Diagnosis (Doctor)
Running `stamp doctor` executes a complete diagnostic checklist checking host OS details, tracking availability of native package managers, verifying manifest integrity, and reporting UNIX compliance status (NO_COLOR, man pages).

### TTY Output Example:
```text
▪ System Diagnosis (Stamp Doctor)

Package Managers:
  Name      Status          Path                  Details
  dnf       ✅ Active       /usr/bin/dnf          Default system manager
  brew      ✅ Active       /usr/local/bin/brew   User-space manager
  flatpak   ❌ Not Found    -                     Executable not found in $PATH

Manifest Integrity:
  Path:     ~/.config/stamp/manifest.toml
  Status:   ✅ Healthy (Valid TOML)
  Packages: 3 tracked

UNIX Compliance:
  NO_COLOR: ❌ Not set
  Man Page: ❌ Not found — run 'stamp man install'
```

### JSON Output Example (`stamp doctor --json`):
```json
{
  "system": "fedora",
  "package_managers": [
    {"name": "dnf", "active": true, "path": "/usr/bin/dnf", "details": "Default system manager"},
    {"name": "brew", "active": true, "path": "/usr/local/bin/brew", "details": "User-space manager"},
    {"name": "flatpak", "active": false, "path": "", "details": "Executable not found in $PATH"}
  ],
  "manifest": {
    "path": "/home/user/.config/stamp/manifest.toml",
    "valid": true,
    "packages_count": 3
  },
  "no_color": false,
  "man_page": {
    "installed": false,
    "path": ""
  }
}
```

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
- **Minimum:* * 90% overall project coverage.

## Boundaries
- **Always:** Use `context.Context` for all shell executions (`os/exec`).
- **Always:** Return meaningful delta states (added, removed, unchanged).
- **Always:** Every flag SHOULD have a single-character short form.
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

## `stamp hello` — Welcome Command (C1)

### Objective
Provide a friendly entry point for new users. When run without arguments, stamp should guide users toward useful first steps. `stamp hello` shows identity, purpose, and suggested next operations.

### Trigger Behavior
Running `stamp` with no arguments or subcommand prints:
```
Don't know where to start? Try:

  stamp hello    — Learn about stamp and next steps
  stamp --help   — See all available commands
```

### Command: `stamp hello`

#### Flags
None. No arguments.

#### Behavior
1. Prints ASCII logo (same as README header).
2. Prints a short "about" paragraph.
3. Suggests three initial operations:
   ```
   For a fresh installation:
     stamp init          — Create manifest and baseline
     stamp doctor        — Verify system is ready
     stamp man install   — Install man pages for offline help
   ```

#### TTY Output Example
```text

                              
         █▄
        ▄██▄      ▄
   ▄██▀█ ██ ▄▀▀█▄ ███▄███▄ ████▄
   ▀███▄ ██ ▄█▀██ ██ ██ ██ ██ ██
  █▄▄██▀▄██▄▀█▄██▄██ ██ ▀█▄████▀
                           ██
                           ▀

  stamp — A lightweight yet powerful wrapper for your native package managers.

  For a fresh installation, try:

    stamp init          — Create manifest and take initial snapshot
    stamp doctor        — Verify system configuration
    stamp man install   — Install offline documentation

  Need help? Run:  stamp --help
```

#### JSON Output
Not applicable. `--json` flag not supported for this command.

### Business Rules
- No args expected (cobra.NoArgs).
- Pure informational. No system state modification.
- Must display ASCII logo exactly as in README header.

---

## `stamp info <pkg>` — Package Info Command (C2)

### Objective
Allow users to query detailed information about a package across all available package managers. Useful for discovering which managers provide a package and getting version/description details.

### Command: `stamp info <pkg>`

#### Flags
| Flag | Short | Required | Description |
| :--- | :--- | :---: | :--- |
| `--manager <name>` | `-m` | No | Scope info query to a single manager. |

#### Behavior
1. If `--manager` is specified: query only that manager.
2. If no `--manager`: query all available adapters.
3. Returns package information (description, version, repository, homepage) from each manager that provides the package.
4. If no manager has the package: print "not found" message.

#### TTY Output Example
```
$ stamp info ripgrep
ripgrep
  dnf:      v14.1.0  (updates)
  brew:     v14.1.0  (core)
  flatpak:  not available

$ stamp info nonexistent
nonexistent: not found in any package manager
```

#### JSON Output Example (`--json`)
```json
{
  "package": "ripgrep",
  "results": [
    {"manager": "dnf", "found": true, "version": "14.1.0", "source": "updates"},
    {"manager": "brew", "found": true, "version": "14.1.0", "source": "core"},
    {"manager": "flatpak", "found": false}
  ]
}
```

### MVP Scope
For MVP, `stamp info` shows whether a package is available in each manager and its version string (as reported by the manager's search/list output). Full metadata (description, homepage) is deferred.

### Business Rules
- Positional arg `pkg` is required (cobra.ExactArgs(1)).
- Adapter.Search() result may contain version info depending on manager. Display what's available.
- If `--manager` specified but manager not found: error "unknown manager".
- If no adapters available: error "no package managers available".

---

## `stamp man check` — Man Page Version Verification (C3)

### Objective
Verify that the installed stamp man page matches the current stamp binary version. If man pages are missing, outdated, or not installed, warn the user and recommend `stamp man install`.

### Command: `stamp man check`

#### Flags
None.

#### Behavior
1. Read the installed man page at standard system paths (`/usr/local/share/man/man1/stamp.1`, `/usr/share/man/man1/stamp.1`, `/opt/homebrew/share/man/man1/stamp.1`).
2. If not found at any path: print message and recommend `stamp man install`.
3. Parse the man page header for the version string (embedded as `.TH STAMP 1 "vX.Y.Z"`).
4. Compare against `cli.Version` (the built-in binary version).
5. Match: print "✅ Man page is up to date (vX.Y.Z)".
6. Mismatch: print "⚠️ Man page is outdated (installed vA.B.C, current vX.Y.Z). Run 'stamp man install' to update."
7. Exit 0 (informational — outdated man page is a warning, not a failure).

#### TTY Output Examples
```
$ stamp man check
✅ Man page is up to date (v1.2.3)

$ stamp man check
⚠️ Man page is outdated (installed v1.1.0, current v1.2.3). Run 'stamp man install' to update.

$ stamp man check
❌ Man page not installed. Run 'stamp man install' to install.
```

#### JSON Output (`--json`)
```json
{"installed": true, "man_version": "1.2.3", "binary_version": "1.2.3", "match": true}
```
```json
{"installed": false, "error": "not found"}
```

### Doctor Integration
Add to `stamp doctor` UNIX Compliance section:
```
Man Page: ⚠️ Outdated (man v1.1.0, binary v1.2.3) — run 'stamp man install'
```
Detection logic reused from `stamp man check`.

### Business Rules
- No args expected (cobra.NoArgs).
- Version comparison uses semantic versioning (string equality, not semver comparison for MVP).
- Non-zero exit only on actual errors (e.g. permission denied reading man path). Mismatch = exit 0.
- Doctor integration reuses the same underlying detection function.

---

## Deferred Decisions
- **Landing page design:** Visual style, screenshot/GIF format, and exact content layout for `docs/index.html` to be discussed before implementation begins.

## Success Criteria
1. **Init:** Running `stamp init` creates the correct XDG directories and an empty `manifest.toml`.
2. **Reconcile (No Drift):** If system state matches the last snapshot, `reconcile` exits cleanly.
3. **Reconcile (Drift):** If `flatpak install com.spotify.Client` is run externally, `stamp reconcile` detects this one new package, prompts, and adds it to `manifest.toml`.
4. **Restore:** Running `stamp restore` successfully adds repositories *before* executing the respective package manager install commands concurrently.
5. **Notes:** A user can pass `--note "reason"` to `stamp install` or `stamp edit`, which will be correctly saved in the TOML manifest.
6. **Doctor:** `stamp doctor` reports manager status, manifest health, and UNIX compliance in both TTY and JSON.
7. **Man Pages:** `stamp man` generates a valid troff man page; `stamp man install` installs it to the system.
8. **Completions:** `stamp completion bash|zsh|fish|powershell` generates valid shell completion scripts.
