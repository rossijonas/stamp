package state

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "stamp-state-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	snap := Snapshot{
		Manager:   "brew",
		Packages:  []string{"lazygit", "ripgrep", "jq"},
		UpdatedAt: time.Now().UTC().Truncate(time.Second), // Truncate to avoid millisecond serialisation differences
	}

	err = Save(tempDir, snap)
	require.NoError(t, err)

	loaded, err := Load(tempDir, "brew")
	require.NoError(t, err)

	assert.Equal(t, snap.Manager, loaded.Manager)
	assert.Equal(t, snap.Packages, loaded.Packages)
	assert.True(t, snap.UpdatedAt.Equal(loaded.UpdatedAt.UTC()))
}

func TestLoadNotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "stamp-state-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	_, err = Load(tempDir, "nonexistent")
	require.Error(t, err)
	assert.True(t, os.IsNotExist(errors.Unwrap(err)))
}

func TestDiff(t *testing.T) {
	tests := []struct {
		name        string
		oldPackages []string
		newPackages []string
		expected    *Delta
	}{
		{
			name:        "no changes",
			oldPackages: []string{"a", "b", "c"},
			newPackages: []string{"a", "b", "c"},
			expected: &Delta{
				Manager: "dnf",
				Added:   []string{},
				Removed: []string{},
			},
		},
		{
			name:        "only additions",
			oldPackages: []string{"a", "b"},
			newPackages: []string{"a", "b", "c", "d"},
			expected: &Delta{
				Manager: "dnf",
				Added:   []string{"c", "d"},
				Removed: []string{},
			},
		},
		{
			name:        "only removals",
			oldPackages: []string{"a", "b", "c"},
			newPackages: []string{"a"},
			expected: &Delta{
				Manager: "dnf",
				Added:   []string{},
				Removed: []string{"b", "c"},
			},
		},
		{
			name:        "additions and removals",
			oldPackages: []string{"a", "b", "c"},
			newPackages: []string{"b", "c", "d", "e"},
			expected: &Delta{
				Manager: "dnf",
				Added:   []string{"d", "e"},
				Removed: []string{"a"},
			},
		},
		{
			name:        "empty to full",
			oldPackages: []string{},
			newPackages: []string{"a", "b"},
			expected: &Delta{
				Manager: "dnf",
				Added:   []string{"a", "b"},
				Removed: []string{},
			},
		},
		{
			name:        "full to empty",
			oldPackages: []string{"a", "b"},
			newPackages: []string{},
			expected: &Delta{
				Manager: "dnf",
				Added:   []string{},
				Removed: []string{"a", "b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldS := Snapshot{Manager: "dnf", Packages: tt.oldPackages}
			newS := Snapshot{Manager: "dnf", Packages: tt.newPackages}

			result := Diff(oldS, newS)
			assert.Equal(t, tt.expected.Manager, result.Manager)
			assert.ElementsMatch(t, tt.expected.Added, result.Added)
			assert.ElementsMatch(t, tt.expected.Removed, result.Removed)
			assert.Empty(t, result.AddedRepos)
			assert.Empty(t, result.RemovedRepos)
		})
	}
}

func TestDiff_ReposAdded(t *testing.T) {
	t.Parallel()
	oldS := Snapshot{
		Manager:      "brew",
		Packages:     []string{"git"},
		Repositories: []manager.RepositoryInfo{},
	}
	newS := Snapshot{
		Manager:  "brew",
		Packages: []string{"git"},
		Repositories: []manager.RepositoryInfo{
			{Name: "aovestdipaperino/tap"},
			{Name: "yvgude/lean-ctx"},
		},
	}

	result := Diff(oldS, newS)
	assert.Equal(t, "brew", result.Manager)
	assert.Empty(t, result.Added)
	assert.Empty(t, result.Removed)
	assert.Len(t, result.AddedRepos, 2)
	assert.Empty(t, result.RemovedRepos)
}

func TestDiff_ReposRemoved(t *testing.T) {
	t.Parallel()
	oldS := Snapshot{
		Manager:  "brew",
		Packages: []string{"git"},
		Repositories: []manager.RepositoryInfo{
			{Name: "old-tap"},
		},
	}
	newS := Snapshot{
		Manager:      "brew",
		Packages:     []string{"git"},
		Repositories: []manager.RepositoryInfo{},
	}

	result := Diff(oldS, newS)
	assert.Len(t, result.RemovedRepos, 1)
	assert.Empty(t, result.AddedRepos)
}

func TestDiff_ReposMixed(t *testing.T) {
	t.Parallel()
	oldS := Snapshot{
		Manager: "brew",
		Repositories: []manager.RepositoryInfo{
			{Name: "a"}, {Name: "b"},
		},
	}
	newS := Snapshot{
		Manager: "brew",
		Repositories: []manager.RepositoryInfo{
			{Name: "b"}, {Name: "c"},
		},
	}

	result := Diff(oldS, newS)
	assert.Len(t, result.AddedRepos, 1)
	assert.Equal(t, "c", result.AddedRepos[0].Name)
	assert.Len(t, result.RemovedRepos, 1)
	assert.Equal(t, "a", result.RemovedRepos[0].Name)
}

func TestDiffAll_Empty(t *testing.T) {
	deltas := DiffAll(nil, nil)
	assert.Empty(t, deltas)
}

func TestDiffAll(t *testing.T) {
	oldSnaps := []Snapshot{
		{
			Manager:  "brew",
			Packages: []string{"lazygit", "jq"},
			Repositories: []manager.RepositoryInfo{
				{Name: "homebrew/core"},
			},
		},
		{
			Manager:  "dnf",
			Packages: []string{"htop", "curl"},
		},
	}

	newSnaps := []Snapshot{
		{
			Manager:  "brew",
			Packages: []string{"lazygit", "ripgrep"},
			Repositories: []manager.RepositoryInfo{
				{Name: "homebrew/core"},
				{Name: "aovestdipaperino/tap"},
			},
		},
		{
			Manager:  "dnf",
			Packages: []string{"htop", "curl", "tmux"},
		},
		{
			Manager:  "flatpak",
			Packages: []string{"spotify"},
		},
	}

	deltas := DiffAll(oldSnaps, newSnaps)
	require.Len(t, deltas, 3)

	brewDelta := deltas[0]
	assert.Equal(t, "brew", brewDelta.Manager)
	assert.ElementsMatch(t, []string{"ripgrep"}, brewDelta.Added)
	assert.ElementsMatch(t, []string{"jq"}, brewDelta.Removed)
	assert.Len(t, brewDelta.AddedRepos, 1)
	assert.Equal(t, "aovestdipaperino/tap", brewDelta.AddedRepos[0].Name)
	assert.Empty(t, brewDelta.RemovedRepos)

	dnfDelta := deltas[1]
	assert.Equal(t, "dnf", dnfDelta.Manager)
	assert.ElementsMatch(t, []string{"tmux"}, dnfDelta.Added)
	assert.Empty(t, dnfDelta.Removed)
	assert.Empty(t, dnfDelta.AddedRepos)
	assert.Empty(t, dnfDelta.RemovedRepos)

	flatpakDelta := deltas[2]
	assert.Equal(t, "flatpak", flatpakDelta.Manager)
	assert.ElementsMatch(t, []string{"spotify"}, flatpakDelta.Added)
	assert.Empty(t, flatpakDelta.Removed)
	assert.Empty(t, flatpakDelta.AddedRepos)
	assert.Empty(t, flatpakDelta.RemovedRepos)
}

func TestCurrent_Success(t *testing.T) {
	ctx := context.Background()

	mockBrew := &manager.Mock{
		ManagerName:   "brew",
		InstalledPkgs: []string{"lazygit", "jq"},
	}

	mockDNF := &manager.Mock{
		ManagerName:   "dnf",
		InstalledPkgs: []string{"htop", "curl"},
	}

	adapters := []manager.Adapter{mockBrew, mockDNF}

	snapshots, err := Current(ctx, adapters)
	require.NoError(t, err)
	require.Len(t, snapshots, 2)

	snapMap := make(map[string]Snapshot)
	for _, s := range snapshots {
		snapMap[s.Manager] = s
	}

	brewSnap, ok := snapMap["brew"]
	require.True(t, ok)
	assert.ElementsMatch(t, []string{"lazygit", "jq"}, brewSnap.Packages)

	dnfSnap, ok := snapMap["dnf"]
	require.True(t, ok)
	assert.ElementsMatch(t, []string{"htop", "curl"}, dnfSnap.Packages)
}

func TestCurrent_Error(t *testing.T) {
	ctx := context.Background()

	mockBrew := &manager.Mock{
		ManagerName:   "brew",
		InstalledPkgs: []string{"lazygit"},
	}

	mockDNF := &manager.Mock{
		ManagerName: "dnf",
		ListErr:     errors.New("dnf command failed"),
	}

	adapters := []manager.Adapter{mockBrew, mockDNF}

	_, err := Current(ctx, adapters)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dnf command failed")
}

func TestSnapshotDir(t *testing.T) {
	// Custom XDG_DATA_HOME to isolate tests
	tempDir, err := os.MkdirTemp("", "stamp-xdg-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	oldXdg := os.Getenv("XDG_DATA_HOME")
	err = os.Setenv("XDG_DATA_HOME", tempDir)
	require.NoError(t, err)
	defer func() { _ = os.Setenv("XDG_DATA_HOME", oldXdg) }()

	dir, err := SnapshotDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tempDir, "stamp", "snapshots"), dir)

	// Confirm directory exists
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestXdgStateDir_Fallback(t *testing.T) {
	oldXdg := os.Getenv("XDG_DATA_HOME")
	err := os.Setenv("XDG_DATA_HOME", "")
	require.NoError(t, err)
	defer func() { _ = os.Setenv("XDG_DATA_HOME", oldXdg) }()

	dir := xdgStateDir()
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".local", "share", "stamp"), dir)
}

func TestSnapshotDir_Error(t *testing.T) {
	// Custom XDG_DATA_HOME pointing to a temp file, so os.MkdirAll fails
	tempFile, err := os.CreateTemp("", "stamp-xdg-error-*")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()

	oldXdg := os.Getenv("XDG_DATA_HOME")
	err = os.Setenv("XDG_DATA_HOME", tempFile.Name())
	require.NoError(t, err)
	defer func() { _ = os.Setenv("XDG_DATA_HOME", oldXdg) }()

	_, err = SnapshotDir()
	require.Error(t, err)
}

func TestSnapshotDirPath_NoCreate(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	p := SnapshotDirPath()
	assert.Equal(t, filepath.Join(tmpDir, "stamp", "snapshots"), p)
	_, err := os.Stat(p)
	assert.True(t, os.IsNotExist(err), "SnapshotDirPath should not create the directory")
}

func TestBackupSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	snapDir := filepath.Join(tmpDir, "snapshots")
	require.NoError(t, os.MkdirAll(snapDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(snapDir, "brew.json"), []byte("{}"), 0600))

	backupPath, err := BackupSnapshots(snapDir)
	require.NoError(t, err)
	assert.Contains(t, backupPath, snapDir)
	assert.Contains(t, backupPath, ".bak")

	_, err = os.Stat(snapDir)
	assert.True(t, os.IsNotExist(err), "original snapshots dir should be renamed")

	entries, err := os.ReadDir(backupPath)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "brew.json", entries[0].Name())
}

func TestBackupSnapshots_NoDir(t *testing.T) {
	_, err := BackupSnapshots("/nonexistent/snapshots")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to backup snapshots")
}

func TestSave_WriteFileError(t *testing.T) {
	// Attempt to save to an invalid directory (should fail on os.WriteFile)
	snap := Snapshot{
		Manager:  "brew",
		Packages: []string{"lazygit"},
	}
	err := Save("/nonexistent-directory/snapshots", snap)
	require.Error(t, err)
}

func TestDiff_SingleElementChanges(t *testing.T) {
	oldS := Snapshot{Manager: "dnf", Packages: []string{"a"}}
	newS := Snapshot{Manager: "dnf", Packages: []string{"b"}}
	result := Diff(oldS, newS)
	assert.ElementsMatch(t, []string{"b"}, result.Added)
	assert.ElementsMatch(t, []string{"a"}, result.Removed)
}

func TestSave_InvalidManagerName(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "stamp-state-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	snap := Snapshot{
		Manager:  "../evil",
		Packages: []string{},
	}
	err = Save(tempDir, snap)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid manager name")
}

func TestLoad_InvalidManagerName(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "stamp-state-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	_, err = Load(tempDir, "../evil")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid manager name")
}

func TestLoad_UnmarshalError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "stamp-state-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Write invalid JSON file
	path := filepath.Join(tempDir, "brew.json")
	//nolint:gosec // permissions are restricted to 0600
	err = os.WriteFile(path, []byte("{invalid-json}"), 0600)
	require.NoError(t, err)

	_, err = Load(tempDir, "brew")
	require.Error(t, err)
}
