package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
	"github.com/rossijonas/stamp/internal/state"
)

// runReconcileReport compares old and current snapshots, reports drift, and
// either tracks it (dryRun=false) or previews it (dryRun=true).
func runReconcileReport(cmd *cobra.Command, app *AppContext, snapDir string, oldSnaps, currentSnaps []state.Snapshot, dryRun bool) error {
	deltas := state.DiffAll(oldSnaps, currentSnaps)
	discovered, discoveredRepos := collectDiscovered(deltas)
	// Reconcile is a fallback for packages installed outside stamp:
	// drift already tracked in the manifest is not new drift.
	discovered, discoveredRepos = filterDiscoveredByManifest(discovered, discoveredRepos, app.manifest)
	missing := missingFromDeltas(deltas, app.manifest)
	errOut := cmd.ErrOrStderr()

	if len(discovered) == 0 && len(discoveredRepos) == 0 {
		// Nothing new to track. Missing manifest packages still get a
		// warning, but the new snapshot is recorded either way — it
		// reflects reality, and the manifest remains the intent.
		if len(missing) > 0 {
			renderMissing(errOut, missing)
		} else {
			renderNoDrift(errOut)
		}
		if !dryRun {
			saveCurrentSnaps(errOut, snapDir, currentSnaps)
		}
		return nil
	}

	renderDiscovered(errOut, discovered, discoveredRepos)
	if len(missing) > 0 {
		renderMissing(errOut, missing)
	}

	if dryRun {
		renderDryRunHint(errOut)
		return nil
	}

	// Back up the current manifest before mutating it. Copy-based so the
	// original stays in place — reconcile can run on a no-save path.
	if _, err := manifest.CopyBackup(app.manifestPath); err != nil {
		_, _ = fmt.Fprintf(errOut, "warning: failed to back up manifest: %v\n", err)
	}

	saveCurrentSnaps(errOut, snapDir, currentSnaps)

	trackedCount, trackedReposCount, err := saveAndTrack(discovered, discoveredRepos, app)
	if err != nil {
		return err
	}

	if n, err := manifest.RotateBackups(app.manifestPath, app.config.Backup.ManifestPolicy()); err != nil {
		_, _ = fmt.Fprintf(errOut, "warning: failed to rotate manifest backups: %v\n", err)
	} else if n > 0 {
		_, _ = fmt.Fprintf(errOut, "rotated %d manifest backup(s)\n", n)
	}

	renderTrackedSummary(errOut, trackedCount, trackedReposCount)
	return nil
}

// prepareReconcile loads the snapshot directory and old/current snapshots.
func prepareReconcile(cmd *cobra.Command, adapters []manager.Adapter) (snapDir string, oldSnaps, currentSnaps []state.Snapshot, err error) {
	// Always frame reconcile as a fallback and surface reliability
	// caveats before any drift is reported.
	renderReconcileBanner(cmd.ErrOrStderr())
	renderReliabilityNotes(cmd.ErrOrStderr(), adapters)

	snapDir, err = state.SnapshotDir()
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to access snapshot directory: %w", err)
	}

	oldSnaps, err = loadOldSnapshots(adapters, snapDir)
	if err != nil {
		return "", nil, nil, err
	}

	currentSnaps, err = state.Current(cmd.Context(), adapters)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to fetch current package state: %w", err)
	}
	printSnapshotWarnings(cmd.ErrOrStderr(), currentSnaps)
	return snapDir, oldSnaps, currentSnaps, nil
}

func newReconcileCmd() *cobra.Command {
	var (
		managerFlag string
		dryRun      bool
	)

	cmd := &cobra.Command{
		Use:   "reconcile",
		Short: "Detect packages installed outside stamp and add them to the manifest",
		Example: `  # track packages installed outside stamp
  stamp reconcile

  # preview drift without tracking
  stamp reconcile --dry-run

  # reconcile a single package manager
  stamp reconcile -m dnf

  # missing manifest packages are reported as a warning`,
		Long: `Compare the current system package state against the last snapshot.
Any new packages found are auto-tracked to the manifest.
Use --dry-run to preview drift without tracking.

Before tracking, the current manifest is timestamp-backed up, and old
manifest backups are pruned per the [backup] policy in config.toml.
Dry-run performs no writes, no backups, and no rotation.`,
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
				return catErr(ErrUnavailable, "no package managers available")
			}

			snapDir, oldSnaps, currentSnaps, err := prepareReconcile(cmd, adapters)
			if err != nil {
				return err
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

			return runReconcileReport(cmd, app, snapDir, oldSnaps, currentSnaps, dryRun)
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to reconcile")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "preview drift without tracking")
	return cmd
}
