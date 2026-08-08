package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
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
