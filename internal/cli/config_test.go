package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_FileNotFound(t *testing.T) {
	t.Parallel()
	cfg, err := LoadConfig("/nonexistent/path/config.toml")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, []string{"dnf", "flatpak", "brew"}, cfg.Precedence)
	assert.Empty(t, cfg.Rules)
}

func TestLoadConfig_ValidFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.toml")
	content := []byte(`
precedence = ["brew", "dnf"]

[[rules]]
pattern = "^com\\..*"
prefer = "flatpak"
`)
	err := os.WriteFile(path, content, 0600)
	require.NoError(t, err)

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, []string{"brew", "dnf"}, cfg.Precedence)
	require.Len(t, cfg.Rules, 1)
	assert.Equal(t, "^com\\..*", cfg.Rules[0].Pattern)
	assert.Equal(t, "flatpak", cfg.Rules[0].Prefer)
}

func TestLoadConfig_InvalidTOML(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.toml")
	err := os.WriteFile(path, []byte("invalid [[toml\n"), 0600)
	require.NoError(t, err)

	_, err = LoadConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config")
}

func TestLoadConfig_ReadError(t *testing.T) {
	t.Parallel()
	_, err := LoadConfig("/proc/1/root/config.toml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config")
}

func TestNewAppContext_ConfigParseError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	cPath := filepath.Join(tmpDir, "config.toml")
	mPath := filepath.Join(tmpDir, "manifest.toml")

	require.NoError(t, os.WriteFile(cPath, []byte("invalid [[toml\n"), 0600))

	_, err := newAppContext(false, false, false, nil, cPath, mPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid configuration")
	assert.Equal(t, ExitConfig, exitCodeFor(err))
}

func TestNewAppContext_ManifestParseError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")

	require.NoError(t, os.WriteFile(mPath, []byte("invalid [[toml\n"), 0600))

	ctx, err := newAppContext(false, false, false, nil, "/nonexistent/config", mPath)
	require.NoError(t, err)
	require.NotNil(t, ctx)
	require.Error(t, ctx.manifestErr)
	assert.Contains(t, ctx.manifestErr.Error(), "failed to parse manifest")
}

func TestDefaultBackupConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultBackupConfig()
	assert.Equal(t, 10, cfg.MaxManifestBackups)
	assert.Equal(t, 3, cfg.MinManifestBackups)
	assert.Equal(t, 10, cfg.MaxSnapshotBackups)
	assert.Equal(t, 3, cfg.MinSnapshotBackups)
	assert.Equal(t, 7, cfg.MinManifestBackupAgeDays)
	assert.Equal(t, 30, cfg.MaxManifestBackupAgeDays)
	assert.Equal(t, 7, cfg.MinSnapshotBackupAgeDays)
	assert.Equal(t, 30, cfg.MaxSnapshotBackupAgeDays)
}

func TestLoadConfig_BackupDefaultsWhenAbsent(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.toml")
	err := os.WriteFile(path, []byte("precedence = [\"dnf\"]\n"), 0600)
	require.NoError(t, err)

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, DefaultBackupConfig(), cfg.Backup)
}

func TestLoadConfig_BackupExplicit(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.toml")
	content := []byte(`
[backup]
max_manifest_backups = 5
min_manifest_backups = 4
max_snapshot_backups = 3
min_snapshot_backups = 2
min_manifest_backup_age_days = 2
max_manifest_backup_age_days = 20
min_snapshot_backup_age_days = 1
max_snapshot_backup_age_days = 15
`)
	err := os.WriteFile(path, content, 0600)
	require.NoError(t, err)

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, 5, cfg.Backup.MaxManifestBackups)
	assert.Equal(t, 4, cfg.Backup.MinManifestBackups)
	assert.Equal(t, 3, cfg.Backup.MaxSnapshotBackups)
	assert.Equal(t, 2, cfg.Backup.MinSnapshotBackups)
	assert.Equal(t, 2, cfg.Backup.MinManifestBackupAgeDays)
	assert.Equal(t, 20, cfg.Backup.MaxManifestBackupAgeDays)
	assert.Equal(t, 1, cfg.Backup.MinSnapshotBackupAgeDays)
	assert.Equal(t, 15, cfg.Backup.MaxSnapshotBackupAgeDays)
}

func TestLoadConfig_BackupZeroIsUnlimited(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.toml")
	err := os.WriteFile(path, []byte(`
[backup]
max_manifest_backups = 0
min_manifest_backups = 0
max_snapshot_backups = 0
min_snapshot_backups = 0
min_manifest_backup_age_days = 0
max_manifest_backup_age_days = 0
min_snapshot_backup_age_days = 0
max_snapshot_backup_age_days = 0
`), 0600)
	require.NoError(t, err)

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	assert.Zero(t, cfg.Backup.MaxManifestBackups)
	assert.Zero(t, cfg.Backup.MinManifestBackups)
	assert.Zero(t, cfg.Backup.MaxSnapshotBackups)
	assert.Zero(t, cfg.Backup.MinSnapshotBackups)
	assert.Zero(t, cfg.Backup.MinManifestBackupAgeDays)
	assert.Zero(t, cfg.Backup.MaxManifestBackupAgeDays)
	assert.Zero(t, cfg.Backup.MinSnapshotBackupAgeDays)
	assert.Zero(t, cfg.Backup.MaxSnapshotBackupAgeDays)
}

func TestLoadConfig_BackupPartialKeepsDefaults(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.toml")
	err := os.WriteFile(path, []byte(`
[backup]
max_manifest_backups = 2
`), 0600)
	require.NoError(t, err)

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, 2, cfg.Backup.MaxManifestBackups)
	assert.Equal(t, DefaultBackupConfig().MaxSnapshotBackups, cfg.Backup.MaxSnapshotBackups)
	assert.Equal(t, DefaultBackupConfig().MinManifestBackupAgeDays, cfg.Backup.MinManifestBackupAgeDays)
}

func TestDefaultConfigTOML(t *testing.T) {
	t.Parallel()
	tomlStr := DefaultConfigTOML()
	require.Contains(t, tomlStr, "[backup]")
	require.Contains(t, tomlStr, "max_manifest_backups")
	require.Contains(t, tomlStr, "min_manifest_backups")
	require.Contains(t, tomlStr, "max_snapshot_backups")
	require.Contains(t, tomlStr, "min_snapshot_backups")
	require.Contains(t, tomlStr, "min_manifest_backup_age_days")
	require.Contains(t, tomlStr, "max_manifest_backup_age_days")
	require.Contains(t, tomlStr, "min_snapshot_backup_age_days")
	require.Contains(t, tomlStr, "max_snapshot_backup_age_days")

	var cfg Config
	require.NoError(t, toml.Unmarshal([]byte(tomlStr), &cfg))
	assert.Equal(t, DefaultBackupConfig(), cfg.Backup)
	assert.Equal(t, []string{"dnf", "flatpak", "brew"}, cfg.Precedence)
}

func TestBackupConfig_ManifestPolicy(t *testing.T) {
	t.Parallel()
	c := BackupConfig{
		MaxManifestBackups:       5,
		MinManifestBackups:       2,
		MinManifestBackupAgeDays: 7,
		MaxManifestBackupAgeDays: 30,
	}
	p := c.ManifestPolicy()
	assert.Equal(t, 5, p.MaxKeep)
	assert.Equal(t, 2, p.MinKeep)
	assert.Equal(t, 7*24*time.Hour, p.MinAge)
	assert.Equal(t, 30*24*time.Hour, p.MaxAge)
}

func TestBackupConfig_SnapshotPolicy(t *testing.T) {
	t.Parallel()
	c := BackupConfig{
		MaxSnapshotBackups:       3,
		MinSnapshotBackups:       1,
		MinSnapshotBackupAgeDays: 2,
		MaxSnapshotBackupAgeDays: 15,
	}
	p := c.SnapshotPolicy()
	assert.Equal(t, 3, p.MaxKeep)
	assert.Equal(t, 1, p.MinKeep)
	assert.Equal(t, 2*24*time.Hour, p.MinAge)
	assert.Equal(t, 15*24*time.Hour, p.MaxAge)
}

func TestBackupConfig_PolicyZeroAxes(t *testing.T) {
	t.Parallel()
	c := BackupConfig{}
	p := c.ManifestPolicy()
	assert.Zero(t, p.MaxKeep)
	assert.Zero(t, p.MinAge)
	assert.Zero(t, p.MaxAge)
}

func TestWriteConfigAtomic_Success(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, writeConfigAtomic(path))
	//nolint:gosec // test reads back the file it just wrote
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var cfg Config
	require.NoError(t, toml.Unmarshal(data, &cfg))
	assert.Equal(t, DefaultBackupConfig(), cfg.Backup)
}

func TestWriteConfigAtomic_CreateTempError(t *testing.T) {
	roDir := t.TempDir()
	//nolint:gosec // simulate a write-protected dir to force a temp-create error
	require.NoError(t, os.Chmod(roDir, 0500))
	t.Cleanup(func() { _ = os.Chmod(roDir, 0700) }) //nolint:gosec // restore perms for cleanup

	err := writeConfigAtomic(filepath.Join(roDir, "config.toml"))
	require.Error(t, err)
}

func TestWriteConfigAtomic_RenameError(t *testing.T) {
	dir := t.TempDir()
	// Target is an existing directory, so the rename cannot overwrite it.
	target := filepath.Join(dir, "config.toml")
	require.NoError(t, os.Mkdir(target, 0750))

	err := writeConfigAtomic(target)
	require.Error(t, err)
	// Temp file must have been cleaned up.
	leftovers, err := filepath.Glob(filepath.Join(dir, "config.toml.*.tmp"))
	require.NoError(t, err)
	assert.Empty(t, leftovers)
}
