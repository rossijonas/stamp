package cli

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
	"github.com/rossijonas/stamp/internal/state"
)

// missingFromSystem returns manifest entries (excluding groups and casks)
// whose manager adapter is active but whose ListInstalled output lacks the
// package. A manager whose ListInstalled fails is skipped: we cannot compare
// against it, and the check is best-effort diagnostics, not a hard error.
// Results are deduplicated and sorted by (manager, name) for deterministic
// output across doctor, ls --type missing, and tests.
func missingFromSystem(ctx context.Context, adapters []manager.Adapter, m *manifest.Manifest) []manifest.Package {
	if m == nil || len(adapters) == 0 {
		return nil
	}

	byManager := make(map[string][]manifest.Package)
	for _, p := range m.Packages {
		if p.Group || p.Cask {
			continue
		}
		byManager[p.Manager] = append(byManager[p.Manager], p)
	}
	if len(byManager) == 0 {
		return nil
	}

	var active []manager.Adapter
	for _, a := range adapters {
		if _, ok := byManager[a.Name()]; ok {
			active = append(active, a)
		}
	}
	if len(active) == 0 {
		return nil
	}

	type result struct {
		manager string
		present map[string]struct{}
	}
	ch := make(chan result, len(active))
	var wg sync.WaitGroup
	for _, a := range active {
		wg.Add(1)
		go func(a manager.Adapter) {
			defer wg.Done()
			candidates := byManager[a.Name()]
			names := make([]string, 0, len(candidates))
			for _, p := range candidates {
				names = append(names, p.Name)
			}
			// Adapters with a real presence check win: their system view
			// includes dependency-installed packages and provides aliases
			// that ListInstalled matching cannot see.
			if c, ok := a.(manager.InstalledChecker); ok {
				absent, err := c.CheckInstalled(ctx, names)
				if err == nil {
					absentSet := make(map[string]struct{}, len(absent))
					for _, n := range absent {
						absentSet[n] = struct{}{}
					}
					present := make(map[string]struct{}, len(names)-len(absentSet))
					for _, n := range names {
						if _, gone := absentSet[n]; !gone {
							present[n] = struct{}{}
						}
					}
					ch <- result{manager: a.Name(), present: present}
					return
				}
				// Fall through to the best-effort legacy path below.
			}
			pkgs, err := a.ListInstalled(ctx)
			if err != nil {
				return
			}
			present := make(map[string]struct{}, len(pkgs))
			for _, p := range pkgs {
				present[p] = struct{}{}
			}
			ch <- result{manager: a.Name(), present: present}
		}(a)
	}
	wg.Wait()
	close(ch)

	var missing []manifest.Package
	for r := range ch {
		for _, p := range byManager[r.manager] {
			if _, isPresent := r.present[p.Name]; !isPresent {
				missing = append(missing, p)
			}
		}
	}
	return dedupeAndSortMissing(missing)
}

// missingFromDeltas returns manifest entries (excluding groups and casks)
// that appear in a delta's Removed set for the same manager. Reconcile feeds
// it state.DiffAll output, which already compares the last snapshot against
// the live system, so a package removed via the native manager surfaces here.
func missingFromDeltas(deltas []state.Delta, m *manifest.Manifest) []manifest.Package {
	if m == nil || len(deltas) == 0 {
		return nil
	}

	removed := make(map[string]map[string]struct{})
	for _, d := range deltas {
		if len(d.Removed) == 0 {
			continue
		}
		set := make(map[string]struct{}, len(d.Removed))
		for _, p := range d.Removed {
			set[p] = struct{}{}
		}
		removed[d.Manager] = set
	}

	var missing []manifest.Package
	for _, p := range m.Packages {
		if p.Group || p.Cask {
			continue
		}
		if set, ok := removed[p.Manager]; ok {
			if _, gone := set[p.Name]; gone {
				missing = append(missing, p)
			}
		}
	}
	return dedupeAndSortMissing(missing)
}

// dedupeAndSortMissing returns a defensive copy of pkgs with duplicate
// (manager, name) entries removed, sorted by manager then name.
func dedupeAndSortMissing(pkgs []manifest.Package) []manifest.Package {
	if len(pkgs) == 0 {
		return nil
	}
	out := slices.Clone(pkgs)
	slices.SortFunc(out, func(a, b manifest.Package) int {
		if a.Manager != b.Manager {
			return strings.Compare(a.Manager, b.Manager)
		}
		return strings.Compare(a.Name, b.Name)
	})
	seen := make(map[[2]string]struct{}, len(out))
	uniq := out[:0]
	for _, p := range out {
		key := [2]string{p.Manager, p.Name}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		uniq = append(uniq, p)
	}
	return uniq
}
