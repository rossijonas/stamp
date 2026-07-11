package cli

import (
	"bufio"
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
)

func newRestoreCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore all tracked repositories and packages from the manifest",
		Long: `Read the manifest and restore your system state.
It first adds all tracked repositories sequentially,
then installs all tracked packages concurrently across package managers.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)

			if len(app.manifest.Packages) == 0 && len(app.manifest.Repositories) == 0 {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Nothing to restore")
				return nil
			}

			if dryRun {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "▪ Dry Run (Preview):")
				if len(app.manifest.Repositories) > 0 {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Repositories:")
					for _, r := range app.manifest.Repositories {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  - %s (%s) %s\n", r.Name, r.Manager, r.URL)
					}
				}
				if len(app.manifest.Packages) > 0 {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Packages:")
					for _, p := range app.manifest.Packages {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  - %s (%s)\n", p.Name, p.Manager)
					}
				}
				return nil
			}

			track := app.yes
			if !track {
				if !isTerminal(cmd.InOrStdin()) {
					track = true
				} else {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Restore %d repository(ies) and %d package(s)? [Y/n]: ", len(app.manifest.Repositories), len(app.manifest.Packages))
					reader := bufio.NewReader(cmd.InOrStdin())
					response, err := reader.ReadString('\n')
					if err != nil {
						_, _ = fmt.Fprintln(cmd.ErrOrStderr())
						track = false
					} else {
						response = strings.TrimSpace(response)
						track = response == "" || strings.EqualFold(response, "y") || strings.EqualFold(response, "yes")
					}
				}
			}

			if !track {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Restore cancelled")
				return nil
			}

			// Phase 1: Restore Repositories (Sequentially)
			if len(app.manifest.Repositories) > 0 {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Phase 1: Restoring Repositories...")
				for _, r := range app.manifest.Repositories {
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
			if len(app.manifest.Packages) > 0 {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Phase 2: Restoring Packages...")

				// Group packages by manager
				byManager := make(map[string][]string)
				for _, p := range app.manifest.Packages {
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

			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Restore completed successfully")
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview repositories and packages to restore")
	return cmd
}
