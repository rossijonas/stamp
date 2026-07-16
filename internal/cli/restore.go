package cli

import (
	"fmt"
	"sync"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
	"github.com/rossijonas/stamp/internal/state"
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
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "▪ Dry Run (Preview):")
				if len(repos) > 0 {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Repositories:")
					for _, r := range repos {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  - %s (%s) %s\n", r.Name, r.Manager, r.URL)
					}
				}
				if len(pkgs) > 0 {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Packages:")
					for _, p := range pkgs {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  - %s (%s)\n", p.Name, p.Manager)
					}
				}
				return nil
			}

			// Phase 1: Restore Repositories (Sequentially)
			if len(repos) > 0 {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Phase 1: Restoring Repositories...")
				for _, r := range repos {
					var adapter manager.Adapter
					for _, a := range app.adapters {
						if a.Name() == r.Manager {
							adapter = a
							break
						}
					}
					if adapter == nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  warning: manager %s not available for repository %s\n", r.Manager, r.Name)
						continue
					}
					if err := adapter.AddRepo(cmd.Context(), r.Name, r.URL); err != nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  warning: failed to add repository %s (%s): %v\n", r.Name, r.Manager, err)
					} else {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  restored repository %s via %s\n", r.Name, r.Manager)
					}
				}
			}

			// Phase 2: Restore Packages (Concurrently by Manager)
			if len(pkgs) > 0 {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Phase 2: Restoring Packages...")

				// Group packages by manager
				byManager := make(map[string][]string)
				for _, p := range pkgs {
					byManager[p.Manager] = append(byManager[p.Manager], p.Name)
				}

				type restoreError struct {
					Manager string
					Pkg     string
					Err     error
				}

				var errors []restoreError
				var errMu sync.Mutex
				var wg sync.WaitGroup

				for mName, pkgs := range byManager {
					var adapter manager.Adapter
					for _, a := range app.adapters {
						if a.Name() == mName {
							adapter = a
							break
						}
					}

					if adapter == nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  warning: manager %s not available, skipping %d package(s)\n", mName, len(pkgs))
						continue
					}

					wg.Add(1)
					go func(a manager.Adapter, pNames []string) {
						defer wg.Done()
						for _, pName := range pNames {
							if err := a.Install(cmd.Context(), pName); err != nil {
								errMu.Lock()
								errors = append(errors, restoreError{
									Manager: a.Name(),
									Pkg:     pName,
									Err:     err,
								})
								errMu.Unlock()
							} else {
								_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  installed %s via %s\n", pName, a.Name())
							}
						}
					}(adapter, pkgs)
				}

				wg.Wait()

				if len(errors) > 0 {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Some packages failed to restore:")
					for _, e := range errors {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  - %s (%s): %v\n", e.Pkg, e.Manager, e.Err)
					}
					return fmt.Errorf("failed to restore %d package(s)", len(errors))
				}
			}

			// Save snapshots to align baseline for next reconcile
			snapDir, err := state.SnapshotDir()
			if err == nil {
				currentSnaps, err := state.Current(cmd.Context(), app.adapters)
				if err == nil {
					for _, s := range currentSnaps {
						if err := state.Save(snapDir, s); err != nil {
							_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to save snapshot for %s: %v\n", s.Manager, err)
						}
					}
				}
			}

			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Restore completed successfully")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "preview repositories and packages to restore")
	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to restore")
	return cmd
}
