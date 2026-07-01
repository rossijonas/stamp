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
*   **Files:** `go.mod`, `.gitignore`, `Taskfile.yml`, `.golangci.yml`, `CONTRIBUTING.md`, `AGENTS.md`.

**Task 2: Manifest Manager (TOML)**
*   **Description:** Implement `internal/manifest` to read, write, and manipulate `manifest.toml`, including support for the `notes` field.
*   **Acceptance:** Can serialize/deserialize TOML and add/remove packages.
*   **Verify:** `task test` passes for `internal/manifest`.
*   **Files:** `internal/manifest/*.go`

### Phase 2: The Active Wrapper Flow
Build the ability to actually modify the system (Install, Remove, Search) as the primary usage model.

**Task 3: Package Manager Interfaces & Mocks**
*   **Description:** Define the `PackageManager` interface. Implement `MockManager` for testing.
*   **Acceptance:** Interface defines `Name()`, `ListInstalled()`, `Install()`, `Remove()`, `Search()`, `AddRepo()`, `RemoveRepo()`.
*   **Verify:** `task test` passes for `internal/manager` mocks.
*   **Files:** `internal/manager/manager.go`, `internal/manager/mock.go`

**Task 4: Native Adapters (Write Operations)**
*   **Description:** Implement `Install()`, `Remove()`, `Search()`, `AddRepo()`, and `RemoveRepo()` for `dnf`, `brew`, and `flatpak`.
*   **Acceptance:** Adapters can execute system and repository modifications.
*   **Verify:** Tests pass.
*   **Files:** `internal/manager/*.go`

**Task 5: Active CLI Commands**
*   **Description:** Wire up `stamp install/add`, `stamp remove/uninstall/delete/del`, and `stamp search` in Cobra. Implement the `stamp repo` command group. Ensure aliases are properly registered using Cobra's `Aliases` array.
*   **Acceptance:** Users can install packages and repositories via `stamp`, updating the manifest automatically.
*   **Verify:** Manual test of `stamp install <test-pkg>`
*   **Files:** `internal/cli/*.go`

### Phase 3: The Safety Net Flow
Build the read-only safety net: checking the system state and calculating the delta.

**Task 6: Native Adapters (Read-Only)**
*   **Description:** Implement `ListInstalled()` for `dnf`, `brew`, and `flatpak` using abstracted shell execution (`os/exec`).
*   **Acceptance:** Adapters correctly parse `dnf repoquery`, `brew leaves`, and `flatpak list`.
*   **Verify:** Unit tests pass using mocked string outputs.
*   **Files:** `internal/manager/dnf.go`, `internal/manager/brew.go`, `internal/manager/flatpak.go`

**Task 7: State Engine (Snapshotting)**
*   **Description:** Implement `internal/state` to save JSON snapshots and calculate deltas (Added/Removed) against the current `PackageManager` outputs.
*   **Acceptance:** Engine can accurately report which packages were added since the last snapshot.
*   **Verify:** `task test` passes with 100% coverage on diffing logic.
*   **Files:** `internal/state/*.go`

**Task 8: The `reconcile` Command (Cobra)**
*   **Description:** Wire up `cmd/stamp/main.go` and `internal/cli/reconcile.go`.
*   **Acceptance:** Running `stamp reconcile` fetches the state, calculates the delta, and prompts the user to add new packages to the manifest.
*   **Verify:** Manual test: `go run cmd/stamp/main.go reconcile`
*   **Files:** `cmd/stamp/main.go`, `internal/cli/*.go`

### Phase 4: Restore & UNIX Compliance
Build the environment reconstruction logic and final touches.

**Task 9: The `restore` Command**
*   **Description:** Implement the environment reconstruction logic.
*   **Acceptance:** `stamp restore` parses the manifest, restores all tracked repositories first, and then executes concurrent package installs.
*   **Verify:** Manual test with `--dry-run` flag.
*   **Files:** `internal/cli/restore.go`

**Task 10: CLI Polish & GitHub Pages Documentation**
*   **Description:** Implement `stamp completion` for shell autocompletion. Implement a hidden `stamp generate-docs` command (or hook it into Taskfile) to auto-generate markdown files using `cobra/doc` for GitHub Pages. Ensure `NO_COLOR` compliance and strict `stdout`/`stderr` separation.
*   **Acceptance:** User can generate markdown docs and shell completions.
*   **Verify:** `task docs` generates valid markdown files in `docs/usage/`.
*   **Files:** `internal/cli/completion.go`, `internal/cli/docs.go`, `Taskfile.yml`

