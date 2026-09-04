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
		if err := adapter.AddRepo(manager.WithYes(ctx), r.Name, r.URL); err != nil {
			_, _ = fmt.Fprintf(w, "  warning: failed to add repository %s (%s): %v\n", r.Name, r.Manager, err)
		} else {
			_, _ = fmt.Fprintf(w, "  restored repository %s via %s\n", r.Name, r.Manager)
		}
	}
}

// installRestorePackage installs one package, wrapping the context with cask or
// group flags when the manifest entry declares them.
func installRestorePackage(ctx context.Context, w io.Writer, a manager.Adapter, pName string, matches []manifest.Package) error {
	installCtx := ctx
	for _, p := range matches {
		if p.Name == pName && p.Manager == a.Name() {
			if p.Cask {
				installCtx = manager.WithCask(ctx)
			}
			if p.Group {
				installCtx = manager.WithGroup(ctx)
			}
			break
		}
	}
	if err := a.Install(manager.WithYes(installCtx), pName); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(w, "  installed %s via %s\n", pName, a.Name())
	return nil
}

// restorePackageGroup installs one manager's packages concurrently, collecting
// any errors.
func restorePackageGroup(ctx context.Context, w io.Writer, a manager.Adapter, names []string, matches []manifest.Package, errMu *sync.Mutex, errors *[]restoreError) {
	for _, pName := range names {
		if err := installRestorePackage(ctx, w, a, pName, matches); err != nil {
			errMu.Lock()
			*errors = append(*errors, restoreError{Manager: a.Name(), Pkg: pName, Err: err})
			errMu.Unlock()
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
			restorePackageGroup(ctx, w, a, names, pkgs, &errMu, &errors)
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
	printSnapshotWarnings(w, currentSnaps)

	for _, s := range currentSnaps {
		if err := state.Save(snapDir, s); err != nil {
			_, _ = fmt.Fprintf(w, "warning: failed to save snapshot for %s: %v\n", s.Manager, err)
		}
	}
}
