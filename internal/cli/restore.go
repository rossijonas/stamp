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
It first adds all tracked repositories sequentially, then installs all
tracked packages across package managers concurrently, batched per
manager into a single native invocation (falling back to per-package
installs for managers without batch support).`,
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

			targets := restoreAdapters(app.adapters, repos, pkgs)

			// Authenticate before Phase 1 (repos), then re-validate right before
			// the parallel package phase; serialize when sudo cannot cache so
			// concurrent prompts never race.
			sudoPreflight(cmd, targets, cmd.ErrOrStderr())

			// Interrupted (SIGINT) during auth: abort cleanly rather than start Phase 1.
			if cmd.Context().Err() != nil {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "aborted")
				return nil
			}

			restoreRepositories(cmd.Context(), cmd.ErrOrStderr(), app.adapters, repos)

			parallelOK := sudoPreflight(cmd, targets, cmd.ErrOrStderr())

			// Interrupted (SIGINT) during Phase 1/auth: abort cleanly rather than start Phase 2.
			if cmd.Context().Err() != nil {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "aborted")
				return nil
			}

			errs := restorePackages(cmd.Context(), cmd.ErrOrStderr(), app.adapters, pkgs, !parallelOK)
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
