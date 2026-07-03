// Package manifest handles the reading, writing, and manipulation
// of the stamp intention manifest file (manifest.toml).
package manifest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// Package represents a single installed application or tool in the manifest.
type Package struct {
	Name     string `toml:"name"`
	Manager  string `toml:"manager"`
	Category string `toml:"category,omitempty"`
	Notes    string `toml:"notes,omitempty"`
}

// Repository represents a tracked third-party repository or tap.
type Repository struct {
	Name    string `toml:"name"`
	Manager string `toml:"manager"`
	URL     string `toml:"url,omitempty"`
}

// Manifest represents the structure of the user's intended state.
type Manifest struct {
	Version      int          `toml:"version"`
	System       string       `toml:"system,omitempty"`
	UpdatedAt    time.Time    `toml:"updated_at"`
	Repositories []Repository `toml:"repositories,omitempty"`
	Packages     []Package    `toml:"packages"`
}

// Load reads a manifest from the given path.
func Load(path string) (*Manifest, error) {
	//nolint:gosec // path is resolved securely via internal config, not user input
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("manifest not found at %s: %w", path, err)
		}
		return nil, fmt.Errorf("failed to read manifest at %s: %w", path, err)
	}

	var m Manifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &m, nil
}

// Save writes the manifest to the given path, creating directories if necessary.
func (m *Manifest) Save(path string) error {
	m.UpdatedAt = time.Now().UTC()

	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create manifest directory %s: %w", dir, err)
	}

	data, err := toml.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to encode manifest: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	var success bool
	defer func() {
		tmpFile.Close()
		if !success {
			os.Remove(tmpName)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write manifest to %s: %w", tmpName, err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp manifest %s: %w", tmpName, err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to rename temp manifest %s to %s: %w", tmpName, path, err)
	}

	success = true
	return nil
}

// AddRepository appends a new repository to the manifest if it doesn't already exist.
func (m *Manifest) AddRepository(repo Repository) bool {
	if m.HasRepository(repo.Name, repo.Manager) {
		return false
	}
	m.Repositories = append(m.Repositories, repo)
	return true
}

// RemoveRepository removes a repository from the manifest.
func (m *Manifest) RemoveRepository(name, manager string) bool {
	for i, repo := range m.Repositories {
		if repo.Name == name && repo.Manager == manager {
			// Remove element efficiently
			m.Repositories = slices.Delete(m.Repositories, i, i+1)
			return true
		}
	}
	return false
}

// HasRepository checks if a repository is already tracked.
func (m *Manifest) HasRepository(name, manager string) bool {
	for _, repo := range m.Repositories {
		if repo.Name == name && repo.Manager == manager {
			return true
		}
	}
	return false
}

// AddPackage appends a new package to the manifest if it doesn't already exist.
func (m *Manifest) AddPackage(pkg Package) bool {
	if m.HasPackage(pkg.Name, pkg.Manager) {
		return false
	}
	m.Packages = append(m.Packages, pkg)
	return true
}

// RemovePackage removes a package from the manifest.
func (m *Manifest) RemovePackage(name, manager string) bool {
	for i, pkg := range m.Packages {
		if pkg.Name == name && pkg.Manager == manager {
			// Remove element efficiently
			m.Packages = slices.Delete(m.Packages, i, i+1)
			return true
		}
	}
	return false
}

// HasPackage checks if a package is already tracked.
func (m *Manifest) HasPackage(name, manager string) bool {
	for _, pkg := range m.Packages {
		if pkg.Name == name && pkg.Manager == manager {
			return true
		}
	}
	return false
}
