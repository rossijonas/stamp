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
	"sync"
	"time"

	"github.com/rossijonas/stamp/internal/manager"
)

// Snapshot represents a point-in-time record of packages and repositories installed
// by a single package manager.
type Snapshot struct {
	Manager      string    `json:"manager"`
	Packages     []string  `json:"packages"`
	Repositories []string  `json:"repositories,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Delta represents the difference (added and removed packages/repos) between two snapshots.
type Delta struct {
	Manager      string   `json:"manager"`
	Added        []string `json:"added"`
	Removed      []string `json:"removed"`
	AddedRepos   []string `json:"added_repos,omitempty"`
	RemovedRepos []string `json:"removed_repos,omitempty"`
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

// SnapshotDir returns the path to the snapshots directory and creates it if it doesn't exist.
func SnapshotDir() (string, error) {
	dir := filepath.Join(xdgStateDir(), "snapshots")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create snapshot directory: %w", err)
	}
	return dir, nil
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

// Diff calculates the added and removed packages and repositories between an old and new snapshot.
func Diff(oldSnap, newSnap Snapshot) *Delta {
	oldPkgs := slices.Clone(oldSnap.Packages)
	newPkgs := slices.Clone(newSnap.Packages)
	slices.Sort(oldPkgs)
	slices.Sort(newPkgs)

	added, removed := diffSorted(oldPkgs, newPkgs)

	oldRepos := slices.Clone(oldSnap.Repositories)
	newRepos := slices.Clone(newSnap.Repositories)
	slices.Sort(oldRepos)
	slices.Sort(newRepos)

	addedRepos, removedRepos := diffSorted(oldRepos, newRepos)

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
			o = Snapshot{Manager: n.Manager, Packages: []string{}, Repositories: []string{}}
		}
		deltas = append(deltas, *Diff(o, n))
	}

	return deltas
}
