---
---

# ADR-020: CLI Icon TTY Convention

## Status
Accepted

## Date
2026-08-29

## Context
stamp uses unicode glyphs (`▪`, `✓`, `✗`) as visual anchors in CLI output. These
are decoration, not data. Two problems existed before this decision:

1. **Inconsistent gating.** `install`, `reinstall`, and `remove` already gate
   icons behind `isOutputTerminal` (via `statusLine`/`printStatus`), so piped
   output is plain ASCII. Every other command (`doctor`, `update`, `selfupdate`,
   `hello`, `man`, `restore`, `autoreconcile`) emits glyphs unconditionally —
   polluting CI logs,管道 output, and日志 files with non-ASCII characters.

2. **No written convention.** ADR-007 (NO_COLOR) covers ANSI escape sequences
   only. Unicode glyphs are outside `NO_COLOR` scope, but the project lacked a
   rule saying *when* glyphs appear and *how* to gate them. New code had no
   guidance.

## Decision

### Glyph set
Three glyphs, each with a fixed semantic:

| Glyph | Meaning | ASCII fallback |
|-------|---------|----------------|
| `▪` | In-progress / section header | *(omitted — no text emitted for progress)* |
| `✓` | Success / positive status | *(text emitted without glyph)* |
| `✗` | Failure / negative status | *(text emitted without glyph)* |
| `⚠` | Warning | always-on (intentionally ungated, pre-existing) |

`⚠` is outside the ADR glyph set. It was never gated and stays always-on —
TTY detection does not suppress it.

### TTY gating
All glyph-bearing output MUST be gated on `isOutputTerminal(w)`. On a pipe,
CI, or non-interactive session, glyphs are stripped and plain text is emitted
instead. Progress lines (`▪`) are suppressed entirely on pipes — same behaviour
as the existing `statusLine` helper.

### Helper
`iconLine(tty bool, icon, text string) string` in `status.go` is the single
source of truth for glyph substitution. Call sites use it instead of
hand-rolling conditionals:

```go
fmt.Fprintln(w, iconLine(tty, "✓", "auto-reconcile enabled"))
```

On TTY: `✓ auto-reconcile enabled`. On pipe: `auto-reconcile enabled`.

### NO_COLOR relationship
`NO_COLOR` governs ANSI escape sequences (colour, bold, underline). Unicode
glyphs are *not* ANSI — `NO_COLOR` does not suppress them. TTY detection is the
correct mechanism for glyph gating. The two systems are orthogonal:

| Signal | Controls |
|--------|----------|
| `isOutputTerminal` | Unicode glyph visibility |
| `NO_COLOR` | ANSI escape sequence suppression |

## Alternatives Considered

### Inline `if tty` per site
- **Pros:** No helper to learn; obvious at each call site.
- **Cons:** ~30 duplicated conditionals; easy to miss one; inconsistent
  formatting (some `if/else`, some ternary-like).
- **Rejected:** Violates DRY for a pattern that appears 30+ times.

### `fatih/color` library
- **Pros:** Automatic NO_COLOR detection, auto-disable on non-TTY.
- **Cons:** Covers ANSI, not unicode glyphs — wrong tool for this problem.
  Adds a dependency for functionality we handle in 5 lines.
- **Rejected:** Does not solve the actual problem (glyph gating).

### Suppress all non-ASCII on pipe
- **Pros:** Simple — one global check.
- **Cons:** Overly broad — some non-ASCII is data (package names, URLs with
  international characters). Glyphs are the specific concern.
- **Rejected:** Too coarse.

## Consequences
- All stamp output is plain ASCII on pipes/CI. Machine consumers see clean text.
- Interactive terminals retain visual glyphs for status/progress.
- New commands follow the `iconLine` pattern — one rule, one helper.
- ADR-007 (NO_COLOR) remains unchanged — it correctly scopes to ANSI escapes.
