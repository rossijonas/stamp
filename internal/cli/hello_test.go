package cli

import (
	"bytes"
	"io"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

// lineReader returns one line per Read call so bufio.NewReader doesn't
// consume all input at once across multiple promptYesNo calls.
type lineReader struct {
	lines []string
	pos   int
	mu    sync.Mutex
}

func (r *lineReader) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pos >= len(r.lines) {
		return 0, io.EOF
	}
	data := r.lines[r.pos] + "\n"
	r.pos++
	n := copy(p, data)
	return n, nil
}

func newLineReader(lines ...string) *lineReader {
	return &lineReader{lines: lines}
}

func TestSetupCmd_AutoAccept(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}},
	}
	buf, err := execCmd(t, []string{"setup", "--yes"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Stamp Setup Wizard")
	// Auto-accept should not contain prompts
	assert.NotContains(t, output, "[Y/n]:")
}

func TestHelloCmd_StillWorks(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"hello"}, []manager.Adapter{})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Stamp Setup Wizard")
}

func TestRootCmd_DefaultSuggestsSetup(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{}, []manager.Adapter{})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Don't know where to start? Try:")
	assert.Contains(t, output, "stamp setup")
}

func TestSetupCmd_Interactive_AllYes(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}},
	}

	root := NewRootCmd(WithAdapters(adapters), WithConfigPath(""), WithManifestPath(""))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(newLineReader("y", "y", "y"))
	root.SetArgs([]string{"setup"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Setup complete")
}

func TestSetupCmd_Reinit_ShowsWarning(t *testing.T) {
	saveRestoreTerminal(t)

	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}},
	}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	createExistingManifest(t, mPath, "")

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(newLineReader("y", "y", "n"))
	root.SetArgs([]string{"setup"})
	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "already initialized")
	assert.Contains(t, output, "[y/N]")
}

func TestSetupCmd_Reinit_AutoAccept(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}},
	}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	t.Setenv("XDG_DATA_HOME", tmpDir)
	createExistingManifest(t, mPath, "")

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"setup", "--yes"})
	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Stamp Setup Wizard")
	assert.NotContains(t, output, "[y/N]:")
	assert.NotContains(t, output, "re-init aborted")
}

func TestSetupCmd_Reinit_Decline(t *testing.T) {
	saveRestoreTerminal(t)

	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}},
	}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	createExistingManifest(t, mPath, "")

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(newLineReader("y", "y", "n"))
	root.SetArgs([]string{"setup"})
	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "requires initialization to work properly")
}

func TestSetupCmd_Reinit_Accept(t *testing.T) {
	saveRestoreTerminal(t)

	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}},
	}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	t.Setenv("XDG_DATA_HOME", tmpDir)
	createExistingManifest(t, mPath, "")

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(newLineReader("y", "y", "y"))
	root.SetArgs([]string{"setup"})
	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Setup complete")
	assert.Contains(t, output, "existing manifest backed up")
}
