package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestAutoremoveCmd_NonTerminalAborts(t *testing.T) {
	mock := &manager.Mock{ManagerName: "brew", AutoRemoveResult: []string{"libfoo"}}
	_, err := execCmd(t, []string{"autoremove"}, []manager.Adapter{mock})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-interactive mode")
}

func TestCleanCmd_NonTerminalAborts(t *testing.T) {
	mock := &manager.Mock{ManagerName: "brew", CleanResult: []string{"cache-item"}}
	_, err := execCmd(t, []string{"clean"}, []manager.Adapter{mock})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-interactive mode")
}

func TestHoldCmd_NonTerminalAborts(t *testing.T) {
	mock := &manager.Mock{ManagerName: "apt"}
	_, err := execCmd(t, []string{"hold", "nginx", "-m", "apt"}, []manager.Adapter{mock})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-interactive mode")
	assert.Empty(t, mock.HeldPkgs)
}

func TestRepoAddCmd_NonTerminalAborts(t *testing.T) {
	mock := &manager.Mock{ManagerName: "brew"}
	_, err := execCmd(t, []string{"repo", "add", "mytap", "-m", "brew"}, []manager.Adapter{mock})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-interactive mode")
	assert.NotContains(t, mock.TrackedRepos, "mytap")
}

func TestRepoAddCmd_InteractiveAccept(t *testing.T) {
	saveRestoreTerminal(t)
	mock := &manager.Mock{ManagerName: "brew"}
	tmpDir := t.TempDir()
	root := NewRootCmd(WithAdapters([]manager.Adapter{mock}),
		WithManifestPath(filepath.Join(tmpDir, "manifest.toml")),
		WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(newLineReader("y"))
	root.SetArgs([]string{"repo", "add", "mytap", "-m", "brew"})

	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "added repo mytap via brew")
	assert.Contains(t, mock.TrackedRepos, "mytap")
}

func TestRestoreCmd_NonTerminalAborts(t *testing.T) {
	mockBrew := &manager.Mock{ManagerName: "brew"}
	adapters := []manager.Adapter{mockBrew}
	manifestContent := "version = 1\nsystem = \"linux\"\n\n[[packages]]\nname = \"htop\"\nmanager = \"brew\"\n"
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
	assert.Contains(t, err.Error(), "non-interactive mode")
	assert.NotContains(t, mockBrew.InstalledPkgs, "htop")
}

func TestRestoreCmd_InteractiveAccept(t *testing.T) {
	saveRestoreTerminal(t)
	mockBrew := &manager.Mock{ManagerName: "brew"}
	adapters := []manager.Adapter{mockBrew}
	manifestContent := "version = 1\nsystem = \"linux\"\n\n[[packages]]\nname = \"htop\"\nmanager = \"brew\"\n"
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))
	t.Setenv("XDG_DATA_HOME", tmpDir)

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(newLineReader("y"))
	root.SetArgs([]string{"restore"})

	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "Restore tracked repositories and packages? [y/N]: ")
	assert.Contains(t, buf.String(), "Restore completed successfully")
	assert.Contains(t, mockBrew.InstalledPkgs, "htop")
}

func TestRestoreCmd_DryRunNoPrompt(t *testing.T) {
	saveRestoreTerminal(t)
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}
	manifestContent := "version = 1\nsystem = \"linux\"\n\n[[packages]]\nname = \"htop\"\nmanager = \"brew\"\n"
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(newLineReader("n"))
	root.SetArgs([]string{"restore", "--dry-run"})

	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "Dry Run (Preview):")
	assert.NotContains(t, buf.String(), "aborted")
}
