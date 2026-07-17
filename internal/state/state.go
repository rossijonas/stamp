// Package state handles the point-in-time snapshotting of package states
// and calculating the delta (drift) between snapshots.
package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/rossijonas/stamp/internal/manager"
)

// Snapshot represents a point-in-time record of packages and repositories installed
// by a single package manager.
type Snapshot struct {
	Manager      string                   `json:"manager"`
	Packages     []string                 `json:"packages"`
	Repositories []manager.RepositoryInfo `json:"repositories,omitempty"`
	UpdatedAt    time.Time                `json:"updated_at"`
}

// UnmarshalJSON implements json.Unmarshaler for backward compatibility with old
// snapshot format where Repositories was a []string.
func (s *Snapshot) UnmarshalJSON(data []byte) error {
	type snapshotAlias Snapshot
	alias := struct {
		Repositories json.RawMessage `json:"repositories,omitempty"`
		*snapshotAlias
	}{snapshotAlias: (*snapshotAlias)(s)}

	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	if len(alias.Repositories) == 0 {
		s.Repositories = nil
		return nil
	}

	// Try new format first ([]RepositoryInfo)
	var repos []manager.RepositoryInfo
	if err := json.Unmarshal(alias.Repositories, &repos); err == nil {
		s.Repositories = repos
		return nil
	}

	// Fallback to old format ([]string)
	var names []string
	if err := json.Unmarshal(alias.Repositories, &names); err != nil {
		return fmt.Errorf("failed to unmarshal repositories")
	}
	s.Repositories = make([]manager.RepositoryInfo, len(names))
	for i, n := range names {
		s.Repositories[i] = manager.RepositoryInfo{Name: n}
	}
	return nil
}

// Delta represents the difference (added and removed packages/repos) between two snapshots.
type Delta struct {
	Manager      string                   `json:"manager"`
	Added        []string                 `json:"added"`
	Removed      []string                 `json:"removed"`
	AddedRepos   []manager.RepositoryInfo `json:"added_repos,omitempty"`
	RemovedRepos []manager.RepositoryInfo `json:"removed_repos,omitempty"`
}

func xdgStateDir() string {
	d := os.Getenv("XDG_DATA_HOME")
	if d == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "/tmp"
		}
		d = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(d, "stamp")
}

// SnapshotDirPath returns the path to the snapshots directory without creating it.
func SnapshotDirPath() string {
	return filepath.Join(xdgStateDir(), "snapshots")
}

// SnapshotDir returns the path to the snapshots directory and creates it if it doesn't exist.
func SnapshotDir() (string, error) {
	dir := SnapshotDirPath()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create snapshot directory: %w", err)
	}
	return dir, nil
}

// BackupSnapshots renames the snapshots directory to a timestamped backup.
// Format: <path>.<YYYYMMDD>THHMMSSZ.bak
func BackupSnapshots(snapDir string) (string, error) {
	ts := time.Now().UTC().Format("20060102T150405Z")
	backupDir := snapDir + "." + ts + ".bak"
	if err := os.Rename(snapDir, backupDir); err != nil {
		return "", fmt.Errorf("failed to backup snapshots to %s: %w", backupDir, err)
	}
	return backupDir, nil
}

// Save writes a snapshot as a JSON file to disk.
func Save(dir string, snap Snapshot) error {
	if filepath.Base(snap.Manager) != snap.Manager {
		return fmt.Errorf("invalid manager name: %s", snap.Manager)
	}
	path := filepath.Join(dir, fmt.Sprintf("%s.json", snap.Manager))
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot for %s: %w", snap.Manager, err)
	}

	//nolint:gosec // permissions are restricted to 0600
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write snapshot for %s: %w", snap.Manager, err)
	}

	return nil
}

// Load reads a snapshot from disk for the given package manager.
func Load(dir, managerName string) (*Snapshot, error) {
	if filepath.Base(managerName) != managerName {
		return nil, fmt.Errorf("invalid manager name: %s", managerName)
	}
	path := filepath.Join(dir, fmt.Sprintf("%s.json", managerName))
	//nolint:gosec // path is constructed safely inside state package
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot for %s: %w", managerName, err)
	}

	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot for %s: %w", managerName, err)
	}

	return &snap, nil
}

// Current fetches the live list of installed packages and repositories from all adapters concurrently.
func Current(ctx context.Context, adapters []manager.Adapter) ([]Snapshot, error) {
	type result struct {
		snap Snapshot
		err  error
	}

	ch := make(chan result, len(adapters))
	var wg sync.WaitGroup

	for _, adapter := range adapters {
		wg.Add(1)
		go func(a manager.Adapter) {
			defer wg.Done()
			packages, err := a.ListInstalled(ctx)
			if err != nil {
				ch <- result{err: fmt.Errorf("failed to list installed for %s: %w", a.Name(), err)}
				return
			}
			repos, err := a.ListRepos(ctx)
			if err != nil {
				ch <- result{err: fmt.Errorf("failed to list repositories for %s: %w", a.Name(), err)}
				return
			}
			ch <- result{
				snap: Snapshot{
					Manager:      a.Name(),
					Packages:     packages,
					Repositories: repos,
					UpdatedAt:    time.Now(),
				},
			}
		}(adapter)
	}

	wg.Wait()
	close(ch)

	snapshots := make([]Snapshot, 0, len(adapters))
	for r := range ch {
		if r.err != nil {
			return nil, r.err
		}
		snapshots = append(snapshots, r.snap)
	}

	return snapshots, nil
}

// diffSorted computes added/removed between two sorted slices.
func diffSorted(a, b []string) (added, removed []string) {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		o, n := a[i], b[j]
		switch {
		case o < n:
			removed = append(removed, o)
			i++
		case o > n:
			added = append(added, n)
			j++
		default:
			i++
			j++
		}
	}
	for ; i < len(a); i++ {
		removed = append(removed, a[i])
	}
	for ; j < len(b); j++ {
		added = append(added, b[j])
	}
	return added, removed
}

// diffSortedRepos computes added/removed repositories between two sorted slices,
// comparing by Name only (URL changes are not drift).
func diffSortedRepos(a, b []manager.RepositoryInfo) (added, removed []manager.RepositoryInfo) {
	slices.SortFunc(a, func(x, y manager.RepositoryInfo) int {
		return strings.Compare(x.Name, y.Name)
	})
	slices.SortFunc(b, func(x, y manager.RepositoryInfo) int {
		return strings.Compare(x.Name, y.Name)
	})

	i, j := 0, 0
	for i < len(a) && j < len(b) {
		o, n := a[i], b[j]
		switch {
		case o.Name < n.Name:
			removed = append(removed, o)
			i++
		case o.Name > n.Name:
			added = append(added, n)
			j++
		default:
			i++
			j++
		}
	}
	for ; i < len(a); i++ {
		removed = append(removed, a[i])
	}
	for ; j < len(b); j++ {
		added = append(added, b[j])
	}
	return added, removed
}

// Diff calculates the added and removed packages and repositories between an old and new snapshot.
func Diff(oldSnap, newSnap Snapshot) *Delta {
	oldPkgs := slices.Clone(oldSnap.Packages)
	newPkgs := slices.Clone(newSnap.Packages)
	slices.Sort(oldPkgs)
	slices.Sort(newPkgs)

	added, removed := diffSorted(oldPkgs, newPkgs)

	addedRepos, removedRepos := diffSortedRepos(oldSnap.Repositories, newSnap.Repositories)

	return &Delta{
		Manager:      newSnap.Manager,
		Added:        added,
		Removed:      removed,
		AddedRepos:   addedRepos,
		RemovedRepos: removedRepos,
	}
}

// DiffAll calculates the diffs across all snapshots.
// Returns a slice of Deltas mapped by manager.
func DiffAll(oldSnaps, newSnaps []Snapshot) []Delta {
	oldMap := make(map[string]Snapshot)
	for _, s := range oldSnaps {
		oldMap[s.Manager] = s
	}

	deltas := make([]Delta, 0, len(newSnaps))
	for _, n := range newSnaps {
		o, ok := oldMap[n.Manager]
		if !ok {
			// If no old snapshot exists for this manager, treat all as added
			o = Snapshot{Manager: n.Manager, Packages: []string{}, Repositories: []manager.RepositoryInfo{}}
		}
		deltas = append(deltas, *Diff(o, n))
	}

	return deltas
}
