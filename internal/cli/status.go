package cli

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// isOutputTerminal reports whether the given writer is connected to a
// terminal. Declared as a variable so it can be overridden in tests. Status
// lines use this so glyphs (▪, ✓) appear only on interactive terminals while
// piped/CI/log output stays plain ASCII.
var isOutputTerminal = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// statusLine renders the status/progress line for install, remove, and
// reinstall operations. When tty is true the line carries the ▪/✓ glyphs;
// when false it is plain ASCII, safe for pipes, logs, and CI.
//
// A progress line (done=false) is emitted only on a terminal; the piped form
// deliberately produces no progress output. A note is appended as
// ` (note: <note>)` whenever non-empty in both forms — the note is data, the
// glyphs are decoration.
func statusLine(tty, done bool, verb, target, mgr, note string) string {
	if !done {
		if !tty {
			return ""
		}
		return fmt.Sprintf("▪ %s %s via %s...", verb, target, mgr)
	}
	line := fmt.Sprintf("%s %s via %s", verb, target, mgr)
	if tty {
		line = "✓ " + line
	}
	if note != "" {
		line += " (note: " + note + ")"
	}
	return line
}

// printStatus emits a progress/completion line to w when statusLine is
// non-empty. It collapses the six repeated "if line := statusLine(...)" sites
// across the install/remove/reinstall command builders into one call.
func printStatus(w io.Writer, tty, done bool, verb, target, mgr, note string) {
	if line := statusLine(tty, done, verb, target, mgr, note); line != "" {
		_, _ = fmt.Fprintln(w, line)
	}
}
