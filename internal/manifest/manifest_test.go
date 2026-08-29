package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/backup"
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

func TestManifestSetNote(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		packages []Package
		target   string
		manager  string
		note     string
		want     bool
		wantNote string
	}{
		{
			name:     "overwrites existing note",
			packages: []Package{{Name: "htop", Manager: "dnf", Notes: "old reason"}},
			target:   "htop", manager: "dnf", note: "new reason",
			want: true, wantNote: "new reason",
		},
		{
			name:     "sets note when empty",
			packages: []Package{{Name: "htop", Manager: "dnf"}},
			target:   "htop", manager: "dnf", note: "reason",
			want: true, wantNote: "reason",
		},
		{
			name:     "not found returns false",
			packages: []Package{{Name: "htop", Manager: "dnf"}},
			target:   "missing", manager: "dnf", note: "reason",
			want: false, wantNote: "",
		},
		{
			name:     "manager mismatch not found",
			packages: []Package{{Name: "htop", Manager: "dnf"}},
			target:   "htop", manager: "brew", note: "reason",
			want: false, wantNote: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &Manifest{Packages: tt.packages}
			got := m.SetNote(tt.target, tt.manager, tt.note)
			assert.Equal(t, tt.want, got)
			for _, p := range m.Packages {
				if p.Name == tt.target && p.Manager == tt.manager {
					assert.Equal(t, tt.wantNote, p.Notes)
				}
			}
		})
	}
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

// TestManifestBackup_SameSecondCollision guards against a same-second re-init:
// the second backup must get a suffixed path rather than colliding.
func TestManifestBackup_SameSecondCollision(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")

	first := ""
	for i := 0; i < 3; i++ {
		require.NoError(t, os.WriteFile(mPath, []byte("version = 1\n"), 0600))
		p, err := Backup(mPath)
		require.NoError(t, err)
		if i > 0 {
			assert.NotEqual(t, first, p, "backups must not collide")
		}
		first = p
	}
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

func TestManifestCopyBackup_KeepsOriginal(t *testing.T) {
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

	backupPath, err := CopyBackup(mPath)
	require.NoError(t, err)
	assert.Contains(t, backupPath, ".bak")

	_, err = os.Stat(mPath)
	require.NoError(t, err, "original manifest must remain")

	//nolint:gosec // path is a controlled temp file
	data, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(data))
}

func TestManifestCopyBackup_NoOriginal(t *testing.T) {
	t.Parallel()
	_, err := CopyBackup("/nonexistent/manifest.toml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read manifest for backup")
}

func TestManifestCopyBackup_TempCreateError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	roDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.Mkdir(roDir, 0700))

	mPath := filepath.Join(roDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte("version = 1\n"), 0600))
	require.NoError(t, os.Chmod(roDir, 0500))       //nolint:gosec // readable but not writable
	t.Cleanup(func() { _ = os.Chmod(roDir, 0700) }) //nolint:gosec // restore perms for cleanup

	_, err := CopyBackup(mPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create temp backup")
}

func TestManifestCopyBackup_SameSecondCollision(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")

	first := ""
	for i := 0; i < 3; i++ {
		require.NoError(t, os.WriteFile(mPath, []byte("version = 1\n"), 0600))
		p, err := CopyBackup(mPath)
		require.NoError(t, err)
		if i > 0 {
			assert.NotEqual(t, first, p, "copy backups must not collide")
		}
		first = p
	}
}

func TestManifestRotateBackups(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte("version = 1\n"), 0600))

	for _, age := range []int{1, 2, 3, 4} {
		ts := time.Now().UTC().Add(-time.Duration(age) * 24 * time.Hour).Format("20060102T150405Z")
		require.NoError(t, os.WriteFile(fmt.Sprintf("%s.%s.bak", mPath, ts), []byte("old"), 0600))
	}

	n, err := RotateBackups(mPath, backup.Policy{MaxKeep: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}

func TestManifestRotateBackups_NoBackups(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	n, err := RotateBackups(mPath, backup.Policy{MaxKeep: 2})
	require.NoError(t, err)
	assert.Zero(t, n)
}

func TestManifestRotateBackups_InvalidGlob(t *testing.T) {
	t.Parallel()
	_, err := RotateBackups("[", backup.Policy{MaxKeep: 1})
	require.Error(t, err)
}

func TestPackageOriginEffective(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		origin   string
		expected string
	}{
		{name: "empty defaults to stamped", origin: "", expected: OriginStamped},
		{name: "explicit stamped", origin: OriginStamped, expected: OriginStamped},
		{name: "explicit reconciled", origin: OriginReconciled, expected: OriginReconciled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Package{Name: "htop", Manager: "dnf", Origin: tt.origin}
			assert.Equal(t, tt.expected, p.OriginEffective())
		})
	}
}

func TestRepositoryOriginEffective(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		origin   string
		expected string
	}{
		{name: "empty defaults to stamped", origin: "", expected: OriginStamped},
		{name: "explicit stamped", origin: OriginStamped, expected: OriginStamped},
		{name: "explicit reconciled", origin: OriginReconciled, expected: OriginReconciled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Repository{Name: "flathub", Manager: "flatpak", Origin: tt.origin}
			assert.Equal(t, tt.expected, r.OriginEffective())
		})
	}
}

func TestManifestOriginRoundTrip(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.toml")

	m := &Manifest{
		Version: 1,
		System:  "linux",
		Packages: []Package{
			{Name: "htop", Manager: "dnf", Origin: OriginStamped},
			{Name: "vim", Manager: "dnf", Origin: OriginReconciled},
			{Name: "git", Manager: "apt"},
		},
		Repositories: []Repository{
			{Name: "flathub", Manager: "flatpak", URL: "https://dl.flathub.org/repo/flathub.flatpakrepo", Origin: OriginStamped},
			{Name: "copr", Manager: "dnf", Origin: OriginReconciled},
		},
	}
	require.NoError(t, m.Save(manifestPath))

	loaded, err := Load(manifestPath)
	require.NoError(t, err)
	require.Len(t, loaded.Packages, 3)
	assert.Equal(t, OriginStamped, loaded.Packages[0].Origin)
	assert.Equal(t, OriginReconciled, loaded.Packages[1].Origin)
	assert.Equal(t, OriginStamped, loaded.Packages[2].OriginEffective(), "absent origin effective is stamped")
	require.Len(t, loaded.Repositories, 2)
	assert.Equal(t, OriginStamped, loaded.Repositories[0].Origin)
	assert.Equal(t, OriginReconciled, loaded.Repositories[1].OriginEffective())
}

func TestManifestOriginAbsentDefaultsStamped(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.toml")

	content := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "dnf"

[[repositories]]
name = "flathub"
manager = "flatpak"
`
	require.NoError(t, os.WriteFile(manifestPath, []byte(content), 0600))

	loaded, err := Load(manifestPath)
	require.NoError(t, err)
	require.Len(t, loaded.Packages, 1)
	assert.Equal(t, OriginStamped, loaded.Packages[0].OriginEffective())
	require.Len(t, loaded.Repositories, 1)
	assert.Equal(t, OriginStamped, loaded.Repositories[0].OriginEffective())
}
