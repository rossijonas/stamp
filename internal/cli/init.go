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
		Use:     "init",
		Short:   "Initialize manifest.toml and take baseline snapshot",
		Example: "  stamp init\n  stamp init -y",
		Long: `Create the stamp configuration directory, an empty manifest.toml,
and take a baseline snapshot of currently installed packages
for each available package manager.

If stamp is already initialized, the existing manifest and snapshots
are always backed up before creating a fresh state. Use -y to skip
the confirmation prompt.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)

			forceYes, _ := cmd.Flags().GetBool("yes")
			autoAccept := forceYes || (app != nil && app.yes)

			// Check if already initialized
			isInit := false
			if _, err := os.Stat(app.manifestPath); err == nil {
				isInit = true
			}

			if isInit && !autoAccept && isTerminal(cmd.InOrStdin()) {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "⚠ Stamp is already initialized on this system.")
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  This will re-write manifest.toml and baseline snapshots.")
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  The existing manifest will be backed up before continuing.")
				if !promptYesNo(cmd.ErrOrStderr(), cmd.InOrStdin(), "Continue? [y/N]: ", false) {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "re-init aborted")
					return nil
				}
			}

			if isInit {
				snapDir := state.SnapshotDirPath()
				if _, err := os.Stat(snapDir); err == nil {
					bakPath, bakErr := state.BackupSnapshots(snapDir)
					if bakErr != nil {
						return fmt.Errorf("failed to backup snapshots: %w", bakErr)
					}
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "existing snapshots backed up to %s\n", bakPath)
				}

				bakPath, bakErr := manifest.Backup(app.manifestPath)
				if bakErr != nil {
					return fmt.Errorf("failed to backup manifest: %w", bakErr)
				}
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "existing manifest backed up to %s\n", bakPath)
			}

			configDir := filepath.Dir(app.manifestPath)
			if err := os.MkdirAll(configDir, 0750); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}

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
