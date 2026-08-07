---
---

# Implementation Plan: Stamp (Intent Tracker)

## 1. Development Standards & Repository Structure

To ensure `stamp` is maintainable, idiomatic, and accessible to contributors (human and AI), we will enforce the following standards:

### Repository Layout
Based on standard Go project layout conventions:
*   `cmd/stamp/` - Application entrypoint. Contains `main.go`. Minimal logic.
*   `internal/` - Core business logic. Un-importable by external modules.
    *   `internal/cli/` - Cobra command definitions.
    *   `internal/manager/` - Package manager adapters (dnf, brew, flatpak).
    *   `internal/state/` - Snapshotting and diffing logic.
    *   `internal/manifest/` - TOML parsing.
*   `tools/docgen/` - Build-time documentation generation tool.
*   `docs/` - Architecture Decision Records (ADRs), specs, and plans.
*   `testdata/` - Static JSON/TOML fixtures for unit tests.

### Development Tools
We will use modern, Go-centric tooling:
*   **Taskfile (`task`)**: Replaces `Makefile`. YAML-based, self-documenting task runner (`task build`, `task test`, `task lint`, `task docs`).
*   **golangci-lint**: The standard Go linter. Configured via `.golangci.yml`.
*   **stretchr/testify**: Used for all test assertions (`assert` and `require`) and mock generation.

### Contribution Documentation
To guide future development:
*   **`README.md`**: User-facing documentation.
*   **`CONTRIBUTING.md`**: Developer-facing documentation explaining how to install `task`, run tests, and format commit messages (Conventional Commits: `feat:`, `fix:`, `chore:`).
*   **`AGENTS.md`**: AI-facing instructions mandating the use of `testify`, table-driven tests, and adherence to the Spec-Driven workflow.

---

## 2. Technical Task Breakdown (Vertical Slicing)

We are building `stamp` using vertical slices. We will not build "all the managers" and then "all the commands". We will build the foundation, then the passive observer flow, then the active wrapper flow.

### Phase 1: Project Foundation & Data Models
Establish the repo structure, tooling, and core data types.

**Task 1: Repository Scaffolding & Tooling**
*   **Description:** Set up the Go module, `.gitignore`, `.golangci.yml`, and `Taskfile.yml`. Create `CONTRIBUTING.md` and `AGENTS.md`.
*   **Acceptance:** Running `task` prints available commands. Linter runs successfully.
*   **Verify:** `task lint`
*   **Status:** ✓ Completed

**Task 2: Manifest Manager (TOML)**
*   **Description:** Implement `internal/manifest` to read, write, and manipulate `manifest.toml`, including support for the `notes` field.
*   **Acceptance:** Can serialize/deserialize TOML and add/remove packages and repositories.
*   **Verify:** `task test` passes for `internal/manifest`.
*   **Status:** ✓ Completed

**Task 2.5: Pre-requisite Fixes (Security & CI)**
*   **Description:** Upgrade the project toolchain to Go 1.26 to resolve standard library CVEs identified by `govulncheck`. Fix the duplicate authorization header bug in the GitHub Actions pipeline.
*   **Acceptance:** `ci.yml` uses native `go run govulncheck` to prevent double checkout. Project builds with Go 1.26.
*   **Verify:** GitHub Actions pipeline passes cleanly.
*   **Status:** ✓ Completed

### Phase 2: The Active Wrapper Flow
Build the ability to actually modify the system (Install, Remove, Search) as the primary usage model.

**Task 3: Package Manager Interfaces & Mocks**
*   **Description:** Define the `PackageManager` interface. Implement `MockManager` for testing.
*   **Acceptance:** Interface defines `Name()`, `ListInstalled()`, `Install()`, `Remove()`, `Search()`, `AddRepo()`, `RemoveRepo()`.
*   **Verify:** `task test` passes for `internal/manager` mocks.
*   **Status:** ✓ Completed

**Task 4: Native Adapters (Write Operations)**
*   **Description:** Implement `Install()`, `Remove()`, `Search()`, `AddRepo()`, and `RemoveRepo()` for `dnf`, `brew`, and `flatpak`.
*   **Acceptance:** Adapters can execute system and repository modifications.
*   **Verify:** Tests pass.
*   **Status:** ✓ Completed

**Task 5: Active CLI Commands**
*   **Description:** Wire up `stamp install/add`, `stamp remove/uninstall/delete/del`, and `stamp search` in Cobra. Implement the `stamp repo` command group. Ensure aliases are properly registered using Cobra's `Aliases` array, and the 3-tier resolution engine parses `config.toml` precedence and regex-based matching rules. Supports the global `--yes` / `-y` flag.
*   **Acceptance:** Users can install packages and repositories via `stamp`, updating the manifest automatically.
*   **Verify:** Manual test of `stamp install <test-pkg>`
*   **Status:** ✓ Completed

### Phase 3: The Safety Net Flow
Build the read-only safety net: checking the system state and calculating the delta.

**Task 6: Native Adapters (Read-Only)**
*   **Description:** Implement `ListInstalled()` for `dnf`, `brew`, and `flatpak` using abstracted shell execution (`os/exec`).
*   **Acceptance:** Adapters correctly parse `dnf repoquery`, `brew leaves`, and `flatpak list`.
*   **Verify:** Unit tests pass using mocked string outputs.
*   **Status:** ✓ Completed

**Task 7: State Engine (Snapshotting)**
*   **Description:** Implement `internal/state` to save JSON snapshots and calculate deltas (Added/Removed) against the current `PackageManager` outputs.
*   **Acceptance:** Engine can accurately report which packages were added since the last snapshot.
*   **Verify:** `task test` passes with 100% coverage on diffing logic.
*   **Status:** ✓ Completed

**Task 8: The `reconcile` Command (Cobra)**
*   **Description:** Wire up `cmd/stamp/main.go` and `internal/cli/reconcile.go`. Supports the `--yes` / `-y` flag to auto-track detected packages without prompting.
*   **Acceptance:** Running `stamp reconcile` fetches the state, calculates the delta, and prompts the user (or auto-tracks) to add new packages to the manifest.
*   **Verify:** Manual test: `go run cmd/stamp/main.go reconcile`
*   **Status:** ✓ Completed

### Phase 4: Restore & UNIX Compliance
Build the environment reconstruction logic and final touches.

**Task 9: The `restore` Command**
*   **Description:** Implement the environment reconstruction logic. Supports the `--yes` / `-y` flag to bypass safety confirmation prompts.
*   **Acceptance:** `stamp restore` parses the manifest, restores all tracked repositories first, and then executes concurrent package installs.
*   **Verify:** Manual test with `--dry-run` flag.
*   **Status:** ✓ Completed

**Task 10: CLI Polish and Documentation**
*   **Description:** Implement `stamp doctor`, `stamp completion`, `stamp man`, NO_COLOR compliance, doc generation pipeline, landing page, and flag standardization.
*   **Status:** ✓ Completed

#### Task 10 Subtasks

| Subtask | Description | Status |
| :--- | :--- | :---: |
| 10a | `stamp doctor` command with TTY/JSON output | ✓ |
| 10b | `stamp completion` shell autocompletion (bash/zsh/fish/powershell) | ✓ |
| 10c | `stamp man` man page generation and install | ✓ |
| 10d | NO_COLOR compliance | ✓ |
| 10e | Doc generation pipeline (`task docs` + CI enforcement) | ✓ |
| 10f | Flag standardization (short forms, actions-as-subcommands) | ✓ |
| 10h | Uninstall documentation in README.md (standard + hard uninstall) | ✓ |

**Task 11: Self-Update Subcommand**
*   **Description:** Implement `stamp self-update/self-upgrade` that checks the current binary version against the GitHub releases API, downloads the latest binary for the host OS/arch, verifies SHA-256 checksums, and replaces itself atomically with permission preservation. After update, automatically re-installs shell completions and man pages.
*   **Acceptance:** User can run `stamp self-update --check` to check, and `stamp self-update` to apply. Post-update hooks complete successfully.
*   **Verify:** Unit tests mock the release API, checksum verification, and binary swap logic. Run `task check`.
*   **Files:** `internal/cli/selfupdate.go`, `internal/cli/selfupdate_test.go`
*   **Status:** ✓ Completed

**Task 12: `stamp hello` Welcome Command**
*   **Description:** Implement a welcome command that prints the ASCII logo, a brief project description, and suggests next steps for new users.
*   **Acceptance:** Running `stamp hello` displays logo, about text, and suggests `stamp init`, `stamp doctor`, `stamp man install`.
*   **Status:** ✓ Completed

**Task 13: `stamp info` Package Info Command**
*   **Status:** ✓ Completed

**Task 13: `stamp info` Package Info Command**
*   **Description:** Implement a command to show detailed package information across all package managers. Supports `--manager` flag to scope to a specific manager.
*   **Acceptance:** Running `stamp info htop` shows package details from all managers that have it.
*   **Status:** ✓ Completed

**Task 14: `stamp man check` Version Verification**
*   **Description:** Implement a subcommand within `stamp man` that verifies the installed man page version matches the stamp binary version.
*   **Acceptance:** Running `stamp man check` reports whether man pages are current, outdated, or missing.
*   **Status:** ✓ Completed

**Task 15: Per-Manager Flag Support**
*   **Description:** Add `--manager`, `-m` flag to `stamp list`, `stamp reconcile`, `stamp restore`, `stamp doctor`, and `stamp update` to scope operations to a single package manager.
*   **Status:** ⚠ Partial

| Subtask | Description | Status |
| :--- | :--- | :---: |
| 15a | `stamp reconcile -m` | ✓ |
| 15b | `stamp restore -m` | ✓ |
| 15c | `stamp doctor -m` | ✓ |
| 15d | `stamp list -m` | ✓ (via Task 22) |
| 15e | `stamp update -m` | ✓ (Task 23) |

#### Phase 4c — Infrastructure

**Task 16: Multi-Platform Integration Testing**
*   **Description:** Add CI matrix testing across Fedora, Ubuntu, Arch Linux, macOS, and Windows using Docker containers and parallel pipeline jobs. Each environment runs the full test suite against real package managers.
*   **Acceptance:** CI passes on all target platforms for every PR.
*   **Verify:** Green CI status on all matrix jobs.
*   **Status:** * Research needed

**Task 17: Package Manager Feature Audit**
*   **Description:** Audit each supported package manager for important features not yet covered by stamp. Specifically: Homebrew `cask` (GUI apps), `brew services`, `dnf groupinstall`, flatpak remotes management. Determine which are critical for adoption.
*   **Acceptance:** Documented findings with recommendations for each manager.
*   **Verify:** Report in docs/decisions/ or FEATURE_MATRIX.md.
*   **Status:** * Research needed

**Task 18: `stamp reinstall` Command**
*   **Description:** Implement a reinstall command that looks up a package in the manifest, resolves its recorded manager, and executes the native reinstallation. No `-m` flag needed — manager resolved from manifest.
*   **Acceptance:** `stamp reinstall htop` reinstalls `htop` using the manager recorded in the manifest. Accepts global `-y`.
*   **Status:** ✓ Completed

**Task 19: Generate Missing Usage & Man Pages**
*   **Description:** Run `task docs` to auto-generate missing `docs/usage/` pages (`stamp_hello.md`, `stamp_info.md`, `stamp_reinstall.md`) and populate `docs/man/` with system man page files.
*   **Acceptance:** Every registered subcommand has a corresponding `docs/usage/*.md` page. `docs/man/stamp.1` exists and is up to date.
*   **Status:** ✓ Completed

#### Phase 4b — Medium Features

**Task 20: Create GitHub Pages Landing Page**
*   **Description:** Create `docs/index.html` as a custom landing page for GitHub Pages. Content requirements defined in SPEC.md → Project Landing Page. Source tagline and features from README.md.
*   **Acceptance:** Navigating to `https://rossijonas.github.io/stamp/` displays the project landing page.
*   **Status:** ~ Pending

#### Phase 4a — Quick Wins

**Task 21: `stamp init` Command**
*   **Description:** Initialize `manifest.toml` and take baseline snapshot of current system packages. Create XDG directories (`~/.config/stamp`, `~/.local/share/stamp/snapshots`). Suggested by `stamp hello` output.
*   **Acceptance:** Running `stamp init` creates config dir, snapshot dir, empty manifest.toml, and baseline snapshot for each available manager.
*   **Status:** ✓ Completed

**Task 22: `stamp list` Command (alias `ls`)**
*   **Description:** List all intentionally installed packages from the manifest. Supports `--json, -j` and `--manager, -m` flags.
*   **Acceptance:** Running `stamp list` prints tracked packages; `stamp list --json` outputs JSON; `stamp list -m brew` filters by manager.
*   **Status:** ✓ Completed

**Task 23: `stamp update` Command (alias `upgrade`)**
*   **Description:** Run system upgrades across all available managers in parallel. Supports `--manager, -m` flag to scope to a single manager.
*   **Acceptance:** Running `stamp update` executes native update/upgrade commands concurrently per manager. Errors from one manager don't block others. Non-zero exit if any manager fails.
*   **Verify:** `task test` passes, manual test: `stamp update` shows per-manager results.
*   **Files:** `internal/cli/update.go`, `internal/cli/update_test.go`, `internal/manager/dnf.go`, `internal/manager/brew.go`, `internal/manager/flatpak.go`, `internal/manager/mock.go`, `internal/manager/manager.go`
*   **Status:** ✓ Completed

**Task 24: Migrate `stamp hello` to `stamp setup` Wizard**
*   **Description:** Replace `stamp hello` with `stamp setup` interactive wizard. Keep `hello` as alias. Run completion, man install, init (with prompts, default Yes), then doctor (no prompt). Support `-y` flag for scripting.
*   **Acceptance:** `stamp setup -y` runs all steps without prompts. `stamp hello` continues to work as alias.
*   **Status:** ✓ Completed

**Task 25: Add Shell Completion Check to `stamp doctor`**
*   **Description:** Check common shell completion paths (bash, zsh, fish) and report status in doctor TTY and JSON output.
*   **Acceptance:** `stamp doctor` shows ✓ or ✗ for completions in both TTY and JSON modes.
*   **Status:** ~ Pending

**Task 25b: Re-init Guard for `stamp init` with Mandatory Backup**
*   **Description:** Add re-init guard to `stamp init`: detect existing manifest, warn user, prompt for confirmation (default No). On confirmation, **always** backup manifest + snapshots (`<path>.<ts>.bak`) before creating fresh state. Update `stamp setup` wizard to detect initialized state and adjust prompt wording. `-y` flag bypasses prompt. Backup runs unconditionally on confirmed re-init.
*   **Acceptance:** `stamp init` on initialized system shows warning, prompts with default No. Accepting creates timestamped backups and fresh state. Declining aborts cleanly. `-y` skips prompt. Wizard shows adjusted prompt when already initialized.
*   **Verify:** `task test` passes.
*   **Files:** `internal/cli/init.go`, `internal/cli/init_test.go`, `internal/cli/hello.go`, `internal/cli/hello_test.go`, `internal/manifest/manifest.go`, `internal/manifest/manifest_test.go`, `internal/state/state.go`, `internal/state/state_test.go`
*   **Status:** ✓ Completed

**Task 26: Add `yum` as Alias to `dnf` Manager**
*   **Description:** Automatically detect `yum` when `dnf` is unavailable (RHEL/CentOS 7). Use resolved command name for all exec calls.
*   **Acceptance:** `stamp` works on systems with only `yum` installed.
*   **Status:** ✓ Completed

### Phase 5: Project Licensing & Governance
Ensure maximum community and enterprise reach.

**Task 11: Relicense to Apache-2.0**
*   **Description:** Transition project license from AGPL-3.0 to Apache-2.0 to simplify integration and adoption. Update files and documentation.
*   **Acceptance:** LICENSE contains Apache-2.0 text, README links to correct license, and ADR-003 is merged.
*   **Verify:** `task check` passes.
*   **Status:** ✓ Completed

### Phase 6: Reconcile Behavior Stabilisation & Feature Completion

Deliver the final design for `stamp reconcile` and `stamp reinstall` based on real-world testing feedback.

**Task 27: Reconcile — Auto-Track and `--dry-run`**
*   **Description:** Remove interactive prompt from reconcile. Auto-track all discovered drift. Add `--dry-run` / `-d` flag for preview mode without saving manifest or snapshots. Fix snapshot save timing to persist on no-drift.
*   **Acceptance:** `stamp reconcile` auto-tracks without prompting. `stamp reconcile --dry-run` shows drift but does not save. `-y` accepted for backward compatibility (no-op). Snapshot updated on no-drift to accurately track subsequent removals.
*   **Verify:** `task test` passes, manual test of `--dry-run` flag.
*   **Files:** `internal/cli/reconcile.go`, `internal/cli/reconcile_test.go`
*   **Depends on:** Task 7 (state engine), Task 8 (reconcile command), Issue #39 (adapter fixes)
*   **Status:** ✓ Completed

**Task 28: Reinstall — Support Pre-Existing Packages**
*   **Description:** Extend `stamp reinstall <pkg>` to handle packages NOT in the manifest. Resolve manager via resolution engine, run native reinstall, append to manifest, save snapshot. Add `Reinstall()` to `Adapter` interface.
*   **Acceptance:** `stamp reinstall htop` works for both manifest-tracked and pre-existing (manifest-absent) packages. Pre-existing packages are recorded in manifest.
*   **Verify:** `task test` passes, manual test: install package outside stamp → `stamp init` → `stamp reinstall pkg` → `stamp list` shows it.
*   **Files:** `internal/cli/reinstall.go`, `internal/cli/reinstall_test.go`
*   **Depends on:** Task 27 (reconcile spec), Issue #39 (adapter fixes)
*   **Status:** ✓ Completed

**Task 29: Flag and Compliance Updates**
*   **Description:** Update global flag documentation to reflect reconcile's deterministic behavior. Ensure `--dry-run` is registered on reconcile and restore. Ensure docs are up to date.
*   **Acceptance:** `--dry-run` flag documented in usage and help. `-y` flag documented as backward-compatible no-op for reconcile. Auto-generated docs match code.
*   **Verify:** `task docs` generates correct usage pages.
*   **Files:** `docs/usage/stamp_reconcile.md`, `internal/cli/reconcile.go` (after code done)
*   **Depends on:** Task 27
*   **Status:** ✓ Completed

**Task 30: `stamp auto-reconcile` Command** ✓
*   **Description:** Implement a subcommand to install or remove automated reconcile timers.

**Task 33: Docker-Based Integration Testing**
*   **Description:** Set up Docker-based integration smoke tests for Ubuntu and Fedora containers.

**Task 34: Post-Release Integration CI Pipelines**
*   **Description:** Create 3 separate GitHub Actions workflows triggered on `release: [published]`, one per target platform (ubuntu, debian, fedora). Each workflow downloads the just-published linux/amd64 binary from the release, builds the Docker test image, and runs the integration test suite. Each workflow produces its own badge. Badges displayed in README.md next to existing CI badge.
*   **Acceptance:** After a release, all 3 workflows execute and badges update to green/passing.
*   **Verify:** Trigger `workflow_dispatch` on each workflow after merge.
*   **Files:** `.github/workflows/test-integration-ubuntu.yml`, `.github/workflows/test-integration-debian.yml`, `.github/workflows/test-integration-fedora.yml`, `README.md`
*   **Status:** ✓ Completed Each container installs stamp and runs real package manager operations (search, install, remove, repo add/remove) via the native adapter (apt on Ubuntu, dnf/flatpak on Fedora, brew on both). Uses Taskfile tasks for execution. Works with both Docker and Podman (via podman-docker) on Fedora.
*   **Acceptance:** `task test:integration` builds the binary and runs smoke tests in Ubuntu 24.04 and Fedora 41 containers without errors.
*   **Verify:** `task test:integration`
*   **Files:** `test/Dockerfile.ubuntu`, `test/Dockerfile.fedora`, `test/integration/ubuntu.sh`, `test/integration/fedora.sh`, `.dockerignore`, `Taskfile.yml`
*   **Status:** ✓ Completed

**Task 32: APT Package Manager Adapter (#46)**
*   **Description:** Implement APT adapter for Debian/Ubuntu systems. Covers all `Adapter` interface methods: ListInstalled (with dpkg-query fallback excluding rc packages), Install, Reinstall, Remove, Search (apt-cache), Info (apt show / apt-cache show), AddRepo (hybrid PPA via add-apt-repository + custom URL via .list file), RemoveRepo, ListRepos (file parsing), Update (two-phase: update + upgrade), Doctor (not supported). Reuses `sudoCmd` from DNF adapter for all write operations.
*   **Acceptance:** All adapter methods work with mocked executors. APT is auto-detected on Debian/Ubuntu systems.
*   **Verify:** `task test` passes, `task check` passes.
*   **Files:** `internal/manager/apt.go`, `internal/manager/apt_test.go`, `internal/cli/root.go`, `internal/cli/repo.go`
*   **Status:** ✓ Completed On Linux, creates systemd user service + timer files in `~/.config/systemd/user/`. On macOS, creates launchd plist in `~/Library/LaunchAgents/`. Supports `--period`, `-p` flag (hourly/daily/weekly, default daily).
*   **Acceptance:** `stamp auto-reconcile on` installs the timer. `stamp auto-reconcile off` removes it. Timer runs `stamp reconcile` at the configured interval. Pre-configured timer files available in `contrib/`.
*   **Verify:** Manual test: `stamp auto-reconcile on --period daily` creates timer, `stamp auto-reconcile off` removes it.
*   **Files:** `internal/cli/autoreconcile.go`, `internal/cli/autoreconcile_test.go`, `contrib/systemd/stamp-reconcile.service`, `contrib/systemd/stamp-reconcile.timer`, `contrib/launchd/com.rossijonas.stamp.reconcile.plist`
*   **Depends on:** Task 27
*   **Status:** ~ Pending

### Phase 7: Signal Handling & Clean Abort

Deliver reliable Ctrl+C behavior for all commands that spawn privileged children.

**Task 43: Clean SIGINT Abort for All Commands (#170)**
*   **Description:** Install a shared two-phase SIGINT handler (`cancelOnInterrupt`) once in the root command's `PersistentPreRunE`, covering every runnable command. First SIGINT: newline + `stty echo` restore + cancel the command context so `exec.CommandContext` kills the running child. Second SIGINT: SIGKILL the entire job process group and exit 130, guaranteeing no orphaned privileged processes survive. Remove the now-redundant inline handler from `update.go`. Documented in ADR-014.
*   **Acceptance:** Ctrl+C at the sudo password prompt aborts cleanly for `reinstall`, `install`, `remove`, `hold`, `unhold`, `clean`, `autoremove`, `update`, `restore`, and `repo add/remove`; no orphaned `sudo`/`dnf`/`apt` processes remain; terminal echo restored; second Ctrl+C force-exits with 130.
*   **Verify:** `task check` passes; manual smoke: `stamp reinstall -m dnf htop`, Ctrl+C at prompt, then `pgrep -a dnf` empty.
*   **Files:** `internal/cli/signal.go` (new), `internal/cli/signal_test.go` (new), `internal/cli/root.go`, `internal/cli/update.go`, `internal/cli/cmd_test.go`, `docs/decisions/ADR-014-signal-handling-and-clean-abort.md`, `docs/usage/installing-packages.md`
*   **Status:** ✓ Completed

### Phase & Task Progress Summary

| Phase | Task | Description | Status |
| :--- | :--- | :--- | :---: |
| 1 | 1 | Repository Scaffolding & Tooling | ✓ |
| 1 | 2 | Manifest Manager (TOML) | ✓ |
| 1 | 2.5 | Pre-requisite Fixes (Security & CI) | ✓ |
| 2 | 3 | Package Manager Interfaces & Mocks | ✓ |
| 2 | 4 | Native Adapters (Write Operations) | ✓ |
| 2 | 5 | Active CLI Commands | ✓ |
| 3 | 6 | Native Adapters (Read-Only) | ✓ |
| 3 | 7 | State Engine (Snapshotting) | ✓ |
| 3 | 8 | The `reconcile` Command | ✓ |
| 4 | 9 | The `restore` Command | ✓ |
| 4 | 10 | CLI Polish, Manpages, GitHub Pages & Landing Page | ~ |
| 4 | 10a | `stamp doctor` command | ✓ |
| 4 | 10b | `stamp completion` shell autocompletion | ✓ |
| 4 | 10c | `stamp man` generation and install | ✓ |
| 4 | 10d | NO_COLOR compliance | ✓ |
| 4 | 10e | Doc generation pipeline (task docs) | ✓ |
| 4 | 10f | Flag standardization (short forms, subcommands) | ✓ |
| 4 | 10h | Uninstall documentation in README.md | ✓ |
| 4 | 11 | Self-Update Subcommand | ✓ |
| 4 | 12 | `stamp hello` welcome command | ✓ |
| 4 | 13 | `stamp info` package info command | ✓ |
| 4 | 14 | `stamp man check` version verification | ✓ |
| 4 | 15 | Per-manager flags for reconcile/restore/doctor/list | ⚠ Partial |
| 4 | 16 | Multi-platform integration testing | * |
| 4 | 17 | Package manager feature audit | * |
| 4 | 18 | `stamp reinstall` command | ✓ |
| 4 | 19 | Generate missing usage & man pages | ✓ |
| 4 | 20 | Create GitHub Pages landing page | ~ |
| 4 | 21 | `stamp init` command | ✓ |
| 4 | 22 | `stamp list` command (alias `ls`) | ✓ |
| 4 | 23 | `stamp update` command (alias `upgrade`) | ✓ |
| 4 | 24 | Migrate `stamp hello` to `stamp setup` wizard (#59) | ✓ |
| 4 | 25 | Add shell completion check to `stamp doctor` (#60) | ✓ |
| 4 | 26 | Add `yum` as alias to `dnf` manager (#61) | ✓ |
| 5 | — | Relicense to Apache-2.0 | ✓ |
| 6 | 27 | Reconcile — Auto-Track and `--dry-run` | ✓ |
| 6 | 28 | Reinstall — Support Pre-Existing Packages | ✓ |
| 6 | 29 | Flag and Compliance Updates | ✓ |
| 6 | 30 | `stamp auto-reconcile` Command | ✓ |
| 4 | 32 | APT package manager adapter (#46) | ✓ |
| 4 | 33 | Docker-based integration testing | ✓ |
| 4 | 34 | Post-release integration CI pipelines (ubuntu/debian/fedora) | ✓ |
| 4 | 35 | Zypper package manager adapter (openSUSE) (#124) | ✓ Complete |
| 4 | 36 | Snap package manager adapter (#47) | ✓ Complete |
| 4 | 37 | Pacman package manager adapter (Arch Linux) (#49) | ✓ Complete |
| 4 | 38 | MacPorts package manager adapter (macOS) (#48) | ✓ Complete |
| 6 | 39 | Go adapter code review fixes | ✓ Complete |
| 6 | 39a | goBinDir: GOBIN + multi-entry GOPATH resolution | ✓ |
| 6 | 39b | Remove: os.Stat + os.Remove instead of exec rm | ✓ |
| 6 | 39c | ListInstalled: recover module paths via go version -m | ✓ |
| 6 | 39d | Update: batch reinstall for recoverable module paths | ✓ |
| 6 | 39e | Search: error instead of fake results, warning on stderr | ✓ |
| 6 | 39f | CLI validation: delegate to ValidatePackageForManager | ✓ |
| 6 | 39g | Integration tests (unit coverage only — Go not in Docker) | ✓ |
| 6 | 39h | Docs: SPEC, FEATURE_MATRIX, usage pages, landing page | ✓ |
| 6 | 40 | Pipx adapter (Python CLI tools via pipx) | ✓ Complete |
| 6 | 40a | pipx.go: all Adapter methods, JSON + text list parsing | ✓ |
| 6 | 40b | pipx_test.go: table-driven tests for all operations | ✓ |
| 6 | 40c | Detection in root.go + docs | ✓ |
| 6 | 41 | Uv adapter (Python CLI tools via uv tool) | ✓ Complete |
| 6 | 41a | uv.go: all Adapter methods, uv tool list parsing | ✓ |
| 6 | 41b | uv_test.go: table-driven tests for all operations | ✓ |
| 6 | 41c | Detection in root.go + docs | ✓ |
| 6 | 42 | Two-phase update check + confirm | ~ Planned |
| 6 | 42a | CheckUpdate interface + UpdateInfo type | ~ |
| 6 | 42b | 12 adapter implementations (CheckUpdate) | ~ |
| 6 | 42c | CLI update.go rework with --check and prompt | ~ |
| 6 | 42d | ADR-011: update check + confirm | ✓ |
| 6 | 42e | Docs: SPEC, FEATURE_MATRIX, usage/update.md | ✓ |
| 7 | 43 | Clean SIGINT abort for all commands (#170) | ✓ |
| 8 | 44 | Fail-closed consent for destructive commands (#168) | ✓ |
| 8 | 44a | Native transaction preview (`Previewer`: dnf/apt/pacman/brew/flatpak/zypper/npm) | ✓ |
| 8 | 44b | `WithYes` consent context + `ErrConfirmationRequired` in all adapters | ✓ |
| 8 | 44c | CLI gate: install/remove/reinstall/restore/update/autoremove/clean/hold/unhold/repo | ✓ |
| 8 | 44d | ADR-015: fail-closed consent model | ✓ |
| 8 | 45 | Unified ADR-011-style preview contract (#168) | ✓ |
| 8 | 45a | `Preview{Output,Noop}` + combined-output dry-runs | ✓ |
| 8 | 45b | Adapter-owned no-op (remove/reinstall via own `ListInstalled`) + warn-and-prompt | ✓ |
| 8 | 45c | dnf `makecache` refresh + `CheckUpdate` unrecognized-output guards | ✓ |
| 8 | 45d | ADR-016: unified preview contract | ✓ |
| 9 | 46 | Docs site: live latest-version pill in nav (#171) | ✓ |
| 9 | 46a | `main.js` GitHub release fetch + 24h cache + graceful hide | ✓ |
| 9 | 46b | Nav hook (`#version-pill`) + pill CSS | ✓ |

### Phase 8: Fail-Closed Consent for Destructive Commands (#168)

Deliver native package-manager UX (preview → prompt, `-y` to skip) for all destructive commands, with fail-closed behavior in non-interactive environments.

**Task 44: Fail-Closed Consent for Destructive Commands (#168)**
*   **Description:** Shared CLI confirmation gate (`confirmDestructive`/`requireConsent` in `internal/cli/confirm.go`): refresh → native preview → prompt (`[y/N]`, default no) → run, with `-y/--yes` skipping everything. Non-interactive runs without `-y` abort. Destructive adapter methods require `manager.WithYes(ctx)` and return `ErrConfirmationRequired` otherwise (fail closed at the privileged boundary). New optional `manager.Previewer` interface gives read-only dry-run previews (dnf `--assumeno`, apt `--assume-no`, pacman `--print`, brew/flatpak/zypper `--dry-run`, npm `--dry-run`); managers without a native dry-run fall back to `Info`. `stamp restore` gains the prompt it previously only documented; `stamp update` prompt is now fail-closed (no silent CI proceed). Documented in ADR-015.
*   **Acceptance:** `stamp reinstall htop -m dnf` shows a transaction preview and prompts; decline installs nothing. `-y` skips refresh/preview/prompt. Non-TTY without `-y` aborts for every destructive command. Adapters refuse destructive calls without consent.
*   **Verify:** `task check` passes; manual: `echo | stamp install htop` → "aborted", no install.
*   **Files:** `internal/cli/confirm.go` (new), `internal/cli/confirm_test.go` (new), `internal/cli/confirm_cmd_test.go` (new), `internal/cli/require_consent_test.go` (new), `internal/manager/consent_test.go` (new), `internal/manager/preview_test.go` (new), all `internal/manager/*.go` adapters, `internal/manager/mock.go`, `internal/cli/{install,reinstall,restore,restore_exec,update,autoremove,clean,hold,repo}.go`, `docs/decisions/ADR-015-fail-closed-consent.md`, README, `docs/project/spec.md`
*   **Status:** ✓ Completed

### Phase 8b: Unified Preview Contract (ADR-011 model)

Standardize install/remove/reinstall previews on the typed, adapter-owned model `update` already uses (ADR-011), removing CLI heuristics.

**Task 45: Unified ADR-011-style preview contract (#168)**
*   **Description:** Replace the raw `(string, error)` previews with `manager.Preview{Output, Noop}` returned by the adapters; combined stdout+stderr dry-runs (`WithCombinedOutput` → `cmd.CombinedOutput`) fix dnf5's stderr transaction UI. Adapters own all no-op decisions (remove/reinstall check their own `ListInstalled`); the CLI renders verbatim, warns-and-prompts on preview error (no `Info` fallback), and fails fast on `Noop`. Update hardening: dnf/yum `Refresh` runs `makecache`; dnf/apt/pacman `CheckUpdate` surface unrecognized output (`parser may be outdated`). Documented in ADR-016.
*   **Acceptance:** `stamp remove -m dnf htop` and `stamp reinstall -m dnf htop` preview the real native transaction; install no-op fails fast; preview errors warn-and-prompt; `stamp update` refresh is real for dnf.
*   **Verify:** `task check` passes; manual smokes for remove/reinstall/install/update.
*   **Files:** `internal/manager/{exec,manager,mock,dnf,apt,pacman,brew,flatpak,zypper,npm}.go`, `internal/cli/confirm.go`, manager/cli tests, `docs/decisions/ADR-016-unified-preview-contract.md`, `docs/decisions/ADR-015-fail-closed-consent.md`, `docs/project/spec.md`
*   **Status:** ✓ Completed

### Phase 9: Docs Site Live Version Indicator (#171)

**Task 46: Live latest-version pill in nav (#171)**
*   **Description:** Show the latest release tag on every docs page via a nav-bar text pill. `main.js` fetches `https://api.github.com/repos/rossijonas/stamp/releases/latest`, reads `tag_name`, renders it into `#version-pill` with `textContent` (no injection surface), and caches it in `localStorage` for 24h to stay under GitHub's 60 req/hr unauthenticated limit. On any failure (non-2xx, rate limit, malformed payload, storage unavailable) the pill is hidden — the static site degrades gracefully. Text pill only (nav), shields.io/badge row deferred.
*   **Acceptance:** All pages show `vX.Y.Z` in the nav; blocking the API hides the pill without breaking the site; `task check` and Jekyll build remain green.
*   **Verify:** `bundle exec jekyll build`; manual `jekyll serve` (pill renders, hides when API blocked).
*   **Files:** `docs/assets/js/main.js`, `docs/_includes/nav.html`, `docs/assets/css/style.css`
*   **Status:** ✓ Completed

### Phase 10: Manifest Origin Segmentation, Backup Rotation & Config Generation (#177)

Manifest entries gain an `origin` provenance field (`stamped`/`reconciled`),
backup rotation adopts a logrotate-aligned 3-axis retention policy
(count / min-age floor / max-age ceiling), and `stamp init` generates a
`config.toml` template when absent. Full spec: `docs/project/spec.md`
(Data Model + Backup Retention Policy). ADR: `docs/decisions/ADR-017-manifest-origin-and-backup-rotation.md`.

**Task 47: Backup config model (#177)**
*   **Description:** Add `BackupConfig` with 8 fields to `internal/cli/config.go`: `max_manifest_backups`, `min_manifest_backups`, `max_snapshot_backups`, `min_snapshot_backups`, `min_manifest_backup_age_days`, `max_manifest_backup_age_days`, `min_snapshot_backup_age_days`, `max_snapshot_backup_age_days`. Defaults `10/3/10/3/7/30/7/30`; `0` = unlimited per axis. `DefaultBackupConfig()` and commented `DefaultConfigTOML()` template (written only if absent, non-fatal). `LoadConfig` merges `[backup]` onto defaults. `ManifestPolicy()`/`SnapshotPolicy()` convert to `backup.Policy`.
*   **Acceptance:** `LoadConfig` returns defaults when `[backup]` absent; parses explicit values; `0`/absent = unlimited per axis.
*   **Verify:** `go test ./internal/cli/ -run Config` ; `task check`.
*   **Files:** `internal/cli/config.go`, `internal/cli/config_test.go`
*   **Status:** ✓ Completed

**Task 48: Origin field on manifest (#177)**
*   **Description:** Add optional `Origin string toml:"origin,omitempty"` to `manifest.Package` and `manifest.Repository`; constants `OriginStamped = "stamped"`, `OriginReconciled = "reconciled"`; accessor `OriginEffective()` returning `stamped` when empty. `Load`/`Save` round-trip without migration.
*   **Acceptance:** Old manifests load with effective origin `stamped`; explicit values preserved on save.
*   **Verify:** `go test ./internal/manifest/` (100% coverage on new funcs).
*   **Files:** `internal/manifest/manifest.go`, `internal/manifest/manifest_test.go`
*   **Status:** ✓ Completed

**Task 49: Origin write sites (#177)**
*   **Description:** Set origin at the 4 manifest-mutation sites: `install.go` and `reinstall.go` (existing branch) → `OriginStamped`; `repo.go` (`repo add`) → `OriginStamped`; `reconcile_report.go` (`saveAndTrack`) → `OriginReconciled`. `override.go` and `tap.go` do not mutate the manifest and are untouched.
*   **Acceptance:** `stamp install`/`stamp repo add` produce `origin = "stamped"`; `stamp reconcile` drift produces `origin = "reconciled"` in the manifest.
*   **Verify:** Golden-file CLI tests per command; `task check`.
*   **Files:** `internal/cli/install.go`, `internal/cli/reinstall.go`, `internal/cli/repo.go`, `internal/cli/reconcile_report.go` + tests
*   **Status:** ✓ Completed

**Task 50: Manifest backup + rotation (#177)**
*   **Description:** `manifest.CopyBackup(path)` copy-based atomic timestamped backup (original kept). `manifest.RotateBackups(path, policy BackupConfig) (int, error)` globs `<path>.<TS>.bak`, `time.Parse` sorts, applies retention precedence (min-age floor protect → **min-count floor keep ≥ N** → max-age ceiling delete → count cap trim, shared deletion budget of `len(entries)-MinKeep`), uses `os.Remove`. `0` axes = unlimited. No backups present = no-op. Retention logic lives in the shared `internal/backup` package (`Policy{MaxKeep,MinKeep,MinAge,MaxAge}` + `Rotate`), used by both manifest and state.
*   **Acceptance:** Table tests cover: protected-by-min-age, min-count floor survives all-ancient ceiling, min > max precedence, max-age ceiling, count cap, `0` axes, none-present, mixed. 100% coverage.
*   **Verify:** `go test ./internal/manifest/` with race detector.
*   **Files:** `internal/manifest/manifest.go`, `internal/manifest/manifest_test.go`
*   **Status:** ✓ Completed

**Task 51: Snapshot backup rotation (#177)**
*   **Description:** `state.RotateSnapshotBackups(snapDir string, policy BackupConfig) (int, error)` globs `snapshots.*.bak/` dirs, same retention precedence, uses `os.RemoveAll`. `0` axes = unlimited.
*   **Acceptance:** Table tests mirror Task 50 for directories. 100% coverage.
*   **Verify:** `go test ./internal/state/` with race detector.
*   **Files:** `internal/state/state.go`, `internal/state/state_test.go`
*   **Status:** ✓ Completed

**Task 52: Reconcile backup hooks (#177)**
*   **Description:** In `reconcile.go`, non-dry-run: `CopyBackup(manifestPath)` before diff; after save `RotateBackups(manifestPath, cfg.Backup)`. Dry-run short-circuits (no backup/rotation/write). `Long` text documents rotation.
*   **Acceptance:** Reconcile drift produces a manifest backup and prunes per policy; `--dry-run` produces no backup/rotation. Full path coverage (high-risk command).
*   **Verify:** `task check`; manual reconcile smoke on a scratch `~/.config/stamp`.
*   **Files:** `internal/cli/reconcile.go`, `internal/cli/reconcile_test.go`
*   **Status:** ✓ Completed

**Task 53: Init config gen + rotation (#177)**
*   **Description:** Fresh init writes `DefaultConfigTOML()` to `config.toml` if absent (non-fatal, `0600`). Re-init confirmed: existing `manifest.Backup()` (rename) + `state.BackupSnapshots()` (rename), then `RotateBackups` + `RotateSnapshotBackups`. Dry-run short-circuits. `Long` text documents config generation + rotation.
*   **Acceptance:** Fresh init creates `config.toml` with `[backup]`; existing config untouched; re-init rotates manifest + snapshot backups; dry-run writes nothing.
*   **Verify:** `task check`; manual init/re-init on scratch dirs.
*   **Files:** `internal/cli/init.go`, `internal/cli/init_test.go`
*   **Status:** ✓ Completed

**Task 54: Docs + ADR + AGENTS.md fix (#177)**
*   **Description:** Spec updated (origin field, provenance subsection, Backup Retention Policy, init/reconcile bullets). ADR-017 records the 3 structural decisions (origin enum + absence=stamped; copy-vs-rename for reconcile; logrotate-aligned 3-axis retention + floor trade-off). `AGENTS.md` stale `docs/SPEC.md` → `docs/project/spec.md`. `docs/usage/configuration.md` gains `[backup]` + logrotate mapping + auto-create note. Run `task docs` to regenerate CLI reference.
*   **Acceptance:** Docs consistent with spec; ADR-017 present; no stale `docs/SPEC.md` references remain.
*   **Verify:** `grep -r "docs/SPEC.md" .` returns nothing; `task docs`.
*   **Files:** `docs/project/spec.md`, `docs/IMPLEMENTATION_PLAN.md`, `AGENTS.md`, `docs/usage/configuration.md`, `docs/decisions/ADR-017-manifest-origin-and-backup-rotation.md`, `docs/usage/` (generated)
*   **Status:** ✓ Completed

**Task 55: Quality gates (#177)**
*   **Description:** Run `task check` (lint, race tests ≥90% coverage, govulncheck); fix until green. Confirm no file exceeds 1,000 lines and errors are wrapped/logged-OR-returned.
*   **Acceptance:** `task check` passes.
*   **Verify:** `task check`
*   **Files:** — (validation only)
*   **Status:** ✓ Completed

### Phase 11: `stamp list --type` Origin & Entity Filtering (#178)

**Task 56: `--type` flag + validation + filters (#178)**
*   **Description:** Add `--type, -t` flag to `stamp list`/`ls`. `validateListType` accepts `packages`, `repos`, `stamped`, `reconciled`, `stamped-packages`, `stamped-repos`, `reconciled-packages`, `reconciled-repos` (empty = default) and rejects anything else with the valid-types list. `filterPackages`/`filterRepositories` filter by manager and/or origin, using `OriginEffective()` so pre-origin manifests default to `stamped`.
*   **Acceptance:** All 8 values + `""` accepted; invalid value returns `unknown type "<v>"; valid types: ...`; origin filter treats absent `origin` as `stamped`.
*   **Verify:** `go test ./internal/cli/ -run 'TestValidateListType|TestFilterPackages|TestFilterRepositories'`
*   **Files:** `internal/cli/list.go`, `internal/cli/list_test.go`
*   **Status:** ✓ Completed

**Task 57: Dispatch + rendering (#178)**
*   **Description:** Switch dispatch per type: package-only, repo-only, and combined (`stamped`/`reconciled`) views. TTY renders `name (manager) — note` for packages (note rendering now matches docs) and `name (manager) url` for repos. JSON keeps capitalized keys; combined emits a flat array of package+repo objects; empty combined prints `nothing tracked`.
*   **Acceptance:** `stamp list` default output unchanged except notes render; `-t repos` matches `stamp repo list` format; combined prints packages before repos; `-t` × `-m` compose; `-t` × `-j` compose.
*   **Verify:** `go test ./internal/cli/ -run 'TestListCmd_Type|TestListCmd_NoteRendering|TestListCmd_TypePackagesEqualsDefault'`
*   **Files:** `internal/cli/list.go`, `internal/cli/list_test.go`
*   **Status:** ✓ Completed

**Task 58: Shell completion + cobra race mitigation (#178)**
*   **Description:** `RegisterFlagCompletionFunc("type", ...)` returns the 8 values. Cobra's completion registry is process-global and `prepareCustomAnnotationsForFlags` writes flag annotations under a read lock (upstream bug, unfixed through v1.10.2) — a data race between concurrent completion generation and command execution. Production is unaffected (single-process CLI); the test suite mitigates it by keeping all completion-generating tests (`stamp completion`/`setup`/`hello`) non-parallel so they run in the sequential phase before the parallel batch. Cobra upgraded to v1.10.2 alongside (pflag 1.0.9, yaml migration).
*   **Acceptance:** `completion` tests race-clean under `-race`; `-t` completes the 8 values; completion-generating tests are non-parallel with the constraint documented.
*   **Verify:** `go test -race ./internal/cli/`
*   **Files:** `internal/cli/list.go`, `internal/cli/completion_test.go`, `internal/cli/hello_test.go`, `go.mod`, `go.sum`
*   **Status:** ✓ Completed

**Task 59: Docs + regenerated CLI refs (#178)**
*   **Description:** `docs/usage/listing-packages.md` gains "Understanding Origins" + `--type` examples + corrected capitalized JSON keys; `docs/getting-started/index.md` and `README.md` gain a stamped/reconciled explanation; `docs/project/spec.md` updates the `stamp list` row and adds a List Command section; `docs/project/features.md` gains the `--type` flag row. `task docs` regenerates `docs/usage/stamp_list.md` and man pages.
*   **Acceptance:** Docs consistent with behavior; generated CLI reference shows `--type`.
*   **Verify:** `task docs`; `grep --type docs/usage/stamp_list.md`
*   **Files:** `docs/project/spec.md`, `docs/project/features.md`, `docs/usage/listing-packages.md`, `docs/getting-started/index.md`, `README.md`, `docs/usage/` + `docs/man/` (generated)
*   **Status:** ✓ Completed

**Task 60: Quality gates (#178)**
*   **Description:** Run `task check` (lint, race tests ≥90% coverage, govulncheck); fix until green.
*   **Acceptance:** `task check` passes.
*   **Verify:** `task check`
*   **Files:** — (validation only)
*   **Status:** ✓ Completed

### Phase 12: `stamp manifest history` + `stamp manifest diff` (#179)

**Task 61: backup.List + Rotate refactor (#179)**
*   **Description:** Export `backup.List(pattern) ([]Entry, error)` returning parsed timestamped backups (with collision index), newest first. Extend `tsPattern` to capture the collision suffix. Refactor `Rotate` to consume `List` (behavior unchanged).
*   **Acceptance:** `List` parses timestamps + collision suffixes, sorts newest first, skips invalid names; existing `TestRotate_*` stay green.
*   **Verify:** `go test ./internal/backup/ -race`
*   **Files:** `internal/backup/backup.go`, `internal/backup/backup_test.go`
*   **Status:** ✓ Completed

**Task 62: manifest history subcommand (#179)**
*   **Description:** `stamp manifest history` lists current + backups newest-first with package/repo counts, short SHA-256 hash, `*` = current, `(unchanged)` marker, TTY + JSON. Corrupted backups skipped with warning; missing manifest errors `manifest not found; run stamp init first`.
*   **Acceptance:** TTY/JSON output shapes match spec; no-backups hint; unchanged marker; corrupted skip.
*   **Verify:** `go test ./internal/cli/ -run 'TestManifestHistory'`
*   **Files:** `internal/cli/manifest.go`, `internal/cli/manifest_test.go`
*   **Status:** ✓ Completed

**Task 63: manifest diff subcommand + filters (#179)**
*   **Description:** `stamp manifest diff [ts|hash]` compares current vs a backup (default most recent) for packages AND repos (name+manager identity). Arg resolution: empty → newest; pure hex ≥6 → hash prefix (ambiguous → error); else timestamp (human or compact). `--manager, -m` + `--origin` filters both sets. TTY `+/-` + summary; JSON with `kind` field.
*   **Acceptance:** all edge cases (no backup, invalid/unknown/ambiguous ref, corrupted baseline, empty diff, filters) covered by tests.
*   **Verify:** `go test ./internal/cli/ -run 'TestManifestDiff'`
*   **Files:** `internal/cli/manifest.go`, `internal/cli/manifest_test.go`
*   **Status:** ✓ Completed

**Task 64: manifest group registration + `--origin` completion (#179)**
*   **Description:** Register `newManifestCmd()` in `root.go` after `newListCmd()`; `--origin` flag completion (`stamped`/`reconciled`) registered like `--type` (see Task 58 for the cobra race mitigation).
*   **Acceptance:** `manifest` group registered; `--origin` completes; completion tests race-clean.
*   **Verify:** `go test -race ./internal/cli/`
*   **Files:** `internal/cli/manifest.go`, `internal/cli/root.go`, `internal/cli/manifest_test.go`
*   **Status:** ✓ Completed

**Task 65: Docs + regenerated CLI refs (#179)**
*   **Description:** New `docs/usage/managing-manifests.md` (history/diff/restore/origins); linked from `docs/usage/index.md` under Recovery; `docs/project/spec.md` command row + Manifest Management subsection; `docs/project/features.md` rows. `task docs` regenerates `docs/usage/stamp_manifest*` and man pages.
*   **Acceptance:** Docs consistent with behavior; generated CLI reference shows `manifest history`/`diff`.
*   **Verify:** `task docs`; `grep manifest docs/usage/stamp.md`
*   **Files:** `docs/usage/managing-manifests.md`, `docs/usage/index.md`, `docs/project/spec.md`, `docs/project/features.md`, `docs/IMPLEMENTATION_PLAN.md`, `docs/usage/` + `docs/man/` (generated)
*   **Status:** ✓ Completed

### Phase 13: sysexits Exit Codes (#177/#178/#179)

**Task 66: exit.go + error mapping (#177/#178/#179)**
*   **Description:** New `internal/cli/exit.go`: category sentinels (`ErrUsage`, `ErrData`, `ErrNoInput`, `ErrUnavailable`, `ErrCanTCreate`, `ErrConfig`), `categorizedError` (preserves message, matches via `Is`), `catErr`, `exitCodeFor`. `Execute()` maps via `exitCodeFor` (default 1, POSIX catchall); `SetFlagErrorFunc` → 64. Map sites: config parse 78, manifestErr 65, saveManifest 73, list `--type` 64, manifest history/diff (78/64/66/65), reconcile (69), init (73), repo validation (64).
*   **Acceptance:** Every mapped error exits its sysexits code; message text unchanged; flag-parse errors → 64.
*   **Verify:** `go test ./internal/cli/ -run 'TestExitCodeFor|TestCategorizedError|TestFlagParseErrorIsUsage'`
*   **Files:** `internal/cli/exit.go`, `internal/cli/exit_test.go`, `root.go`, `list.go`, `manifest.go`, `reconcile.go`, `reconcile_report.go`, `init.go`, `repo.go`
*   **Status:** ✓ Completed

**Task 67: Docs (exit codes) (#177/#178/#179)**
*   **Description:** `docs/project/spec.md` "Exit Codes" section — sysexits table + category mapping + default-1 note + non-fatal backup failures.
*   **Acceptance:** Spec documents the public exit-code contract.
*   **Verify:** `grep -n "Exit Codes" docs/project/spec.md`
*   **Files:** `docs/project/spec.md`
*   **Status:** ✓ Completed

**Task 68: Quality gates (exit codes) (#177/#178/#179)**
*   **Description:** `task check` green; confirm error messages unchanged (existing string assertions pass) and no dead sentinels.
*   **Acceptance:** `task check` passes.
*   **Verify:** `task check`
*   **Files:** — (validation only)
*   **Status:** ✓ Completed

### Phase 14: Missing-package drift visibility (#182)

Manifest packages removed via the native manager (e.g. `dnf remove`) become
visible: `stamp reconcile` warns on stderr, `stamp doctor` reports them under
"Manifest Integrity", and `stamp ls --type missing` lists them. Warning-only —
no mutation, no auto-reinstall; `stamp restore` remains the convergence tool.
Full spec: `docs/project/spec.md` (List Command / §`stamp reconcile` /
§`stamp doctor`). ADR: `docs/decisions/ADR-018-missing-package-drift-warning.md`.

**Task 69: Shared missing helpers (#182)**
*   **Description:** New `internal/cli/missing.go`: `missingFromSystem(ctx, adapters, m)` (concurrent, error-tolerant per-manager `ListInstalled`), `missingFromDeltas(deltas, m)` (intersect `Delta.Removed` with manifest), `dedupeAndSortMissing`. Both exclude `Group`/`Cask` entries and return `[]manifest.Package` for reuse of list rendering/filtering.
*   **Acceptance:** Table tests: found / none / list-error-skipped / inactive-skipped / group-cask excluded / nil manifest / dedupe / sort / multi-manager. 100% coverage.
*   **Verify:** `go test ./internal/cli/ -run 'TestMissing|TestDedupeAndSort'` with race detector.
*   **Files:** `internal/cli/missing.go`, `internal/cli/missing_test.go`
*   **Status:** ✓ Completed

**Task 70: `stamp ls --type missing` (#182)**
*   **Description:** Add `missing` to `validListTypes` in `list.go`; RunE branch queries `missingFromSystem`, applies `-m` filter, renders via `renderPackages` (`--json` supported); empty → `no missing packages`. Completion includes `missing`.
*   **Acceptance:** TTY + JSON + `-m` filter + empty-manifest + list-error-skipped paths covered.
*   **Verify:** `go test ./internal/cli/ -run 'TestListCmd_TypeMissing|TestListTypeCompletion'`
*   **Files:** `internal/cli/list.go`, `internal/cli/list_test.go`
*   **Status:** ✓ Completed

**Task 71: Reconcile missing warning (#182)**
*   **Description:** `reconcile.go` computes `missingFromDeltas` after `DiffAll`; `renderMissing` (new, `reconcile_display.go`) prints `N manifest package(s) not installed: name (mgr), ...` with `stamp ls --type missing` / `stamp restore` hint (names inline ≤ 5). Fires on no-drift and drift paths, including `--dry-run`; snapshot still saved; tracking unchanged.
*   **Acceptance:** Full path coverage (high-risk command): no-drift+missing, drift+missing, dry-run, none-missing unchanged, group skip, render branches.
*   **Verify:** `go test ./internal/cli/ -run 'TestReconcile|TestRenderMissing'` with race detector.
*   **Files:** `internal/cli/reconcile.go`, `internal/cli/reconcile_display.go`, `internal/cli/reconcile_test.go`
*   **Status:** ✓ Completed

**Task 72: Doctor missing section (#182)**
*   **Description:** `manifestStatus` gains `Missing []manifest.Package json:"missing,omitempty"`; full `doctor` run populates it via `missingFromSystem` when the manifest is valid; TTY renders a `Missing:` block under "Manifest Integrity"; `--json` emits `manifest.missing`. Manifest `✓ Healthy` status unchanged (missing ≠ invalid).
*   **Acceptance:** TTY + JSON + list-error-skipped + none-missing paths covered; existing `TestDoctor_Manifest_Healthy` regression green.
*   **Verify:** `go test ./internal/cli/ -run 'TestDoctor'` with race detector.
*   **Files:** `internal/cli/doctor.go`, `internal/cli/doctor_display.go`, `internal/cli/doctor_test.go`
*   **Status:** ✓ Completed

**Task 73: Docs + ADR (#182)**
*   **Description:** Spec updates (§list `missing` row + system-aware note, §`stamp reconcile` warning bullet, §`stamp doctor` manifest-vs-system bullet); `docs/usage/listing-packages.md`, `docs/usage/reconcile.md`, `docs/usage/doctor.md` examples; new ADR-018. No cobra help-text change → generated refs unaffected.
*   **Acceptance:** Docs consistent with behavior; `grep missing docs/usage/listing-packages.md` hits.
*   **Verify:** `task docs`; review rendered pages.
*   **Files:** `docs/project/spec.md`, `docs/usage/listing-packages.md`, `docs/usage/reconcile.md`, `docs/usage/doctor.md`, `docs/decisions/ADR-018-missing-package-drift-warning.md`, `docs/IMPLEMENTATION_PLAN.md`
*   **Status:** ✓ Completed

**Task 74: Quality gates (#182)**
*   **Description:** `task check` green (lint + vet + tests + race + coverage ≥90%).
*   **Acceptance:** `task check` passes.
*   **Verify:** `task check`
*   **Files:** — (validation only)
*   **Status:** ☐ Pending
