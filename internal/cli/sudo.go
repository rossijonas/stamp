package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
)

// sudoReady and sudoEnsure are seams over the manager sudo primitives so
// preflight behavior can be tested without invoking sudo.
var (
	sudoReady  = manager.SudoReady
	sudoEnsure = manager.EnsureSudo
)

// needsSudo reports whether any of the given adapters runs commands through sudo.
func needsSudo(adapters []manager.Adapter) bool {
	for _, a := range adapters {
		switch a.Name() {
		case "dnf", "apt", "zypper", "pacman", "paru", "macports", "snap", "npm":
			return true
		}
	}
	return false
}

// sudoWarnedAnnotation marks a command that already reported a sudo
// validation failure, so repeated preflights warn once.
const sudoWarnedAnnotation = "stamp.sudoWarned"

// sudoPreflight makes a command's privileged phase predictable without stamp
// ever handling the password:
//
//   - canceled context → no-op, no child spawned (return false so callers that
//     proceed serialize rather than race);
//   - no selected manager needs sudo → no-op, parallel OK;
//   - sudo already usable (NOPASSWD or a valid credential cache) → no-op,
//     parallel OK, no side effects;
//   - not ready on a non-interactive terminal → no-op; sudoCmd adds -n and the
//     real command fails fast;
//   - not ready on a terminal → authenticate once with `sudo -v` (sudo prompts
//     natively), then re-probe. A valid cache afterward → parallel OK; still
//     not ready (non-caching sudo policy) → return false so the caller
//     serializes and never races concurrent prompts.
//
// The returned bool reports whether the privileged phase may run in parallel.
func sudoPreflight(cmd *cobra.Command, adapters []manager.Adapter, errOut io.Writer) bool {
	if cmd.Context().Err() != nil {
		return false
	}
	if !needsSudo(adapters) || sudoReady(cmd.Context()) {
		return true
	}
	if !isTerminal(cmd.InOrStdin()) {
		return true
	}
	if err := sudoEnsure(cmd.Context()); err != nil {
		warnSudoOnce(cmd, errOut, err)
		return false
	}
	return sudoReady(cmd.Context())
}

// warnSudoOnce reports a sudo validation failure at most once per command and
// stays quiet when the context is already canceled — the caller prints its own
// "aborted" line in that case.
func warnSudoOnce(cmd *cobra.Command, errOut io.Writer, err error) {
	if cmd.Context().Err() != nil {
		return
	}
	if cmd.Annotations == nil {
		cmd.Annotations = make(map[string]string)
	}
	if _, ok := cmd.Annotations[sudoWarnedAnnotation]; ok {
		return
	}
	cmd.Annotations[sudoWarnedAnnotation] = "true"
	_, _ = fmt.Fprintf(errOut, "  warning: sudo validation failed: %v\n", err)
}
