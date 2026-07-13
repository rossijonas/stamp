package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestInitCmd_CreatesManifest(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "stamp", "manifest.toml")
	cPath := filepath.Join(tmpDir, "stamp", "config.toml")

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	root.SetArgs([]string{"init"})
	err := root.Execute()
	require.NoError(t, err)

	_, err = os.Stat(mPath)
	require.NoError(t, err)
}

func TestInitCmd_SnapshotsBaseline(t *testing.T) {
	mockBrew := &manager.Mock{
		ManagerName:   "brew",
		InstalledPkgs: []string{"lazygit", "jq"},
	}
	adapters := []manager.Adapter{mockBrew}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	t.Setenv("XDG_DATA_HOME", tmpDir)

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"init"})
	err := root.Execute()
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "manifest initialized and system baseline snapshot taken")

	snapPath := filepath.Join(tmpDir, "stamp", "snapshots", "brew.json")
	_, err = os.Stat(snapPath)
	require.NoError(t, err)
}

func TestInitCmd_ExistingManifest(t *testing.T) {
	mockBrew := &manager.Mock{
		ManagerName:   "brew",
		InstalledPkgs: []string{"htop"},
	}
	adapters := []manager.Adapter{mockBrew}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	t.Setenv("XDG_DATA_HOME", tmpDir)

	require.NoError(t, os.WriteFile(mPath, []byte(`version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "brew"
`), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"init"})
	err := root.Execute()
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "manifest initialized and system baseline snapshot taken")

	//nolint:gosec // path is a controlled temp file
	data, err := os.ReadFile(mPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "htop")
}

func TestInitCmd_NoAdapters(t *testing.T) {
	adapters := []manager.Adapter{}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"init"})
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "manifest initialized and system baseline snapshot taken")

	_, err = os.Stat(mPath)
	require.NoError(t, err)
}
