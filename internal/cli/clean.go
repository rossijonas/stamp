package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
)

// renderCleanResult prints one adapter's clean outcome. It reports whether
// any work is pending or was performed.
func renderCleanResult(w io.Writer, adapterName string, result []string, dryRun bool) bool {
	if dryRun {
		if len(result) == 0 {
			_, _ = fmt.Fprintf(w, "  %s: nothing to clean\n", adapterName)
			return false
		}
		_, _ = fmt.Fprintf(w, "  %s: would clean %d item(s)\n", adapterName, len(result))
		return true
	}
	if len(result) == 0 {
		_, _ = fmt.Fprintf(w, "  %s: cleaned\n", adapterName)
	} else {
		_, _ = fmt.Fprintf(w, "  %s: cleaned %d item(s)\n", adapterName, len(result))
	}
	return true
}

// runCleanAdapter runs Clean for one adapter and reports whether work is
// pending or was performed. Unsupported managers are skipped; other failures
// surface as warnings on stderr.
func runCleanAdapter(ctx context.Context, w io.Writer, a manager.Adapter, dryRun bool) (hasWork bool) {
	result, err := a.Clean(manager.WithYes(ctx), dryRun)
	if err != nil {
		if errors.Is(err, manager.ErrNotSupported) {
			return false
		}
		_, _ = fmt.Fprintf(w, "warning: %s clean failed: %v\n", a.Name(), err)
		return false
	}
	return renderCleanResult(w, a.Name(), result, dryRun)
}

func newCleanCmd() *cobra.Command {
	var managerFlag string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean package caches and temporary files",
		Example: `  # clean package caches
  stamp clean

  # preview what would be cleaned
  stamp clean --dry-run

  # scope to a single package manager
  stamp clean -m brew`,
		Long: `Remove locally cached package files across all package managers.
Use --dry-run to preview what would be cleaned without deleting.

Scoped to a single manager with the --manager flag.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)
			errOut := cmd.ErrOrStderr()

			targets, err := resolveManagerTarget(app.adapters, managerFlag)
			if err != nil {
				return err
			}

			hasWork := false
			// Real cleanups require explicit consent; dry-run is read-only.
			if !dryRun {
				if err := requireConsent(cmd, "Proceed with clean"); err != nil {
					return handleConsent(err)
				}
			}
			for _, a := range targets {
				if runCleanAdapter(cmd.Context(), errOut, a, dryRun) {
					hasWork = true
				}
			}

			if !hasWork {
				_, _ = fmt.Fprintln(errOut, "nothing to clean")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "preview what would be cleaned")
	return cmd
}
