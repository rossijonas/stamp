package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manifest"
	"github.com/rossijonas/stamp/internal/state"
)

func newReconcileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reconcile",
		Short: "Detect packages installed outside stamp and add them to the manifest",
		Long: `Compare the current system package state against the last snapshot.
Any new packages found are surfaced as potential intentional installs
and can be added to the manifest.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)

			if len(app.adapters) == 0 {
				return fmt.Errorf("no package managers available")
			}

			snapDir, err := state.SnapshotDir()
			if err != nil {
				return fmt.Errorf("failed to access snapshot directory: %w", err)
			}

			oldSnaps := make([]state.Snapshot, 0, len(app.adapters))
			for _, a := range app.adapters {
				snap, err := state.Load(snapDir, a.Name())
				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						continue
					}
					return fmt.Errorf("failed to load snapshot for %s: %w", a.Name(), err)
				}
				oldSnaps = append(oldSnaps, *snap)
			}

			currentSnaps, err := state.Current(cmd.Context(), app.adapters)
			if err != nil {
				return fmt.Errorf("failed to fetch current package state: %w", err)
			}

			for _, s := range currentSnaps {
				if err := state.Save(snapDir, s); err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to save snapshot for %s: %v\n", s.Manager, err)
				}
			}

			if len(oldSnaps) == 0 {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "initial baseline snapshot taken")
				return nil
			}

			deltas := state.DiffAll(oldSnaps, currentSnaps)

			type discoveredPkg struct {
				name    string
				manager string
			}
			var discovered []discoveredPkg
			for _, d := range deltas {
				for _, p := range d.Added {
					discovered = append(discovered, discoveredPkg{name: p, manager: d.Manager})
				}
			}

			if len(discovered) == 0 {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "No drift detected")
				return nil
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Discovered %d new package(s):\n", len(discovered))
			for _, p := range discovered {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  - %s (%s)\n", p.name, p.manager)
			}

			track := app.yes
			if !track {
				if !isTerminal(cmd.InOrStdin()) {
					track = true
				} else {
					_, _ = fmt.Fprint(cmd.ErrOrStderr(), "Track these packages? [Y/n]: ")
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
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Packages not tracked")
				return nil
			}

			trackedCount := 0
			for _, p := range discovered {
				if app.manifest.AddPackage(manifest.Package{
					Name:    p.name,
					Manager: p.manager,
				}) {
					trackedCount++
				}
			}

			if trackedCount > 0 {
				if err := app.saveManifest(); err != nil {
					return fmt.Errorf("failed to save manifest: %w", err)
				}
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Tracked %d package(s)\n", trackedCount)
			return nil
		},
	}

	return cmd
}

// isTerminal reports whether the given reader is connected to a terminal.
func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}
