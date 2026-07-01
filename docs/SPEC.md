# Spec: Stamp (Intent Tracker)

## Objective
Build a CLI tool that captures a solo developer's package installation intent across fragmented package managers (`dnf`, `brew`, `flatpak`) into a portable, version-controllable TOML manifest. The primary workflow is using `stamp install` as a unified wrapper to guarantee total traceability from day one. It also acts as a passive safety net, allowing developers to track changes retroactively via local snapshot diffing (`stamp reconcile`) if they bypass the tool. It fully supports tracking custom repositories (taps, remotes).

## Tech Stack
- **Language:** Go 1.22+
- **CLI Framework:** `spf13/cobra` (industry standard for Go CLIs)
- **Manifest Parsing:** `pelletier/go-toml/v2`
- **Output/UI:** Standard `fmt` and `log` (keeping it simple for MVP)

## Command Blueprint
The complete surface area of the CLI, including aliases and flags.

**Global Flags:**
*   `--verbose`, `-v`: Enable debug/verbose logging.
*   `--json`: Output results in machine-readable JSON format.

**Core Commands:**
| Command | Aliases | Flags | Description |
| :--- | :--- | :--- | :--- |
| `stamp init` | | | Initializes `manifest.toml` and takes baseline snapshot. |
| `stamp install <pkg>` | `add` | `--via <manager>`, `--note <text>` | Installs natively and records intent. |
| `stamp remove <pkg>` | `uninstall`, `rm`, `delete`, `del` | `--via <manager>` | Removes natively and untracks. |
| `stamp search <query>` | | `--via <manager>` | Searches across managers. |
| `stamp reconcile` | | | Detects drift, prompts user, records intent. |
| `stamp restore` | | `--dry-run` | Reinstalls repos and packages on a new machine. |
| `stamp update` | `upgrade` | `--manager <name>` | Runs system upgrades across all managers in parallel. |
| `stamp list` | `ls` | `--json` | Lists all intentionally installed packages. |
| `stamp doctor` | | | Checks manager availability and manifest integrity. |

**Repository Commands:**
| Command | Aliases | Flags | Description |
| :--- | :--- | :--- | :--- |
| `stamp repo add <name> [url]` | `install` | `--via <manager>` | Adds custom repository and records it. |
| `stamp repo remove <name>` | `uninstall`, `rm`, `delete`, `del` | `--via <manager>` | Removes a repository and untracks it. |
| `stamp repo list` | `ls` | `--json` | Lists all tracked repositories. |

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

## Success Criteria
1. **Init:** Running `stamp init` creates the correct XDG directories and an empty `manifest.toml`.
2. **Reconcile (No Drift):** If system state matches the last snapshot, `reconcile` exits cleanly.
3. **Reconcile (Drift):** If `flatpak install com.spotify.Client` is run externally, `stamp reconcile` detects this one new package, prompts, and adds it to `manifest.toml`.
4. **Restore:** Running `stamp restore` successfully adds repositories *before* executing the respective package manager install commands concurrently.
5. **Notes:** A user can pass `--note "reason"` to `stamp install` or `stamp edit`, which will be correctly saved in the TOML manifest.
