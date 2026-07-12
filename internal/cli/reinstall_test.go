package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestReinstallCmd_Success(t *testing.T) {
	mockBrew := &manager.Mock{
		ManagerName: "brew",
	}
	adapters := []manager.Adapter{mockBrew}

	manifestContent := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "brew"
`
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))

	// Setup snapshots path for snapshot alignment verification
	t.Setenv("XDG_DATA_HOME", tmpDir)

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reinstall", "htop"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "reinstalled htop via brew")
	assert.Contains(t, mockBrew.InstalledPkgs, "htop")

	// Verify snapshot was created automatically
	snapPath := filepath.Join(tmpDir, "stamp", "snapshots", "brew.json")
	_, err = os.Stat(snapPath)
	require.NoError(t, err)
}

func TestReinstallCmd_NotTracked(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}

	buf, err := execCmd(t, []string{"reinstall", "htop"}, adapters)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not tracked in the manifest")
	assert.Empty(t, buf.String())
}

func TestReinstallCmd_ManagerNotAvailable(t *testing.T) {
	t.Parallel()
	// htop is tracked via dnf, but we only have brew adapter available
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}

	manifestContent := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "dnf"
`
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reinstall", "htop"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not available on this system")
}

func TestReinstallCmd_Failures(t *testing.T) {
	t.Parallel()
	mockBrew := &manager.Mock{
		ManagerName: "brew",
		InstallErr:  errors.New("connection reset"),
	}
	adapters := []manager.Adapter{mockBrew}

	manifestContent := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "brew"
`
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reinstall", "htop"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reinstall failed")
}

func TestReinstallCmd_InvalidName(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"reinstall", "-invalid"}, []manager.Adapter{})
	require.Error(t, err)
}

func TestReinstallCmd_CorruptedManifest(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	require.NoError(t, os.WriteFile(mPath, []byte("invalid [[toml\n"), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	root.SetArgs([]string{"reinstall", "htop"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}
