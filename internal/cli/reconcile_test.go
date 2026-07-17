package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/state"
)

func TestReconcile_NoDrift(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "jq"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No drift detected")
}

func TestReconcile_DriftAndAutoTrack(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "jq"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "ripgrep (brew)")
	assert.Contains(t, output, "Tracked 1 package(s)")
}

func TestReconcile_MultipleManagers(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "ripgrep"},
		},
		&manager.Mock{
			ManagerName:   "dnf",
			InstalledPkgs: []string{"htop", "tmux", "curl"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
		{Manager: "dnf", Packages: []string{"htop"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 3 new package(s)")
	assert.Contains(t, output, "ripgrep (brew)")
	assert.Contains(t, output, "tmux (dnf)")
	assert.Contains(t, output, "curl (dnf)")
	assert.Contains(t, output, "Tracked 3 package(s)")
}

func TestReconcile_FirstRun(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq"},
		},
	}

	snapDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "initial baseline snapshot taken")
}

func TestReconcile_FirstRunDryRun(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq"},
		},
	}

	snapDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile", "--dry-run"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "No baseline snapshot exists")
}

func TestReconcile_NoAdapters(t *testing.T) {
	_, err := execCmd(t, []string{"reconcile"}, []manager.Adapter{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no package managers available")
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

func TestReconcile_AlreadyTracked(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "jq"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	manifestDir := t.TempDir()
	manifestPath := filepath.Join(manifestDir, "manifest.toml")
	manifestContent := `version = 1
system = "linux"

[[packages]]
name = "ripgrep"
manager = "brew"
`

	require.NoError(t, os.MkdirAll(manifestDir, 0700))
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(manifestDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "Tracked 0 package(s)")
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

func TestReconcile_ManagerFlag_Success(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq", "ripgrep"},
		},
		&manager.Mock{
			ManagerName:   "dnf",
			InstalledPkgs: []string{"htop"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
		{Manager: "dnf", Packages: []string{"htop"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile", "-m", "brew"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 2 new package(s)")
	assert.Contains(t, output, "jq")
	assert.Contains(t, output, "ripgrep")
}

func TestReconcile_ManagerFlag_NotFound(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	_, err := execCmd(t, []string{"reconcile", "-m", "nonexistent"}, adapters)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available on this system")
}

func TestReconcile_RepoDrift(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:    "brew",
			InstalledPkgs:  []string{"lazygit"},
			InstalledRepos: []manager.RepositoryInfo{{Name: "aovestdipaperino/tap"}, {Name: "yvgude/lean-ctx"}},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 2 new repository(ies)")
	assert.Contains(t, output, "aovestdipaperino/tap (brew)")
	assert.Contains(t, output, "yvgude/lean-ctx (brew)")
	assert.Contains(t, output, "Tracked 0 package(s), 2 repository(ies)")
}

func TestReconcile_RepoAndPackageDrift(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:    "brew",
			InstalledPkgs:  []string{"lazygit", "ripgrep"},
			InstalledRepos: []manager.RepositoryInfo{{Name: "aovestdipaperino/tap"}},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "Discovered 1 new repository(ies)")
	assert.Contains(t, output, "Tracked 1 package(s), 1 repository(ies)")
}

func TestReconcile_DryRun(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "jq"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile", "--dry-run"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "Use `stamp reconcile` without --dry-run to track")
	assert.NotContains(t, output, "Tracked")
}

func TestReconcile_DryRun_NoDrift(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "jq"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile", "--dry-run"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No drift detected")
}

func TestReconcile_YFlag_Compatibility(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "jq"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile", "--yes"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "Tracked 1 package(s)")
}

func TestReconcile_SnapshotsUpdatedOnNoDrift(t *testing.T) {
	// Simulate: package in baseline → removed externally → reconcile (no drift, save snapshot without it) → reinstalled → reconcile detects drift
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{},
		},
	}

	// Old snapshot has lazygit (which was externally removed)
	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	// First reconcile: no drift (lazygit removed, nothing added), snapshot saved without lazygit
	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No drift detected")

	// Manually verify snapshot no longer has lazygit
	loaded, err := state.Load(filepath.Join(snapDir, "stamp", "snapshots"), "brew")
	require.NoError(t, err)
	assert.NotContains(t, loaded.Packages, "lazygit")

	// Simulate: lazygit reinstalled externally
	adapters[0].(*manager.Mock).InstalledPkgs = []string{"lazygit"}

	// Second reconcile: should now detect lazygit as added
	buf2, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	output := buf2.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "lazygit (brew)")
	assert.Contains(t, output, "Tracked 1 package(s)")
}

func TestReconcile_DryRun_ShortFlag(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "jq"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile", "-d"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "Use `stamp reconcile` without --dry-run to track")
	assert.NotContains(t, output, "Tracked")
}

// setupSnapshots saves snapshots to {tmpDir}/stamp/snapshots/ and returns tmpDir.
func setupSnapshots(t *testing.T, snaps []state.Snapshot) string {
	t.Helper()
	dir := t.TempDir()
	for _, s := range snaps {
		snapDir := filepath.Join(dir, "stamp", "snapshots")
		require.NoError(t, os.MkdirAll(snapDir, 0700))
		require.NoError(t, state.Save(snapDir, s))
	}
	return dir
}
