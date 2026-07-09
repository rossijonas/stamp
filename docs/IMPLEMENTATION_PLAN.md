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
*   `docs/` - Architecture Decision Records (ADRs), specs, and plans.
*   `testdata/` - Static JSON/TOML fixtures for unit tests.

### Development Tools
We will use modern, Go-centric tooling:
*   **Taskfile (`task`)**: Replaces `Makefile`. YAML-based, self-documenting task runner (`task build`, `task test`, `task lint`).
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
*   **Status:** ⏳ Pending

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
*   **Status:** ⏳ Pending

**Task 8: The `reconcile` Command (Cobra)**
*   **Description:** Wire up `cmd/stamp/main.go` and `internal/cli/reconcile.go`. Supports the `--yes` / `-y` flag to auto-track detected packages without prompting.
*   **Acceptance:** Running `stamp reconcile` fetches the state, calculates the delta, and prompts the user (or auto-tracks) to add new packages to the manifest.
*   **Verify:** Manual test: `go run cmd/stamp/main.go reconcile`
*   **Status:** ⏳ Pending

### Phase 4: Restore & UNIX Compliance
Build the environment reconstruction logic and final touches.

**Task 9: The `restore` Command**
*   **Description:** Implement the environment reconstruction logic. Supports the `--yes` / `-y` flag to bypass safety confirmation prompts.
*   **Acceptance:** `stamp restore` parses the manifest, restores all tracked repositories first, and then executes concurrent package installs.
*   **Verify:** Manual test with `--dry-run` flag.
*   **Status:** ⏳ Pending

**Task 10: CLI Polish, Manpages, GitHub Pages & Landing Page**
*   **Description:** Implement `stamp completion` for shell autocompletion. Implement a documentation generation pipeline (invoked via `task docs` or a command) to auto-generate both Markdown files (for GitHub Pages) and troff `man` pages (for native UNIX documentation) using `cobra/doc`. Create a landing page served via GitHub Pages (`/docs` folder on main branch) at `https://rossijonas.github.io/stamp/`. Ensure `NO_COLOR` compliance and strict `stdout`/`stderr` separation.
*   **Landing page scope (to be detailed before implementation):**
    - Custom `index.html` + `assets/style.css` in `docs/`
    - Hero section with ASCII logo, tagline, install one-liner
    - Selling paragraph + example workflow section
    - Screenshots / animated GIFs of tool usage (format and content TBD)
    - Links to auto-generated CLI reference (`docs/usage/`)
*   **Acceptance:** User can run diagnostics with `stamp doctor --json`, load shell completions, and run `man stamp` locally. Landing page renders at project URL with links to usage docs.
*   **Verify:** `task docs` generates valid markdown files in `docs/usage/` and `.1` files in `docs/man/`. GitHub Pages URL serves a styled landing page.
*   **Status:** ⏳ Pending

### Phase 5: Project Licensing & Governance
Ensure maximum community and enterprise reach.

**Task 11: Relicense to Apache-2.0**
*   **Description:** Transition project license from AGPL-3.0 to Apache-2.0 to simplify integration and adoption. Update files and documentation.
*   **Acceptance:** LICENSE contains Apache-2.0 text, README links to correct license, and ADR-003 is merged.
*   **Verify:** `task check` passes.
*   **Status:** ✅ Completed

