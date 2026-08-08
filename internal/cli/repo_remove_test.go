package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

// repoRemoveRecorder captures RemoveRepo invocations and records the manager
// used, so tests can assert manifest-driven resolution.
type repoRemoveRecorder struct {
	mockAdapter
	name string
}

func newRepoRemoveRecorder(managerName string) *repoRemoveRecorder {
	return &repoRemoveRecorder{mockAdapter: mockAdapter{name: managerName}}
}

func (r *repoRemoveRecorder) RemoveRepo(_ context.Context, name string) error {
	r.name = name
	return nil
}

func TestRepoRemoveCmd_ManagerFromManifest(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	manifestContent := `version = 1
system = "linux"

[[repositories]]
name = "enpass"
manager = "dnf"
url = "https://yum.enpass.io/enpass-yum.repo"
`
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0o600))

	rec := newRepoRemoveRecorder("dnf")
	buf := new(bytes.Buffer)
	root := NewRootCmd(WithAdapters([]manager.Adapter{rec}), WithConfigPath(cPath), WithManifestPath(mPath))
	r, w, err := os.Pipe()
	require.NoError(t, err)
	root.SetIn(r)
	_ = w.Close()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"repo", "remove", "enpass", "-y"})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "removed repo enpass via dnf")

	// manifest entry removed
	//nolint:gosec // mPath comes from t.TempDir in the test
	data, err := os.ReadFile(mPath)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "enpass")
}

func TestRepoRemoveCmd_NoManagerNotTracked(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	rec := newRepoRemoveRecorder("dnf")
	root := NewRootCmd(WithAdapters([]manager.Adapter{rec}), WithConfigPath(cPath), WithManifestPath(mPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"repo", "remove", "notracked"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not tracked")
	assert.Equal(t, ExitUsage, exitCodeFor(err))
}

func TestRepoRemoveCmd_ExplicitManager(t *testing.T) {
	t.Parallel()

	rec := newRepoRemoveRecorder("dnf")
	buf, err := execCmd(t, []string{"repo", "remove", "petersen/cava", "-m", "dnf", "-y"},
		[]manager.Adapter{rec})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "removed repo petersen/cava via dnf")
}

func TestRepoRemoveCmd_ManagerFlagNotRequired(t *testing.T) {
	t.Parallel()

	repo := lookupCmd(NewRootCmd().Commands(), "repo")
	rm := lookupCmd(repo.Commands(), "remove")
	require.NotNil(t, rm)
	mFlag := rm.Flag("manager")
	require.NotNil(t, mFlag)
	// cobra reports a flag as required through ValidateRequiredFlags
	require.NoError(t, rm.ValidateRequiredFlags(), "manager flag must not be required when the repo is tracked in the manifest")
}
