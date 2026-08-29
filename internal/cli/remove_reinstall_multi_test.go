package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

func TestRemoveMany_RequiresManager(t *testing.T) {
	_, err := execCmd(t, []string{"remove", "htop", "atop", "-y"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple packages require --manager")
}

func TestRemoveMany_HappyPath(t *testing.T) {
	buf, err := execCmd(t, []string{"remove", "-m", "brew", "htop", "atop", "-y"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "removed 2 package(s) via brew")
}

func TestRemoveMany_CapabilityError(t *testing.T) {
	_, err := execCmd(t, []string{"remove", "-m", "dnf", "htop", "atop", "-y"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support removing multiple packages at once")
}

func TestRemoveMany_GroupRejected(t *testing.T) {
	_, err := execCmd(t, []string{"remove", "-m", "dnf", "htop", "atop", "--group", "-y"}, []manager.Adapter{&manager.Mock{ManagerName: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--group supports a single package")
}

func TestRemoveMany_InteractiveSkipsFilter(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "dnf", PreviewNoop: true}}

	buf, err := execCmd(t, []string{"remove", "-m", "dnf", "htop", "atop"}, adapters)
	require.NoError(t, err)
	output := buf.String()

	// Interactive (no -y): the absent-package filter must NOT run. Only the
	// gate's batch-level "nothing to do" appears — never per-package.
	assert.NotContains(t, output, "nothing to do: htop via dnf")
	assert.Contains(t, output, "nothing to do: 2 package(s)")
}

func TestRemoveCmd_DnfY_SkipNoop(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "dnf", PreviewNoop: true, InstalledPkgs: []string{"htop"}}}

	tmpDir := t.TempDir()
	root := NewRootCmd(
		WithAdapters(adapters),
		WithManifestPath(tmpDir+"/manifest.toml"),
		WithConfigPath(tmpDir+"/config.toml"),
	)
	content := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "dnf"
`
	require.NoError(t, os.WriteFile(tmpDir+"/manifest.toml", []byte(content), 0600))

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"remove", "htop", "-m", "dnf", "-y"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "nothing to do: htop via dnf")
	assert.NotContains(t, buf.String(), "removed")

	// The package must not be untracked.
	loaded, err := manifest.Load(tmpDir + "/manifest.toml")
	require.NoError(t, err)
	require.Len(t, loaded.Packages, 1)
	assert.Equal(t, "htop", loaded.Packages[0].Name)
}

func TestRemoveMany_DnfY_SkipAbsent(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "dnf", PreviewNoop: true, InstalledPkgs: []string{"htop"}}}

	tmpDir := t.TempDir()
	root := NewRootCmd(
		WithAdapters(adapters),
		WithManifestPath(tmpDir+"/manifest.toml"),
		WithConfigPath(tmpDir+"/config.toml"),
	)
	content := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "dnf"

[[packages]]
name = "atop"
manager = "dnf"
`
	require.NoError(t, os.WriteFile(tmpDir+"/manifest.toml", []byte(content), 0600))

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"remove", "-m", "dnf", "htop", "atop", "-y"})

	err := root.Execute()
	require.NoError(t, err)
	// Both packages preview as no-op: nothing is removed, nothing is untracked.
	assert.Contains(t, buf.String(), "nothing to do: htop via dnf")
	assert.NotContains(t, buf.String(), "removed")

	loaded, err := manifest.Load(tmpDir + "/manifest.toml")
	require.NoError(t, err)
	require.Len(t, loaded.Packages, 2)
}

func TestReinstallMany_RequiresManager(t *testing.T) {
	_, err := execCmd(t, []string{"reinstall", "htop", "atop", "-y"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple packages require --manager")
}

func TestReinstallMany_HappyPath(t *testing.T) {
	buf, err := execCmd(t, []string{"reinstall", "-m", "brew", "htop", "atop", "-y"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "reinstalled 2 package(s) via brew")
}

func TestReinstallMany_CapabilityError(t *testing.T) {
	// mockAdapter lacks BatchReinstaller (like the real snap adapter).
	_, err := execCmd(t, []string{"reinstall", "-m", "snap", "a", "b", "-y"}, []manager.Adapter{&mockAdapter{name: "snap"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support reinstalling multiple packages at once")
}

func TestReinstallMany_TrackedUnderDifferentManagerFailsFast(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "dnf", InstalledPkgs: []string{"htop", "atop"}}}

	tmpDir := t.TempDir()
	root := NewRootCmd(
		WithAdapters(adapters),
		WithManifestPath(tmpDir+"/manifest.toml"),
		WithConfigPath(tmpDir+"/config.toml"),
	)
	// htop is tracked under brew; the batch asks for dnf.
	content := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "brew"
`
	require.NoError(t, os.WriteFile(tmpDir+"/manifest.toml", []byte(content), 0600))

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reinstall", "-m", "dnf", "htop", "atop", "-y"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package htop is tracked under brew, not dnf")
	assert.Contains(t, err.Error(), "-m brew")
	// Fail fast: no reinstall executed, no summary printed.
	assert.NotContains(t, buf.String(), "reinstalled")
}
