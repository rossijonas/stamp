package cli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

// resolverAdapter is a mock adapter that reports a canonical installed name.
type resolverAdapter struct {
	mockAdapter
	name  string
	err   error
	calls int
}

func (r *resolverAdapter) ResolveInstalledName(_ context.Context, _ string) (string, error) {
	r.calls++
	return r.name, r.err
}

var _ manager.InstalledNameResolver = (*resolverAdapter)(nil)

func TestInstallCmd_ResolveAliasTracksCanonicalName(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t)
	cPath := filepath.Join(t.TempDir(), "config.toml")
	adapter := &resolverAdapter{mockAdapter: mockAdapter{name: "dnf"}, name: "nodejs-corepack"}

	buf, err := execCmdAt(t, []string{"install", "yarn", "-m", "dnf", "-y"},
		[]manager.Adapter{adapter}, mPath, cPath)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "note: yarn is provided by nodejs-corepack; tracking nodejs-corepack")
	assert.Equal(t, 1, adapter.calls)

	m := loadManifest(t, mPath)
	assert.True(t, m.HasPackage("nodejs-corepack", "dnf"))
	assert.False(t, m.HasPackage("yarn", "dnf"))
}

func TestInstallCmd_ResolveSameNameIsSilent(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t)
	cPath := filepath.Join(t.TempDir(), "config.toml")
	adapter := &resolverAdapter{mockAdapter: mockAdapter{name: "dnf"}, name: "htop"}

	buf, err := execCmdAt(t, []string{"install", "htop", "-m", "dnf", "-y"},
		[]manager.Adapter{adapter}, mPath, cPath)
	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "note:")
	assert.True(t, loadManifest(t, mPath).HasPackage("htop", "dnf"))
}

func TestInstallCmd_ResolveErrorWarnsAndTracksSpec(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t)
	cPath := filepath.Join(t.TempDir(), "config.toml")
	adapter := &resolverAdapter{mockAdapter: mockAdapter{name: "dnf"}, err: assert.AnError}

	buf, err := execCmdAt(t, []string{"install", "yarn", "-m", "dnf", "-y"},
		[]manager.Adapter{adapter}, mPath, cPath)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "warning: could not resolve installed name for yarn")
	assert.True(t, loadManifest(t, mPath).HasPackage("yarn", "dnf"))
}

func TestInstallCmd_GroupSkipsResolver(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t)
	cPath := filepath.Join(t.TempDir(), "config.toml")
	adapter := &resolverAdapter{mockAdapter: mockAdapter{name: "dnf"}, name: "nodejs-corepack"}

	buf, err := execCmdAt(t, []string{"install", "development-tools", "-m", "dnf", "--group", "-y"},
		[]manager.Adapter{adapter}, mPath, cPath)
	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "warning:")
	assert.Equal(t, 0, adapter.calls)

	m := loadManifest(t, mPath)
	require.True(t, m.HasPackage("development-tools", "dnf"))
	for _, p := range m.Packages {
		if p.Name == "development-tools" {
			assert.True(t, p.Group, "group entries must keep the flag")
		}
	}
}

func TestReinstallCmd_TrackedPackageSkipsResolver(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	mPath := seedManifest(t, manifest.Package{Name: "htop", Manager: "dnf"})
	cPath := filepath.Join(t.TempDir(), "config.toml")
	adapter := &resolverAdapter{mockAdapter: mockAdapter{name: "dnf"}, name: "nodejs-corepack"}

	buf, err := execCmdAt(t, []string{"reinstall", "htop", "-m", "dnf", "-y"},
		[]manager.Adapter{adapter}, mPath, cPath)
	require.NoError(t, err)
	assert.Equal(t, 0, adapter.calls)
	assert.NotContains(t, buf.String(), "note:")

	m := loadManifest(t, mPath)
	assert.True(t, m.HasPackage("htop", "dnf"))
	assert.False(t, m.HasPackage("nodejs-corepack", "dnf"))
}

func TestReinstallCmd_ResolveAliasRecordsCanonicalName(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	mPath := seedManifest(t)
	cPath := filepath.Join(t.TempDir(), "config.toml")
	adapter := &resolverAdapter{mockAdapter: mockAdapter{name: "dnf"}, name: "nodejs-corepack"}

	buf, err := execCmdAt(t, []string{"reinstall", "yarn", "-m", "dnf", "-y"},
		[]manager.Adapter{adapter}, mPath, cPath)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "tracking nodejs-corepack")

	m := loadManifest(t, mPath)
	assert.True(t, m.HasPackage("nodejs-corepack", "dnf"))
	assert.False(t, m.HasPackage("yarn", "dnf"))
}
