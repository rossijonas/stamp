package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"

	"github.com/rossijonas/stamp/internal/backup"
)

// Rule defines a pattern-based routing rule.
type Rule struct {
	Pattern string `toml:"pattern"`
	Prefer  string `toml:"prefer"`
}

// BackupConfig controls timestamped backup retention for the manifest and
// snapshots. Semantics mirror logrotate: count cap (rotate), min-age floor
// (minage), and max-age ceiling (maxage). A value of 0 means unlimited on
// that axis.
type BackupConfig struct {
	MaxManifestBackups       int `toml:"max_manifest_backups"`
	MinManifestBackups       int `toml:"min_manifest_backups"`
	MaxSnapshotBackups       int `toml:"max_snapshot_backups"`
	MinSnapshotBackups       int `toml:"min_snapshot_backups"`
	MinManifestBackupAgeDays int `toml:"min_manifest_backup_age_days"`
	MaxManifestBackupAgeDays int `toml:"max_manifest_backup_age_days"`
	MinSnapshotBackupAgeDays int `toml:"min_snapshot_backup_age_days"`
	MaxSnapshotBackupAgeDays int `toml:"max_snapshot_backup_age_days"`
}

// DefaultBackupConfig returns the logrotate-aligned retention defaults.
func DefaultBackupConfig() BackupConfig {
	return BackupConfig{
		MaxManifestBackups:       10,
		MinManifestBackups:       3,
		MaxSnapshotBackups:       10,
		MinSnapshotBackups:       3,
		MinManifestBackupAgeDays: 7,
		MaxManifestBackupAgeDays: 30,
		MinSnapshotBackupAgeDays: 7,
		MaxSnapshotBackupAgeDays: 30,
	}
}

// Validate returns nil when the [backup] retention policy is well-formed, or
// an errors.Join of every misconfigured axis. A value of 0 means unlimited on
// the count axes and disabled on the age axes, so a min-vs-max comparison only
// applies when both sides are non-zero. Misconfiguration is reported by
// `stamp doctor`, never by LoadConfig — an invalid file must not brick the
// other commands (see docs/project/spec.md "Misconfiguration").
func (c BackupConfig) Validate() error {
	return errors.Join(
		backupAxisError(c.MinManifestBackups, c.MaxManifestBackups, "min_manifest_backups", "max_manifest_backups"),
		backupAxisError(c.MinSnapshotBackups, c.MaxSnapshotBackups, "min_snapshot_backups", "max_snapshot_backups"),
		backupAxisError(c.MinManifestBackupAgeDays, c.MaxManifestBackupAgeDays, "min_manifest_backup_age_days", "max_manifest_backup_age_days"),
		backupAxisError(c.MinSnapshotBackupAgeDays, c.MaxSnapshotBackupAgeDays, "min_snapshot_backup_age_days", "max_snapshot_backup_age_days"),
	)
}

// backupAxisError validates one (min, max) axis pair: negatives are rejected,
// and a min floor above a non-zero max cap is a contradiction. Zero disables
// the check on that axis, mirroring the unlimited/disabled semantics.
func backupAxisError(minVal, maxVal int, minKey, maxKey string) error {
	if minVal < 0 {
		return fmt.Errorf("negative value for %s", minKey)
	}
	if maxVal < 0 {
		return fmt.Errorf("negative value for %s", maxKey)
	}
	if minVal > 0 && maxVal > 0 && minVal > maxVal {
		return fmt.Errorf("%s (%d) exceeds %s (%d)", minKey, minVal, maxKey, maxVal)
	}
	return nil
}

// Config represents the user's stamp configuration.
type Config struct {
	Precedence []string     `toml:"precedence"`
	Rules      []Rule       `toml:"rules"`
	Backup     BackupConfig `toml:"backup"`
}

// LoadConfig reads and parses the config.toml file.
// Returns default values if the file does not exist.
func LoadConfig(path string) (*Config, error) {
	cfg := &Config{
		Precedence: []string{"dnf", "flatpak", "brew"},
		Backup:     DefaultBackupConfig(),
	}

	//nolint:gosec // path is resolved internally via XDG config dir
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return cfg, nil
}

// DefaultConfigTOML returns a commented config template with the [backup]
// retention defaults. Written by stamp init when config.toml does not exist.
// The [backup] block is generated from DefaultBackupConfig so the template
// and the parsed defaults can never drift apart.
func DefaultConfigTOML() string {
	backupBlock, err := toml.Marshal(map[string]any{"backup": DefaultBackupConfig()})
	if err != nil {
		panic("failed to marshal backup defaults: " + err.Error()) // unreachable: static int fields
	}
	return `# ~/.config/stamp/config.toml

# Global priority order (highest to lowest)
precedence = ["dnf", "flatpak", "brew"]

# Backup retention policy (logrotate-style)
# A value of 0 on any key means unlimited on that axis.
# min_*_backup_age_days protects recent backups (never deleted).
# min_*_backups always keeps at least this many backups (newest survive).
# max_*_backup_age_days always deletes backups older than this.
# max_*_backups caps how many eligible backups are kept.
` + string(backupBlock) + `
# Pattern-based routing rules override the global precedence
# [[rules]]
# pattern = "^com\\..*|^org\\..*"
# prefer = "flatpak"
`
}

// writeConfigAtomic writes the default config template to path via a temp
// file and atomic rename, so an interrupted write never leaves a partial
// config.toml. Restrictive perms keep any future secrets safe.
func writeConfigAtomic(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}
	//nolint:gosec // path is resolved internally via XDG config dir, not user input
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp config: %w", err)
	}
	tmpName := tmp.Name()
	success := false
	defer func() {
		_ = tmp.Close()
		if !success {
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(0600); err != nil {
		return fmt.Errorf("failed to set config permissions: %w", err)
	}
	if _, err := tmp.Write([]byte(DefaultConfigTOML())); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp config %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to rename config to %s: %w", path, err)
	}
	success = true
	return nil
}

// ManifestPolicy converts the manifest retention keys into a backup.Policy.
func (c BackupConfig) ManifestPolicy() backup.Policy {
	return backup.Policy{
		MaxKeep: c.MaxManifestBackups,
		MinKeep: c.MinManifestBackups,
		MinAge:  time.Duration(c.MinManifestBackupAgeDays) * 24 * time.Hour,
		MaxAge:  time.Duration(c.MaxManifestBackupAgeDays) * 24 * time.Hour,
	}
}

// SnapshotPolicy converts the snapshot retention keys into a backup.Policy.
func (c BackupConfig) SnapshotPolicy() backup.Policy {
	return backup.Policy{
		MaxKeep: c.MaxSnapshotBackups,
		MinKeep: c.MinSnapshotBackups,
		MinAge:  time.Duration(c.MinSnapshotBackupAgeDays) * 24 * time.Hour,
		MaxAge:  time.Duration(c.MaxSnapshotBackupAgeDays) * 24 * time.Hour,
	}
}
