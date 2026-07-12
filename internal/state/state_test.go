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
		})
	}
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
		},
		{
			Manager:  "dnf",
			Packages: []string{"htop", "curl"},
		},
	}

	newSnaps := []Snapshot{
		{
			Manager:  "brew",
			Packages: []string{"lazygit", "ripgrep"}, // jq removed, ripgrep added
		},
		{
			Manager:  "dnf",
			Packages: []string{"htop", "curl", "tmux"}, // htop, curl kept, tmux added
		},
		{
			Manager:  "flatpak",
			Packages: []string{"spotify"}, // new manager detected, all added
		},
	}

	deltas := DiffAll(oldSnaps, newSnaps)
	require.Len(t, deltas, 3)

	expectedMap := map[string]*Delta{
		"brew": {
			Manager: "brew",
			Added:   []string{"ripgrep"},
			Removed: []string{"jq"},
		},
		"dnf": {
			Manager: "dnf",
			Added:   []string{"tmux"},
			Removed: []string{},
		},
		"flatpak": {
			Manager: "flatpak",
			Added:   []string{"spotify"},
			Removed: []string{},
		},
	}

	for _, d := range deltas {
		exp, ok := expectedMap[d.Manager]
		require.True(t, ok, "unexpected manager delta: %s", d.Manager)
		assert.ElementsMatch(t, exp.Added, d.Added)
		assert.ElementsMatch(t, exp.Removed, d.Removed)
	}
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
