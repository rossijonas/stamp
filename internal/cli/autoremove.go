package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
)

// resolveManagerTarget narrows adapters to a single manager when managerFlag
// is set, returning the plain "not available" error so the exit code stays 1.
func resolveManagerTarget(adapters []manager.Adapter, managerFlag string) ([]manager.Adapter, error) {
	if managerFlag == "" {
		return adapters, nil
	}
	resolved := manager.ResolveManager(managerFlag)
	for _, a := range adapters {
		if a.Name() == resolved {
			return []manager.Adapter{a}, nil
		}
	}
	return nil, fmt.Errorf("manager %q not available", managerFlag)
}

// renderAutoremoveResult prints one adapter's autoremove outcome. It reports
// whether any work is pending or was performed.
func renderAutoremoveResult(w io.Writer, adapterName string, pkgs []string, dryRun bool) bool {
	if dryRun {
		if len(pkgs) == 0 {
			_, _ = fmt.Fprintf(w, "  %s: no orphaned packages\n", adapterName)
			return false
		}
		_, _ = fmt.Fprintf(w, "  %s: would remove %d package(s)\n", adapterName, len(pkgs))
		for _, p := range pkgs {
			_, _ = fmt.Fprintf(w, "    - %s\n", p)
		}
		return true
	}
	if len(pkgs) == 0 {
		_, _ = fmt.Fprintf(w, "  %s: cleaned\n", adapterName)
	} else {
		_, _ = fmt.Fprintf(w, "  %s: removed %d package(s)\n", adapterName, len(pkgs))
	}
	return true
}

// runAutoremoveAdapter runs AutoRemove for one adapter and reports whether
// work is pending or was performed. Unsupported managers are skipped; other
// failures surface as warnings on stderr.
func runAutoremoveAdapter(ctx context.Context, w io.Writer, a manager.Adapter, dryRun bool) (hasWork bool) {
	pkgs, err := a.AutoRemove(manager.WithYes(ctx), dryRun)
	if err != nil {
		if errors.Is(err, manager.ErrNotSupported) {
			return false
		}
		_, _ = fmt.Fprintf(w, "warning: %s autoremove failed: %v\n", a.Name(), err)
		return false
	}
	return renderAutoremoveResult(w, a.Name(), pkgs, dryRun)
}

func newAutoremoveCmd() *cobra.Command {
	var managerFlag string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "autoremove",
		Short: "Remove orphaned packages and unused dependencies",
		Example: `  # remove orphaned dependencies
  stamp autoremove

  # preview what would be removed
  stamp autoremove --dry-run

  # scope to a single package manager
  stamp autoremove -m brew`,
		Long: `Remove orphaned packages and unused dependencies across all
package managers. Use --dry-run to preview what would be removed.

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
			// Real removals require explicit consent; dry-run is read-only.
			if !dryRun {
				if err := requireConsent(cmd, "Proceed with autoremove"); err != nil {
					return handleConsent(err)
				}
			}
			for _, a := range targets {
				if runAutoremoveAdapter(cmd.Context(), errOut, a, dryRun) {
					hasWork = true
				}
			}

			if !hasWork {
				_, _ = fmt.Fprintln(errOut, "no orphaned packages found")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "preview orphans without removing")
	return cmd
}
