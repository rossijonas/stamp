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
// group flags when the manifest entry declares them. Echoes on success.
func installRestorePackage(ctx context.Context, w io.Writer, a manager.Adapter, pName string, matches []manifest.Package, outMu *sync.Mutex) error {
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
	outMu.Lock()
	_, _ = fmt.Fprintf(w, "  installed %s via %s\n", pName, a.Name())
	outMu.Unlock()
	return nil
}

// restorePackageGroup installs one manager's packages. When the adapter
// implements BatchInstaller, packages are batched in a single native
// invocation per manager (excluding group packages, which are installed
// individually). On batch failure, all packages are retried individually to
// attribute per-package errors. Adapters without BatchInstaller fall back to
// sequential per-package installs.
func restorePackageGroup(ctx context.Context, w io.Writer, a manager.Adapter, names []string, matches []manifest.Package, errMu, outMu *sync.Mutex, errors *[]restoreError) {
	bi, canBatch := a.(manager.BatchInstaller)
	if !canBatch {
		restorePackageGroupSequential(ctx, w, a, names, matches, errMu, outMu, errors)
		return
	}

	groups, regular := splitGroupPackages(names, matches, a.Name())
	installGroupsIndividually(ctx, w, a, groups, matches, errMu, outMu, errors)

	if len(regular) == 0 {
		return
	}

	casks := brewCasks(ctx, a, regular)
	installBatchOrRetry(ctx, w, a, bi, regular, casks, errMu, outMu, errors)
}

// splitGroupPackages separates package names into group (manifest Group=true)
// and regular lists.
func splitGroupPackages(names []string, matches []manifest.Package, mgr string) (groups, regular []string) {
	for _, n := range names {
		isGroup := false
		for _, p := range matches {
			if p.Name == n && p.Manager == mgr && p.Group {
				isGroup = true
				break
			}
		}
		if isGroup {
			groups = append(groups, n)
		} else {
			regular = append(regular, n)
		}
	}
	return
}

// installGroupsIndividually installs group packages one-by-one (dnf rejects
// groups in batch).
func installGroupsIndividually(ctx context.Context, w io.Writer, a manager.Adapter, groups []string, matches []manifest.Package, errMu, outMu *sync.Mutex, errors *[]restoreError) {
	for _, g := range groups {
		if err := installRestorePackage(ctx, w, a, g, matches, outMu); err != nil {
			errMu.Lock()
			*errors = append(*errors, restoreError{Manager: a.Name(), Pkg: g, Err: err})
			errMu.Unlock()
		}
	}
}

// installBatchOrRetry attempts a single batch install. If all packages are
// casks, WithCask is set. On mixed cask/formula, falls back to per-package.
// On batch failure, retries all individually with per-package error
// attribution.
func installBatchOrRetry(ctx context.Context, w io.Writer, a manager.Adapter, bi manager.BatchInstaller, regular []string, casks map[string]bool, errMu, outMu *sync.Mutex, errors *[]restoreError) {
	caskCount := countCasks(casks)
	allCask := caskCount == len(regular)
	mixed := caskCount > 0 && caskCount < len(regular)

	if mixed {
		installPackagesWithCask(ctx, w, a, regular, casks, errMu, outMu, errors)
		return
	}

	batchCtx := manager.WithYes(ctx)
	if allCask {
		batchCtx = manager.WithCask(batchCtx)
	}

	if err := bi.InstallMany(batchCtx, regular...); err != nil {
		installPackagesWithCask(ctx, w, a, regular, casks, errMu, outMu, errors)
		return
	}
	outMu.Lock()
	_, _ = fmt.Fprintf(w, "  restored %d package(s) via %s\n", len(regular), a.Name())
	outMu.Unlock()
}

// installPackagesWithCask installs each package individually, stacking
// WithCask for packages the live cask map flags. Used for the mixed
// cask/formula fallback and for per-package error attribution after a batch
// failure. Echoes on success.
func installPackagesWithCask(ctx context.Context, w io.Writer, a manager.Adapter, pkgs []string, casks map[string]bool, errMu, outMu *sync.Mutex, errors *[]restoreError) {
	for _, p := range pkgs {
		pkgCtx := manager.WithYes(ctx)
		if casks[p] {
			pkgCtx = manager.WithYes(manager.WithCask(ctx))
		}
		if err := a.Install(pkgCtx, p); err != nil {
			errMu.Lock()
			*errors = append(*errors, restoreError{Manager: a.Name(), Pkg: p, Err: err})
			errMu.Unlock()
			continue
		}
		outMu.Lock()
		_, _ = fmt.Fprintf(w, "  installed %s via %s\n", p, a.Name())
		outMu.Unlock()
	}
}

// restorePackageGroupSequential installs packages one-by-one for adapters
// that do not implement BatchInstaller.
func restorePackageGroupSequential(ctx context.Context, w io.Writer, a manager.Adapter, names []string, matches []manifest.Package, errMu, outMu *sync.Mutex, errors *[]restoreError) {
	for _, pName := range names {
		if err := installRestorePackage(ctx, w, a, pName, matches, outMu); err != nil {
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
	var outMu sync.Mutex
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
			restorePackageGroup(ctx, w, a, names, pkgs, &errMu, &outMu, &errors)
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
