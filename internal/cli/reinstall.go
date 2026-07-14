package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
	"github.com/rossijonas/stamp/internal/state"
)

func newReinstallCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:   "reinstall <package>",
		Short: "Reinstall a package and record it in the manifest",
		Long: `Look up the package in the manifest to find its recorded package manager,
then execute the native reinstallation command. If the package is not
tracked in the manifest, resolve the manager and track it.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
			}

			pkgName := args[0]
			if err := manager.ValidatePackageName(pkgName); err != nil {
				return fmt.Errorf("invalid package name: %w", err)
			}

			// Look up in manifest
			var recordedManager string
			for _, p := range app.manifest.Packages {
				if p.Name == pkgName {
					recordedManager = p.Manager
					break
				}
			}

			var adapter manager.Adapter
			isPreExisting := recordedManager == ""

			if !isPreExisting {
				// Manifest-tracked: find adapter by recorded manager
				for _, a := range app.adapters {
					if a.Name() == recordedManager {
						adapter = a
						break
					}
				}
				if adapter == nil {
					return fmt.Errorf("manager %q is not available on this system", recordedManager)
				}
			} else {
				// Pre-existing: resolve via 3-tier engine
				resolver := NewResolver(app.adapters, app.config)
				resolved, err := resolver.Resolve(pkgName, managerFlag)
				if err != nil {
					return fmt.Errorf("cannot resolve manager for %q: %w", pkgName, err)
				}
				adapter = resolved
			}

			// Execute native reinstall
			if err := adapter.Reinstall(cmd.Context(), pkgName); err != nil {
				return fmt.Errorf("reinstall failed: %w", err)
			}

			// Save snapshots to align baseline
			snapDir, err := state.SnapshotDir()
			if err == nil {
				currentSnaps, err := state.Current(cmd.Context(), app.adapters)
				if err == nil {
					for _, s := range currentSnaps {
						_ = state.Save(snapDir, s)
					}
				}
			}

			// Add to manifest if pre-existing
			if isPreExisting {
				app.manifest.AddPackage(manifest.Package{
					Name:    pkgName,
					Manager: adapter.Name(),
				})
			}

			// Save manifest
			if err := app.saveManifest(); err != nil {
				return fmt.Errorf("failed to save manifest: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "reinstalled %s via %s\n", pkgName, adapter.Name())
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use (pre-existing packages only)")
	return cmd
}
