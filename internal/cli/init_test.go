package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestInitCmd_CreatesConfigTemplate(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	root.SetArgs([]string{"init"})
	require.NoError(t, root.Execute())

	//nolint:gosec // path is a controlled temp file
	data, err := os.ReadFile(cPath)
	require.NoError(t, err, "config.toml must be created on fresh init")
	content := string(data)
	assert.Contains(t, content, "[backup]")
	assert.Contains(t, content, "max_manifest_backups = 10")
	assert.Contains(t, content, "precedence")
}

func TestInitCmd_DoesNotOverwriteExistingConfig(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	require.NoError(t, os.WriteFile(cPath, []byte("precedence = [\"brew\"]\n"), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	root.SetArgs([]string{"init"})
	require.NoError(t, root.Execute())

	//nolint:gosec // path is a controlled temp file
	data, err := os.ReadFile(cPath)
	require.NoError(t, err)
	assert.Equal(t, "precedence = [\"brew\"]\n", string(data), "existing config must not be overwritten")
}

func TestInitCmd_Reinit_RotatesManifestBackups(t *testing.T) {
	saveRestoreTerminal(t)

	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	t.Setenv("XDG_DATA_HOME", tmpDir)
	createExistingManifest(t, mPath, "")

	// Pre-seed 4 backups + a config capping manifest backups at 2. Age floors
	// are disabled so the count cap alone drives rotation.
	for _, age := range []int{1, 2, 3, 4} {
		ts := time.Now().UTC().Add(-time.Duration(age) * 24 * time.Hour).Format("20060102T150405Z")
		require.NoError(t, os.WriteFile(fmt.Sprintf("%s.%s.bak", mPath, ts), []byte("old"), 0600))
	}
	require.NoError(t, os.WriteFile(cPath, []byte("[backup]\nmax_manifest_backups = 2\nmin_manifest_backups = 0\nmin_manifest_backup_age_days = 0\nmax_manifest_backup_age_days = 0\n"), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader("y\n"))
	root.SetArgs([]string{"init"})
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "rotated 3 manifest backup(s)")

	backups, _ := filepath.Glob(filepath.Join(tmpDir, "manifest.toml.*.bak"))
	assert.Len(t, backups, 2, "re-init rotation must cap manifest backups at the configured max")
}

func TestInitCmd_Reinit_RotatesSnapshotBackups(t *testing.T) {
	saveRestoreTerminal(t)

	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	t.Setenv("XDG_DATA_HOME", tmpDir)
	createExistingManifest(t, mPath, "")

	snapBase := filepath.Join(tmpDir, "stamp", "snapshots")
	require.NoError(t, os.MkdirAll(snapBase, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(snapBase, "brew.json"), []byte(`{"manager":"brew","packages":["htop"]}`), 0600))

	// Pre-seed 3 snapshot backup dirs. Disable age floors/ceilings in config so
	// the count cap alone drives rotation.
	for _, age := range []int{1, 2, 3} {
		ts := time.Now().UTC().Add(-time.Duration(age) * 24 * time.Hour).Format("20060102T150405Z")
		dir := fmt.Sprintf("%s.%s.bak", snapBase, ts)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "brew.json"), []byte("{}"), 0600))
	}
	require.NoError(t, os.WriteFile(cPath, []byte("[backup]\nmax_snapshot_backups = 2\nmin_snapshot_backups = 0\nmin_snapshot_backup_age_days = 0\nmax_snapshot_backup_age_days = 0\n"), 0600))

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
	assert.Contains(t, output, "rotated 2 snapshot backup(s)")

	// 1 (fresh re-init) + 3 (pre-seeded) backups, minus 2 rotated = 2 remain.
	backups, _ := filepath.Glob(filepath.Join(tmpDir, "stamp", "snapshots.*.bak"))
	assert.Len(t, backups, 2, "re-init rotation must cap snapshot backups at the configured max")
}
