package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
)

func newCleanCmd() *cobra.Command {
	var managerFlag string
	var dryRun bool

	cmd := &cobra.Command{
		Use:     "clean",
		Short:   "Clean package caches and temporary files",
		Example: "  stamp clean\n  stamp clean -m brew\n  stamp clean --dry-run",
		Long: `Remove locally cached package files across all package managers.
Use --dry-run to preview what would be cleaned without deleting.

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
				result, err := a.Clean(cmd.Context(), dryRun)
				if err != nil {
					if errors.Is(err, manager.ErrNotSupported) {
						continue
					}
					_, _ = fmt.Fprintf(errOut, "warning: %s clean failed: %v\n", a.Name(), err)
					continue
				}

				if dryRun {
					if len(result) == 0 {
						_, _ = fmt.Fprintf(errOut, "  %s: nothing to clean\n", a.Name())
					} else {
						hasWork = true
						_, _ = fmt.Fprintf(errOut, "  %s: would clean %d item(s)\n", a.Name(), len(result))
					}
				} else {
					hasWork = true
					if len(result) == 0 {
						_, _ = fmt.Fprintf(errOut, "  %s: cleaned\n", a.Name())
					} else {
						_, _ = fmt.Fprintf(errOut, "  %s: cleaned %d item(s)\n", a.Name(), len(result))
					}
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
