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
a default config.toml (only when absent), and take a baseline snapshot
of currently installed packages for each available package manager.

If stamp is already initialized, the existing manifest and snapshots
are always backed up before creating a fresh state, and old backups are
pruned per the [backup] policy in config.toml. Use -y to skip the
confirmation prompt.`,
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
				if !promptYesNo(cmd.Context(), cmd.ErrOrStderr(), cmd.InOrStdin(), "Continue? [y/N]: ", false) {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "re-init aborted")
					return nil
				}
			}

			if isInit {
				snapDir := state.SnapshotDirPath()
				if _, err := os.Stat(snapDir); err == nil {
					bakPath, bakErr := state.BackupSnapshots(snapDir)
					if bakErr != nil {
						return catErr(ErrCanTCreate, "failed to backup snapshots: %w", bakErr)
					}
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "existing snapshots backed up to %s\n", bakPath)

					if n, err := state.RotateSnapshotBackups(snapDir, app.config.Backup.SnapshotPolicy()); err != nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to rotate snapshot backups: %v\n", err)
					} else if n > 0 {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "rotated %d snapshot backup(s)\n", n)
					}
				}

				bakPath, bakErr := manifest.Backup(app.manifestPath)
				if bakErr != nil {
					return catErr(ErrCanTCreate, "failed to backup manifest: %w", bakErr)
				}
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "existing manifest backed up to %s\n", bakPath)

				if n, err := manifest.RotateBackups(app.manifestPath, app.config.Backup.ManifestPolicy()); err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to rotate manifest backups: %v\n", err)
				} else if n > 0 {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "rotated %d manifest backup(s)\n", n)
				}
			}

			configDir := filepath.Dir(app.manifestPath)
			if err := os.MkdirAll(configDir, 0750); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}

			// Write a default config.toml template when absent. Never overwrites
			// an existing config, and failure is non-fatal. Temp-file + atomic
			// rename so an interrupted write can never leave a partial config.
			if app.configPath != "" {
				if _, err := os.Stat(app.configPath); os.IsNotExist(err) {
					if err := writeConfigAtomic(app.configPath); err != nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to write config: %v\n", err)
					}
				}
			}

			m := &manifest.Manifest{
				Version:   1,
				System:    runtime.GOOS,
				Packages:  []manifest.Package{},
				UpdatedAt: time.Now(),
			}
			if err := m.Save(app.manifestPath); err != nil {
				return catErr(ErrCanTCreate, "failed to create manifest: %w", err)
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
			printSnapshotWarnings(cmd.ErrOrStderr(), snaps)

			for _, s := range snaps {
				if err := state.Save(snapDir, s); err != nil {
					return catErr(ErrCanTCreate, "failed to save snapshot for %s: %w", s.Manager, err)
				}
			}

			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "manifest initialized and system baseline snapshot taken")
			return nil
		},
	}

	return cmd
}
