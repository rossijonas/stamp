package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

func TestReinstallCmd_TrackedManagerWinsOverFlag(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	mPath := seedManifest(t, manifest.Package{Name: "htop", Manager: "dnf"})
	cPath := filepath.Join(t.TempDir(), "config.toml")
	dnf := &recordingAdapter{mockAdapter: mockAdapter{name: "dnf"}}
	brew := &recordingAdapter{mockAdapter: mockAdapter{name: "brew"}}

	_, err := execCmdAt(t, []string{"reinstall", "htop", "-m", "brew", "-y"},
		[]manager.Adapter{dnf, brew}, mPath, cPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"htop"}, dnf.reinstal, "tracked manager must win")
	assert.Empty(t, brew.reinstal, "-m must not override a tracked package")
}

func TestReinstallCmd_FlagUsedForPreExisting(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	mPath := seedManifest(t)
	cPath := filepath.Join(t.TempDir(), "config.toml")
	brew := &recordingAdapter{mockAdapter: mockAdapter{name: "brew"}}

	_, err := execCmdAt(t, []string{"reinstall", "htop", "-m", "brew", "-y"},
		[]manager.Adapter{brew}, mPath, cPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"htop"}, brew.reinstal)
}

func TestReinstallCmd_TrackedUnavailableManagerErrors(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	mPath := seedManifest(t, manifest.Package{Name: "htop", Manager: "flatpak"})
	cPath := filepath.Join(t.TempDir(), "config.toml")
	dnf := &recordingAdapter{mockAdapter: mockAdapter{name: "dnf"}}

	_, err := execCmdAt(t, []string{"reinstall", "htop", "-m", "dnf", "-y"},
		[]manager.Adapter{dnf}, mPath, cPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not available on this system")
	assert.Empty(t, dnf.reinstal, "-m must not fall back when the recorded manager is gone")
}
