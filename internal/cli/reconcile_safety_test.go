package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
	"github.com/rossijonas/stamp/internal/state"
)

func TestReconcile_DriftCreatesManifestBackup(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(manifestPath, []byte("version = 1\nsystem = \"linux\"\n"), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile"})

	require.NoError(t, root.Execute())

	backups, err := filepath.Glob(manifestPath + ".*.bak")
	require.NoError(t, err)
	assert.Len(t, backups, 1, "reconcile drift must create a manifest backup")
}

func TestReconcile_DryRunNoBackup(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(manifestPath, []byte("version = 1\nsystem = \"linux\"\n"), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile", "--dry-run"})

	require.NoError(t, root.Execute())

	backups, err := filepath.Glob(manifestPath + ".*.bak")
	require.NoError(t, err)
	assert.Empty(t, backups, "reconcile --dry-run must not create backups")
}

func TestReconcile_NoDriftNoBackup(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(manifestPath, []byte("version = 1\nsystem = \"linux\"\n"), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile"})

	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "No drift detected")

	backups, err := filepath.Glob(manifestPath + ".*.bak")
	require.NoError(t, err)
	assert.Empty(t, backups, "reconcile with no drift must not create backups")
}

func TestReconcile_DriftRotatesBackups(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(manifestPath, []byte("version = 1\nsystem = \"linux\"\n"), 0600))

	// Pre-seed 4 backups, then a config capping manifest backups at 2. The
	// max-age ceiling is disabled so the count cap alone drives rotation.
	for _, age := range []int{20, 40, 60, 80} {
		ts := time.Now().UTC().Add(-time.Duration(age) * 24 * time.Hour).Format("20060102T150405Z")
		require.NoError(t, os.WriteFile(fmt.Sprintf("%s.%s.bak", manifestPath, ts), []byte("old"), 0600))
	}
	configContent := "[backup]\nmax_manifest_backups = 2\nmin_manifest_backups = 0\nmax_manifest_backup_age_days = 0\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile"})

	require.NoError(t, root.Execute())
	output := buf.String()
	assert.Contains(t, output, "rotated 2 manifest backup(s)")

	backups, err := filepath.Glob(manifestPath + ".*.bak")
	require.NoError(t, err)
	assert.Len(t, backups, 3, "rotation must cap manifest backups at the configured max plus the min-age-protected fresh backup")
}

func TestReconcile_DriftDefaultMinKeepSurvives(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(manifestPath, []byte("version = 1\nsystem = \"linux\"\n"), 0600))

	// All backups older than the default 30-day ceiling — the min-count floor
	// (default 3) must stop the ceiling from wiping everything.
	for _, age := range []int{40, 50, 60, 70} {
		ts := time.Now().UTC().Add(-time.Duration(age) * 24 * time.Hour).Format("20060102T150405Z")
		require.NoError(t, os.WriteFile(fmt.Sprintf("%s.%s.bak", manifestPath, ts), []byte("old"), 0600))
	}

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile"})

	require.NoError(t, root.Execute())

	backups, err := filepath.Glob(manifestPath + ".*.bak")
	require.NoError(t, err)
	assert.Len(t, backups, 3, "default min_manifest_backups=3 must survive the max-age ceiling")
}

func TestReconcile_NoAdapters(t *testing.T) {
	_, err := execCmd(t, []string{"reconcile"}, []manager.Adapter{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no package managers available")
	assert.Equal(t, ExitUnavailable, exitCodeFor(err))
}

func TestReconcile_CorruptSnapshot(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit"},
		},
	}

	snapDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", snapDir)

	snapshotDir := filepath.Join(snapDir, "stamp", "snapshots")
	require.NoError(t, os.MkdirAll(snapshotDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(snapshotDir, "brew.json"), []byte("{invalid"), 0600))

	_, err := execCmd(t, []string{"reconcile"}, adapters)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load snapshot for brew")
}

func TestReconcile_ListInstalledError(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName: "brew",
			ListErr:     assert.AnError,
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "jq"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	_, err := execCmd(t, []string{"reconcile"}, adapters)
	require.Error(t, err)
}

func TestReconcile_CorruptedManifest(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	require.NoError(t, os.WriteFile(mPath, []byte("invalid [[toml\n"), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}

func TestReconcile_SnapshotDirError(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}
	t.Setenv("XDG_DATA_HOME", "/root")

	_, err := execCmd(t, []string{"reconcile"}, adapters)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to access snapshot directory")
}

func TestReconcile_ManifestSaveError(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})
	t.Setenv("XDG_DATA_HOME", snapDir)

	manifestDir := t.TempDir()
	manifestPath := filepath.Join(manifestDir, "manifest.toml")
	require.NoError(t, os.WriteFile(manifestPath, []byte("version = 1\nsystem = \"linux\"\n"), 0600))
	//nolint:gosec // test fixture: set directory read-only to trigger save error
	require.NoError(t, os.Chmod(manifestDir, 0500))
	t.Cleanup(func() {
		_ = os.Chmod(manifestDir, 0700) //nolint:gosec // restore permissions
	})

	configPath := filepath.Join(manifestDir, "config.toml")
	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(configPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save manifest")
}

func TestReconcile_ShowsWarningOnListReposError(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit"},
			ListReposErr:  errors.New("brew tap list failed"),
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{}},
	})
	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "warning: brew: failed to list repositories: brew tap list failed")
}

func TestRenderMissing(t *testing.T) {
	t.Run("small list inlines names", func(t *testing.T) {
		buf := new(bytes.Buffer)
		renderMissing(buf, []manifest.Package{
			{Name: "foo", Manager: "dnf"},
			{Name: "bar", Manager: "brew"},
		})
		out := buf.String()
		assert.Contains(t, out, "2 manifest package(s) not installed: foo (dnf), bar (brew)")
		assert.Contains(t, out, "stamp ls --type missing")
	})

	t.Run("large list omits inline names", func(t *testing.T) {
		buf := new(bytes.Buffer)
		renderMissing(buf, []manifest.Package{
			{Name: "p1", Manager: "dnf"},
			{Name: "p2", Manager: "dnf"},
			{Name: "p3", Manager: "dnf"},
			{Name: "p4", Manager: "dnf"},
			{Name: "p5", Manager: "dnf"},
			{Name: "p6", Manager: "dnf"},
		})
		out := buf.String()
		assert.Contains(t, out, "6 manifest package(s) not installed")
		assert.NotContains(t, out, "p1 (dnf)")
	})

	t.Run("empty input writes nothing", func(t *testing.T) {
		buf := new(bytes.Buffer)
		renderMissing(buf, nil)
		assert.Empty(t, buf.String())
	})
}
