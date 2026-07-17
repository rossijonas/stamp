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
*   **Status:** ✅ Completed

**Task 2: Manifest Manager (TOML)**
*   **Description:** Implement `internal/manifest` to read, write, and manipulate `manifest.toml`, including support for the `notes` field.
*   **Acceptance:** Can serialize/deserialize TOML and add/remove packages and repositories.
*   **Verify:** `task test` passes for `internal/manifest`.
*   **Status:** ✅ Completed

**Task 2.5: Pre-requisite Fixes (Security & CI)**
*   **Description:** Upgrade the project toolchain to Go 1.26 to resolve standard library CVEs identified by `govulncheck`. Fix the duplicate authorization header bug in the GitHub Actions pipeline.
*   **Acceptance:** `ci.yml` uses native `go run govulncheck` to prevent double checkout. Project builds with Go 1.26.
*   **Verify:** GitHub Actions pipeline passes cleanly.
*   **Status:** ✅ Completed

### Phase 2: The Active Wrapper Flow
Build the ability to actually modify the system (Install, Remove, Search) as the primary usage model.

**Task 3: Package Manager Interfaces & Mocks**
*   **Description:** Define the `PackageManager` interface. Implement `MockManager` for testing.
*   **Acceptance:** Interface defines `Name()`, `ListInstalled()`, `Install()`, `Remove()`, `Search()`, `AddRepo()`, `RemoveRepo()`.
*   **Verify:** `task test` passes for `internal/manager` mocks.
*   **Status:** ✅ Completed

**Task 4: Native Adapters (Write Operations)**
*   **Description:** Implement `Install()`, `Remove()`, `Search()`, `AddRepo()`, and `RemoveRepo()` for `dnf`, `brew`, and `flatpak`.
*   **Acceptance:** Adapters can execute system and repository modifications.
*   **Verify:** Tests pass.
*   **Status:** ✅ Completed

**Task 5: Active CLI Commands**
*   **Description:** Wire up `stamp install/add`, `stamp remove/uninstall/delete/del`, and `stamp search` in Cobra. Implement the `stamp repo` command group. Ensure aliases are properly registered using Cobra's `Aliases` array, and the 3-tier resolution engine parses `config.toml` precedence and regex-based matching rules. Supports the global `--yes` / `-y` flag.
*   **Acceptance:** Users can install packages and repositories via `stamp`, updating the manifest automatically.
*   **Verify:** Manual test of `stamp install <test-pkg>`
*   **Status:** ✅ Completed

### Phase 3: The Safety Net Flow
Build the read-only safety net: checking the system state and calculating the delta.

**Task 6: Native Adapters (Read-Only)**
*   **Description:** Implement `ListInstalled()` for `dnf`, `brew`, and `flatpak` using abstracted shell execution (`os/exec`).
*   **Acceptance:** Adapters correctly parse `dnf repoquery`, `brew leaves`, and `flatpak list`.
*   **Verify:** Unit tests pass using mocked string outputs.
*   **Status:** ✅ Completed

**Task 7: State Engine (Snapshotting)**
*   **Description:** Implement `internal/state` to save JSON snapshots and calculate deltas (Added/Removed) against the current `PackageManager` outputs.
*   **Acceptance:** Engine can accurately report which packages were added since the last snapshot.
*   **Verify:** `task test` passes with 100% coverage on diffing logic.
*   **Status:** ✅ Completed

**Task 8: The `reconcile` Command (Cobra)**
*   **Description:** Wire up `cmd/stamp/main.go` and `internal/cli/reconcile.go`. Supports the `--yes` / `-y` flag to auto-track detected packages without prompting.
*   **Acceptance:** Running `stamp reconcile` fetches the state, calculates the delta, and prompts the user (or auto-tracks) to add new packages to the manifest.
*   **Verify:** Manual test: `go run cmd/stamp/main.go reconcile`
*   **Status:** ✅ Completed

### Phase 4: Restore & UNIX Compliance
Build the environment reconstruction logic and final touches.

**Task 9: The `restore` Command**
*   **Description:** Implement the environment reconstruction logic. Supports the `--yes` / `-y` flag to bypass safety confirmation prompts.
*   **Acceptance:** `stamp restore` parses the manifest, restores all tracked repositories first, and then executes concurrent package installs.
*   **Verify:** Manual test with `--dry-run` flag.
*   **Status:** ✅ Completed

**Task 10: CLI Polish and Documentation**
*   **Description:** Implement `stamp doctor`, `stamp completion`, `stamp man`, NO_COLOR compliance, doc generation pipeline, landing page, and flag standardization.
*   **Status:** ✅ Completed

#### Task 10 Subtasks

| Subtask | Description | Status |
| :--- | :--- | :---: |
| 10a | `stamp doctor` command with TTY/JSON output | ✅ |
| 10b | `stamp completion` shell autocompletion (bash/zsh/fish/powershell) | ✅ |
| 10c | `stamp man` man page generation and install | ✅ |
| 10d | NO_COLOR compliance | ✅ |
| 10e | Doc generation pipeline (`task docs` + CI enforcement) | ✅ |
| 10f | Flag standardization (short forms, actions-as-subcommands) | ✅ |
| 10h | Uninstall documentation in README.md (standard + hard uninstall) | ✅ |

**Task 11: Self-Update Subcommand**
*   **Description:** Implement `stamp self-update/self-upgrade` that checks the current binary version against the GitHub releases API, downloads the latest binary for the host OS/arch, and replaces itself atomically. Supports `--check, -c` flag to query without downloading.
*   **Acceptance:** User can run `stamp self-update --check` to check, and `stamp self-update` to apply.
*   **Verify:** Unit tests mock the release API and verify binary swap logic.
*   **Status:** ⏳ Pending

**Task 12: `stamp hello` Welcome Command**
*   **Description:** Implement a welcome command that prints the ASCII logo, a brief project description, and suggests next steps for new users.
*   **Acceptance:** Running `stamp hello` displays logo, about text, and suggests `stamp init`, `stamp doctor`, `stamp man install`.
*   **Status:** ✅ Completed

**Task 13: `stamp info` Package Info Command**
*   **Status:** ✅ Completed

**Task 13: `stamp info` Package Info Command**
*   **Description:** Implement a command to show detailed package information across all package managers. Supports `--manager` flag to scope to a specific manager.
*   **Acceptance:** Running `stamp info htop` shows package details from all managers that have it.
*   **Status:** ✅ Completed

**Task 14: `stamp man check` Version Verification**
*   **Description:** Implement a subcommand within `stamp man` that verifies the installed man page version matches the stamp binary version.
*   **Acceptance:** Running `stamp man check` reports whether man pages are current, outdated, or missing.
*   **Status:** ✅ Completed

**Task 15: Per-Manager Flag Support**
*   **Description:** Add `--manager`, `-m` flag to `stamp list`, `stamp reconcile`, `stamp restore`, `stamp doctor`, and `stamp update` to scope operations to a single package manager.
*   **Status:** ⚠️ Partial

| Subtask | Description | Status |
| :--- | :--- | :---: |
| 15a | `stamp reconcile -m` | ✅ |
| 15b | `stamp restore -m` | ✅ |
| 15c | `stamp doctor -m` | ✅ |
| 15d | `stamp list -m` | ✅ (via Task 22) |
| 15e | `stamp update -m` | ⏳ Pending (Task 23) |

#### Phase 4c — Infrastructure

**Task 16: Multi-Platform Integration Testing**
*   **Description:** Add CI matrix testing across Fedora, Ubuntu, Arch Linux, macOS, and Windows using Docker containers and parallel pipeline jobs. Each environment runs the full test suite against real package managers.
*   **Acceptance:** CI passes on all target platforms for every PR.
*   **Verify:** Green CI status on all matrix jobs.
*   **Status:** 📝 Research needed

**Task 17: Package Manager Feature Audit**
*   **Description:** Audit each supported package manager for important features not yet covered by stamp. Specifically: Homebrew `cask` (GUI apps), `brew services`, `dnf groupinstall`, flatpak remotes management. Determine which are critical for adoption.
*   **Acceptance:** Documented findings with recommendations for each manager.
*   **Verify:** Report in docs/decisions/ or FEATURE_MATRIX.md.
*   **Status:** 📝 Research needed

**Task 18: `stamp reinstall` Command**
*   **Description:** Implement a reinstall command that looks up a package in the manifest, resolves its recorded manager, and executes the native reinstallation. No `-m` flag needed — manager resolved from manifest.
*   **Acceptance:** `stamp reinstall htop` reinstalls `htop` using the manager recorded in the manifest. Accepts global `-y`.
*   **Status:** ✅ Completed

**Task 19: Generate Missing Usage & Man Pages**
*   **Description:** Run `task docs` to auto-generate missing `docs/usage/` pages (`stamp_hello.md`, `stamp_info.md`, `stamp_reinstall.md`) and populate `docs/man/` with system man page files.
*   **Acceptance:** Every registered subcommand has a corresponding `docs/usage/*.md` page. `docs/man/stamp.1` exists and is up to date.
*   **Status:** ✅ Completed

#### Phase 4b — Medium Features

**Task 20: Create GitHub Pages Landing Page**
*   **Description:** Create `docs/index.html` as a custom landing page for GitHub Pages. Content requirements defined in SPEC.md → Project Landing Page. Source tagline and features from README.md.
*   **Acceptance:** Navigating to `https://rossijonas.github.io/stamp/` displays the project landing page.
*   **Status:** ⏳ Pending

#### Phase 4a — Quick Wins

**Task 21: `stamp init` Command**
*   **Description:** Initialize `manifest.toml` and take baseline snapshot of current system packages. Create XDG directories (`~/.config/stamp`, `~/.local/share/stamp/snapshots`). Suggested by `stamp hello` output.
*   **Acceptance:** Running `stamp init` creates config dir, snapshot dir, empty manifest.toml, and baseline snapshot for each available manager.
*   **Status:** ✅ Completed

**Task 22: `stamp list` Command (alias `ls`)**
*   **Description:** List all intentionally installed packages from the manifest. Supports `--json, -j` and `--manager, -m` flags.
*   **Acceptance:** Running `stamp list` prints tracked packages; `stamp list --json` outputs JSON; `stamp list -m brew` filters by manager.
*   **Status:** ✅ Completed

**Task 23: `stamp update` Command (alias `upgrade`)**
*   **Description:** Run system upgrades across all available managers in parallel. Supports `--manager, -m` flag to scope to a single manager.
*   **Acceptance:** Running `stamp update` executes native update/upgrade commands concurrently per manager.
*   **Status:** ⏳ Pending

**Task 24: Migrate `stamp hello` to `stamp setup` Wizard**
*   **Description:** Replace `stamp hello` with `stamp setup` interactive wizard. Keep `hello` as alias. Run completion, man install, init (with prompts, default Yes), then doctor (no prompt). Support `-y` flag for scripting.
*   **Acceptance:** `stamp setup -y` runs all steps without prompts. `stamp hello` continues to work as alias.
*   **Status:** ✅ Completed

**Task 25: Add Shell Completion Check to `stamp doctor`**
*   **Description:** Check common shell completion paths (bash, zsh, fish) and report status in doctor TTY and JSON output.
*   **Acceptance:** `stamp doctor` shows ✅ or ❌ for completions in both TTY and JSON modes.
*   **Status:** ⏳ Pending

**Task 25b: Re-init Guard for `stamp init` with Mandatory Backup**
*   **Description:** Add re-init guard to `stamp init`: detect existing manifest, warn user, prompt for confirmation (default No). On confirmation, **always** backup manifest + snapshots (`<path>.<ts>.bak`) before creating fresh state. Update `stamp setup` wizard to detect initialized state and adjust prompt wording. `-y` flag bypasses prompt. Backup runs unconditionally on confirmed re-init.
*   **Acceptance:** `stamp init` on initialized system shows warning, prompts with default No. Accepting creates timestamped backups and fresh state. Declining aborts cleanly. `-y` skips prompt. Wizard shows adjusted prompt when already initialized.
*   **Verify:** `task test` passes.
*   **Files:** `internal/cli/init.go`, `internal/cli/init_test.go`, `internal/cli/hello.go`, `internal/cli/hello_test.go`, `internal/manifest/manifest.go`, `internal/manifest/manifest_test.go`, `internal/state/state.go`, `internal/state/state_test.go`
*   **Status:** ✅ Completed

**Task 26: Add `yum` as Alias to `dnf` Manager**
*   **Description:** Automatically detect `yum` when `dnf` is unavailable (RHEL/CentOS 7). Use resolved command name for all exec calls.
*   **Acceptance:** `stamp` works on systems with only `yum` installed.
*   **Status:** ✅ Completed

### Phase 5: Project Licensing & Governance
Ensure maximum community and enterprise reach.

**Task 11: Relicense to Apache-2.0**
*   **Description:** Transition project license from AGPL-3.0 to Apache-2.0 to simplify integration and adoption. Update files and documentation.
*   **Acceptance:** LICENSE contains Apache-2.0 text, README links to correct license, and ADR-003 is merged.
*   **Verify:** `task check` passes.
*   **Status:** ✅ Completed

### Phase 6: Reconcile Behavior Stabilisation & Feature Completion

Deliver the final design for `stamp reconcile` and `stamp reinstall` based on real-world testing feedback.

**Task 27: Reconcile — Auto-Track and `--dry-run`**
*   **Description:** Remove interactive prompt from reconcile. Auto-track all discovered drift. Add `--dry-run` / `-d` flag for preview mode without saving manifest or snapshots. Fix snapshot save timing to persist on no-drift.
*   **Acceptance:** `stamp reconcile` auto-tracks without prompting. `stamp reconcile --dry-run` shows drift but does not save. `-y` accepted for backward compatibility (no-op). Snapshot updated on no-drift to accurately track subsequent removals.
*   **Verify:** `task test` passes, manual test of `--dry-run` flag.
*   **Files:** `internal/cli/reconcile.go`, `internal/cli/reconcile_test.go`
*   **Depends on:** Task 7 (state engine), Task 8 (reconcile command), Issue #39 (adapter fixes)
*   **Status:** ✅ Completed

**Task 28: Reinstall — Support Pre-Existing Packages**
*   **Description:** Extend `stamp reinstall <pkg>` to handle packages NOT in the manifest. Resolve manager via resolution engine, run native reinstall, append to manifest, save snapshot. Add `Reinstall()` to `Adapter` interface.
*   **Acceptance:** `stamp reinstall htop` works for both manifest-tracked and pre-existing (manifest-absent) packages. Pre-existing packages are recorded in manifest.
*   **Verify:** `task test` passes, manual test: install package outside stamp → `stamp init` → `stamp reinstall pkg` → `stamp list` shows it.
*   **Files:** `internal/cli/reinstall.go`, `internal/cli/reinstall_test.go`
*   **Depends on:** Task 27 (reconcile spec), Issue #39 (adapter fixes)
*   **Status:** ✅ Completed

**Task 29: Flag and Compliance Updates**
*   **Description:** Update global flag documentation to reflect reconcile's deterministic behavior. Ensure `--dry-run` is registered on reconcile and restore. Ensure docs are up to date.
*   **Acceptance:** `--dry-run` flag documented in usage and help. `-y` flag documented as backward-compatible no-op for reconcile. Auto-generated docs match code.
*   **Verify:** `task docs` generates correct usage pages.
*   **Files:** `docs/usage/stamp_reconcile.md`, `internal/cli/reconcile.go` (after code done)
*   **Depends on:** Task 27
*   **Status:** ✅ Completed

**Task 30: `stamp auto-reconcile` Command**
*   **Description:** Implement a subcommand to install or remove automated reconcile timers. On Linux, creates systemd user service + timer files in `~/.config/systemd/user/`. On macOS, creates launchd plist in `~/Library/LaunchAgents/`. Supports `--period`, `-p` flag (hourly/daily/weekly, default daily).
*   **Acceptance:** `stamp auto-reconcile on` installs the timer. `stamp auto-reconcile off` removes it. Timer runs `stamp reconcile` at the configured interval. Pre-configured timer files available in `contrib/`.
*   **Verify:** Manual test: `stamp auto-reconcile on --period daily` creates timer, `stamp auto-reconcile off` removes it.
*   **Files:** `internal/cli/autoreconcile.go`, `internal/cli/autoreconcile_test.go`, `contrib/systemd/stamp-reconcile.service`, `contrib/systemd/stamp-reconcile.timer`, `contrib/launchd/com.rossijonas.stamp.reconcile.plist`
*   **Depends on:** Task 27
*   **Status:** ⏳ Pending

### Phase & Task Progress Summary

| Phase | Task | Description | Status |
| :--- | :--- | :--- | :---: |
| 1 | 1 | Repository Scaffolding & Tooling | ✅ |
| 1 | 2 | Manifest Manager (TOML) | ✅ |
| 1 | 2.5 | Pre-requisite Fixes (Security & CI) | ✅ |
| 2 | 3 | Package Manager Interfaces & Mocks | ✅ |
| 2 | 4 | Native Adapters (Write Operations) | ✅ |
| 2 | 5 | Active CLI Commands | ✅ |
| 3 | 6 | Native Adapters (Read-Only) | ✅ |
| 3 | 7 | State Engine (Snapshotting) | ✅ |
| 3 | 8 | The `reconcile` Command | ✅ |
| 4 | 9 | The `restore` Command | ✅ |
| 4 | 10 | CLI Polish, Manpages, GitHub Pages & Landing Page | ⏳ |
| 4 | 10a | `stamp doctor` command | ✅ |
| 4 | 10b | `stamp completion` shell autocompletion | ✅ |
| 4 | 10c | `stamp man` generation and install | ✅ |
| 4 | 10d | NO_COLOR compliance | ✅ |
| 4 | 10e | Doc generation pipeline (task docs) | ✅ |
| 4 | 10f | Flag standardization (short forms, subcommands) | ✅ |
| 4 | 10h | Uninstall documentation in README.md | ✅ |
| 4 | 11 | Self-Update Subcommand | ⏳ |
| 4 | 12 | `stamp hello` welcome command | ✅ |
| 4 | 13 | `stamp info` package info command | ✅ |
| 4 | 14 | `stamp man check` version verification | ✅ |
| 4 | 15 | Per-manager flags for reconcile/restore/doctor/list | ⚠️ Partial |
| 4 | 16 | Multi-platform integration testing | 📝 |
| 4 | 17 | Package manager feature audit | 📝 |
| 4 | 18 | `stamp reinstall` command | ✅ |
| 4 | 19 | Generate missing usage & man pages | ✅ |
| 4 | 20 | Create GitHub Pages landing page | ⏳ |
| 4 | 21 | `stamp init` command | ✅ |
| 4 | 22 | `stamp list` command (alias `ls`) | ✅ |
| 4 | 23 | `stamp update` command (alias `upgrade`) | ⏳ |
| 4 | 24 | Migrate `stamp hello` to `stamp setup` wizard (#59) | ⏳ |
| 4 | 25 | Add shell completion check to `stamp doctor` (#60) | ⏳ |
| 4 | 26 | Add `yum` as alias to `dnf` manager (#61) | ✅ |
| 5 | — | Relicense to Apache-2.0 | ✅ |
| 6 | 27 | Reconcile — Auto-Track and `--dry-run` | ✅ |
| 6 | 28 | Reinstall — Support Pre-Existing Packages | ✅ |
| 6 | 29 | Flag and Compliance Updates | ✅ |
| 6 | 30 | `stamp auto-reconcile` Command | ⏳ Pending |
