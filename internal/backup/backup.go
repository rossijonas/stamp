// Package backup provides a shared, logrotate-aligned retention policy for
// timestamped backups. Both manifest backups (files) and snapshot backups
// (directories) rotate through the same precedence rules.
package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"
)

// BackupTimeLayout is the timestamp format embedded in backup names
// (<path>.<YYYYMMDDTHHMMSSZ>.bak), chosen so names sort lexicographically by age.
const BackupTimeLayout = "20060102T150405Z"

// Policy is a logrotate-aligned retention policy. A zero value on any axis
// means unlimited on that axis.
type Policy struct {
	// MaxKeep is the count cap (logrotate rotate).
	MaxKeep int
	// MinKeep is the count floor: at least this many backups are always kept,
	// even when the max-age ceiling or count cap would delete more. The newest
	// backups survive. Zero disables the floor.
	MinKeep int
	// MinAge is the floor: backups younger than this are never deleted.
	MinAge time.Duration
	// MaxAge is the ceiling: eligible backups older than this are always deleted.
	MaxAge time.Duration
}

type entry struct {
	path string
	age  time.Duration
}

// Entry is a timestamped backup matched by a glob pattern. Index is the
// numeric collision suffix appended when multiple backups share a timestamp
// (see manifest.uniqueBackupPath); it is 0 when no suffix is present.
type Entry struct {
	Path  string
	Time  time.Time
	Index int
}

// tsPattern extracts the timestamp (and optional collision suffix) from a
// backup name: <path>.<YYYYMMDDTHHMMSSZ>.<N>.bak or <path>.<YYYYMMDDTHHMMSSZ>.bak.
var tsPattern = regexp.MustCompile(`\.(\d{8}T\d{6}Z)(?:\.(\d+))?\.bak$`)

// List returns the timestamped backup entries matched by globPattern, sorted
// newest first. Names that do not parse as backups are skipped. A malformed
// glob pattern is the only error case.
func List(globPattern string) ([]Entry, error) {
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to expand backup glob: %w", err)
	}

	entries := make([]Entry, 0, len(matches))
	for _, m := range matches {
		sub := tsPattern.FindStringSubmatch(m)
		if sub == nil {
			continue
		}
		ts, err := time.Parse(BackupTimeLayout, sub[1])
		if err != nil {
			continue
		}
		index := 0
		if len(sub) > 2 && sub[2] != "" {
			index, _ = strconv.Atoi(sub[2])
		}
		entries = append(entries, Entry{Path: m, Time: ts, Index: index})
	}
	sort.Slice(entries, func(i, j int) bool {
		if !entries[i].Time.Equal(entries[j].Time) {
			return entries[i].Time.After(entries[j].Time)
		}
		return entries[i].Index > entries[j].Index
	})
	return entries, nil
}

// Rotate prunes timestamped backup entries matched by globPattern according to
// the retention precedence: min-age floor (protect) > min-count floor (keep at
// least MinKeep) > max-age ceiling (delete) > count cap (trim). A shared
// deletion budget of len(entries)-MinKeep bounds both deletion passes so the
// newest MinKeep backups always survive. Unparseable names are left untouched.
// Returns the number of entries removed.
func Rotate(globPattern string, p Policy) (int, error) {
	found, err := List(globPattern)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	entries := make([]entry, 0, len(found))
	for _, e := range found {
		entries = append(entries, entry{path: e.Path, age: now.Sub(e.Time)})
	}
	// List returns newest-first; rotation scans oldest-first.
	sort.Slice(entries, func(i, j int) bool { return entries[i].age > entries[j].age })

	// budget is how many backups may be deleted in total across both passes.
	// MinKeep backups always survive (the newest, since scans are oldest-first).
	budget := len(entries) - p.MinKeep
	if budget < 0 {
		budget = 0
	}

	removed := 0
	kept := make([]entry, 0, len(entries))
	for _, e := range entries {
		protected := p.MinAge > 0 && e.age < p.MinAge
		overCeiling := p.MaxAge > 0 && e.age > p.MaxAge
		if !protected && overCeiling && budget > 0 {
			if err := os.RemoveAll(e.path); err != nil {
				return removed, fmt.Errorf("failed to remove backup %s: %w", e.path, err)
			}
			removed++
			budget--
			continue
		}
		kept = append(kept, e)
	}

	if p.MaxKeep > 0 {
		eligible := 0
		for _, e := range kept {
			if e.age >= p.MinAge {
				eligible++
			}
		}
		// kept is sorted oldest-first and all eligible entries (age >= MinAge)
		// precede protected ones, so trimming from the front only ever removes
		// eligible backups.
		surplus := eligible - p.MaxKeep
		for i := 0; i < len(kept) && surplus > 0 && budget > 0; i++ {
			if err := os.RemoveAll(kept[i].path); err != nil {
				return removed, fmt.Errorf("failed to remove backup %s: %w", kept[i].path, err)
			}
			removed++
			surplus--
			budget--
		}
	}

	return removed, nil
}
