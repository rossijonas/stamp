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
)

func saveRestoreTerminal(t *testing.T) {
	t.Helper()
	old := isTerminal
	isTerminal = func(_ io.Reader) bool { return true }
	t.Cleanup(func() { isTerminal = old })
}

func createExistingManifest(t *testing.T, path, content string) {
	t.Helper()
	if content == "" {
		content = `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "brew"
`
	}
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
}

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
	root.SetArgs([]string{"init", "--yes"})
	err := root.Execute()
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "manifest initialized and system baseline snapshot taken")
	assert.Contains(t, buf.String(), "existing manifest backed up")

	//nolint:gosec // path is a controlled temp file
	data, err := os.ReadFile(mPath)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "htop")

	// Verify backup exists
	backups, _ := filepath.Glob(filepath.Join(tmpDir, "manifest.toml.*.bak"))
	assert.Len(t, backups, 1)
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

func TestInitCmd_Reinit_InteractiveDecline(t *testing.T) { //nolint:dupl // test structure is intentional
	saveRestoreTerminal(t)

	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	createExistingManifest(t, mPath, "")

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader("n\n"))
	root.SetArgs([]string{"init"})
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "re-init aborted")

	//nolint:gosec // path is a controlled temp file
	data, err := os.ReadFile(mPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "htop")
}

func TestInitCmd_Reinit_InteractiveAccept(t *testing.T) {
	saveRestoreTerminal(t)

	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	t.Setenv("XDG_DATA_HOME", tmpDir)
	createExistingManifest(t, mPath, "")

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader("y\n"))
	root.SetArgs([]string{"init"})
	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "manifest initialized and system baseline snapshot taken")
	assert.Contains(t, output, "existing manifest backed up")
	assert.NotContains(t, output, "re-init aborted")
}

func TestInitCmd_Reinit_AutoAcceptFlag(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	t.Setenv("XDG_DATA_HOME", tmpDir)
	createExistingManifest(t, mPath, "")

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"init", "--yes"})
	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "manifest initialized and system baseline snapshot taken")
	assert.Contains(t, output, "existing manifest backed up")
	assert.NotContains(t, output, "Continue?")
	assert.NotContains(t, output, "re-init aborted")
}

func TestInitCmd_Reinit_NonInteractive(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	t.Setenv("XDG_DATA_HOME", tmpDir)
	createExistingManifest(t, mPath, "")

	// Non-TTY: stdin pipe (as done by execCmd)
	buf := new(bytes.Buffer)
	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	r, w, err := os.Pipe()
	require.NoError(t, err)
	root.SetIn(r)
	require.NoError(t, w.Close())
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"init"})
	err = root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "manifest initialized and system baseline snapshot taken")
	assert.Contains(t, output, "existing manifest backed up")
	assert.NotContains(t, output, "Continue?")
}

func TestInitCmd_Reinit_BackupCreated(t *testing.T) {
	saveRestoreTerminal(t)

	originalContent := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "brew"
`
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	t.Setenv("XDG_DATA_HOME", tmpDir)
	createExistingManifest(t, mPath, originalContent)

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader("y\n"))
	root.SetArgs([]string{"init"})
	err := root.Execute()
	require.NoError(t, err)

	// Find the backup file
	backups, _ := filepath.Glob(filepath.Join(tmpDir, "manifest.toml.*.bak"))
	require.Len(t, backups, 1)

	data, err := os.ReadFile(backups[0])
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(data))
}

func TestInitCmd_Reinit_SnapshotsBackedUp(t *testing.T) {
	saveRestoreTerminal(t)

	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	t.Setenv("XDG_DATA_HOME", tmpDir)
	createExistingManifest(t, mPath, "")

	// Create pre-existing snapshots
	snapDir := filepath.Join(tmpDir, "stamp", "snapshots")
	require.NoError(t, os.MkdirAll(snapDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(snapDir, "brew.json"), []byte(`{"manager":"brew","packages":["htop"]}`), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader("y\n"))
	root.SetArgs([]string{"init"})
	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "existing snapshots backed up")

	// snapshots dir is recreated by SnapshotDir() after backup, check backup instead
	backups, _ := filepath.Glob(filepath.Join(tmpDir, "stamp", "snapshots.*.bak"))
	assert.Len(t, backups, 1, "snapshots backup directory should exist")
}

func TestInitCmd_Reinit_FreshManifest(t *testing.T) {
	saveRestoreTerminal(t)

	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	t.Setenv("XDG_DATA_HOME", tmpDir)
	createExistingManifest(t, mPath, "")

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader("y\n"))
	root.SetArgs([]string{"init"})
	err := root.Execute()
	require.NoError(t, err)

	//nolint:gosec // path is a controlled temp file
	data, err := os.ReadFile(mPath)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "htop")
	assert.Contains(t, string(data), "packages = []")
}

func TestInitCmd_FirstInit_NoPrompt(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}}}
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
	output := buf.String()
	assert.Contains(t, output, "manifest initialized and system baseline snapshot taken")
	assert.NotContains(t, output, "Continue?")
	assert.NotContains(t, output, "backed up")
}
