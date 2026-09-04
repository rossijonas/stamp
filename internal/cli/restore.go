package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// confirmRestore gates the restore on user consent. Non-interactive runs
// without -y abort (fail closed); interrupted runs abort cleanly.
func confirmRestore(cmd *cobra.Command, app *AppContext) error {
	if app.yes {
		return nil
	}
	if cmd.Context().Err() != nil {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "aborted")
		return nil
	}
	if err := consentPrompt(cmd.Context(), cmd.ErrOrStderr(), cmd.InOrStdin(),
		"Restore tracked repositories and packages? [y/N]: "); err != nil {
		return handleConsent(err)
	}
	return nil
}

func newRestoreCmd() *cobra.Command {
	var dryRun bool
	var managerFlag string

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore all tracked repositories and packages from the manifest",
		Example: `  # restore all repositories and packages from the manifest
  stamp restore

  # skip confirmation and proceed immediately
  stamp restore -y

  # preview what would be restored without making changes
  stamp restore --dry-run

  # restore only packages from a specific manager
  stamp restore -m brew`,
		Long: `Read the manifest and restore your system state.
It first adds all tracked repositories sequentially,
then installs all tracked packages concurrently across package managers.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
			}

			repos := filterRepositories(app.manifest.Repositories, managerFlag, "")
			pkgs := filterPackages(app.manifest.Packages, managerFlag, "")

			if len(pkgs) == 0 && len(repos) == 0 {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Nothing to restore")
				return nil
			}

			if dryRun {
				renderRestoreDryRun(cmd.ErrOrStderr(), repos, pkgs)
				return nil
			}

			if err := confirmRestore(cmd, app); err != nil {
				return err
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
