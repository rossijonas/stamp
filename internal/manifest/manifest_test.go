package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestAddAndRemove(t *testing.T) {
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
	_, err := Load("/path/that/does/not/exist.toml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest not found")
}
