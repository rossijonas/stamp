# Spec: Stamp (Intent Tracker)

## Objective
Build a CLI tool that captures a solo developer's package installation intent across fragmented package managers (`dnf`, `brew`, `flatpak`) into a portable, version-controllable TOML manifest. The primary workflow is using `stamp install` as a unified wrapper to guarantee total traceability from day one. It also acts as a passive safety net, allowing developers to track changes retroactively via local snapshot diffing (`stamp reconcile`) if they bypass the tool. It fully supports tracking custom repositories (taps, remotes).

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

**Core Commands:**
| Command | Aliases | Flags | Description |
| :--- | :--- | :--- | :--- |
| `stamp init` | | | Initializes `manifest.toml` and takes baseline snapshot. |
| `stamp install <pkg>` | `add` | `--manager, -m <name>`, `--note <text>` | Installs natively and records intent. |
| `stamp remove <pkg>` | `uninstall`, `rm`, `delete`, `del` | `--manager, -m <name>` | Removes natively and untracks. |
| `stamp search <query>` | | `--manager, -m <name>` | Searches across managers. |
| `stamp reconcile` | | | Detects drift, prompts user, records intent. |
| `stamp restore` | | `--dry-run` | Reinstalls repos and packages on a new machine. |
| `stamp update` | `upgrade` | `--manager, -m <name>` | Runs system upgrades across all managers in parallel. |
| `stamp list` | `ls` | `--json` | Lists all intentionally installed packages. |
| `stamp doctor` | | `--json` | Checks manager availability and manifest integrity. |

**Repository Commands:**
| Command | Aliases | Flags | Description |
| :--- | :--- | :--- | :--- |
| `stamp repo add <name> [url]` | `install` | `--manager, -m <name> (Required)` | Adds custom repository and records it. |
| `stamp repo remove <name>` | `uninstall`, `rm`, `delete`, `del` | `--manager, -m <name> (Required)` | Removes a repository and untracks it. |
| `stamp repo list` | `ls` | `--json` | Lists all tracked repositories. |

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
Running `stamp doctor` executes a complete diagnostic checklist checking host OS details, tracking availability of native package managers, and verifying manifest integrity.

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
  Packages: 14 tracked
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
    "path": "/home/rossijonas/.config/stamp/manifest.toml",
    "valid": true,
    "packages_count": 14
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
├── docs/              → ADRs and specifications
└── README.md
```

## Code Style
Idiomatic Go with strict error wrapping and interface-driven design for testability.

## Testing Strategy
- **Framework:** standard `testing` package + `stretchr/testify` for assertions.
- **Test Locations:** Co-located with source (`state_test.go` next to `state.go`).
- **Core Coverage:** 100% on `internal/state/` and `internal/manifest/`.
- **Mocks:** Mock the `PackageManager` interface.

## Boundaries
- **Always:** Use `context.Context` for all shell executions (`os/exec`).
- **Always:** Return meaningful delta states (added, removed, unchanged).
- **Ask first:** Before adding any third-party dependencies beyond `cobra` and `go-toml`.
- **Ask first:** Before changing the structure of the `manifest.toml`.
- **Never:** Mutate the actual system state (run native installs) during a `reconcile` or `list` command.

## UNIX Compliance & Documentation Strategy
To be a "good UNIX citizen", `stamp` must adhere to:
- **POSIX Syntax:** Handled natively by `spf13/cobra`.
- **XDG Base Directory:** Config in `~/.config/stamp`, state in `~/.local/share/stamp`.
- **Exit Codes:** `0` for success, `>0` for failures (e.g., standard `sysexits`).
- **I/O Separation:** Informational output/UI to `stdout`, errors to `stderr`.
- **NO_COLOR:** Respect the `NO_COLOR=1` environment variable.
- **Auto-Generated Docs:** Usage documentation for GitHub Pages must be auto-generated from the codebase using `github.com/spf13/cobra/doc` to ensure docs and code never drift.
- **UNIX Man Pages:** System reference pages (Section 1) must be auto-generated to `docs/man/` using `cobra/doc` so users can run `man stamp` locally on Unix systems.

## Success Criteria
1. **Init:** Running `stamp init` creates the correct XDG directories and an empty `manifest.toml`.
2. **Reconcile (No Drift):** If system state matches the last snapshot, `reconcile` exits cleanly.
3. **Reconcile (Drift):** If `flatpak install com.spotify.Client` is run externally, `stamp reconcile` detects this one new package, prompts, and adds it to `manifest.toml`.
4. **Restore:** Running `stamp restore` successfully adds repositories *before* executing the respective package manager install commands concurrently.
5. **Notes:** A user can pass `--note "reason"` to `stamp install` or `stamp edit`, which will be correctly saved in the TOML manifest.
