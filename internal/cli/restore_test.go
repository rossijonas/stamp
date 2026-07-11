package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestRestore_Empty(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}
	buf, err := execCmd(t, []string{"restore"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Nothing to restore")
}

func TestRestore_DryRun(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}

	// Setup manifest with repo and package
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

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"restore", "--yes"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Phase 1: Restoring Repositories...")
	assert.Contains(t, output, "  restored repository my-tap via brew")
	assert.Contains(t, output, "Phase 2: Restoring Packages...")
	assert.Contains(t, output, "  installed htop via brew")
	assert.Contains(t, output, "Restore completed successfully")

	// Verify state actually changed in the mock
	assert.Contains(t, mockBrew.TrackedRepos, "my-tap")
	assert.Contains(t, mockBrew.InstalledPkgs, "htop")
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
	root.SetArgs([]string{"restore", "--yes"})

	err := root.Execute()
	require.Error(t, err)
	output := buf.String()

	assert.Contains(t, output, "  warning: failed to add repository my-tap (brew): invalid url")
	assert.Contains(t, output, "Some packages failed to restore:")
	assert.Contains(t, output, "  - htop (brew): network timeout")
	assert.Contains(t, err.Error(), "failed to restore 1 package(s)")
}

func TestRestore_NonTTYAutoTrack(t *testing.T) {
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

	// No --yes, non-TTY input. Should auto-track.
	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader("n\n")) // non-terminal inputs default to auto-accept
	root.SetArgs([]string{"restore"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Restore completed successfully")
	assert.Contains(t, mockBrew.InstalledPkgs, "htop")
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
manager = "dnf" # not in adapters
url = "https://github.com/my-tap"

[[packages]]
name = "htop"
manager = "flatpak" # not in adapters
`
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"restore", "--yes"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "warning: manager dnf not available for repository my-tap")
	assert.Contains(t, output, "warning: manager flatpak not available, skipping 1 package(s)")
}

func TestRestore_InteractiveConfirm(t *testing.T) {
	oldIsTerminal := isTerminal
	isTerminal = func(_ io.Reader) bool { return true }
	defer func() { isTerminal = oldIsTerminal }()

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
	root.SetIn(strings.NewReader("y\n")) // mock confirm
	root.SetArgs([]string{"restore"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Restore completed successfully")
	assert.Contains(t, mockBrew.InstalledPkgs, "htop")
}

func TestRestore_InteractiveCancel(t *testing.T) {
	oldIsTerminal := isTerminal
	isTerminal = func(_ io.Reader) bool { return true }
	defer func() { isTerminal = oldIsTerminal }()

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
	root.SetIn(strings.NewReader("n\n")) // mock cancel
	root.SetArgs([]string{"restore"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Restore cancelled")
	assert.NotContains(t, mockBrew.InstalledPkgs, "htop")
}

func TestRestore_InteractiveReadError(t *testing.T) {
	oldIsTerminal := isTerminal
	isTerminal = func(_ io.Reader) bool { return true }
	defer func() { isTerminal = oldIsTerminal }()

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
	root.SetIn(strings.NewReader("")) // empty reader triggers read error on ReadString
	root.SetArgs([]string{"restore"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Restore cancelled")
	assert.NotContains(t, mockBrew.InstalledPkgs, "htop")
}
