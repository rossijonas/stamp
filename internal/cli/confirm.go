package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
)

// previewMode selects the destructive operation being previewed.
type previewMode int

const (
	previewInstall previewMode = iota
	previewRemove
	previewReinstall
)

// confirmDestructive implements the shared confirmation gate for destructive
// package operations (install, remove, reinstall):
//
//   - yes → return true immediately (skip refresh/preview/prompt).
//   - otherwise refresh metadata (best-effort, install/reinstall only), render
//     the adapter's transaction preview, then prompt with a default of "no".
//
// A no-op preview (adapter asserts no transaction would occur) fails fast
// without prompting. A preview that cannot be rendered warns and still prompts
// — the prompt is the consent gate. A non-terminal input without explicit
// consent returns false (fail closed), so pipelines and CI never silently
// mutate the system. Callers must only run the destructive operation when this
// returns true.
func confirmDestructive(ctx context.Context, w io.Writer, in io.Reader, yes bool,
	adapter manager.Adapter, mode previewMode, verb, pkg string) bool {
	if yes {
		return true
	}

	if mode != previewRemove {
		if err := adapter.Refresh(ctx); err != nil {
			_, _ = fmt.Fprintf(w, "  ⚠ %s: refresh failed: %v\n", adapter.Name(), err)
		}
	}

	pv, previewErr := previewOutput(ctx, adapter, mode, pkg)

	// Interrupted (SIGINT) during refresh/preview: abort without prompting.
	if ctx.Err() != nil {
		_, _ = fmt.Fprintln(w, "aborted")
		return false
	}

	switch {
	case previewErr != nil:
		// Could not render a preview: warn and prompt anyway — the prompt is
		// the consent gate, so this degrades to confirm-without-preview.
		_, _ = fmt.Fprintf(w, "  ⚠ %s: could not render preview: %v\n", adapter.Name(), previewErr)
	case pv.Noop:
		if strings.TrimSpace(pv.Output) != "" {
			_, _ = fmt.Fprintln(w, pv.Output)
		}
		_, _ = fmt.Fprintf(w, "  nothing to do: %s via %s\n", pkg, adapter.Name())
		return false
	case strings.TrimSpace(pv.Output) != "":
		_, _ = fmt.Fprintln(w, pv.Output)
	}

	msg := fmt.Sprintf("%s %s via %s? [y/N]: ", verb, pkg, adapter.Name())
	if !promptYesNo(ctx, w, in, msg, false) {
		_, _ = fmt.Fprintln(w, "aborted")
		return false
	}
	return true
}

// requireConsent prompts for confirmation unless the global -y/--yes flag is
// set. Non-interactive input without -y fails closed (returns false), so
// pipelines and CI never silently mutate the system. Callers must only run
// destructive operations when this returns true.
func requireConsent(cmd *cobra.Command, verb string) bool {
	app := appFromCtx(cmd)
	if app != nil && app.yes {
		return true
	}
	if cmd.Context().Err() != nil {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "aborted")
		return false
	}
	if !promptYesNo(cmd.Context(), cmd.ErrOrStderr(), cmd.InOrStdin(), fmt.Sprintf("%s? [y/N]: ", verb), false) {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "aborted")
		return false
	}
	return true
}

// previewOutput renders the adapter-owned transaction preview for the given
// operation. Managers that do not implement manager.Previewer have no preview:
// the gate falls back to warn-and-prompt.
func previewOutput(ctx context.Context, adapter manager.Adapter, mode previewMode, pkg string) (manager.Preview, error) {
	p, ok := adapter.(manager.Previewer)
	if !ok {
		return manager.Preview{}, fmt.Errorf("%w: no transaction preview", manager.ErrNotSupported)
	}
	switch mode {
	case previewInstall:
		return p.PreviewInstall(ctx, pkg)
	case previewRemove:
		return p.PreviewRemove(ctx, pkg)
	case previewReinstall:
		return p.PreviewReinstall(ctx, pkg)
	}
	return manager.Preview{}, fmt.Errorf("%w: unknown preview mode", manager.ErrNotSupported)
}
