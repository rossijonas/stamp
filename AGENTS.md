# AI Agent Rules for Stamp

If you are an AI agent working in this repository, you **MUST** adhere to the following rules:

## Workflow
1. **Spec-Driven Development:** Do not write code without an agreed-upon specification in `docs/SPEC.md` and an implementation plan in `docs/IMPLEMENTATION_PLAN.md`.
2. **Vertical Slices:** Deliver working features vertically. Do not build all horizontal layers (e.g., all managers, then all commands) at once.
3. **Read-Only Plan Mode:** When planning, do not execute commands or modify files until the human approves the plan.
4. **Documentation Must Accompany Changes:** Every pull request that adds, changes, or removes functionality MUST include corresponding updates to the documentation site (`docs/`) and/or README. Documentation changes must be in the same PR as the code they describe — not deferred to a follow-up.

## Go Standards
1. **No `pkg/` directory:** `stamp` is a CLI application, not an external library. Business logic goes in `internal/`.
2. **Naming:** Use lowercase, semantic package names. Avoid `utils` or `helpers`.
3. **Interface-Driven & Decoupling:** Abstract external dependencies (like package managers or shell executions) behind interfaces to enable easy mocking. Cross-package dependencies must be passed structurally or injected via parameters — never imported via circular paths — to prevent cyclic dependency chains between `manager`, `cli`, and `state`.
4. **File Length & Structure (Anti-God-File):** No single source or test file may exceed 1,000 lines. Large tables or logic blocks must be extracted into focused sub-modules or per-adapter file pairs.
5. **Error Handling:** Wrap errors with context (`fmt.Errorf("failed to do X: %w", err)`). Errors must be logged OR returned, never both. Error strings must be lowercase without trailing punctuation.
6. **Code Style:** Handle errors first (early return) to keep the happy path un-indented. Use `:=` for non-zero values, `var` for zero-value initialization.
7. **Safety & File Writes:** Always initialize maps before use (`make(map[K]V)`). Return defensive copies (`slices.Clone`) of internal data structures to prevent caller mutation. Files modifying completions, manifests, or snapshots must use the temp-file + atomic rename pattern to prevent partial writes.
8. **Execution Gating (TTY-Awareness):** Privileged command execution must detect interactive vs non-interactive environments. In non-interactive mode, inject appropriate flags (e.g. `-n` for `sudo`) to prevent hangs in CI, pipelines, or containers.
9. **Design Patterns:** Constructors should be explicit (no `init()` functions unless strictly required by a framework like Cobra).

## Testing
1. **Framework:** Use the standard `testing` package + `github.com/stretchr/testify` (`assert` and `require`).
2. **Mocks:** Use `testify/mock` for mocking internal interfaces.
3. **Structure & Localization:** Use Table-Driven Tests for multiple scenarios. Manager adapter tests must be localized into their own per-adapter file pairs (e.g., `dnf_test.go`, `brew_test.go`, `flatpak_test.go`). Do not bloat `manager_test.go` beyond 1,000 lines.
4. **Coverage & Complexity:** Overall project test coverage MUST remain above **90%**. Core logic packages demand 100%. Functions exceeding a cyclomatic complexity of 15 must have corresponding unit tests before merge. High-risk commands (`stamp reconcile`, `stamp restore`, `stamp doctor`) must always be introduced or modified with complete path coverage.
5. **Assessment Reference:** Consult `docs/assessments/` for periodic quality reviews. Issues labeled `assessment-001` track recommendations from the latest code assessment report.

## Tools
1. Use `task` instead of `make`.
2. Validate code quality with `task check`.

<!-- lean-ctx -->
## lean-ctx

lean-ctx is active — the MCP tools replace native equivalents.
Full rules: LEAN-CTX.md (open on demand — do not auto-load).
<!-- /lean-ctx -->

<!-- CODEGRAPH_START -->
## CodeGraph

In repositories indexed by CodeGraph (a `.codegraph/` directory exists at the repo root), reach for it BEFORE grep/find or reading files when you need to understand or locate code:

- **MCP tool** (when available): `codegraph_explore` answers most code questions in one call — the relevant symbols' verbatim source plus the call paths between them, including dynamic-dispatch hops grep can't follow. Name a file or symbol in the query to read its current line-numbered source. If it's listed but deferred, load it by name via tool search.
- **Shell** (always works): `codegraph explore "<symbol names or question>"` prints the same output.

If there is no `.codegraph/` directory, skip CodeGraph entirely — indexing is the user's decision.
<!-- CODEGRAPH_END -->
