package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestInstallCmd_NonTerminalAborts(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"install", "htop"}, []manager.Adapter{&manager.Mock{ManagerName: "dnf"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "aborted")
	assert.NotContains(t, buf.String(), "installed htop via dnf")
}

func TestInstallCmd_InteractiveAccept(t *testing.T) {
	saveRestoreTerminal(t)
	mockDNF := &manager.Mock{ManagerName: "dnf"}
	tmpDir := t.TempDir()
	root := NewRootCmd(WithAdapters([]manager.Adapter{mockDNF}),
		WithManifestPath(filepath.Join(tmpDir, "manifest.toml")),
		WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(newLineReader("y"))
	root.SetArgs([]string{"install", "htop"})

	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "Install htop via dnf? [y/N]: ")
	assert.Contains(t, buf.String(), "installed htop via dnf")
	assert.Contains(t, mockDNF.InstalledPkgs, "htop")
}

func TestInstallCmd_InteractiveDecline(t *testing.T) {
	saveRestoreTerminal(t)
	mockDNF := &manager.Mock{ManagerName: "dnf"}
	tmpDir := t.TempDir()
	root := NewRootCmd(WithAdapters([]manager.Adapter{mockDNF}),
		WithManifestPath(filepath.Join(tmpDir, "manifest.toml")),
		WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(newLineReader("n"))
	root.SetArgs([]string{"install", "htop"})

	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "aborted")
	assert.NotContains(t, mockDNF.InstalledPkgs, "htop")
}

func TestInstallCmd_YesFlagSkipsPrompt(t *testing.T) {
	t.Parallel()
	mockDNF := &manager.Mock{ManagerName: "dnf"}
	buf, err := execCmd(t, []string{"install", "htop", "-y"}, []manager.Adapter{mockDNF})
	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "[y/N]")
	assert.Contains(t, mockDNF.InstalledPkgs, "htop")
}

func TestRemoveCmd_NonTerminalAborts(t *testing.T) {
	t.Parallel()
	mockDNF := &manager.Mock{ManagerName: "dnf", InstalledPkgs: []string{"htop"}}
	buf, err := execCmd(t, []string{"remove", "htop"}, []manager.Adapter{mockDNF})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "aborted")
	assert.Contains(t, mockDNF.InstalledPkgs, "htop")
}

func TestRemoveCmd_InteractiveAccept(t *testing.T) {
	saveRestoreTerminal(t)
	mockDNF := &manager.Mock{ManagerName: "dnf", InstalledPkgs: []string{"htop"}}
	tmpDir := t.TempDir()
	root := NewRootCmd(WithAdapters([]manager.Adapter{mockDNF}),
		WithManifestPath(filepath.Join(tmpDir, "manifest.toml")),
		WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(newLineReader("y"))
	root.SetArgs([]string{"remove", "htop"})

	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "Remove htop via dnf? [y/N]: ")
	assert.Contains(t, buf.String(), "removed htop via dnf")
	assert.NotContains(t, mockDNF.InstalledPkgs, "htop")
}

func TestReinstallCmd_NonTerminalAborts(t *testing.T) {
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
	root.SetArgs([]string{"reinstall", "htop"})

	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "aborted")
	assert.NotContains(t, buf.String(), "reinstalled htop via brew")
}

func TestReinstallCmd_InteractiveAccept(t *testing.T) {
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
	root.SetArgs([]string{"reinstall", "htop"})

	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "Reinstall: htop@1.0.0")
	assert.Contains(t, buf.String(), "Reinstall htop via brew? [y/N]: ")
	assert.Contains(t, buf.String(), "reinstalled htop via brew")
	assert.Contains(t, mockBrew.InstalledPkgs, "htop")
}

func TestInstallCmd_NoopDoesNotPrompt(t *testing.T) {
	t.Parallel()
	a := &manager.Mock{ManagerName: "dnf", PreviewNoop: true}
	buf, err := execCmd(t, []string{"install", "htop"}, []manager.Adapter{a})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "nothing to do: htop via dnf")
	assert.NotContains(t, buf.String(), "[y/N]: ")
	assert.NotContains(t, buf.String(), "installed htop via dnf")
	assert.NotContains(t, buf.String(), "aborted")
}

func TestRemoveCmd_NoopDoesNotPrompt(t *testing.T) {
	t.Parallel()
	a := &manager.Mock{ManagerName: "dnf", PreviewNoop: true}
	buf, err := execCmd(t, []string{"remove", "htop"}, []manager.Adapter{a})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "nothing to do: htop via dnf")
	assert.NotContains(t, buf.String(), "[y/N]: ")
	assert.NotContains(t, buf.String(), "removed htop via dnf")
	assert.NotContains(t, buf.String(), "aborted")
}

// blockingPreviewAdapter blocks in its transaction preview until the context is
// canceled — the confirmation gate equivalent of a slow sudo/package-manager
// preview that the user interrupts with SIGINT.
type blockingPreviewAdapter struct {
	mockAdapter
	previewStarted chan struct{}
}

func (a *blockingPreviewAdapter) PreviewInstall(ctx context.Context, _ string) (manager.Preview, error) {
	close(a.previewStarted)
	<-ctx.Done()
	return manager.Preview{}, ctx.Err()
}

func (a *blockingPreviewAdapter) PreviewRemove(ctx context.Context, _ string) (manager.Preview, error) {
	close(a.previewStarted)
	<-ctx.Done()
	return manager.Preview{}, ctx.Err()
}

func (a *blockingPreviewAdapter) PreviewReinstall(ctx context.Context, _ string) (manager.Preview, error) {
	close(a.previewStarted)
	<-ctx.Done()
	return manager.Preview{}, ctx.Err()
}

// TestInstallCmd_SigintDuringPreviewAborts proves the reported side effect is
// fixed: SIGINT while the preview is running (e.g. at the sudo password prompt)
// cancels the context and the gate must abort — not proceed to the
// confirmation prompt.
func TestInstallCmd_SigintDuringPreviewAborts(t *testing.T) {
	sigCh := make(chan os.Signal, 1)
	old := notifySigint
	notifySigint = func() chan os.Signal { return sigCh }
	t.Cleanup(func() { notifySigint = old })

	started := make(chan struct{})
	adapter := &blockingPreviewAdapter{mockAdapter: mockAdapter{name: "dnf"}, previewStarted: started}
	tmpDir := t.TempDir()
	root := NewRootCmd(
		WithAdapters([]manager.Adapter{adapter}),
		WithConfigPath(filepath.Join(tmpDir, "config.toml")),
		WithManifestPath(filepath.Join(tmpDir, "manifest.toml")),
	)
	buf := new(bytes.Buffer)
	r, w, err := os.Pipe()
	require.NoError(t, err)
	root.SetIn(r)
	_ = w.Close()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"install", "htop"})

	errCh := make(chan error, 1)
	go func() { errCh <- root.Execute() }()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("preview never started")
	}

	sigCh <- os.Interrupt

	select {
	case err := <-errCh:
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "aborted")
		assert.NotContains(t, buf.String(), "[y/N]: ")
	case <-time.After(5 * time.Second):
		t.Fatal("command did not abort on SIGINT")
	}
}
