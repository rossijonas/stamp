package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
)

func newAutoremoveCmd() *cobra.Command {
	var managerFlag string
	var dryRun bool

	cmd := &cobra.Command{
		Use:     "autoremove",
		Short:   "Remove orphaned packages and unused dependencies",
		Example: "  stamp autoremove\n  stamp autoremove -m brew\n  stamp autoremove --dry-run",
		Long: `Remove orphaned packages and unused dependencies across all
package managers. Use --dry-run to preview what would be removed.

Scoped to a single manager with the --manager flag.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)
			errOut := cmd.ErrOrStderr()

			targets := app.adapters
			if managerFlag != "" {
				resolved := manager.ResolveManager(managerFlag)
				var found bool
				for _, a := range targets {
					if a.Name() == resolved {
						targets = []manager.Adapter{a}
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("manager %q not available", managerFlag)
				}
			}

			hasWork := false
			for _, a := range targets {
				pkgs, err := a.AutoRemove(cmd.Context(), dryRun)
				if err != nil {
					if errors.Is(err, manager.ErrNotSupported) {
						continue
					}
					_, _ = fmt.Fprintf(errOut, "warning: %s autoremove failed: %v\n", a.Name(), err)
					continue
				}

				if dryRun {
					if len(pkgs) == 0 {
						_, _ = fmt.Fprintf(errOut, "  %s: no orphaned packages\n", a.Name())
					} else {
						hasWork = true
						_, _ = fmt.Fprintf(errOut, "  %s: would remove %d package(s)\n", a.Name(), len(pkgs))
						for _, p := range pkgs {
							_, _ = fmt.Fprintf(errOut, "    - %s\n", p)
						}
					}
				} else {
					hasWork = true
					if len(pkgs) == 0 {
						_, _ = fmt.Fprintf(errOut, "  %s: cleaned\n", a.Name())
					} else {
						_, _ = fmt.Fprintf(errOut, "  %s: removed %d package(s)\n", a.Name(), len(pkgs))
					}
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
