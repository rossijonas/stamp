package cli

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
	"github.com/rossijonas/stamp/internal/state"
)

type restoreError struct {
	Manager string
	Pkg     string
	Err     error
}

func restoreRepositories(ctx context.Context, w io.Writer, adapters []manager.Adapter, repos []manifest.Repository) {
	if len(repos) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w, "Phase 1: Restoring Repositories...")
	for _, r := range repos {
		var adapter manager.Adapter
		for _, a := range adapters {
			if a.Name() == r.Manager {
				adapter = a
				break
			}
		}
		if adapter == nil {
			_, _ = fmt.Fprintf(w, "  warning: manager %s not available for repository %s\n", r.Manager, r.Name)
			continue
		}
		if err := adapter.AddRepo(ctx, r.Name, r.URL); err != nil {
			_, _ = fmt.Fprintf(w, "  warning: failed to add repository %s (%s): %v\n", r.Name, r.Manager, err)
		} else {
			_, _ = fmt.Fprintf(w, "  restored repository %s via %s\n", r.Name, r.Manager)
		}
	}
}

func restorePackages(ctx context.Context, w io.Writer, adapters []manager.Adapter, pkgs []manifest.Package) []restoreError {
	if len(pkgs) == 0 {
		return nil
	}
	_, _ = fmt.Fprintln(w, "Phase 2: Restoring Packages...")

	byManager := make(map[string][]string)
	for _, p := range pkgs {
		byManager[p.Manager] = append(byManager[p.Manager], p.Name)
	}

	var errors []restoreError
	var errMu sync.Mutex
	var wg sync.WaitGroup

	for mName, pNames := range byManager {
		var adapter manager.Adapter
		for _, a := range adapters {
			if a.Name() == mName {
				adapter = a
				break
			}
		}
		if adapter == nil {
			_, _ = fmt.Fprintf(w, "  warning: manager %s not available, skipping %d package(s)\n", mName, len(pNames))
			continue
		}

		wg.Add(1)
		go func(a manager.Adapter, names []string) {
			defer wg.Done()
			for _, pName := range names {
				if err := a.Install(ctx, pName); err != nil {
					errMu.Lock()
					errors = append(errors, restoreError{Manager: a.Name(), Pkg: pName, Err: err})
					errMu.Unlock()
				} else {
					_, _ = fmt.Fprintf(w, "  installed %s via %s\n", pName, a.Name())
				}
			}
		}(adapter, pNames)
	}

	wg.Wait()
	return errors
}

func restoreSaveSnapshots(ctx context.Context, w io.Writer, adapters []manager.Adapter) {
	snapDir, err := state.SnapshotDir()
	if err != nil {
		return
	}
	currentSnaps, err := state.Current(ctx, adapters)
	if err != nil {
		return
	}
	for _, s := range currentSnaps {
		if err := state.Save(snapDir, s); err != nil {
			_, _ = fmt.Fprintf(w, "warning: failed to save snapshot for %s: %v\n", s.Manager, err)
		}
	}
}
