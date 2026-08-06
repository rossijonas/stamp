package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/backup"
	"github.com/rossijonas/stamp/internal/manifest"
)

// runManifestCmd executes `stamp manifest <args>` against the given manifest
// and config, returning stdout, stderr, and any error separately.
func runManifestCmd(t *testing.T, mPath, cPath string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := NewRootCmd(WithManifestPath(mPath), WithConfigPath(cPath))
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetArgs(append([]string{"manifest"}, args...))
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

func newTestManifest(pkgs []manifest.Package, repos []manifest.Repository) *manifest.Manifest {
	return &manifest.Manifest{
		Version:      1,
		System:       "linux",
		UpdatedAt:    time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
		Packages:     pkgs,
		Repositories: repos,
	}
}

func writeManifestFile(t *testing.T, path string, m *manifest.Manifest) {
	t.Helper()
	data, err := toml.Marshal(m)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0600))
}

// writeManifestBackup writes a timestamped backup of the manifest and returns
// its path.
func writeManifestBackup(t *testing.T, mPath string, ts time.Time, m *manifest.Manifest) string {
	t.Helper()
	name := fmt.Sprintf("%s.%s.bak", mPath, ts.UTC().Format(backup.BackupTimeLayout))
	data, err := toml.Marshal(m)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(name, data, 0600))
	return name
}

func writeManifestAndConfig(t *testing.T, m *manifest.Manifest) (mPath, cPath string) {
	t.Helper()
	tmpDir := t.TempDir()
	mPath = filepath.Join(tmpDir, "manifest.toml")
	cPath = filepath.Join(tmpDir, "config.toml")
	writeManifestFile(t, mPath, m)
	return mPath, cPath
}

var hex12 = regexp.MustCompile(`[0-9a-f]{12}`)

func TestManifestHistoryCmd_NoBackups(t *testing.T) {
	mPath, cPath := writeManifestAndConfig(t, newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf"}},
		nil,
	))
	stdout, _, err := runManifestCmd(t, mPath, cPath, "history")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Available manifest backups:")
	assert.Contains(t, stdout, "* 2026-08-04T12:00:00Z")
	assert.Contains(t, stdout, "(current)")
	assert.Contains(t, stdout, "No backups found. Backups are created on re-init and reconcile.")
}

func TestManifestHistoryCmd_WithBackups(t *testing.T) {
	current := newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf"}, {Name: "lazygit", Manager: "brew"}},
		[]manifest.Repository{{Name: "my-tap", Manager: "brew"}},
	)
	mPath, cPath := writeManifestAndConfig(t, current)

	older := newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf"}, {Name: "jq", Manager: "apt"}, {Name: "vim", Manager: "apt"}},
		nil,
	)
	newer := newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf"}},
		[]manifest.Repository{{Name: "my-tap", Manager: "brew"}},
	)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), older)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 3, 18, 30, 0, 0, time.UTC), newer)

	stdout, _, err := runManifestCmd(t, mPath, cPath, "history")
	require.NoError(t, err)

	curIdx := strings.Index(stdout, "* 2026-08-04T12:00:00Z")
	newerIdx := strings.Index(stdout, "2026-08-03T18:30:00Z")
	olderIdx := strings.Index(stdout, "2026-08-02T09:15:00Z")
	require.Greater(t, curIdx, -1)
	require.Greater(t, newerIdx, -1)
	require.Greater(t, olderIdx, -1)
	assert.Less(t, curIdx, newerIdx, "current first, then newest backup")
	assert.Less(t, newerIdx, olderIdx, "backups sorted newest first")

	assert.Contains(t, stdout, "2 packages, 1 repos  (current)")
	assert.Contains(t, stdout, "1 packages, 1 repos")
	assert.Contains(t, stdout, "3 packages, 0 repos")

	matches := hex12.FindAllString(stdout, -1)
	assert.Len(t, matches, 3, "one hash per row")
}

func TestManifestHistoryCmd_JSON(t *testing.T) {
	current := newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf"}},
		[]manifest.Repository{{Name: "my-tap", Manager: "brew"}},
	)
	mPath, cPath := writeManifestAndConfig(t, current)
	backupM := newTestManifest(
		[]manifest.Package{{Name: "jq", Manager: "apt"}},
		nil,
	)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), backupM)

	stdout, _, err := runManifestCmd(t, mPath, cPath, "history", "-j")
	require.NoError(t, err)
	var entries []historyEntry
	require.NoError(t, json.Unmarshal([]byte(stdout), &entries))
	require.Len(t, entries, 2)
	assert.True(t, entries[0].Current)
	assert.Equal(t, "2026-08-04T12:00:00Z", entries[0].Timestamp)
	assert.Equal(t, 1, entries[0].Packages)
	assert.Equal(t, 1, entries[0].Repos)
	assert.Len(t, entries[0].Hash, hashPrefixLen)
	assert.False(t, entries[1].Current)
	assert.Equal(t, "2026-08-02T09:15:00Z", entries[1].Timestamp)
	assert.Equal(t, 1, entries[1].Packages)
	assert.Zero(t, entries[1].Repos)
}

func TestManifestHistoryCmd_UnchangedMarker(t *testing.T) {
	current := newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf"}},
		nil,
	)
	mPath, cPath := writeManifestAndConfig(t, current)
	// Backup with identical content.
	writeManifestBackup(t, mPath, time.Date(2026, 8, 3, 18, 30, 0, 0, time.UTC), current)

	stdout, _, err := runManifestCmd(t, mPath, cPath, "history")
	require.NoError(t, err)
	assert.Contains(t, stdout, "(unchanged)")
}

func TestManifestHistoryCmd_CorruptedBackupSkipped(t *testing.T) {
	current := newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf"}},
		nil,
	)
	mPath, cPath := writeManifestAndConfig(t, current)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), current)
	bad := fmt.Sprintf("%s.%s.bak", mPath, time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC).Format(backup.BackupTimeLayout))
	require.NoError(t, os.WriteFile(bad, []byte("not [[valid toml"), 0600))

	stdout, stderr, err := runManifestCmd(t, mPath, cPath, "history")
	require.NoError(t, err)
	assert.Contains(t, stderr, "warning: skipping unreadable backup")
	assert.Contains(t, stdout, "2026-08-02T09:15:00Z")
	assert.NotContains(t, stdout, "2026-08-01T09:00:00Z")
}

func TestManifestHistoryCmd_ManifestMissing(t *testing.T) {
	mPath := filepath.Join(t.TempDir(), "manifest.toml")
	cPath := filepath.Join(t.TempDir(), "config.toml")
	_, _, err := runManifestCmd(t, mPath, cPath, "history")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest not found; run stamp init first")
	assert.Equal(t, ExitConfig, exitCodeFor(err))
}

func TestManifestDiffCmd_NoBackups(t *testing.T) {
	mPath, cPath := writeManifestAndConfig(t, newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf"}},
		nil,
	))
	_, _, err := runManifestCmd(t, mPath, cPath, "diff")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no backup to compare against")
	assert.Equal(t, ExitNoInput, exitCodeFor(err))
}

func TestManifestDiffCmd_InvalidTimestamp(t *testing.T) {
	current := newTestManifest([]manifest.Package{{Name: "htop", Manager: "dnf"}}, nil)
	mPath, cPath := writeManifestAndConfig(t, current)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), current)

	_, _, err := runManifestCmd(t, mPath, cPath, "diff", "2026-08-99T00:00:00Z")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no backup found for 2026-08-99T00:00:00Z")
	assert.Equal(t, ExitUsage, exitCodeFor(err))
}

func TestManifestDiffCmd_UnknownTimestamp(t *testing.T) {
	current := newTestManifest([]manifest.Package{{Name: "htop", Manager: "dnf"}}, nil)
	mPath, cPath := writeManifestAndConfig(t, current)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), current)

	// Valid timestamp format but no backup with that timestamp.
	_, _, err := runManifestCmd(t, mPath, cPath, "diff", "2026-08-01T00:00:00Z")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no backup found for 2026-08-01T00:00:00Z")
	assert.Equal(t, ExitNoInput, exitCodeFor(err))
}

func TestManifestDiffCmd_EmptyDiff(t *testing.T) {
	current := newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf"}},
		[]manifest.Repository{{Name: "my-tap", Manager: "brew"}},
	)
	mPath, cPath := writeManifestAndConfig(t, current)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), current)

	stdout, _, err := runManifestCmd(t, mPath, cPath, "diff")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Comparing: current vs 2026-08-02T09:15:00Z")
	assert.Contains(t, stdout, "no differences")
}

func TestManifestDiffCmd_AddedRemoved(t *testing.T) {
	baseline := newTestManifest(
		[]manifest.Package{{Name: "NetworkManager", Manager: "dnf"}},
		nil,
	)
	current := newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf"}, {Name: "lazygit", Manager: "brew"}},
		nil,
	)
	mPath, cPath := writeManifestAndConfig(t, current)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), baseline)

	stdout, _, err := runManifestCmd(t, mPath, cPath, "diff")
	require.NoError(t, err)
	assert.Contains(t, stdout, "+ htop (dnf)")
	assert.Contains(t, stdout, "+ lazygit (brew)")
	assert.Contains(t, stdout, "- NetworkManager (dnf)")
	assert.Contains(t, stdout, "2 added, 1 removed")
}

func TestManifestDiffCmd_Repos(t *testing.T) {
	baseline := newTestManifest(
		nil,
		[]manifest.Repository{{Name: "old-repo", Manager: "dnf", URL: "https://example.com/old"}},
	)
	current := newTestManifest(
		nil,
		[]manifest.Repository{{Name: "my-tap", Manager: "brew"}},
	)
	mPath, cPath := writeManifestAndConfig(t, current)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), baseline)

	stdout, _, err := runManifestCmd(t, mPath, cPath, "diff")
	require.NoError(t, err)
	assert.Contains(t, stdout, "+ my-tap (brew)")
	assert.Contains(t, stdout, "- old-repo (dnf)")
	assert.Contains(t, stdout, "1 added, 1 removed")
}

func TestManifestDiffCmd_SpecificTimestamp(t *testing.T) {
	newer := newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf"}},
		nil,
	)
	older := newTestManifest(
		[]manifest.Package{{Name: "jq", Manager: "apt"}},
		nil,
	)
	current := newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf"}, {Name: "vim", Manager: "apt"}},
		nil,
	)
	mPath, cPath := writeManifestAndConfig(t, current)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), older)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 3, 18, 30, 0, 0, time.UTC), newer)

	// Default picks the most recent backup.
	stdout, _, err := runManifestCmd(t, mPath, cPath, "diff")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Comparing: current vs 2026-08-03T18:30:00Z")
	assert.Contains(t, stdout, "+ vim (apt)")
	assert.NotContains(t, stdout, "+ jq (apt)")

	// Explicit older timestamp picks that backup.
	stdout, _, err = runManifestCmd(t, mPath, cPath, "diff", "2026-08-02T09:15:00Z")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Comparing: current vs 2026-08-02T09:15:00Z")
	assert.Contains(t, stdout, "+ htop (dnf)")
	assert.Contains(t, stdout, "+ vim (apt)")
	assert.Contains(t, stdout, "- jq (apt)")
}

func TestManifestDiffCmd_CompactTimestamp(t *testing.T) {
	current := newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf"}},
		nil,
	)
	mPath, cPath := writeManifestAndConfig(t, current)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), current)

	stdout, _, err := runManifestCmd(t, mPath, cPath, "diff", "20260802T091500Z")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Comparing: current vs 2026-08-02T09:15:00Z")
}

func TestManifestDiffCmd_HashResolution(t *testing.T) {
	target := newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf"}},
		nil,
	)
	other := newTestManifest(
		[]manifest.Package{{Name: "jq", Manager: "apt"}},
		nil,
	)
	current := newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf"}, {Name: "vim", Manager: "apt"}},
		nil,
	)
	mPath, cPath := writeManifestAndConfig(t, current)
	targetPath := writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), target)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 3, 18, 30, 0, 0, time.UTC), other)

	hash, err := contentHash(targetPath)
	require.NoError(t, err)
	stdout, _, err := runManifestCmd(t, mPath, cPath, "diff", hash[:hashPrefixLen])
	require.NoError(t, err)
	assert.Contains(t, stdout, "Comparing: current vs 2026-08-02T09:15:00Z")
	assert.Contains(t, stdout, "+ vim (apt)")
}

func TestManifestDiffCmd_UnknownHash(t *testing.T) {
	current := newTestManifest([]manifest.Package{{Name: "htop", Manager: "dnf"}}, nil)
	mPath, cPath := writeManifestAndConfig(t, current)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), current)

	_, _, err := runManifestCmd(t, mPath, cPath, "diff", "deadbeefdeadbeef")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no backup found for deadbeefdeadbeef")
	assert.Equal(t, ExitNoInput, exitCodeFor(err))
}

func TestManifestDiffCmd_AmbiguousHash(t *testing.T) {
	current := newTestManifest([]manifest.Package{{Name: "htop", Manager: "dnf"}}, nil)
	mPath, cPath := writeManifestAndConfig(t, current)
	// Identical content -> identical hash -> any hash prefix matches both.
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), current)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 3, 18, 30, 0, 0, time.UTC), current)

	hash, err := contentHash(mPath)
	require.NoError(t, err)
	_, _, err = runManifestCmd(t, mPath, cPath, "diff", hash[:hashPrefixLen])
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous hash")
}

func TestManifestDiffCmd_JSON(t *testing.T) {
	baseline := newTestManifest(
		[]manifest.Package{{Name: "NetworkManager", Manager: "dnf", Origin: manifest.OriginReconciled}},
		[]manifest.Repository{{Name: "old-repo", Manager: "dnf"}},
	)
	current := newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf", Origin: manifest.OriginStamped}},
		nil,
	)
	mPath, cPath := writeManifestAndConfig(t, current)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), baseline)

	stdout, _, err := runManifestCmd(t, mPath, cPath, "diff", "-j")
	require.NoError(t, err)
	var res diffResult
	require.NoError(t, json.Unmarshal([]byte(stdout), &res))
	assert.Equal(t, "2026-08-02T09:15:00Z", res.Baseline)
	require.Len(t, res.Added, 1)
	assert.Equal(t, diffItem{Name: "htop", Manager: "dnf", Origin: "stamped", Kind: "package"}, res.Added[0])
	require.Len(t, res.Removed, 2)
	assert.Equal(t, diffItem{Name: "NetworkManager", Manager: "dnf", Origin: "reconciled", Kind: "package"}, res.Removed[0])
	assert.Equal(t, diffItem{Name: "old-repo", Manager: "dnf", Origin: "stamped", Kind: "repo"}, res.Removed[1])
}

func TestManifestDiffCmd_ManagerFilter(t *testing.T) {
	baseline := newTestManifest(
		[]manifest.Package{{Name: "NetworkManager", Manager: "dnf"}},
		nil,
	)
	current := newTestManifest(
		[]manifest.Package{{Name: "htop", Manager: "dnf"}, {Name: "lazygit", Manager: "brew"}},
		nil,
	)
	mPath, cPath := writeManifestAndConfig(t, current)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), baseline)

	stdout, _, err := runManifestCmd(t, mPath, cPath, "diff", "-m", "brew")
	require.NoError(t, err)
	assert.Contains(t, stdout, "+ lazygit (brew)")
	assert.NotContains(t, stdout, "htop")
	assert.NotContains(t, stdout, "NetworkManager")
	assert.Contains(t, stdout, "1 added, 0 removed")
}

func TestManifestDiffCmd_OriginFilter(t *testing.T) {
	baseline := newTestManifest(
		[]manifest.Package{{Name: "NetworkManager", Manager: "dnf", Origin: manifest.OriginReconciled}},
		nil,
	)
	current := newTestManifest(
		[]manifest.Package{
			{Name: "htop", Manager: "dnf", Origin: manifest.OriginStamped},
			{Name: "lazygit", Manager: "brew", Origin: manifest.OriginStamped},
		},
		nil,
	)
	mPath, cPath := writeManifestAndConfig(t, current)
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), baseline)

	stdout, _, err := runManifestCmd(t, mPath, cPath, "diff", "--origin", "stamped")
	require.NoError(t, err)
	assert.Contains(t, stdout, "+ htop (dnf)")
	assert.Contains(t, stdout, "+ lazygit (brew)")
	assert.NotContains(t, stdout, "NetworkManager")
	assert.Contains(t, stdout, "2 added, 0 removed")
}

func TestManifestDiffCmd_InvalidOrigin(t *testing.T) {
	mPath, cPath := writeManifestAndConfig(t, newTestManifest(nil, nil))
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), newTestManifest(nil, nil))

	_, _, err := runManifestCmd(t, mPath, cPath, "diff", "--origin", "bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid origin "bogus"`)
}

func TestManifestDiffCmd_CorruptedBaseline(t *testing.T) {
	current := newTestManifest([]manifest.Package{{Name: "htop", Manager: "dnf"}}, nil)
	mPath, cPath := writeManifestAndConfig(t, current)
	bad := fmt.Sprintf("%s.%s.bak", mPath, time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC).Format(backup.BackupTimeLayout))
	require.NoError(t, os.WriteFile(bad, []byte("not [[valid toml"), 0600))

	_, _, err := runManifestCmd(t, mPath, cPath, "diff")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse backup at")
}

func TestManifestDiffCmd_ManifestMissing(t *testing.T) {
	mPath := filepath.Join(t.TempDir(), "manifest.toml")
	cPath := filepath.Join(t.TempDir(), "config.toml")
	writeManifestBackup(t, mPath, time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC), newTestManifest(nil, nil))

	_, _, err := runManifestCmd(t, mPath, cPath, "diff")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest not found; run stamp init first")
}

func TestDiffManifests(t *testing.T) {
	current := &manifest.Manifest{
		Packages: []manifest.Package{
			{Name: "htop", Manager: "dnf"},
			{Name: "lazygit", Manager: "brew"},
			{Name: "yq", Manager: "dnf"}, // manager changed from brew
		},
		Repositories: []manifest.Repository{{Name: "my-tap", Manager: "brew"}},
	}
	baseline := &manifest.Manifest{
		Packages: []manifest.Package{
			{Name: "htop", Manager: "dnf"},
			{Name: "NetworkManager", Manager: "dnf"},
			{Name: "yq", Manager: "brew"},
		},
		Repositories: []manifest.Repository{{Name: "old-repo", Manager: "dnf"}},
	}

	addedPkgs, removedPkgs, addedRepos, removedRepos := diffManifests(current, baseline)
	assert.ElementsMatch(t, []string{"lazygit", "yq"}, pkgNames(addedPkgs), "lazygit new + yq manager change")
	assert.ElementsMatch(t, []string{"NetworkManager", "yq"}, pkgNames(removedPkgs))
	assert.ElementsMatch(t, []string{"my-tap"}, repoNames(addedRepos))
	assert.ElementsMatch(t, []string{"old-repo"}, repoNames(removedRepos))
}

func pkgNames(pkgs []manifest.Package) []string {
	names := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		names = append(names, p.Name)
	}
	return names
}

func repoNames(repos []manifest.Repository) []string {
	names := make([]string, 0, len(repos))
	for _, r := range repos {
		names = append(names, r.Name)
	}
	return names
}

func TestParseBackupTimestamp(t *testing.T) {
	t.Parallel()
	ts, ok := parseBackupTimestamp("2026-08-02T09:15:00Z")
	require.True(t, ok)
	assert.Equal(t, 2026, ts.Year())

	_, ok = parseBackupTimestamp("20260802T091500Z")
	require.True(t, ok)

	_, ok = parseBackupTimestamp("not-a-timestamp")
	assert.False(t, ok)
}

func TestIsHexHash(t *testing.T) {
	t.Parallel()
	assert.True(t, isHexHash("deadbeef"))
	assert.True(t, isHexHash("abcdef123456"))
	assert.False(t, isHexHash("abc"), "too short")
	assert.False(t, isHexHash("20260802T091500Z"), "compact timestamp has T/Z")
	assert.False(t, isHexHash("2026-08-02T09:15:00Z"), "human timestamp has dashes")
	assert.False(t, isHexHash(""))
}

func TestValidateOriginFlag(t *testing.T) {
	t.Parallel()
	require.NoError(t, validateOriginFlag(""))
	require.NoError(t, validateOriginFlag(manifest.OriginStamped))
	require.NoError(t, validateOriginFlag(manifest.OriginReconciled))
	err := validateOriginFlag("bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stamped")
	assert.Contains(t, err.Error(), "reconciled")
}

func TestManifestCmd_GroupShowsHelp(t *testing.T) {
	mPath, cPath := writeManifestAndConfig(t, newTestManifest(nil, nil))
	stdout, _, err := runManifestCmd(t, mPath, cPath)
	require.NoError(t, err)
	assert.Contains(t, stdout, "history")
	assert.Contains(t, stdout, "diff")
}

func TestManifestOriginCompletion(t *testing.T) {
	t.Parallel()
	got, directive := originCompletion(nil, nil, "")
	assert.ElementsMatch(t, []string{manifest.OriginStamped, manifest.OriginReconciled}, got)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestShortHash(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "abcdef", shortHash("abcdef"), "hash shorter than prefix returned unchanged")
	long := strings.Repeat("a", 64)
	assert.Equal(t, strings.Repeat("a", hashPrefixLen), shortHash(long))
}

func TestContentHash_Error(t *testing.T) {
	t.Parallel()
	_, err := contentHash(filepath.Join(t.TempDir(), "missing.toml"))
	require.Error(t, err)
}

func TestCurrentManifestTimestamp_FromUpdatedAt(t *testing.T) {
	t.Parallel()
	app := &AppContext{
		manifestPath: filepath.Join(t.TempDir(), "manifest.toml"),
		manifest:     &manifest.Manifest{UpdatedAt: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)},
	}
	assert.Equal(t, "2026-08-04T12:00:00Z", currentManifestTimestamp(app))
}

func TestCurrentManifestTimestamp_FallbackToMtime(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "manifest.toml")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0600))
	mtime := time.Now().UTC()
	require.NoError(t, os.Chtimes(path, mtime, mtime))
	app := &AppContext{manifestPath: path, manifest: &manifest.Manifest{}}
	parsed, err := time.Parse(historyTimeLayout, currentManifestTimestamp(app))
	require.NoError(t, err)
	assert.WithinDuration(t, mtime, parsed, time.Minute)
}

func TestCurrentManifestTimestamp_Unknown(t *testing.T) {
	t.Parallel()
	app := &AppContext{
		manifestPath: filepath.Join(t.TempDir(), "missing.toml"),
		manifest:     &manifest.Manifest{},
	}
	assert.Equal(t, "unknown", currentManifestTimestamp(app))
}
