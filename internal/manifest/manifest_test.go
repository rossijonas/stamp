package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestAddAndRemove(t *testing.T) {
	t.Parallel()
	m := &Manifest{Version: 1}

	pkg1 := Package{Name: "htop", Manager: "dnf"}
	pkg2 := Package{Name: "ripgrep", Manager: "brew"}

	// Test Add
	added := m.AddPackage(pkg1)
	assert.True(t, added)
	assert.Len(t, m.Packages, 1)

	// Add Duplicate
	added = m.AddPackage(pkg1)
	assert.False(t, added)
	assert.Len(t, m.Packages, 1)

	// Add second package
	added = m.AddPackage(pkg2)
	assert.True(t, added)
	assert.Len(t, m.Packages, 2)

	// Test HasPackage
	assert.True(t, m.HasPackage("htop", "dnf"))
	assert.False(t, m.HasPackage("htop", "brew"))
	assert.False(t, m.HasPackage("unknown", "dnf"))

	// Test Remove
	removed := m.RemovePackage("htop", "dnf")
	assert.True(t, removed)
	assert.Len(t, m.Packages, 1)
	assert.Equal(t, "ripgrep", m.Packages[0].Name)

	// Remove non-existent
	removed = m.RemovePackage("not-here", "dnf")
	assert.False(t, removed)
	assert.Len(t, m.Packages, 1)
}

func TestManifestLoadAndSave(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.toml")

	m := &Manifest{
		Version: 1,
		System:  "fedora",
		Packages: []Package{
			{Name: "htop", Manager: "dnf", Category: "utils"},
		},
	}

	// Test Save
	err := m.Save(manifestPath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(manifestPath)
	require.NoError(t, err)

	// Test Load
	loaded, err := Load(manifestPath)
	require.NoError(t, err)
	assert.Equal(t, 1, loaded.Version)
	assert.Equal(t, "fedora", loaded.System)
	assert.Len(t, loaded.Packages, 1)
	assert.Equal(t, "htop", loaded.Packages[0].Name)
	assert.Equal(t, "dnf", loaded.Packages[0].Manager)
	assert.Equal(t, "utils", loaded.Packages[0].Category)

	// Check that UpdatedAt was set
	assert.False(t, loaded.UpdatedAt.IsZero())
}

func TestManifestLoadNotFound(t *testing.T) {
	t.Parallel()
	_, err := Load("/path/that/does/not/exist.toml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest not found")
}

func TestManifestLoadReadError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	_, err := Load(tmpDir)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "manifest not found")
	assert.Contains(t, err.Error(), "failed to read manifest")
}

func TestManifestLoadInvalidTOML(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "invalid.toml")

	// Write malformed TOML
	err := os.WriteFile(manifestPath, []byte("invalid = [toml\n"), 0600)
	require.NoError(t, err)

	_, err = Load(manifestPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}

func TestManifestSavePermissionError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0400) // Read-only
	require.NoError(t, err)

	m := &Manifest{Version: 1}
	manifestPath := filepath.Join(readOnlyDir, "manifest.toml")

	err = m.Save(manifestPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create temp file")
}

func TestManifestSaveRenameError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.toml")

	// Create a directory at the target path to make rename fail
	err := os.Mkdir(manifestPath, 0750)
	require.NoError(t, err)

	m := &Manifest{Version: 1}

	err = m.Save(manifestPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to rename temp manifest")

	// Ensure the temp file was removed
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Len(t, files, 1) // Only the "manifest.toml" directory should exist
	assert.Equal(t, "manifest.toml", files[0].Name())
}

func TestManifestBackup_CreatesFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	originalContent := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "dnf"
`
	require.NoError(t, os.WriteFile(mPath, []byte(originalContent), 0600))

	backupPath, err := Backup(mPath)
	require.NoError(t, err)
	assert.Contains(t, backupPath, ".bak")

	_, err = os.Stat(mPath)
	assert.True(t, os.IsNotExist(err), "original manifest should be renamed")

	_, err = os.Stat(backupPath)
	require.NoError(t, err)

	//nolint:gosec // path is a controlled temp file
	data, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(data))
}

func TestManifestBackup_NoOriginal(t *testing.T) {
	t.Parallel()
	_, err := Backup("/nonexistent/manifest.toml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to backup manifest")
}

func TestManifestSaveMkdirError(t *testing.T) {
	t.Parallel()
	tmpFile, err := os.CreateTemp("", "manifest-mkdir-test-*")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_ = tmpFile.Close()

	m := &Manifest{Version: 1}
	err = m.Save(filepath.Join(tmpFile.Name(), "subdir", "manifest.toml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create manifest directory")
}
