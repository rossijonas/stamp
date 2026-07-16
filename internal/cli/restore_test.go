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
	"github.com/rossijonas/stamp/internal/state"
)

func TestRestore_Empty(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}
	buf, err := execCmd(t, []string{"restore"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Nothing to restore")
}

func TestRestore_DryRun(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}

	manifestContent := `version = 1
system = "linux"

[[repositories]]
name = "my-tap"
manager = "brew"
url = "https://github.com/my-tap"

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
	root.SetArgs([]string{"restore", "--dry-run"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "▪ Dry Run (Preview):")
	assert.Contains(t, output, "Repositories:")
	assert.Contains(t, output, "  - my-tap (brew) https://github.com/my-tap")
	assert.Contains(t, output, "Packages:")
	assert.Contains(t, output, "  - htop (brew)")
}

func TestRestore_Success(t *testing.T) {
	mockBrew := &manager.Mock{
		ManagerName: "brew",
	}
	adapters := []manager.Adapter{mockBrew}

	manifestContent := `version = 1
system = "linux"

[[repositories]]
name = "my-tap"
manager = "brew"
url = "https://github.com/my-tap"

[[packages]]
name = "htop"
manager = "brew"
`
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))

	t.Setenv("XDG_DATA_HOME", tmpDir)

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"restore"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Phase 1: Restoring Repositories...")
	assert.Contains(t, output, "  restored repository my-tap via brew")
	assert.Contains(t, output, "Phase 2: Restoring Packages...")
	assert.Contains(t, output, "  installed htop via brew")
	assert.Contains(t, output, "Restore completed successfully")

	assert.Contains(t, mockBrew.TrackedRepos, "my-tap")
	assert.Contains(t, mockBrew.InstalledPkgs, "htop")
}

func TestRestore_SnapshotsSaved(t *testing.T) {
	mockBrew := &manager.Mock{
		ManagerName:   "brew",
		InstalledPkgs: []string{"htop"},
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

	t.Setenv("XDG_DATA_HOME", tmpDir)

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"restore"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Restore completed successfully")

	// Verify snapshot was saved
	snapPath := filepath.Join(tmpDir, "stamp", "snapshots", "brew.json")
	_, err = os.Stat(snapPath)
	require.NoError(t, err)

	// Verify snapshot contains the restored package
	loaded, err := state.Load(filepath.Join(tmpDir, "stamp", "snapshots"), "brew")
	require.NoError(t, err)
	assert.Contains(t, loaded.Packages, "htop")
}

func TestRestore_Failures(t *testing.T) {
	mockBrew := &manager.Mock{
		ManagerName: "brew",
		InstallErr:  errors.New("network timeout"),
		AddRepoErr:  errors.New("invalid url"),
	}
	adapters := []manager.Adapter{mockBrew}

	manifestContent := `version = 1
system = "linux"

[[repositories]]
name = "my-tap"
manager = "brew"
url = "https://github.com/my-tap"

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
	root.SetArgs([]string{"restore"})

	err := root.Execute()
	require.Error(t, err)
	output := buf.String()

	assert.Contains(t, output, "  warning: failed to add repository my-tap (brew): invalid url")
	assert.Contains(t, output, "Some packages failed to restore:")
	assert.Contains(t, output, "  - htop (brew): network timeout")
	assert.Contains(t, err.Error(), "failed to restore 1 package(s)")
}

func TestRestore_UnknownManager(t *testing.T) {
	mockBrew := &manager.Mock{
		ManagerName: "brew",
	}
	adapters := []manager.Adapter{mockBrew}

	manifestContent := `version = 1
system = "linux"

[[repositories]]
name = "my-tap"
manager = "dnf"
url = "https://github.com/my-tap"

[[packages]]
name = "htop"
manager = "flatpak"
`
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"restore"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "warning: manager dnf not available for repository my-tap")
	assert.Contains(t, output, "warning: manager flatpak not available, skipping 1 package(s)")
}

func TestRestore_ManagerFlagNoMatch(t *testing.T) {
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

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"restore", "-m", "nonexistent"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Nothing to restore")
}

func TestRestore_WithManagerFlag(t *testing.T) {
	mockBrew := &manager.Mock{
		ManagerName: "brew",
	}
	mockDNF := &manager.Mock{
		ManagerName: "dnf",
	}
	adapters := []manager.Adapter{mockBrew, mockDNF}

	manifestContent := `version = 1
system = "linux"

[[repositories]]
name = "my-tap"
manager = "brew"
url = "https://github.com/my-tap"

[[repositories]]
name = "fedora-copr"
manager = "dnf"
url = "https://copr.fedorainfracloud.org/coprs/user/repo"

[[packages]]
name = "htop"
manager = "brew"

[[packages]]
name = "tmux"
manager = "dnf"
`
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"restore", "-m", "brew"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "restored repository my-tap via brew")
	assert.Contains(t, output, "installed htop via brew")
	assert.NotContains(t, output, "fedora-copr")
	assert.NotContains(t, output, "tmux")

	assert.Contains(t, mockBrew.InstalledPkgs, "htop")
	assert.Contains(t, mockBrew.TrackedRepos, "my-tap")
	assert.NotContains(t, mockDNF.InstalledPkgs, "tmux")
}

func TestRestore_YFlag_Compatibility(t *testing.T) {
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

	t.Setenv("XDG_DATA_HOME", tmpDir)

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"restore", "--yes"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Restore completed successfully")
}

func TestRestore_CorruptedManifest(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	require.NoError(t, os.WriteFile(mPath, []byte("invalid [[toml\n"), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"restore"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}
