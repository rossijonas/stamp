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

// recordingAdapter captures native arguments for operation-identity tests.
type recordingAdapter struct {
	mockAdapter
	installed []string
	removed   []string
	reinstal  []string
	updated   []string
}

func (r *recordingAdapter) Install(_ context.Context, pkg string) error {
	r.installed = append(r.installed, pkg)
	return nil
}

func (r *recordingAdapter) Remove(_ context.Context, pkg string) error {
	r.removed = append(r.removed, pkg)
	return nil
}

func (r *recordingAdapter) Reinstall(_ context.Context, pkg string) error {
	r.reinstal = append(r.reinstal, pkg)
	return nil
}

func (r *recordingAdapter) Update(_ context.Context, pkg string) error {
	r.updated = append(r.updated, pkg)
	return nil
}

func (r *recordingAdapter) RemoveMany(_ context.Context, pkgs ...string) error {
	r.removed = append(r.removed, pkgs...)
	return nil
}

func TestInstallCmd_RawSpecReachesNative(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t)
	cPath := filepath.Join(t.TempDir(), "config.toml")
	adapter := &recordingAdapter{mockAdapter: mockAdapter{name: "apt"}}

	_, err := execCmdAt(t, []string{"install", "htop=3.2.1", "-m", "apt", "-y"},
		[]manager.Adapter{adapter}, mPath, cPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"htop=3.2.1"}, adapter.installed)
}

func TestRemoveCmd_TrackedSpecReachesNative(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t, manifest.Package{Name: "htop", Manager: "apt"})
	cPath := filepath.Join(t.TempDir(), "config.toml")
	adapter := &recordingAdapter{mockAdapter: mockAdapter{name: "apt"}}

	_, err := execCmdAt(t, []string{"remove", "htop=3.2.1", "-m", "apt", "-y"},
		[]manager.Adapter{adapter}, mPath, cPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"htop"}, adapter.removed)
}

func TestRemoveManyCmd_TrackedSpecsReachNative(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t,
		manifest.Package{Name: "htop", Manager: "apt"},
		manifest.Package{Name: "vim", Manager: "apt"},
	)
	cPath := filepath.Join(t.TempDir(), "config.toml")
	adapter := &recordingAdapter{mockAdapter: mockAdapter{name: "apt"}}

	_, err := execCmdAt(t, []string{"remove", "htop=3.2.1", "vim=9.0", "-m", "apt", "-y"},
		[]manager.Adapter{adapter}, mPath, cPath)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"htop", "vim"}, adapter.removed)
}

func TestReinstallCmd_TrackedSpecReachesNative(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	mPath := seedManifest(t, manifest.Package{Name: "htop", Manager: "apt"})
	cPath := filepath.Join(t.TempDir(), "config.toml")
	adapter := &recordingAdapter{mockAdapter: mockAdapter{name: "apt"}}

	_, err := execCmdAt(t, []string{"reinstall", "htop=3.2.1", "-m", "apt", "-y"},
		[]manager.Adapter{adapter}, mPath, cPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"htop"}, adapter.reinstal)
}

// recordingResolverAdapter records native args and resolves canonical names.
type recordingResolverAdapter struct {
	recordingAdapter
	name  string
	err   error
	calls int
}

func (r *recordingResolverAdapter) ResolveInstalledName(_ context.Context, _ string) (string, error) {
	r.calls++
	return r.name, r.err
}

func TestRemoveCmd_DnfAliasReachesNativeCanonical(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t, manifest.Package{Name: "nodejs-corepack", Manager: "dnf"})
	cPath := filepath.Join(t.TempDir(), "config.toml")
	adapter := &recordingResolverAdapter{recordingAdapter: recordingAdapter{mockAdapter: mockAdapter{name: "dnf"}}, name: "nodejs-corepack"}

	_, err := execCmdAt(t, []string{"remove", "yarn", "-m", "dnf", "-y"},
		[]manager.Adapter{adapter}, mPath, cPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"nodejs-corepack"}, adapter.removed)
	assert.Equal(t, 1, adapter.calls)
	assert.False(t, loadManifest(t, mPath).HasPackage("nodejs-corepack", "dnf"))
}

func TestUpdateCmd_DnfAliasReachesNativeCanonical(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t, manifest.Package{Name: "nodejs-corepack", Manager: "dnf"})
	cPath := filepath.Join(t.TempDir(), "config.toml")
	adapter := &recordingResolverAdapter{recordingAdapter: recordingAdapter{mockAdapter: mockAdapter{name: "dnf"}}, name: "nodejs-corepack"}

	_, err := execCmdAt(t, []string{"update", "-p", "yarn", "-m", "dnf", "-y"},
		[]manager.Adapter{adapter}, mPath, cPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"nodejs-corepack"}, adapter.updated)
	assert.Equal(t, 1, adapter.calls)
}

func TestUpdateCmd_TrackedSpecReachesNative(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t, manifest.Package{Name: "htop", Manager: "apt"})
	cPath := filepath.Join(t.TempDir(), "config.toml")
	adapter := &recordingAdapter{mockAdapter: mockAdapter{name: "apt"}}

	_, err := execCmdAt(t, []string{"update", "-p", "htop=3.2.1", "-m", "apt", "-y"},
		[]manager.Adapter{adapter}, mPath, cPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"htop"}, adapter.updated)
}
