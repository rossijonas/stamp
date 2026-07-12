package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
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

	buf, err := execCmd(t, []string{"reconcile", "--yes"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No drift detected")
}

func TestReconcile_NonTTYAutoTrack(t *testing.T) {
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

	// Do NOT pass --yes. In non-TTY (CI), reconcile should auto-track.
	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "Tracked 1 package(s)")
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

	buf, err := execCmd(t, []string{"reconcile", "--yes"}, adapters)
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

	buf, err := execCmd(t, []string{"reconcile", "--yes"}, adapters)
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

	buf, err := execCmd(t, []string{"reconcile", "--yes"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "initial baseline snapshot taken")
}

func TestReconcile_NoAdapters(t *testing.T) {
	_, err := execCmd(t, []string{"reconcile", "--yes"}, []manager.Adapter{})
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

	// Simulate XDG dir and write invalid JSON into a snapshot
	snapDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", snapDir)

	// Write a corrupted snapshot to trigger error on Load
	// The correct structure needs to be in {snapDir}/stamp/snapshots/brew.json
	snapshotDir := filepath.Join(snapDir, "stamp", "snapshots")
	require.NoError(t, os.MkdirAll(snapshotDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(snapshotDir, "brew.json"), []byte("{invalid"), 0600))

	_, err := execCmd(t, []string{"reconcile", "--yes"}, adapters)
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

	// Pre-create the manifest path with ripgrep already tracked
	manifestDir := t.TempDir()
	manifestPath := filepath.Join(manifestDir, "manifest.toml")
	// Write a manifest with ripgrep already present to test duplicate dedup
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
	root.SetArgs([]string{"reconcile", "--yes"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "Tracked 0 package(s)")
}

func TestIsTerminal(t *testing.T) {
	// Pipe is not a terminal
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()

	assert.False(t, isTerminal(r))
	assert.False(t, isTerminal(w))
	// nil reader is not a terminal
	assert.False(t, isTerminal(nil))
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

	_, err := execCmd(t, []string{"reconcile", "--yes"}, adapters)
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
	root.SetArgs([]string{"reconcile", "--yes"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}

func TestReconcile_Cancel(t *testing.T) {
	oldIsTerminal := isTerminal
	isTerminal = func(_ io.Reader) bool { return true }
	defer func() { isTerminal = oldIsTerminal }()

	adapters := []manager.Adapter{&manager.Mock{
		ManagerName:   "brew",
		InstalledPkgs: []string{"lazygit", "ripgrep"},
	}}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	root := NewRootCmd(WithAdapters(adapters), WithConfigPath(filepath.Join(t.TempDir(), "config.toml")), WithManifestPath(filepath.Join(t.TempDir(), "manifest.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader("n\n"))
	root.SetArgs([]string{"reconcile"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "Packages not tracked")
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

	// Reconcile only brew — should discover ripgrep, ignore dnf changes
	buf, err := execCmd(t, []string{"reconcile", "--yes", "-m", "brew"}, adapters)
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

	_, err := execCmd(t, []string{"reconcile", "--yes", "-m", "nonexistent"}, adapters)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available on this system")
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
