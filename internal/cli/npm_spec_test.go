package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

func TestInstallCmd_NpmVersionedRecordsBaseName(t *testing.T) {
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	_, err := execCmdAt(t, []string{"install", "typescript@5.6.3", "-m", "npm", "-y"},
		[]manager.Adapter{&mockAdapter{name: "npm"}}, mPath, cPath)
	require.NoError(t, err)

	m, err := manifest.Load(mPath)
	require.NoError(t, err)
	assert.True(t, m.HasPackage("typescript", "npm"))
	assert.False(t, m.HasPackage("typescript@5.6.3", "npm"))
}

func TestInstallCmd_NpmScopedRecordsScopedName(t *testing.T) {
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	_, err := execCmdAt(t, []string{"install", "@scope/tool@latest", "-m", "npm", "-y"},
		[]manager.Adapter{&mockAdapter{name: "npm"}}, mPath, cPath)
	require.NoError(t, err)

	m, err := manifest.Load(mPath)
	require.NoError(t, err)
	assert.True(t, m.HasPackage("@scope/tool", "npm"))
}

func TestInstallCmd_NpmScopedWithoutManagerFails(t *testing.T) {
	_, err := execCmd(t, []string{"install", "@scope/tool", "-y"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
	assert.NotContains(t, err.Error(), "Homebrew tap")
}

func TestRemoveCmd_NpmVersionedUntracksBaseName(t *testing.T) {
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	m := &manifest.Manifest{Version: 1}
	m.AddPackage(manifest.Package{Name: "typescript", Manager: "npm"})
	require.NoError(t, m.Save(mPath))

	_, err := execCmdAt(t, []string{"remove", "typescript@5.6.3", "-m", "npm", "-y"},
		[]manager.Adapter{&mockAdapter{name: "npm"}}, mPath, cPath)
	require.NoError(t, err)

	loaded, err := manifest.Load(mPath)
	require.NoError(t, err)
	assert.False(t, loaded.HasPackage("typescript", "npm"))
}

func TestReinstallCmd_NpmScopedRecordsBaseName(t *testing.T) {
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	_, err := execCmdAt(t, []string{"reinstall", "@scope/tool@latest", "-m", "npm", "-y"},
		[]manager.Adapter{&mockAdapter{name: "npm"}}, mPath, cPath)
	require.NoError(t, err)

	m, err := manifest.Load(mPath)
	require.NoError(t, err)
	assert.True(t, m.HasPackage("@scope/tool", "npm"))
	assert.False(t, m.HasPackage("@scope/tool@latest", "npm"))
}

func TestUpdateCmd_NpmScopedPackage(t *testing.T) {
	_, err := execCmd(t, []string{"update", "-p", "@scope/tool@latest", "-m", "npm", "-y"},
		[]manager.Adapter{&mockAdapter{name: "npm"}})
	require.NoError(t, err)
}
