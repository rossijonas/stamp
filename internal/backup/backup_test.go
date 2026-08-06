package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// backupName builds a timestamped backup name <base>.<YYYYMMDDTHHMMSSZ>.bak
// whose embedded timestamp is age before now.
func backupName(base string, age time.Duration) string {
	ts := time.Now().UTC().Add(-age).Format(BackupTimeLayout)
	return fmt.Sprintf("%s.%s.bak", base, ts)
}

func writeBackup(t *testing.T, base string, age time.Duration) string {
	t.Helper()
	name := backupName(base, age)
	require.NoError(t, os.WriteFile(name, []byte("data"), 0600))
	return name
}

func TestRotate_NoBackups(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	n, err := Rotate(base+".*.bak", Policy{MaxKeep: 10})
	require.NoError(t, err)
	assert.Zero(t, n)
}

func TestRotate_NoBackupsMatchGlob(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unrelated.txt"), []byte("x"), 0600))
	base := filepath.Join(dir, "manifest.toml")
	n, err := Rotate(base+".*.bak", Policy{MaxKeep: 10})
	require.NoError(t, err)
	assert.Zero(t, n)
}

func TestRotate_AllProtectedByMinAge(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	for _, age := range []time.Duration{24 * time.Hour, 48 * time.Hour} {
		writeBackup(t, base, age)
	}
	n, err := Rotate(base+".*.bak", Policy{MaxKeep: 1, MinAge: 7 * 24 * time.Hour})
	require.NoError(t, err)
	assert.Zero(t, n)
	files, err := filepath.Glob(base + ".*.bak")
	require.NoError(t, err)
	assert.Len(t, files, 2, "min-age floor protects recent backups even over count cap")
}

func TestRotate_CountCapTrimsOldestEligible(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	names := []string{
		writeBackup(t, base, 1*24*time.Hour),
		writeBackup(t, base, 2*24*time.Hour),
		writeBackup(t, base, 3*24*time.Hour),
		writeBackup(t, base, 4*24*time.Hour),
	}
	n, err := Rotate(base+".*.bak", Policy{MaxKeep: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	_, err = os.Stat(names[2])
	assert.True(t, os.IsNotExist(err), "3-day-old eligible backup removed by count cap")
	_, err = os.Stat(names[3])
	assert.True(t, os.IsNotExist(err), "4-day-old eligible backup removed by count cap")
	_, err = os.Stat(names[0])
	require.NoError(t, err, "youngest backup kept")
	_, err = os.Stat(names[1])
	require.NoError(t, err, "second-youngest backup kept")
}

func TestRotate_MaxAgeCeilingFiresRegardlessOfCount(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	old1 := writeBackup(t, base, 11*24*time.Hour)
	old2 := writeBackup(t, base, 12*24*time.Hour)
	writeBackup(t, base, 2*24*time.Hour)
	writeBackup(t, base, 3*24*time.Hour)
	n, err := Rotate(base+".*.bak", Policy{MaxKeep: 5, MaxAge: 10 * 24 * time.Hour})
	require.NoError(t, err)
	assert.Equal(t, 2, n, "max-age ceiling deletes ancient backups even though count cap is not exceeded")
	_, err = os.Stat(old1)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(old2)
	assert.True(t, os.IsNotExist(err))
}

func TestRotate_ZeroAxesUnlimited(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	for _, age := range []time.Duration{10 * 24 * time.Hour, 20 * 24 * time.Hour, 30 * 24 * time.Hour} {
		writeBackup(t, base, age)
	}
	n, err := Rotate(base+".*.bak", Policy{})
	require.NoError(t, err)
	assert.Zero(t, n, "all-zero policy keeps everything")
}

func TestRotate_CeilingAndCapCombined(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	for _, age := range []time.Duration{
		1 * 24 * time.Hour, 2 * 24 * time.Hour, 3 * 24 * time.Hour,
		12 * 24 * time.Hour, 13 * 24 * time.Hour, 14 * 24 * time.Hour,
	} {
		writeBackup(t, base, age)
	}
	n, err := Rotate(base+".*.bak", Policy{MaxKeep: 2, MaxAge: 10 * 24 * time.Hour})
	require.NoError(t, err)
	assert.Equal(t, 4, n, "3 over ceiling + 1 over count cap removed")
	files, err := filepath.Glob(base + ".*.bak")
	require.NoError(t, err)
	require.Len(t, files, 2)
}

func TestRotate_UnparseableNamesIgnored(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	require.NoError(t, os.WriteFile(base+".garbage.bak", []byte("x"), 0600))
	require.NoError(t, os.WriteFile(base+".NOTATIMESTAMP.bak", []byte("x"), 0600))
	writeBackup(t, base, 20*24*time.Hour)
	n, err := Rotate(base+".*.bak", Policy{MaxAge: 10 * 24 * time.Hour})
	require.NoError(t, err)
	assert.Equal(t, 1, n, "only the parseable old backup is removed")
	_, err = os.Stat(base + ".garbage.bak")
	require.NoError(t, err, "unparseable names are left alone")
}

func TestRotate_MinAgeFloorBeatsMaxAgeCeiling(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	name := writeBackup(t, base, 5*24*time.Hour)
	n, err := Rotate(base+".*.bak", Policy{MinAge: 7 * 24 * time.Hour, MaxAge: 3 * 24 * time.Hour})
	require.NoError(t, err)
	assert.Zero(t, n, "misconfigured floor wins over ceiling on the overlap")
	_, err = os.Stat(name)
	require.NoError(t, err)
}

func TestRotate_RemoveError(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	dirName := fmt.Sprintf("%s.%s.bak", base, time.Now().UTC().Add(-2*24*time.Hour).Format(BackupTimeLayout))
	require.NoError(t, os.MkdirAll(filepath.Join(dirName, "sub"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dirName, "sub", "f.json"), []byte("{}"), 0600))
	n, err := Rotate(base+".*.bak", Policy{MaxKeep: 0, MaxAge: 24 * time.Hour})
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	_, err = os.Stat(dirName)
	require.True(t, os.IsNotExist(err))
}

func TestRotate_InvalidTimestampNameIgnored(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	// Matches the regex but fails time.Parse (month 13).
	require.NoError(t, os.WriteFile(base+".20261301T000000Z.bak", []byte("x"), 0600))
	writeBackup(t, base, 20*24*time.Hour)
	n, err := Rotate(base+".*.bak", Policy{MaxAge: 10 * 24 * time.Hour})
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	_, err = os.Stat(base + ".20261301T000000Z.bak")
	require.NoError(t, err, "invalid-timestamp names are left alone")
}

func TestRotate_CapSkipsProtectedInIteration(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	prot := writeBackup(t, base, 1*24*time.Hour) // protected by min-age
	old2 := writeBackup(t, base, 4*24*time.Hour) // eligible, older
	writeBackup(t, base, 5*24*time.Hour)         // eligible, oldest → trimmed
	n, err := Rotate(base+".*.bak", Policy{MaxKeep: 1, MinAge: 3 * 24 * time.Hour})
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	_, err = os.Stat(prot)
	require.NoError(t, err, "protected backup kept")
	_, err = os.Stat(old2)
	require.NoError(t, err, "newest eligible kept")
}

func TestRotate_InvalidGlob(t *testing.T) {
	t.Parallel()
	_, err := Rotate("[", Policy{MaxKeep: 1})
	require.Error(t, err)
}

func TestRotate_MinKeepSurvivesAllAncient(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	for _, age := range []time.Duration{
		40 * 24 * time.Hour, 50 * 24 * time.Hour, 60 * 24 * time.Hour, 70 * 24 * time.Hour, 80 * 24 * time.Hour,
	} {
		writeBackup(t, base, age)
	}
	n, err := Rotate(base+".*.bak", Policy{MinKeep: 3, MaxAge: 30 * 24 * time.Hour})
	require.NoError(t, err)
	assert.Equal(t, 2, n, "max-age ceiling deletes but never below MinKeep")
	files, err := filepath.Glob(base + ".*.bak")
	require.NoError(t, err)
	assert.Len(t, files, 3, "newest MinKeep backups survive the ceiling")
}

func TestRotate_MinKeepFloorsCap(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	for _, age := range []time.Duration{1, 2, 3, 4} {
		writeBackup(t, base, age*24*time.Hour)
	}
	n, err := Rotate(base+".*.bak", Policy{MinKeep: 3, MaxKeep: 1})
	require.NoError(t, err)
	assert.Equal(t, 1, n, "count cap trims but never below MinKeep")
	files, err := filepath.Glob(base + ".*.bak")
	require.NoError(t, err)
	assert.Len(t, files, 3)
}

func TestRotate_MinKeepOverridesMaxKeep(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	for _, age := range []time.Duration{1, 2, 3, 4, 5} {
		writeBackup(t, base, age*24*time.Hour)
	}
	n, err := Rotate(base+".*.bak", Policy{MinKeep: 4, MaxKeep: 1})
	require.NoError(t, err)
	assert.Equal(t, 1, n, "MinKeep wins over a conflicting MaxKeep")
	files, err := filepath.Glob(base + ".*.bak")
	require.NoError(t, err)
	assert.Len(t, files, 4)
}

func TestRotate_MinKeepMoreThanBackupsIsNoop(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	writeBackup(t, base, 90*24*time.Hour)
	writeBackup(t, base, 95*24*time.Hour)
	n, err := Rotate(base+".*.bak", Policy{MinKeep: 3, MaxAge: 30 * 24 * time.Hour})
	require.NoError(t, err)
	assert.Zero(t, n, "fewer backups than MinKeep means nothing is deleted")
	files, err := filepath.Glob(base + ".*.bak")
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestRotate_MinKeepZeroIsUnlimitedFloor(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	for _, age := range []time.Duration{40, 50, 60, 70, 80} {
		writeBackup(t, base, age*24*time.Hour)
	}
	n, err := Rotate(base+".*.bak", Policy{MinKeep: 0, MaxAge: 30 * 24 * time.Hour})
	require.NoError(t, err)
	assert.Equal(t, 5, n, "MinKeep 0 means no floor, ceiling deletes everything")
	files, err := filepath.Glob(base + ".*.bak")
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestRotate_MinKeepSharedBudgetAcrossCeilingAndCap(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	// 2 ancient (ceiling candidate), 5 fresh (cap candidate).
	writeBackup(t, base, 50*24*time.Hour)
	writeBackup(t, base, 60*24*time.Hour)
	for _, age := range []time.Duration{1, 2, 3, 4, 5} {
		writeBackup(t, base, age*24*time.Hour)
	}
	// Total 7, MinKeep 3 → budget 4. Ceiling takes 2 (ancient), cap takes 2
	// (oldest fresh), 3 survive.
	n, err := Rotate(base+".*.bak", Policy{MinKeep: 3, MaxKeep: 1, MaxAge: 30 * 24 * time.Hour})
	require.NoError(t, err)
	assert.Equal(t, 4, n)
	files, err := filepath.Glob(base + ".*.bak")
	require.NoError(t, err)
	assert.Len(t, files, 3)
}

func TestRotate_MinKeepRespectsMinAgeProtection(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	writeBackup(t, base, 1*24*time.Hour)  // protected by MinAge
	writeBackup(t, base, 20*24*time.Hour) // eligible
	writeBackup(t, base, 30*24*time.Hour) // eligible
	// MinKeep 2: budget = 3-2 = 1. Only the single eligible-not-protected-by-age
	// deletion happens; the MinAge-protected backup is never removed.
	n, err := Rotate(base+".*.bak", Policy{MinKeep: 2, MaxAge: 10 * 24 * time.Hour})
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	files, err := filepath.Glob(base + ".*.bak")
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestRotate_RemoveAllError_Ceiling(t *testing.T) {
	tmpDir := t.TempDir()
	roDir := filepath.Join(tmpDir, "ro")
	require.NoError(t, os.Mkdir(roDir, 0700))

	base := filepath.Join(roDir, "manifest.toml")
	require.NoError(t, os.WriteFile(backupName(base, 20*24*time.Hour), []byte("data"), 0600))
	//nolint:gosec // simulate a write-protected dir to force a removal error
	require.NoError(t, os.Chmod(roDir, 0500))       // readable glob, removal denied
	t.Cleanup(func() { _ = os.Chmod(roDir, 0700) }) //nolint:gosec // restore perms for cleanup

	_, err := Rotate(base+".*.bak", Policy{MaxAge: 10 * 24 * time.Hour})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove backup")
}

func TestRotate_RemoveAllError_Cap(t *testing.T) {
	tmpDir := t.TempDir()
	roDir := filepath.Join(tmpDir, "ro")
	require.NoError(t, os.Mkdir(roDir, 0700))

	base := filepath.Join(roDir, "manifest.toml")
	writeBackup(t, base, 2*24*time.Hour)
	writeBackup(t, base, 3*24*time.Hour)
	//nolint:gosec // simulate a write-protected dir to force a removal error
	require.NoError(t, os.Chmod(roDir, 0500))
	t.Cleanup(func() { _ = os.Chmod(roDir, 0700) }) //nolint:gosec // restore perms for cleanup

	_, err := Rotate(base+".*.bak", Policy{MaxKeep: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove backup")
}

func TestList_NewestFirst(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	writeBackup(t, base, 10*24*time.Hour)
	writeBackup(t, base, 2*24*time.Hour)
	writeBackup(t, base, 5*24*time.Hour)
	entries, err := List(base + ".*.bak")
	require.NoError(t, err)
	require.Len(t, entries, 3)
	assert.True(t, entries[0].Time.After(entries[1].Time), "newest first")
	assert.True(t, entries[1].Time.After(entries[2].Time), "newest first")
}

func TestList_CollisionIndex(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	ts := time.Now().UTC().Add(-24 * time.Hour).Format(BackupTimeLayout)
	name1 := fmt.Sprintf("%s.%s.bak", base, ts)
	name2 := fmt.Sprintf("%s.%s.2.bak", base, ts)
	name3 := fmt.Sprintf("%s.%s.10.bak", base, ts)
	for _, n := range []string{name1, name2, name3} {
		require.NoError(t, os.WriteFile(n, []byte("x"), 0600))
	}
	entries, err := List(base + ".*.bak")
	require.NoError(t, err)
	require.Len(t, entries, 3)
	assert.Equal(t, name3, entries[0].Path, "highest collision index first")
	assert.Equal(t, 10, entries[0].Index)
	assert.Equal(t, name2, entries[1].Path)
	assert.Equal(t, 2, entries[1].Index)
	assert.Equal(t, name1, entries[2].Path)
	assert.Zero(t, entries[2].Index)
}

func TestList_InvalidNamesSkipped(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	require.NoError(t, os.WriteFile(base+".garbage.bak", []byte("x"), 0600))
	require.NoError(t, os.WriteFile(base+".20261301T000000Z.bak", []byte("x"), 0600))
	writeBackup(t, base, 20*24*time.Hour)
	entries, err := List(base + ".*.bak")
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func TestList_Empty(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	entries, err := List(base + ".*.bak")
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestList_InvalidGlob(t *testing.T) {
	t.Parallel()
	_, err := List("[")
	require.Error(t, err)
}

func TestRotate_CollisionSuffixRemoved(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "manifest.toml")
	ts := time.Now().UTC().Add(-40 * 24 * time.Hour).Format(BackupTimeLayout)
	require.NoError(t, os.WriteFile(fmt.Sprintf("%s.%s.2.bak", base, ts), []byte("x"), 0600))
	require.NoError(t, os.WriteFile(fmt.Sprintf("%s.%s.3.bak", base, ts), []byte("x"), 0600))
	n, err := Rotate(base+".*.bak", Policy{MaxAge: 30 * 24 * time.Hour})
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}
