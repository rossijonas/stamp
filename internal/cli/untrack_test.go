package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

// seedManifest writes a manifest containing pkgs and returns its path.
func seedManifest(t *testing.T, pkgs ...manifest.Package) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.toml")
	m := &manifest.Manifest{Version: 1}
	for _, p := range pkgs {
		m.AddPackage(p)
	}
	require.NoError(t, m.Save(path))
	return path
}

func loadManifest(t *testing.T, path string) *manifest.Manifest {
	t.Helper()
	m, err := manifest.Load(path)
	require.NoError(t, err)
	return m
}

func TestUntrackCmd_YesUntracksManifestEntry(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t,
		manifest.Package{Name: "yarn", Manager: "dnf"},
		manifest.Package{Name: "htop", Manager: "dnf"},
	)
	cPath := filepath.Join(t.TempDir(), "config.toml")

	buf, err := execCmdAt(t, []string{"untrack", "yarn", "-m", "dnf", "-y"},
		[]manager.Adapter{}, mPath, cPath)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "untracked yarn (dnf)")

	loaded := loadManifest(t, mPath)
	assert.False(t, loaded.HasPackage("yarn", "dnf"))
	assert.True(t, loaded.HasPackage("htop", "dnf"), "unrelated entries must survive")
}

func TestUntrackCmd_ManagerAlias(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t, manifest.Package{Name: "yarn", Manager: "dnf"})
	cPath := filepath.Join(t.TempDir(), "config.toml")

	_, err := execCmdAt(t, []string{"untrack", "yarn", "-m", "yum", "-y"},
		[]manager.Adapter{}, mPath, cPath)
	require.NoError(t, err)
	assert.False(t, loadManifest(t, mPath).HasPackage("yarn", "dnf"))
}

func TestUntrackCmd_AmbiguousRequiresManager(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t,
		manifest.Package{Name: "tool", Manager: "dnf"},
		manifest.Package{Name: "tool", Manager: "brew"},
	)
	cPath := filepath.Join(t.TempDir(), "config.toml")

	_, err := execCmdAt(t, []string{"untrack", "tool", "-y"},
		[]manager.Adapter{}, mPath, cPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tracked under multiple managers")
	assert.Contains(t, err.Error(), "brew, dnf")
	assert.Equal(t, ExitUsage, exitCodeFor(err))
	assert.True(t, loadManifest(t, mPath).HasPackage("tool", "dnf"), "ambiguous name must not mutate")
}

func TestUntrackCmd_NotTracked(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t, manifest.Package{Name: "htop", Manager: "dnf"})
	cPath := filepath.Join(t.TempDir(), "config.toml")

	_, err := execCmdAt(t, []string{"untrack", "ghost", "-y"},
		[]manager.Adapter{}, mPath, cPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `package "ghost" is not tracked`)
}

func TestUntrackCmd_WrongManager(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t, manifest.Package{Name: "yarn", Manager: "dnf"})
	cPath := filepath.Join(t.TempDir(), "config.toml")

	_, err := execCmdAt(t, []string{"untrack", "yarn", "-m", "brew", "-y"},
		[]manager.Adapter{}, mPath, cPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `not tracked under "brew"`)
	assert.True(t, loadManifest(t, mPath).HasPackage("yarn", "dnf"))
}

func TestUntrackCmd_NonInteractiveRefusesWithoutYes(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t, manifest.Package{Name: "yarn", Manager: "dnf"})
	cPath := filepath.Join(t.TempDir(), "config.toml")

	_, err := execCmdAt(t, []string{"untrack", "yarn", "-m", "dnf"},
		[]manager.Adapter{}, mPath, cPath)
	require.Error(t, err)
	require.ErrorIs(t, err, errNonInteractive)
	assert.Contains(t, err.Error(), "untrack yarn via dnf")
	assert.True(t, loadManifest(t, mPath).HasPackage("yarn", "dnf"), "refused run must not mutate")
}

func TestUntrackCmd_InteractiveAccept(t *testing.T) {
	saveRestoreTerminal(t)
	mPath := seedManifest(t, manifest.Package{Name: "yarn", Manager: "dnf"})
	cPath := filepath.Join(t.TempDir(), "config.toml")

	buf := new(bytes.Buffer)
	root := NewRootCmd(WithAdapters([]manager.Adapter{}), WithManifestPath(mPath), WithConfigPath(cPath))
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(newLineReader("y"))
	root.SetArgs([]string{"untrack", "yarn", "-m", "dnf"})

	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "Untrack yarn (dnf)? [y/N]: ")
	assert.Contains(t, buf.String(), "untracked yarn (dnf)")
	assert.False(t, loadManifest(t, mPath).HasPackage("yarn", "dnf"))
}

func TestUntrackCmd_InteractiveDecline(t *testing.T) {
	saveRestoreTerminal(t)
	mPath := seedManifest(t, manifest.Package{Name: "yarn", Manager: "dnf"})
	cPath := filepath.Join(t.TempDir(), "config.toml")

	buf := new(bytes.Buffer)
	root := NewRootCmd(WithAdapters([]manager.Adapter{}), WithManifestPath(mPath), WithConfigPath(cPath))
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(newLineReader("n"))
	root.SetArgs([]string{"untrack", "yarn", "-m", "dnf"})

	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "aborted")
	assert.True(t, loadManifest(t, mPath).HasPackage("yarn", "dnf"), "declined run must not mutate")
}
