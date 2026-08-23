package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
	"github.com/rossijonas/stamp/internal/state"
)

type discoveredPkg struct {
	Name    string
	Manager string
}

type discoveredRepo struct {
	Name    string
	Manager string
	URL     string
}

func filterAdapters(adapters []manager.Adapter, managerFlag string) ([]manager.Adapter, error) {
	if managerFlag == "" {
		return adapters, nil
	}
	var found manager.Adapter
	for _, a := range adapters {
		if a.Name() == manager.ResolveManager(managerFlag) {
			found = a
			break
		}
	}
	if found == nil {
		return nil, catErr(ErrUnavailable, "manager %q not available on this system", managerFlag)
	}
	return []manager.Adapter{found}, nil
}

func loadOldSnapshots(adapters []manager.Adapter, snapDir string) ([]state.Snapshot, error) {
	oldSnaps := make([]state.Snapshot, 0, len(adapters))
	for _, a := range adapters {
		snap, err := state.Load(snapDir, a.Name())
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("failed to load snapshot for %s: %w", a.Name(), err)
		}
		oldSnaps = append(oldSnaps, *snap)
	}
	return oldSnaps, nil
}

func collectDiscovered(deltas []state.Delta) ([]discoveredPkg, []discoveredRepo) {
	var discovered []discoveredPkg
	var discoveredRepos []discoveredRepo
	for _, d := range deltas {
		for _, p := range d.Added {
			discovered = append(discovered, discoveredPkg{Name: p, Manager: d.Manager})
		}
		for _, r := range d.AddedRepos {
			discoveredRepos = append(discoveredRepos, discoveredRepo{Name: r.Name, Manager: d.Manager, URL: r.URL})
		}
	}
	return discovered, discoveredRepos
}

// filterDiscoveredByManifest drops drift entries already tracked in the
// manifest. Reconcile is a fallback for packages installed outside stamp, so
// entries stamp already knows about (via install/repo add) must not be reported
// as new drift. Matching is by (name, manager), mirroring HasPackage.
func filterDiscoveredByManifest(discovered []discoveredPkg, discoveredRepos []discoveredRepo, m *manifest.Manifest) ([]discoveredPkg, []discoveredRepo) {
	if m == nil {
		return discovered, discoveredRepos
	}
	var pkgs []discoveredPkg
	for _, p := range discovered {
		if !m.HasPackage(p.Name, p.Manager) {
			pkgs = append(pkgs, p)
		}
	}
	var repos []discoveredRepo
	for _, r := range discoveredRepos {
		if !m.HasRepository(r.Name, r.Manager) {
			repos = append(repos, r)
		}
	}
	return pkgs, repos
}

func saveCurrentSnaps(w io.Writer, snapDir string, snaps []state.Snapshot) {
	for _, s := range snaps {
		if err := state.Save(snapDir, s); err != nil {
			_, _ = fmt.Fprintf(w, "warning: failed to save snapshot for %s: %v\n", s.Manager, err)
		}
	}
}

func saveAndTrack(discovered []discoveredPkg, discoveredRepos []discoveredRepo, app *AppContext) (int, int, error) {
	trackedCount := 0
	trackedReposCount := 0
	for _, p := range discovered {
		if app.manifest.AddPackage(manifest.Package{Name: p.Name, Manager: p.Manager, Origin: manifest.OriginReconciled}) {
			trackedCount++
		}
	}
	for _, r := range discoveredRepos {
		if app.manifest.AddRepository(manifest.Repository{Name: r.Name, Manager: r.Manager, URL: r.URL, Origin: manifest.OriginReconciled}) {
			trackedReposCount++
		}
	}
	if trackedCount > 0 || trackedReposCount > 0 {
		if err := app.saveManifest(); err != nil {
			return 0, 0, fmt.Errorf("failed to save manifest: %w", err)
		}
	}
	return trackedCount, trackedReposCount, nil
}
