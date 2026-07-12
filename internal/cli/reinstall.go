package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/state"
)

func newReinstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reinstall <package>",
		Short: "Reinstall a package currently tracked in the manifest",
		Long: `Look up the package in the manifest to find its recorded package manager,
then execute the native reinstallation command for that package.`,
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

			if recordedManager == "" {
				return fmt.Errorf("package %q is not tracked in the manifest", pkgName)
			}

			// Find adapter
			var adapter manager.Adapter
			for _, a := range app.adapters {
				if a.Name() == recordedManager {
					adapter = a
					break
				}
			}

			if adapter == nil {
				return fmt.Errorf("manager %q is not available on this system", recordedManager)
			}

			// Execute native install
			if err := adapter.Install(cmd.Context(), pkgName); err != nil {
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

			// Save manifest to update modified time
			if err := app.saveManifest(); err != nil {
				return fmt.Errorf("failed to save manifest: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "reinstalled %s via %s\n", pkgName, adapter.Name())
			return nil
		},
	}

	return cmd
}
