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

	"github.com/rossijonas/stamp/internal/backup"
)

// Origin values for Package and Repository entries.
const (
	// OriginStamped marks an entry recorded by a direct user action.
	OriginStamped = "stamped"
	// OriginReconciled marks an entry auto-tracked by stamp reconcile.
	OriginReconciled = "reconciled"
)

// Package represents a single installed application or tool in the manifest.
type Package struct {
	Name     string `toml:"name"`
	Manager  string `toml:"manager"`
	Category string `toml:"category,omitempty"`
	Notes    string `toml:"notes,omitempty"`
	Cask     bool   `toml:"cask,omitempty"`
	Group    bool   `toml:"group,omitempty"`
	// Origin records how the entry entered the manifest. Read it via
	// OriginEffective, which defaults an absent value to stamped so pre-origin
	// manifests load without migration.
	Origin string `toml:"origin,omitempty"`
}

// OriginEffective returns the effective origin, defaulting to stamped when
// the field is absent (pre-origin manifests). Consumers must use this method
// rather than reading Origin directly.
func (p Package) OriginEffective() string {
	if p.Origin == "" {
		return OriginStamped
	}
	return p.Origin
}

// Repository represents a tracked third-party repository or tap.
type Repository struct {
	Name    string `toml:"name"`
	Manager string `toml:"manager"`
	URL     string `toml:"url,omitempty"`
	// Origin records how the entry entered the manifest. Read it via
	// OriginEffective, which defaults an absent value to stamped so pre-origin
	// manifests load without migration.
	Origin string `toml:"origin,omitempty"`
}

// OriginEffective returns the effective origin, defaulting to stamped when
// the field is absent (pre-origin manifests). Consumers must use this method
// rather than reading Origin directly.
func (r Repository) OriginEffective() string {
	if r.Origin == "" {
		return OriginStamped
	}
	return r.Origin
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
		_ = tmpFile.Close()
		if !success {
			_ = os.Remove(tmpName)
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

// SetNote updates the note of an existing package. Returns false when no
// package matches name+manager.
func (m *Manifest) SetNote(name, manager, note string) bool {
	for i, pkg := range m.Packages {
		if pkg.Name == name && pkg.Manager == manager {
			m.Packages[i].Notes = note
			return true
		}
	}
	return false
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

// Backup creates a timestamped backup of the manifest file by renaming it.
// Format: <path>.<YYYYMMDD>THHMMSSZ.bak
func Backup(path string) (string, error) {
	ts := time.Now().UTC().Format("20060102T150405Z")
	backupPath := uniqueBackupPath(path, ts)
	if err := os.Rename(path, backupPath); err != nil {
		return "", fmt.Errorf("failed to backup manifest to %s: %w", backupPath, err)
	}
	return backupPath, nil
}

// CopyBackup creates a timestamped copy of the manifest file, leaving the
// original in place. Format: <path>.<YYYYMMDDTHHMMSSZ>.bak. Used by reconcile,
// which must keep the live manifest (the no-drift path never saves it).
func CopyBackup(path string) (string, error) {
	ts := time.Now().UTC().Format(backup.BackupTimeLayout)
	backupPath := uniqueBackupPath(path, ts)

	//nolint:gosec // path is resolved internally via config, not user input
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read manifest for backup: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".bak.*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temp backup: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("failed to write backup to %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("failed to close backup %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, backupPath); err != nil {
		return "", fmt.Errorf("failed to rename backup to %s: %w", backupPath, err)
	}
	return backupPath, nil
}

// RotateBackups prunes manifest backup files (<path>.<TS>.bak) per the given
// retention policy. Returns the number of backups removed.
func RotateBackups(path string, p backup.Policy) (int, error) {
	return backup.Rotate(path+".*.bak", p)
}

// uniqueBackupPath returns <path>.<ts>.bak, appending a numeric suffix if the
// target already exists (see state.BackupSnapshots — same-second collisions).
func uniqueBackupPath(path, ts string) string {
	candidate := path + "." + ts + ".bak"
	for i := 2; pathExists(candidate); i++ {
		candidate = fmt.Sprintf("%s.%s.%d.bak", path, ts, i)
	}
	return candidate
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
