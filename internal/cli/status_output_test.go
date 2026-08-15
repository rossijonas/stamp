package cli

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

// forceOutputTTY overrides isOutputTerminal for the duration of the test so
// the icon (TTY) rendering path is exercised. Tests that use it must not run
// in parallel (package var mutation).
func forceOutputTTY(t *testing.T) {
	t.Helper()
	old := isOutputTerminal
	isOutputTerminal = func(io.Writer) bool { return true }
	t.Cleanup(func() { isOutputTerminal = old })
}

func TestInstallCmd_Status_Plain(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"install", "htop", "-y"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "installed htop via dnf\n")
	assert.NotContains(t, out, "▪")
	assert.NotContains(t, out, "✓")
}

func TestInstallCmd_Status_TTY(t *testing.T) {
	forceOutputTTY(t)
	buf, err := execCmd(t, []string{"install", "htop", "-y"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "▪ installing htop via dnf...")
	assert.Contains(t, out, "✓ installed htop via dnf")
}

func TestInstallCmd_Status_Note_Plain(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"install", "htop", "-y", "--note", "system monitor"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "installed htop via dnf (note: system monitor)\n")
	assert.NotContains(t, out, "▪")
	assert.NotContains(t, out, "✓")
}

func TestInstallCmd_Status_Note_TTY(t *testing.T) {
	forceOutputTTY(t)
	buf, err := execCmd(t, []string{"install", "htop", "-y", "--note", "system monitor"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "✓ installed htop via dnf (note: system monitor)")
}

func TestInstallCmd_Status_Batch_Plain(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"install", "htop", "atop", "-m", "brew", "-y"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "installed 2 package(s) via brew\n")
	assert.NotContains(t, out, "▪")
	assert.NotContains(t, out, "✓")
}

func TestInstallCmd_Status_Batch_TTY(t *testing.T) {
	forceOutputTTY(t)
	buf, err := execCmd(t, []string{"install", "htop", "atop", "-m", "brew", "-y"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "▪ installing 2 package(s) via brew...")
	assert.Contains(t, out, "✓ installed 2 package(s) via brew")
}

func TestRemoveCmd_Status_Plain(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"remove", "htop", "-y"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "removed htop via brew\n")
	assert.NotContains(t, out, "▪")
	assert.NotContains(t, out, "✓")
}

func TestRemoveCmd_Status_TTY(t *testing.T) {
	forceOutputTTY(t)
	buf, err := execCmd(t, []string{"remove", "htop", "-y"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "▪ removing htop via brew...")
	assert.Contains(t, out, "✓ removed htop via brew")
}

func TestRemoveCmd_Status_Batch_TTY(t *testing.T) {
	forceOutputTTY(t)
	buf, err := execCmd(t, []string{"remove", "htop", "atop", "-m", "brew", "-y"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "✓ removed 2 package(s) via brew")
}

func TestReinstallCmd_Status_Plain(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"reinstall", "lazygit", "-m", "brew", "-y"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "reinstalled lazygit via brew\n")
	assert.NotContains(t, out, "✓")
}

func TestReinstallCmd_Status_TTY(t *testing.T) {
	forceOutputTTY(t)
	buf, err := execCmd(t, []string{"reinstall", "lazygit", "-m", "brew", "-y"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "▪ reinstalling lazygit via brew...")
	assert.Contains(t, out, "✓ reinstalled lazygit via brew")
}

func TestReinstallCmd_Status_Batch_TTY(t *testing.T) {
	forceOutputTTY(t)
	buf, err := execCmd(t, []string{"reinstall", "-m", "brew", "htop", "atop", "-y"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "✓ reinstalled 2 package(s) via brew")
}
