---
---

# ADR-007: NO_COLOR Compliance Approach

## Status
Accepted

## Date
2026-07-11

## Context
The NO_COLOR standard (https://no-color.org/) requires that if the `NO_COLOR` environment variable is set (to any value), the program must not output ANSI escape sequences. This is a UNIX compliance requirement for stamp.

Key considerations:
- stamp currently outputs NO ANSI color codes — trivially compliant
- We should make the compliance explicit and future-proof
- Users should be able to verify compliance via `stamp doctor`
- Future additions of color output must check `NO_COLOR` before emitting codes

## Decision

### Implementation

1. **Helper Function:** `NoColor() bool` checks `os.Getenv("NO_COLOR") != ""`
2. **AppContext Field:** `ctx.noColor` caches the value at initialization time for O(1) access throughout the command lifecycle
3. **Reporting:** `stamp doctor` shows the status in both TTY and JSON output

### Usage Pattern for Future Color Code
Any code that adds ANSI color output must check before emitting:
```go
if !app.noColor {
    fmt.Fprint(out, "\033[31m") // red
}
fmt.Fprint(out, "error message")
if !app.noColor {
    fmt.Fprint(out, "\033[0m")  // reset
}
```

### Doctor Reporting

TTY output:
```
UNIX Compliance:
  NO_COLOR: ✅ Set
  NO_COLOR: ❌ Not set
```

JSON output:
```json
{
  "no_color": true
}
```

## Alternatives Considered

### Third-Party Library (e.g. fatih/color)
- **Pros:** Automatic NO_COLOR detection, cleaner API
- **Cons:** Adds a dependency for functionality we can implement in 3 lines of stdlib
- **Rejected:** stdlib approach is simpler and sufficient

### Global Variable with sync.Once
- **Pros:** Thread-safe, read-once
- **Cons:** Unnecessary complexity — env var reads are cheap
- **Rejected:** Simple `os.Getenv` call is fast enough

## Consequences
- stamp is explicitly NO_COLOR compliant
- Future color additions must check the flag before emitting ANSI codes
- The compliance status is visible to users via `stamp doctor`
- No new dependencies required
