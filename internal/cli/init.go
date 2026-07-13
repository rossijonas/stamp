package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manifest"
	"github.com/rossijonas/stamp/internal/state"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize manifest.toml and take baseline snapshot",
		Long: `Create the stamp configuration directory, an empty manifest.toml,
and take a baseline snapshot of currently installed packages
for each available package manager.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)

			configDir := filepath.Dir(app.manifestPath)
			if err := os.MkdirAll(configDir, 0750); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}

			if _, err := os.Stat(app.manifestPath); os.IsNotExist(err) {
				m := &manifest.Manifest{
					Version:   1,
					System:    runtime.GOOS,
					Packages:  []manifest.Package{},
					UpdatedAt: time.Now(),
				}
				if err := m.Save(app.manifestPath); err != nil {
					return fmt.Errorf("failed to create manifest: %w", err)
				}
				app.manifest = m
			}

			snapDir, err := state.SnapshotDir()
			if err != nil {
				return fmt.Errorf("failed to create snapshot directory: %w", err)
			}

			snaps, err := state.Current(cmd.Context(), app.adapters)
			if err != nil {
				return fmt.Errorf("failed to take baseline snapshot: %w", err)
			}

			for _, s := range snaps {
				if err := state.Save(snapDir, s); err != nil {
					return fmt.Errorf("failed to save snapshot for %s: %w", s.Manager, err)
				}
			}

			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "manifest initialized and system baseline snapshot taken")
			return nil
		},
	}

	return cmd
}
