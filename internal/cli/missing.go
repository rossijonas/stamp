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

// managerPresent computes the set of present packages for one adapter. It
// returns ok=false when presence cannot be determined (ListInstalled error)
// and the manager should be skipped entirely.
func managerPresent(ctx context.Context, a manager.Adapter, names []string) (present map[string]struct{}, ok bool) {
	// Adapters with a real presence check win: their system view includes
	// dependency-installed packages and provides aliases that ListInstalled
	// matching cannot see.
	if c, isChecker := a.(manager.InstalledChecker); isChecker {
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
			return present, true
		}
		// Fall through to the best-effort legacy path below.
	}
	pkgs, err := a.ListInstalled(ctx)
	if err != nil {
		return nil, false
	}
	present = make(map[string]struct{}, len(pkgs))
	for _, p := range pkgs {
		present[p] = struct{}{}
	}
	return present, true
}

// buildMissingCandidates groups manifest packages by manager, excluding groups
// and casks, and returns the managers that have an active adapter.
func buildMissingCandidates(adapters []manager.Adapter, m *manifest.Manifest) (byManager map[string][]manifest.Package, active []manager.Adapter) {
	byManager = make(map[string][]manifest.Package)
	for _, p := range m.Packages {
		if p.Group || p.Cask {
			continue
		}
		byManager[p.Manager] = append(byManager[p.Manager], p)
	}
	if len(byManager) == 0 {
		return nil, nil
	}

	for _, a := range adapters {
		if _, ok := byManager[a.Name()]; ok {
			active = append(active, a)
		}
	}
	return byManager, active
}

// queryManagerPresence resolves the present-set for one adapter's candidates
// and sends the result, or skips when presence cannot be determined.
func queryManagerPresence(ctx context.Context, ch chan<- missingResult, a manager.Adapter, candidates []manifest.Package) {
	names := make([]string, 0, len(candidates))
	for _, p := range candidates {
		names = append(names, p.Name)
	}
	present, ok := managerPresent(ctx, a, names)
	if !ok {
		return
	}
	ch <- missingResult{manager: a.Name(), present: present}
}

type missingResult struct {
	manager string
	present map[string]struct{}
}

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

	byManager, active := buildMissingCandidates(adapters, m)
	if len(active) == 0 {
		return nil
	}

	ch := make(chan missingResult, len(active))
	var wg sync.WaitGroup
	for _, a := range active {
		wg.Add(1)
		go func(a manager.Adapter) {
			defer wg.Done()
			queryManagerPresence(ctx, ch, a, byManager[a.Name()])
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

// collectRemoved builds a manager → removed-package-set map from deltas.
func collectRemoved(deltas []state.Delta) map[string]map[string]struct{} {
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
	return removed
}

// missingFromDeltas returns manifest entries (excluding groups and casks)
// that appear in a delta's Removed set for the same manager. Reconcile feeds
// it state.DiffAll output, which already compares the last snapshot against
// the live system, so a package removed via the native manager surfaces here.
func missingFromDeltas(deltas []state.Delta, m *manifest.Manifest) []manifest.Package {
	if m == nil || len(deltas) == 0 {
		return nil
	}

	removed := collectRemoved(deltas)

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
