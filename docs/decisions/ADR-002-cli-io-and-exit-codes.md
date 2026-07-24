---
---

# ADR-002: CLI I/O Separation, Exit Codes, and Flag Constraints

## Status
Accepted

## Date
2026-07-03

## Context
For a CLI tool to be considered a "good UNIX citizen" and be highly scriptable, automated, and composable, it must adhere strictly to established POSIX and GNU command-line conventions. 

Specifically, we must define:
1. How output streams (`stdout` vs `stderr`) are handled.
2. How exit codes map to specific operational and runtime conditions.
3. How flag naming collisions (like `--via` vs `-v` for verbose) are avoided.
4. How ambiguous package installation commands are resolved cleanly without forcing interactive prompts in headless script environments.

## Decision

We will implement the following strict CLI conventions across the entire application:

### 1. I/O Separation & Redirection
*   **`stdout` (Standard Output):** Reserved strictly for successful, pipeable, machine-readable program output (e.g., package listings, search results, or JSON formats). 
*   **`stderr` (Standard Error):** All diagnostics, progress bars, interactive prompts, logging, and error messages MUST go to `stderr`. 
*   **Rule of Silence:** When running in a pipeline or with the `--quiet`/`-q` flag, successful command execution produces zero output to `stdout`.

### 2. Auto-Accept Flag (`--yes` / `-y`)
*   We introduce a global `--yes` / `-y` flag to bypass all interactive prompts (e.g., when removing critical repositories/packages).
*   If `--yes` is specified, `stamp` will auto-accept all options. If the environment is a non-interactive shell (no TTY), `--yes` is assumed by default to prevent hanging scripts.
*   **Note:** `stamp reconcile` does NOT have interactive prompts. It auto-tracks drift deterministically. The `--yes` flag is accepted on reconcile for backward compatibility with scripting, but is functionally a no-op.

### 3. CLI Verbose & Version Flags
*   We reserve `-v` and `--verbose` strictly for global debug logging.
*   The previous `--via` flag is replaced with `--manager` / `-m` to prevent any short-flag collision with `-v`.

### 4. Package Manager Resolution Engine (3-Tier)
When a package or repository command is executed without the `--manager` flag, the engine resolves ambiguity in three sequential tiers:
*   **Tier 1: Explicit Override:** If `--manager <name>` or `-m <name>` is provided, use that manager directly.
*   **Tier 2: Declarative Precedence:** `stamp` queries all managers concurrently. If the package exists in multiple managers, `stamp` reads the user's `config.toml` precedence array:
    ```toml
    precedence = ["dnf", "flatpak", "brew"]
    ```
    And automatically installs it using the manager with the highest priority.
*   **Tier 3: Interactive Fallback:** If there is a tie or no precedence is configured:
    *   In a TTY (interactive terminal): Prompt the user with a select menu.
    *   In a non-TTY (scripting pipeline): Fail immediately with exit status `2` (usage error) prompting the user to supply `--manager`.

### 5. Repository Aliases & Operations
To ensure interface consistency, the exact same aliases will apply to both packages and repositories:
*   **Install/Add:** `stamp install/add` and `stamp repo add/install`
*   **Remove/Uninstall:** `stamp remove/uninstall/delete/del` and `stamp repo remove/uninstall/delete/del`

---

## Alternatives Considered

### Relying on Shell Interception
- **Pros:** Completely hands-off for the user.
- **Cons:** Highly brittle, shell-specific, and prone to breaking on different terminals.
- **Rejected:** Bypasses standard package manager control boundaries.

### Requiring `--manager` on All Write Commands
- **Pros:** Extremely simple to implement. Zero ambiguity logic needed.
- **Cons:** Horrible user experience. A major goal of `stamp` is simplifying multi-manager friction; forcing the user to type `-m brew` on every install defeats the purpose.
- **Rejected:** Violates user-friendliness design principles.

---

## Consequences
- **Testing:** The resolution engine must be fully covered by unit tests using mock configurations.
- **Dependency:** Exposes `cobra` commands to standard UNIX input/output checkers (e.g., checking `IsTerminal`).
- **Compatibility:** Guarantees 100% stable automation support for devops provisioning and dotfile scripts.
