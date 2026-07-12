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
| 10g | GitHub Pages landing page (`docs/index.html`) — content requirements defined in SPEC.md → Project Landing Page; source tagline and features from README.md | ⏳ |
| 10h | Uninstall documentation in README.md (standard + hard uninstall) | ✅ |

**Task 11: Self-Update Subcommand**
*   **Description:** Implement `stamp self-update/self-upgrade` that checks the current binary version against the GitHub releases API, downloads the latest binary for the host OS/arch, and replaces itself atomically. Supports a `--check` flag to query without downloading.
*   **Acceptance:** User can run `stamp self-update --check` to check, and `stamp self-update` to apply.
*   **Verify:** Unit tests mock the release API and verify binary swap logic.
*   **Status:** ⏳ Pending

**Task 12: `stamp hello` Welcome Command**
*   **Description:** Implement a welcome command that prints the ASCII logo, a brief project description, and suggests next steps for new users.
*   **Acceptance:** Running `stamp hello` displays logo, about text, and suggests `stamp init`, `stamp doctor`, `stamp man install`.
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
*   **Acceptance:** All applicable commands accept `-m` flag and limit operations to the specified manager.
*   **Status:** ✅ Completed

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
*   **Status:** ⏳ Pending

**Task 20: Create GitHub Pages Landing Page**
*   **Description:** Create `docs/index.html` as a custom landing page for GitHub Pages. Content requirements defined in SPEC.md → Project Landing Page. Source tagline and features from README.md.
*   **Acceptance:** Navigating to `https://rossijonas.github.io/stamp/` displays the project landing page.
*   **Status:** ⏳ Pending

### Phase 5: Project Licensing & Governance
Ensure maximum community and enterprise reach.

**Task 11: Relicense to Apache-2.0**
*   **Description:** Transition project license from AGPL-3.0 to Apache-2.0 to simplify integration and adoption. Update files and documentation.
*   **Acceptance:** LICENSE contains Apache-2.0 text, README links to correct license, and ADR-003 is merged.
*   **Verify:** `task check` passes.
*   **Status:** ✅ Completed
