package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
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

			adapters := app.adapters
			if managerFlag != "" {
				var found manager.Adapter
				for _, a := range app.adapters {
					if a.Name() == manager.ResolveManager(managerFlag) {
						found = a
						break
					}
				}
				if found == nil {
					return fmt.Errorf("manager %q not available on this system", managerFlag)
				}
				adapters = []manager.Adapter{found}
			}

			if len(adapters) == 0 {
				return fmt.Errorf("no package managers available")
			}

			snapDir, err := state.SnapshotDir()
			if err != nil {
				return fmt.Errorf("failed to access snapshot directory: %w", err)
			}

			oldSnaps := make([]state.Snapshot, 0, len(adapters))
			for _, a := range adapters {
				snap, err := state.Load(snapDir, a.Name())
				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						continue
					}
					return fmt.Errorf("failed to load snapshot for %s: %w", a.Name(), err)
				}
				oldSnaps = append(oldSnaps, *snap)
			}

			currentSnaps, err := state.Current(cmd.Context(), adapters)
			if err != nil {
				return fmt.Errorf("failed to fetch current package state: %w", err)
			}

			if len(oldSnaps) == 0 {
				if dryRun {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "No baseline snapshot exists. Run without --dry-run to take baseline.")
					return nil
				}
				for _, s := range currentSnaps {
					if err := state.Save(snapDir, s); err != nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to save snapshot for %s: %v\n", s.Manager, err)
					}
				}
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "initial baseline snapshot taken")
				return nil
			}

			deltas := state.DiffAll(oldSnaps, currentSnaps)

			type discoveredPkg struct {
				name    string
				manager string
			}
			var discovered []discoveredPkg
			var discoveredRepos []discoveredPkg

			for _, d := range deltas {
				for _, p := range d.Added {
					discovered = append(discovered, discoveredPkg{name: p, manager: d.Manager})
				}
				for _, r := range d.AddedRepos {
					discoveredRepos = append(discoveredRepos, discoveredPkg{name: r, manager: d.Manager})
				}
			}

			if len(discovered) == 0 && len(discoveredRepos) == 0 {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "No drift detected")
				if !dryRun {
					for _, s := range currentSnaps {
						if err := state.Save(snapDir, s); err != nil {
							_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to save snapshot for %s: %v\n", s.Manager, err)
						}
					}
				}
				return nil
			}

			if len(discovered) > 0 {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Discovered %d new package(s):\n", len(discovered))
				for _, p := range discovered {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  - %s (%s)\n", p.name, p.manager)
				}
			}

			if len(discoveredRepos) > 0 {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Discovered %d new repository(ies):\n", len(discoveredRepos))
				for _, r := range discoveredRepos {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  - %s (%s)\n", r.name, r.manager)
				}
			}

			if dryRun {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Use `stamp reconcile` without --dry-run to track")
				return nil
			}

			for _, s := range currentSnaps {
				if err := state.Save(snapDir, s); err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to save snapshot for %s: %v\n", s.Manager, err)
				}
			}

			trackedCount := 0
			trackedReposCount := 0
			for _, p := range discovered {
				if app.manifest.AddPackage(manifest.Package{
					Name:    p.name,
					Manager: p.manager,
				}) {
					trackedCount++
				}
			}

			for _, r := range discoveredRepos {
				if app.manifest.AddRepository(manifest.Repository{
					Name:    r.name,
					Manager: r.manager,
				}) {
					trackedReposCount++
				}
			}

			if trackedCount > 0 || trackedReposCount > 0 {
				if err := app.saveManifest(); err != nil {
					return fmt.Errorf("failed to save manifest: %w", err)
				}
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Tracked %d package(s), %d repository(ies)\n", trackedCount, trackedReposCount)
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to reconcile")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "preview drift without tracking")
	return cmd
}
