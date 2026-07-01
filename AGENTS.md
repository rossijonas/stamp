# AI Agent Rules for Stamp

If you are an AI agent working in this repository, you **MUST** adhere to the following rules:

## Workflow
1. **Spec-Driven Development:** Do not write code without an agreed-upon specification in `docs/SPEC.md` and an implementation plan in `docs/IMPLEMENTATION_PLAN.md`.
2. **Vertical Slices:** Deliver working features vertically. Do not build all horizontal layers (e.g., all managers, then all commands) at once.
3. **Read-Only Plan Mode:** When planning, do not execute commands or modify files until the human approves the plan.

## Go Standards
1. **No `pkg/` directory:** `stamp` is a CLI application, not an external library. Business logic goes in `internal/`.
2. **Naming:** Use lowercase, semantic package names. Avoid `utils` or `helpers`.
3. **Interface-Driven:** Abstract external dependencies (like package managers or shell executions) behind interfaces to enable easy mocking.

## Testing
1. **Framework:** Use the standard `testing` package + `github.com/stretchr/testify` (`assert` and `require`).
2. **Mocks:** Use `testify/mock` for mocking internal interfaces.
3. **Structure:** Use Table-Driven Tests for multiple scenarios.
4. **Coverage:** Core logic packages (like `internal/state` and `internal/manifest`) demand 100% test coverage.

## Tools
1. Use `task` instead of `make`.
2. Validate code quality with `task lint` (golangci-lint).
