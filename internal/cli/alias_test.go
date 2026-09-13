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
	"github.com/rossijonas/stamp/internal/manifest"
)

// execCmdAt runs a command against an explicit manifest path so tests can
// assert the persisted manifest state.
func execCmdAt(t *testing.T, args []string, adapters []manager.Adapter, mPath, cPath string) (*bytes.Buffer, error) {
	t.Helper()
	buf := new(bytes.Buffer)
	root := NewRootCmd(WithAdapters(adapters), WithConfigPath(cPath), WithManifestPath(mPath))
	r, w, err := os.Pipe()
	require.NoError(t, err)
	root.SetIn(r)
	_ = w.Close()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err = root.Execute()
	return buf, err
}

func TestShowAlias(t *testing.T) {
	buf, err := execCmd(t, []string{"show", "htop", "-m", "dnf"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "htop")
}

func TestViewAlias(t *testing.T) {
	buf, err := execCmd(t, []string{"view", "htop", "-m", "dnf"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "htop")
}

func TestRefreshAlias(t *testing.T) {
	buf, err := execCmd(t, []string{"refresh"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestOutdatedCmd(t *testing.T) {
	buf, err := execCmd(t, []string{"outdated"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestCheckUpdateCmd(t *testing.T) {
	buf, err := execCmd(t, []string{"check-update"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestTapCmd(t *testing.T) {
	buf, err := execCmd(t, []string{"tap", "mytap", "-y"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "added tap mytap via brew")
}

func TestTapCmd_BrewNotAvailable(t *testing.T) {
	_, err := execCmd(t, []string{"tap", "mytap", "-y"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "brew is not available")
}

func TestTapCmd_RefusesWithoutYesNonInteractive(t *testing.T) {
	_, err := execCmd(t, []string{"tap", "mytap"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.ErrorIs(t, err, errNonInteractive)
}

func TestUntapCmd(t *testing.T) {
	buf, err := execCmd(t, []string{"untap", "mytap", "-y"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "removed tap mytap via brew")
}

func TestUntapCmd_RefusesWithoutYesNonInteractive(t *testing.T) {
	_, err := execCmd(t, []string{"untap", "mytap"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.ErrorIs(t, err, errNonInteractive)
}

func TestTapCmd_RecordsInManifest(t *testing.T) {
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	buf, err := execCmdAt(t, []string{"tap", "mytap", "-y"}, []manager.Adapter{&mockAdapter{name: "brew"}}, mPath, cPath)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "added tap mytap via brew")

	m, err := manifest.Load(mPath)
	require.NoError(t, err)
	assert.True(t, m.HasRepository("mytap", "brew"), "tap must be recorded in the manifest")
}

func TestUntapCmd_RemovesFromManifest(t *testing.T) {
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	seeded := &manifest.Manifest{
		Version:      1,
		System:       "linux",
		Repositories: []manifest.Repository{{Name: "mytap", Manager: "brew", Origin: manifest.OriginStamped}},
		Packages:     []manifest.Package{},
	}
	require.NoError(t, seeded.Save(mPath))

	buf, err := execCmdAt(t, []string{"untap", "mytap", "-y"}, []manager.Adapter{&mockAdapter{name: "brew"}}, mPath, cPath)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "removed tap mytap via brew")

	m, err := manifest.Load(mPath)
	require.NoError(t, err)
	assert.False(t, m.HasRepository("mytap", "brew"), "untap must remove the tap from the manifest")
}

func TestTapCmd_SingleErrorContext(t *testing.T) {
	// AddRepo already contextualizes ("failed to tap X: ..."); the command must
	// not wrap it a second time.
	_, err := execCmd(t, []string{"tap", "mytap", "-y"},
		[]manager.Adapter{&mockAdapter{name: "brew", err: errors.New("failed to tap mytap: boom")}})
	require.Error(t, err)
	assert.Equal(t, "failed to tap mytap: boom", err.Error())
}

func TestTapsCmd(t *testing.T) {
	buf, err := execCmd(t, []string{"taps"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "no taps added")
}

func TestTapsCmd_BrewNotAvailable(t *testing.T) {
	_, err := execCmd(t, []string{"taps"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "brew is not available")
}
