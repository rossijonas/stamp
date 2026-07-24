---
---

# ADR-003: Relicense Stamp to Apache-2.0

## Status
Accepted

## Date
2026-07-07

## Context
`stamp` was originally licensed under the GNU Affero General Public License Version 3 (AGPL-3.0). While AGPL-3.0 provides strong copyleft protection, it severely limits corporate adoption due to enterprise legal bans on AGPL software. To foster maximum adoption and ease integration, a permissive license is required.

## Decision
Relicense `stamp` under the **Apache License, Version 2.0 (Apache-2.0)**.

## Alternatives Considered

### Keep AGPL-3.0
- **Pros:** Prevents proprietary SaaS/cloud exploitation.
- **Cons:** Blocks developer/corporate adoption.
- **Rejected:** Barrier to corporate adoption outweighs protection benefits for a local developer CLI tool.

### MIT License
- **Pros:** Extremely simple, highly permissive.
- **Cons:** Lacks explicit patent grants.
- **Rejected:** Apache-2.0 offers superior legal protection via explicit patent grants from contributors to users.

## Consequences
- Broadens enterprise adoption and integration.
- Contributors protected by explicit patent licensing clauses.
- Derivatives can be used in proprietary projects.
