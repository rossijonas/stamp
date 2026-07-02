package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManifestAddAndRemoveRepository(t *testing.T) {
	m := &Manifest{Version: 1}

	repo1 := Repository{Name: "flathub", Manager: "flatpak", URL: "https://dl.flathub.org/repo/flathub.flatpakrepo"}
	repo2 := Repository{Name: "hashicorp/tap", Manager: "brew"}

	// Test Add
	added := m.AddRepository(repo1)
	assert.True(t, added)
	assert.Len(t, m.Repositories, 1)

	// Add Duplicate
	added = m.AddRepository(repo1)
	assert.False(t, added)
	assert.Len(t, m.Repositories, 1)

	// Add second repository
	added = m.AddRepository(repo2)
	assert.True(t, added)
	assert.Len(t, m.Repositories, 2)

	// Test HasRepository
	assert.True(t, m.HasRepository("flathub", "flatpak"))
	assert.False(t, m.HasRepository("flathub", "brew"))
	assert.False(t, m.HasRepository("unknown", "dnf"))

	// Test Remove
	removed := m.RemoveRepository("flathub", "flatpak")
	assert.True(t, removed)
	assert.Len(t, m.Repositories, 1)
	assert.Equal(t, "hashicorp/tap", m.Repositories[0].Name)

	// Remove non-existent
	removed = m.RemoveRepository("not-here", "dnf")
	assert.False(t, removed)
	assert.Len(t, m.Repositories, 1)
}
