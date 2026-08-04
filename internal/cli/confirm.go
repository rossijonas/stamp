package cli

import (
	"context"
	"errors"
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

// Sentinel results for the confirmation gates.
var (
	// errStopClean is returned when the operation must not run but the command
	// is still a clean no-op (exit 0): an interactive decline, a no-op preview,
	// or a SIGINT abort. Callers map it to nil via handleConsent.
	errStopClean = errors.New("aborted")

	// errNonInteractive is returned when a destructive operation needs consent
	// but the input is not a terminal and -y/--yes was not passed. Callers
	// propagate it so the command exits non-zero — a forgotten -y in a script
	// or CI pipeline must fail loud instead of silently doing nothing.
	errNonInteractive = errors.New("refusing to run without -y in non-interactive mode")
)

// consentPrompt prompts with a default of "no". It returns nil when accepted,
// errStopClean when the user declines (TTY), and errNonInteractive when the
// input is not a terminal (there is no way to consent).
func consentPrompt(ctx context.Context, out io.Writer, in io.Reader, msg string) error {
	if !isTerminal(in) {
		return errNonInteractive
	}
	if !promptYesNo(ctx, out, in, msg, false) {
		return errStopClean
	}
	return nil
}

// handleConsent maps a gate error to the caller's exit behavior: nil when the
// operation may proceed or stopped cleanly (exit 0), and the original error
// when the operation was refused non-interactively (exit non-zero).
func handleConsent(err error) error {
	if err == nil || errors.Is(err, errStopClean) {
		return nil
	}
	return err
}

// confirmDestructive implements the shared confirmation gate for destructive
// package operations (install, remove, reinstall):
//
//   - yes → return nil immediately (skip refresh/preview/prompt).
//   - otherwise refresh metadata (best-effort, install/reinstall only), render
//     the adapter's transaction preview, then prompt with a default of "no".
//
// A no-op preview (adapter asserts no transaction would occur), an interactive
// decline, or a SIGINT abort all return errStopClean — the caller stops with
// exit 0. A non-terminal input without explicit consent returns errNonInteractive
// so callers exit non-zero (fail closed and fail loud for pipelines/CI).
func confirmDestructive(ctx context.Context, w io.Writer, in io.Reader, yes bool,
	adapter manager.Adapter, mode previewMode, verb, pkg string) error {
	if yes {
		return nil
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
		return errStopClean
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
		return errStopClean
	case strings.TrimSpace(pv.Output) != "":
		_, _ = fmt.Fprintln(w, pv.Output)
	}

	msg := fmt.Sprintf("%s %s via %s? [y/N]: ", verb, pkg, adapter.Name())
	if err := consentPrompt(ctx, w, in, msg); err != nil {
		if errors.Is(err, errNonInteractive) {
			return fmt.Errorf("%w (%s %s via %s)", errNonInteractive, verb, pkg, adapter.Name())
		}
		_, _ = fmt.Fprintln(w, "aborted")
		return errStopClean
	}
	return nil
}

// requireConsent prompts for confirmation unless the global -y/--yes flag is
// set. Non-interactive input without -y returns errNonInteractive so callers
// exit non-zero; an interactive decline returns errStopClean (exit 0). Callers
// must only run destructive operations when this returns nil.
func requireConsent(cmd *cobra.Command, verb string) error {
	app := appFromCtx(cmd)
	if app != nil && app.yes {
		return nil
	}
	if cmd.Context().Err() != nil {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "aborted")
		return errStopClean
	}
	if err := consentPrompt(cmd.Context(), cmd.ErrOrStderr(), cmd.InOrStdin(), fmt.Sprintf("%s? [y/N]: ", verb)); err != nil {
		if errors.Is(err, errNonInteractive) {
			return fmt.Errorf("%w (%s)", errNonInteractive, verb)
		}
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "aborted")
		return errStopClean
	}
	return nil
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
