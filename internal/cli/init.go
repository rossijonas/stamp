package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manifest"
	"github.com/rossijonas/stamp/internal/state"
)

// backupExistingState backs up snapshots first, then the manifest, and
// rotates old backups per the configured policies. Failures to rotate are
// warnings; failures to back up are fatal.
func backupExistingState(app *AppContext, w io.Writer) error {
	snapDir := state.SnapshotDirPath()
	if _, err := os.Stat(snapDir); err == nil {
		bakPath, bakErr := state.BackupSnapshots(snapDir)
		if bakErr != nil {
			return catErr(ErrCanTCreate, "failed to backup snapshots: %w", bakErr)
		}
		_, _ = fmt.Fprintf(w, "existing snapshots backed up to %s\n", bakPath)

		if n, err := state.RotateSnapshotBackups(snapDir, app.config.Backup.SnapshotPolicy()); err != nil {
			_, _ = fmt.Fprintf(w, "warning: failed to rotate snapshot backups: %v\n", err)
		} else if n > 0 {
			_, _ = fmt.Fprintf(w, "rotated %d snapshot backup(s)\n", n)
		}
	}

	bakPath, bakErr := manifest.Backup(app.manifestPath)
	if bakErr != nil {
		return catErr(ErrCanTCreate, "failed to backup manifest: %w", bakErr)
	}
	_, _ = fmt.Fprintf(w, "existing manifest backed up to %s\n", bakPath)

	if n, err := manifest.RotateBackups(app.manifestPath, app.config.Backup.ManifestPolicy()); err != nil {
		_, _ = fmt.Fprintf(w, "warning: failed to rotate manifest backups: %v\n", err)
	} else if n > 0 {
		_, _ = fmt.Fprintf(w, "rotated %d manifest backup(s)\n", n)
	}
	return nil
}

// createFreshManifest ensures the config directory and a default config.toml
// exist, then writes an empty manifest. The config write is non-fatal.
func createFreshManifest(app *AppContext, w io.Writer) error {
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
				_, _ = fmt.Fprintf(w, "warning: failed to write config: %v\n", err)
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
	return nil
}

// takeBaselineSnapshot records the current installed state for every adapter.
func takeBaselineSnapshot(ctx context.Context, app *AppContext, w io.Writer) error {
	snapDir, err := state.SnapshotDir()
	if err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	snaps, err := state.Current(ctx, app.adapters)
	if err != nil {
		return fmt.Errorf("failed to take baseline snapshot: %w", err)
	}
	printSnapshotWarnings(w, snaps)

	for _, s := range snaps {
		if err := state.Save(snapDir, s); err != nil {
			return catErr(ErrCanTCreate, "failed to save snapshot for %s: %w", s.Manager, err)
		}
	}
	return nil
}

// confirmReinit prompts before re-initializing an existing stamp install.
// It returns false when the user declines or is not prompted.
func confirmReinit(cmd *cobra.Command, app *AppContext, autoAccept bool) (proceed, isInit bool) {
	isInit = alreadyInitialized(app.manifestPath)
	if !isInit || autoAccept || !isTerminal(cmd.InOrStdin()) {
		return true, isInit
	}
	errOut := cmd.ErrOrStderr()
	_, _ = fmt.Fprintln(errOut, "⚠ Stamp is already initialized on this system.")
	_, _ = fmt.Fprintln(errOut, "  This will re-write manifest.toml and baseline snapshots.")
	_, _ = fmt.Fprintln(errOut, "  The existing manifest will be backed up before continuing.")
	if !promptYesNo(cmd.Context(), errOut, cmd.InOrStdin(), "Continue? [y/N]: ", false) {
		_, _ = fmt.Fprintln(errOut, "re-init aborted")
		return false, isInit
	}
	return true, isInit
}

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize manifest.toml and take baseline snapshot",
		Example: `  # initialize stamp: config dirs, manifest, and baseline snapshot
  stamp init

  # non-interactive initialization for scripting
  stamp init -y`,
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
			errOut := cmd.ErrOrStderr()

			forceYes, _ := cmd.Flags().GetBool("yes")
			autoAccept := forceYes || (app != nil && app.yes)

			proceed, isInit := confirmReinit(cmd, app, autoAccept)
			if !proceed {
				return nil
			}

			if isInit {
				if err := backupExistingState(app, errOut); err != nil {
					return err
				}
			}

			if err := createFreshManifest(app, errOut); err != nil {
				return err
			}

			if err := takeBaselineSnapshot(cmd.Context(), app, errOut); err != nil {
				return err
			}

			_, _ = fmt.Fprintln(errOut, "manifest initialized and system baseline snapshot taken")
			return nil
		},
	}

	return cmd
}
