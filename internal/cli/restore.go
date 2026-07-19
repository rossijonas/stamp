package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manifest"
)

func newRestoreCmd() *cobra.Command {
	var dryRun bool
	var managerFlag string

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore all tracked repositories and packages from the manifest",
		Long: `Read the manifest and restore your system state.
It first adds all tracked repositories sequentially,
then installs all tracked packages concurrently across package managers.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
			}

			repos := app.manifest.Repositories
			pkgs := app.manifest.Packages

			if managerFlag != "" {
				var filteredRepos []manifest.Repository
				for _, r := range repos {
					if r.Manager == managerFlag {
						filteredRepos = append(filteredRepos, r)
					}
				}
				repos = filteredRepos

				var filteredPkgs []manifest.Package
				for _, p := range pkgs {
					if p.Manager == managerFlag {
						filteredPkgs = append(filteredPkgs, p)
					}
				}
				pkgs = filteredPkgs
			}

			if len(pkgs) == 0 && len(repos) == 0 {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Nothing to restore")
				return nil
			}

			if dryRun {
				renderRestoreDryRun(cmd.ErrOrStderr(), repos, pkgs)
				return nil
			}

			restoreRepositories(cmd.Context(), cmd.ErrOrStderr(), app.adapters, repos)

			errs := restorePackages(cmd.Context(), cmd.ErrOrStderr(), app.adapters, pkgs)
			if len(errs) > 0 {
				renderRestoreErrors(cmd.ErrOrStderr(), errs)
				return fmt.Errorf("failed to restore %d package(s)", len(errs))
			}

			restoreSaveSnapshots(cmd.Context(), cmd.ErrOrStderr(), app.adapters)
			renderRestoreComplete(cmd.ErrOrStderr())
			return nil
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "preview repositories and packages to restore")
	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to restore")
	return cmd
}
