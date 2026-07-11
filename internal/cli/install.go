package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

func newInstallCmd() *cobra.Command {
	var managerFlag string
	var note string

	cmd := &cobra.Command{
		Use:     "install <package>",
		Aliases: []string{"add"},
		Short:   "Install a package and record intent",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
			}
			pkgName := args[0]

			if err := manager.ValidatePackageName(pkgName); err != nil {
				return fmt.Errorf("invalid package name: %w", err)
			}

			r := NewResolver(app.adapters, app.config)
			adapter, err := r.Resolve(pkgName, managerFlag)
			if err != nil {
				return fmt.Errorf("cannot resolve package manager: %w", err)
			}

			if err := adapter.Install(cmd.Context(), pkgName); err != nil {
				return fmt.Errorf("install failed: %w", err)
			}

			app.manifest.AddPackage(manifest.Package{
				Name:    pkgName,
				Manager: adapter.Name(),
				Notes:   note,
			})

			if err := app.saveManifest(); err != nil {
				return fmt.Errorf("failed to save manifest: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "installed %s via %s\n", pkgName, adapter.Name())
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use")
	cmd.Flags().StringVar(&note, "note", "", "annotation for this package")
	return cmd
}

func newRemoveCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:     "remove <package>",
		Aliases: []string{"uninstall", "rm", "delete", "del"},
		Short:   "Remove a package and untrack it",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
			}
			pkgName := args[0]

			var adapter manager.Adapter

			// Check manifest first: if package is tracked, use its recorded manager
			if managerFlag == "" {
				for _, p := range app.manifest.Packages {
					if p.Name == pkgName {
						for _, a := range app.adapters {
							if a.Name() == p.Manager {
								adapter = a
								break
							}
						}
						break
					}
				}
			}

			// Fall back to explicit flag or first available adapter
			if adapter == nil {
				switch {
				case managerFlag != "":
					for _, a := range app.adapters {
						if a.Name() == managerFlag {
							adapter = a
							break
						}
					}
					if adapter == nil {
						return fmt.Errorf("unknown manager %q", managerFlag)
					}
				case len(app.adapters) > 0:
					adapter = app.adapters[0]
				default:
					return fmt.Errorf("no package managers available")
				}
			}

			if err := adapter.Remove(cmd.Context(), pkgName); err != nil {
				return fmt.Errorf("remove failed: %w", err)
			}

			app.manifest.RemovePackage(pkgName, adapter.Name())
			if err := app.saveManifest(); err != nil {
				return fmt.Errorf("failed to save manifest: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "removed %s via %s\n", pkgName, adapter.Name())
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use")
	return cmd
}

func newSearchCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for packages across managers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			query := args[0]

			targets := app.adapters
			if managerFlag != "" {
				var found bool
				for _, a := range app.adapters {
					if a.Name() == managerFlag {
						targets = []manager.Adapter{a}
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("unknown manager %q", managerFlag)
				}
			}

			var results []string
			for _, a := range targets {
				pkgs, err := a.Search(cmd.Context(), query)
				if err != nil {
					continue
				}
				for _, p := range pkgs {
					results = append(results, fmt.Sprintf("%s (%s)", p, a.Name()))
				}
			}

			if len(results) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no results found")
				return nil
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), strings.Join(results, "\n"))
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to search")
	return cmd
}
