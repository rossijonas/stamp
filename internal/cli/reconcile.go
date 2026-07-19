package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/state"
)

func newReconcileCmd() *cobra.Command {
	var (
		managerFlag string
		dryRun      bool
	)

	cmd := &cobra.Command{
		Use:   "reconcile",
		Short: "Detect packages installed outside stamp and add them to the manifest",
		Long: `Compare the current system package state against the last snapshot.
Any new packages found are auto-tracked to the manifest.
Use --dry-run to preview drift without tracking.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
			}

			adapters, err := filterAdapters(app.adapters, managerFlag)
			if err != nil {
				return err
			}
			if len(adapters) == 0 {
				return fmt.Errorf("no package managers available")
			}

			snapDir, err := state.SnapshotDir()
			if err != nil {
				return fmt.Errorf("failed to access snapshot directory: %w", err)
			}

			oldSnaps, err := loadOldSnapshots(adapters, snapDir)
			if err != nil {
				return err
			}

			currentSnaps, err := state.Current(cmd.Context(), adapters)
			if err != nil {
				return fmt.Errorf("failed to fetch current package state: %w", err)
			}

			if len(oldSnaps) == 0 {
				if dryRun {
					renderNoBaselineDryRun(cmd.ErrOrStderr())
					return nil
				}
				saveCurrentSnaps(cmd.ErrOrStderr(), snapDir, currentSnaps)
				renderBaselineTaken(cmd.ErrOrStderr())
				return nil
			}

			deltas := state.DiffAll(oldSnaps, currentSnaps)
			discovered, discoveredRepos := collectDiscovered(deltas)

			if len(discovered) == 0 && len(discoveredRepos) == 0 {
				renderNoDrift(cmd.ErrOrStderr())
				if !dryRun {
					saveCurrentSnaps(cmd.ErrOrStderr(), snapDir, currentSnaps)
				}
				return nil
			}

			renderDiscovered(cmd.ErrOrStderr(), discovered, discoveredRepos)

			if dryRun {
				renderDryRunHint(cmd.ErrOrStderr())
				return nil
			}

			saveCurrentSnaps(cmd.ErrOrStderr(), snapDir, currentSnaps)

			trackedCount, trackedReposCount, err := saveAndTrack(discovered, discoveredRepos, app)
			if err != nil {
				return err
			}

			renderTrackedSummary(cmd.ErrOrStderr(), trackedCount, trackedReposCount)
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to reconcile")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "preview drift without tracking")
	return cmd
}
