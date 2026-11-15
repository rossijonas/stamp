package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

func newInstallCmd() *cobra.Command {
	var managerFlag string
	var note string
	var groupInstall bool

	cmd := &cobra.Command{
		Use:     "install <package>",
		Aliases: []string{"add"},
		Short:   "Install a package and record intent",
		Example: `  # install htop using the default system manager
  stamp install htop

  # install from a specific manager
  stamp install spotify -m flatpak

  # install a tap-qualified Homebrew formula (auto-routes to brew)
  stamp install nklmilojevic/sofka/sofka

  # install multiple packages in one command (per-manager batch, -m required)
  stamp install htop atop btop -m dnf

  # install a DNF package group (by group ID, see 'dnf group list')
  stamp install development-tools -m dnf --group

  # add a note so you remember why later
  stamp add lazygit -m brew --note "better git TUI"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(cmd, appFromCtx(cmd), args, managerFlag, note, groupInstall)
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use")
	cmd.Flags().StringVarP(&note, "note", "n", "", "annotation for this package")
	cmd.Flags().BoolVarP(&groupInstall, "group", "g", false, "install a DNF package group (by group ID)")
	return cmd
}

// runInstall performs a single- or multi-package install: flatpak argument
// validation, tap-qualified routing, manager resolution, package validation,
// and dispatch to the single or batch install path.
func runInstall(cmd *cobra.Command, app *AppContext, args []string, managerFlag, note string, groupInstall bool) error {
	if app.manifestErr != nil {
		return app.manifestErr
	}

	if len(args) > 1 && managerFlag != "" && manager.ResolveManager(managerFlag) == "flatpak" && !strings.Contains(args[0], ".") {
		return catErr(ErrUsage, "flatpak install takes a remote and app ID separately; use: stamp install <app-id> -m flatpak")
	}
	if len(args) > 1 {
		return installMany(cmd, app, args, managerFlag, note, groupInstall)
	}
	pkgName := args[0]

	if strings.Count(pkgName, "/") == 1 && !strings.HasPrefix(pkgName, "@") && (managerFlag == "" || manager.ResolveManager(managerFlag) == "brew") {
		return catErr(ErrUsage, "%q looks like a Homebrew tap; use 'stamp tap %s'", pkgName, pkgName)
	}

	r := NewResolver(app.adapters, app.config)
	adapter, err := r.Resolve(pkgName, managerFlag)
	if err != nil {
		return fmt.Errorf("cannot resolve package manager: %w", err)
	}

	if groupInstall {
		if !isValidGroupID(pkgName) {
			return catErr(ErrUsage, "group IDs contain only a-z0-9_- (see 'dnf group list')")
		}
	} else if err := manager.ValidatePackageForManager(adapter.Name(), pkgName); err != nil {
		return err
	}

	cask := detectBrewCask(cmd.Context(), adapter, pkgName)
	return runSingleInstall(cmd, app, adapter, pkgName, note, cask, groupInstall)
}

// runSingleInstall performs one package install: group validation, context
// flag stacking, confirmation gate, native install, manifest record, save, and
// the completion status line. Shared by the single-package install path.
func runSingleInstall(cmd *cobra.Command, app *AppContext, adapter manager.Adapter, pkgName, note string, cask, group bool) error {
	if err := validateGroupSupport(adapter, group); err != nil {
		return err
	}

	installCtx := applyOpFlags(cmd.Context(), cask, group)

	// Confirmation gate: prompts unless -y is passed. Non-interactive
	// runs without -y abort (fail closed, non-zero exit).
	if err := confirmDestructive(installCtx, cmd.ErrOrStderr(), cmd.InOrStdin(), app.yes,
		adapter, previewInstall, "Install", pkgName); err != nil {
		return handleConsent(err)
	}

	errOut := cmd.ErrOrStderr()
	tty := isOutputTerminal(errOut)
	printStatus(errOut, tty, false, "installing", pkgName, adapter.Name(), "")

	if err := adapter.Install(manager.WithYes(installCtx), pkgName); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	manifestName := manager.TrackedName(adapter.Name(), pkgName)
	if !group {
		// Group IDs are not packages: resolving them would query the manager
		// for a name it cannot install and warn spuriously.
		manifestName = trackedInstallName(cmd.Context(), adapter, pkgName, cmd.ErrOrStderr())
	}

	app.manifest.AddPackage(manifest.Package{
		Name:    manifestName,
		Manager: adapter.Name(),
		Notes:   note,
		Cask:    cask,
		Group:   group,
		Origin:  manifest.OriginStamped,
	})

	if err := app.saveManifest(); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	printStatus(errOut, tty, true, "installed", pkgName, adapter.Name(), note)
	return nil
}

// installMany installs multiple packages in a single native invocation.
// Requires -m so every package goes to the same manager; only managers with
// native multi-package support participate (see manager.BatchInstaller).
// brewBatchContext computes the cask map, detects mixed cask/formula batches,
// and returns the install context with cask flag set when all packages are casks.
func brewBatchContext(cmd *cobra.Command, adapter manager.Adapter, pkgs []string) (installCtx context.Context, mixed bool, casks map[string]bool) {
	casks = brewCasks(cmd.Context(), adapter, pkgs)
	caskCount := countCasks(casks)
	mixed = caskCount > 0 && caskCount < len(pkgs)
	installCtx = cmd.Context()
	if caskCount == len(pkgs) {
		installCtx = manager.WithCask(cmd.Context())
	}
	return
}

// execBatchInstall runs the native install via batch or per-package (mixed cask).
func execBatchInstall(ctx context.Context, adapter manager.Adapter, bi manager.BatchInstaller, mixed bool, casks map[string]bool, pkgs []string) error {
	if mixed {
		for _, p := range pkgs {
			pkgCtx := manager.WithYes(ctx)
			if casks[p] {
				pkgCtx = manager.WithYes(manager.WithCask(ctx))
			}
			if err := adapter.Install(pkgCtx, p); err != nil {
				return fmt.Errorf("install failed: %w", err)
			}
		}
		return nil
	}
	if err := bi.InstallMany(manager.WithYes(ctx), pkgs...); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}
	return nil
}

func installMany(cmd *cobra.Command, app *AppContext, pkgs []string, managerFlag, note string, groupInstall bool) error {
	if managerFlag == "" {
		return catErr(ErrUsage, "multiple packages require --manager")
	}
	if groupInstall {
		return catErr(ErrUsage, "--group supports a single package")
	}

	adapter, err := resolveAdapterByFlag(app.adapters, managerFlag)
	if err != nil {
		return err
	}
	if err := validateBatchPackages(adapter, pkgs); err != nil {
		return err
	}

	bi, ok := adapter.(manager.BatchInstaller)
	if !ok {
		return catErr(ErrUnavailable, "manager %s does not support installing multiple packages at once", adapter.Name())
	}

	installCtx, mixed, casks := brewBatchContext(cmd, adapter, pkgs)

	if err := confirmDestructiveMany(installCtx, cmd.ErrOrStderr(), cmd.InOrStdin(), app.yes,
		adapter, previewInstall, "Install", pkgs); err != nil {
		return handleConsent(err)
	}

	errOut := cmd.ErrOrStderr()
	tty := isOutputTerminal(errOut)
	target := fmt.Sprintf("%d package(s)", len(pkgs))
	printStatus(errOut, tty, false, "installing", target, adapter.Name(), "")

	if err := execBatchInstall(installCtx, adapter, bi, mixed, casks, pkgs); err != nil {
		return err
	}
	addTrackedAll(cmd.Context(), app, adapter, pkgs, note, cmd.ErrOrStderr())

	if err := app.saveManifest(); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}
	printStatus(errOut, tty, true, "installed", target, adapter.Name(), note)
	return nil
}

// countCasks returns how many packages in the map are casks.
func countCasks(casks map[string]bool) int {
	n := 0
	for _, isCask := range casks {
		if isCask {
			n++
		}
	}
	return n
}

// isValidGroupID reports whether s is a DNF group ID: lowercase letters,
// digits, '-' and '_' only.
func isValidGroupID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

// validateBatchPackages validates every package name against the chosen
// manager before any native command runs, so a bad name aborts the whole
// operation upfront.
func validateBatchPackages(adapter manager.Adapter, pkgs []string) error {
	for _, p := range pkgs {
		if err := manager.ValidatePackageForManager(adapter.Name(), p); err != nil {
			return err
		}
	}
	return nil
}

// trackedInstallName returns the manifest identity for a successful install.
// Adapters implementing InstalledNameResolver may report the native name the
// install actually added (e.g. dnf resolving "yarn" to "nodejs-corepack");
// every other adapter keeps the requested spec. Resolution failures warn and
// fall back to the spec: the system already changed, so the command still
// records intent.
func trackedInstallName(ctx context.Context, adapter manager.Adapter, spec string, errOut io.Writer) string {
	tracked := manager.TrackedName(adapter.Name(), spec)
	resolver, ok := adapter.(manager.InstalledNameResolver)
	if !ok {
		return tracked
	}
	name, err := resolver.ResolveInstalledName(ctx, spec)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "  warning: could not resolve installed name for %s: %v\n", spec, err)
		return tracked
	}
	if name == "" || name == spec {
		return tracked
	}
	_, _ = fmt.Fprintf(errOut, "  note: %s is provided by %s; tracking %s\n", spec, name, name)
	return name
}

// maintenanceName returns the native argument for remove/reinstall/update:
// manifest identity first, then adapter resolution, then lexical identity.
// Install keeps the raw spec; maintenance commands use the tracked name.
func maintenanceName(ctx context.Context, app *AppContext, adapter manager.Adapter, spec string) string {
	tracked := manager.TrackedName(adapter.Name(), spec)
	for _, p := range app.manifest.Packages {
		if p.Manager == adapter.Name() && (p.Name == spec || p.Name == tracked) {
			return p.Name
		}
	}
	if resolver, ok := adapter.(manager.InstalledNameResolver); ok {
		if resolved, err := resolver.ResolveInstalledName(ctx, spec); err == nil && resolved != "" {
			return resolved
		}
	}
	return tracked
}

// maintenanceNames maps every spec in a batch through maintenanceName.
func maintenanceNames(ctx context.Context, app *AppContext, adapter manager.Adapter, specs []string) []string {
	out := make([]string, len(specs))
	for i, spec := range specs {
		out[i] = maintenanceName(ctx, app, adapter, spec)
	}
	return out
}

// addTrackedAll records every installed package in the manifest. Called once
// after a batch install succeeds (saveManifest persists it).
func addTrackedAll(ctx context.Context, app *AppContext, adapter manager.Adapter, pkgs []string, note string, errOut io.Writer) {
	for _, p := range pkgs {
		name := trackedInstallName(ctx, adapter, p, errOut)
		app.manifest.AddPackage(manifest.Package{Name: name, Manager: adapter.Name(), Notes: note, Origin: manifest.OriginStamped})
	}
}

// resolveAdapterByFlag returns the adapter whose name matches -m, or an
// ErrUnavailable error. Used by the multi-package paths where a single manager
// must own the whole batch.
func resolveAdapterByFlag(adapters []manager.Adapter, managerFlag string) (manager.Adapter, error) {
	for _, a := range adapters {
		if a.Name() == manager.ResolveManager(managerFlag) {
			return a, nil
		}
	}
	return nil, catErr(ErrUnavailable, "manager %q is not available on this system", managerFlag)
}

// caskDetector reports whether a brew package is a cask. Only *manager.Brew
// implements it today; abstracting keeps brewCasks testable without the real
// adapter's unexported executor.
type caskDetector interface {
	IsCask(ctx context.Context, pkg string) (bool, error)
}

// brewCasks reports per-package cask status for cask-aware adapters. IsCask
// lookup failures are treated as non-cask (best-effort), matching the
// single-package install path.
func brewCasks(ctx context.Context, adapter manager.Adapter, pkgs []string) map[string]bool {
	detector, ok := adapter.(caskDetector)
	if !ok {
		return nil
	}
	m := make(map[string]bool, len(pkgs))
	for _, p := range pkgs {
		if strings.Contains(p, "/") {
			// Tap-qualified: keep to the formula path (see detectBrewCask).
			continue
		}
		if isCask, err := detector.IsCask(ctx, p); err == nil {
			m[p] = isCask
		}
	}
	return m
}

// detectBrewCask reports whether pkg is a brew cask on the given adapter. Only
// *manager.Brew implements cask detection, so a failed type assertion is a
// non-cask result (matching the previous Name()=="brew" guard).
func detectBrewCask(ctx context.Context, adapter manager.Adapter, pkg string) bool {
	if strings.Contains(pkg, "/") {
		// Tap-qualified: keep to the formula path and skip `brew info --cask`,
		// which could otherwise tap/load the untrusted tap as a side effect.
		return false
	}
	d, ok := adapter.(caskDetector)
	if !ok {
		return false
	}
	isCask, err := d.IsCask(ctx, pkg)
	return err == nil && isCask
}

// validateGroupSupport rejects --group for managers that do not support
// package groups (only dnf and yum do). The error text preserves the existing
// message even though yum is accepted.
func validateGroupSupport(adapter manager.Adapter, group bool) error {
	if !group {
		return nil
	}
	if n := adapter.Name(); n != "dnf" && n != "yum" {
		return fmt.Errorf("--group is only supported for dnf")
	}
	return nil
}

// applyOpFlags stacks the --cask and --group context markers onto ctx.
func applyOpFlags(ctx context.Context, cask, group bool) context.Context {
	if cask {
		ctx = manager.WithCask(ctx)
	}
	if group {
		ctx = manager.WithGroup(ctx)
	}
	return ctx
}

func newRemoveCmd() *cobra.Command {
	var managerFlag string
	var groupRemove bool

	cmd := &cobra.Command{
		Use:     "remove <package>",
		Aliases: []string{"uninstall", "rm", "delete", "del"},
		Short:   "Remove a package and untrack it",
		Example: `  # remove using the manager recorded in the manifest
  stamp remove htop

  # specify a manager explicitly
  stamp remove lazygit -m brew

  # remove a tap-qualified Homebrew formula
  stamp remove nklmilojevic/sofka/sofka

  # remove a DNF package group (by group ID)
  stamp remove development-tools -m dnf --group

  # all these aliases behave the same way
  stamp uninstall htop
  stamp rm htop
  stamp delete htop
  stamp del htop

  # remove multiple packages in one command (per-manager batch, -m required)
  stamp remove htop atop btop -m dnf`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
			}

			if len(args) > 1 {
				return removeMany(cmd, app, args, managerFlag, groupRemove)
			}
			pkgName := args[0]

			adapter, isCask, canonical, err := resolveRemoveTarget(cmd.Context(), app, pkgName, managerFlag)
			if err != nil {
				return err
			}

			return runSingleRemove(cmd, app, adapter, pkgName, canonical, isCask, groupRemove)
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use")
	cmd.Flags().BoolVarP(&groupRemove, "group", "g", false, "remove a DNF package group (by group ID)")
	return cmd
}

// findTrackedAdapter looks up a tracked package in the manifest and returns the
// matching adapter plus its recorded cask flag. found is false when the package
// is not tracked.
func findTrackedAdapter(pkgs []manifest.Package, adapters []manager.Adapter, name string) (manager.Adapter, bool, bool) {
	for _, p := range pkgs {
		tracked := p.Name == name || p.Name == manager.TrackedName(p.Manager, name)
		if !tracked {
			continue
		}
		for _, a := range adapters {
			if a.Name() == p.Manager {
				return a, p.Cask, true
			}
		}
	}
	return nil, false, false
}

// resolveRemoveTarget selects the adapter for a single-package remove: the
// manifest-recorded manager when available, otherwise the -m manager, otherwise
// the first adapter, otherwise ErrUnavailable. The -m path reports "unknown
// manager %q" (distinct from resolveAdapterByFlag's message).
// findTrackedResolved tries adapters that can resolve an installed name (e.g.
// dnf NEVRA/provides specs) and matches the result against the manifest.
func findTrackedResolved(ctx context.Context, pkgs []manifest.Package, adapters []manager.Adapter, name string) (manager.Adapter, bool, string, bool) {
	for _, a := range adapters {
		resolver, ok := a.(manager.InstalledNameResolver)
		if !ok {
			continue
		}
		resolved, err := resolver.ResolveInstalledName(ctx, name)
		if err != nil {
			continue
		}
		if adapter, cask, found := findTrackedAdapter(pkgs, []manager.Adapter{a}, resolved); found {
			return adapter, cask, resolved, true
		}
	}
	return nil, false, "", false
}

func resolveRemoveTarget(ctx context.Context, app *AppContext, pkgName, managerFlag string) (manager.Adapter, bool, string, error) {
	if managerFlag == "" {
		if a, cask, found := findTrackedAdapter(app.manifest.Packages, app.adapters, pkgName); found {
			return a, cask, "", nil
		}
		if a, cask, canonical, found := findTrackedResolved(ctx, app.manifest.Packages, app.adapters, pkgName); found {
			return a, cask, canonical, nil
		}
		if isTapQualifiedRef(pkgName) {
			if a := brewAdapter(app.adapters); a != nil {
				return a, false, "", nil
			}
		}
		if len(app.adapters) > 0 {
			return app.adapters[0], false, "", nil
		}
		return nil, false, "", catErr(ErrUnavailable, "no package managers available")
	}

	want := manager.ResolveManager(managerFlag)
	for _, a := range app.adapters {
		if a.Name() == want {
			return a, false, "", nil
		}
	}
	return nil, false, "", fmt.Errorf("unknown manager %q", managerFlag)
}

// runSingleRemove performs one package removal: group validation, context flag
// stacking, confirmation gate, native remove, untrack, save, and the completion
// status line.
func runSingleRemove(cmd *cobra.Command, app *AppContext, adapter manager.Adapter, pkgName, canonical string, isCask, group bool) error {
	if err := validateGroupSupport(adapter, group); err != nil {
		return err
	}

	removeCtx := applyOpFlags(cmd.Context(), isCask && adapter.Name() == "brew", group)
	removeName := canonical
	if removeName == "" {
		removeName = pkgName
		if !group {
			removeName = maintenanceName(cmd.Context(), app, adapter, pkgName)
		}
	}

	if err := confirmDestructive(removeCtx, cmd.ErrOrStderr(), cmd.InOrStdin(), app.yes,
		adapter, previewRemove, "Remove", removeName); err != nil {
		return handleConsent(err)
	}

	// In -y mode the gate skipped the preview; pre-check so an absent package
	// is not falsely reported as removed and untracked.
	if app.yes {
		pv, pErr := previewOutput(removeCtx, adapter, previewRemove, removeName)
		if pErr == nil && pv.Noop {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  nothing to do: %s via %s\n", pkgName, adapter.Name())
			return nil
		}
	}

	errOut := cmd.ErrOrStderr()
	tty := isOutputTerminal(errOut)
	printStatus(errOut, tty, false, "removing", pkgName, adapter.Name(), "")

	if err := adapter.Remove(manager.WithYes(removeCtx), removeName); err != nil {
		return fmt.Errorf("remove failed: %w", err)
	}

	app.manifest.RemovePackage(removeName, adapter.Name())
	if err := app.saveManifest(); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	printStatus(errOut, tty, true, "removed", pkgName, adapter.Name(), "")
	return nil
}

// removeMany removes multiple packages in a single native invocation. Requires
// -m so every package is removed from the same manager; only managers with
// native multi-package support participate (see manager.BatchRemover).
func removeMany(cmd *cobra.Command, app *AppContext, pkgs []string, managerFlag string, groupRemove bool) error {
	if managerFlag == "" {
		return catErr(ErrUsage, "multiple packages require --manager")
	}
	if groupRemove {
		return catErr(ErrUsage, "--group supports a single package")
	}

	adapter, err := resolveAdapterByFlag(app.adapters, managerFlag)
	if err != nil {
		return err
	}
	if err := validateBatchPackages(adapter, pkgs); err != nil {
		return err
	}
	pkgs = maintenanceNames(cmd.Context(), app, adapter, pkgs)

	br, ok := adapter.(manager.BatchRemover)
	if !ok {
		return catErr(ErrUnavailable, "manager %s does not support removing multiple packages at once", adapter.Name())
	}

	// In -y mode the gate skips the per-package preview; drop not-installed
	// packages so they are not falsely reported as removed and untracked.
	if app.yes {
		if pkgs = filterAbsentPkgs(cmd.Context(), cmd.ErrOrStderr(), adapter, pkgs); pkgs == nil {
			return nil
		}
	}

	// Brew: --cask is batch-wide; a mixed cask/formula batch falls back to
	// per-package single removals.
	casks := brewCasks(cmd.Context(), adapter, pkgs)
	caskCount := countCasks(casks)
	mixed := caskCount > 0 && caskCount < len(pkgs)

	removeCtx := cmd.Context()
	if caskCount == len(pkgs) {
		removeCtx = manager.WithCask(cmd.Context())
	}

	if err := confirmDestructiveMany(removeCtx, cmd.ErrOrStderr(), cmd.InOrStdin(), app.yes,
		adapter, previewRemove, "Remove", pkgs); err != nil {
		return handleConsent(err)
	}

	errOut := cmd.ErrOrStderr()
	tty := isOutputTerminal(errOut)
	target := fmt.Sprintf("%d package(s)", len(pkgs))
	printStatus(errOut, tty, false, "removing", target, adapter.Name(), "")

	if err := execBatchRemove(cmd.Context(), removeCtx, adapter, br, pkgs, mixed, casks); err != nil {
		return err
	}
	removeTrackedAll(app, pkgs, adapter.Name())

	if err := app.saveManifest(); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}
	printStatus(errOut, tty, true, "removed", target, adapter.Name(), "")
	return nil
}

// removeTrackedAll untracks every removed package from the manifest. Called
// once after a batch remove succeeds (saveManifest persists it).
func removeTrackedAll(app *AppContext, pkgs []string, mgr string) {
	for _, p := range pkgs {
		app.manifest.RemovePackage(manager.TrackedName(mgr, p), mgr)
	}
}

// execBatchRemove runs the native remove operation for a batch. A mixed
// cask/formula batch falls back to per-package single removals; a uniform
// batch uses the native batch invocation.
func execBatchRemove(ctx context.Context, removeCtx context.Context, adapter manager.Adapter, br manager.BatchRemover, pkgs []string, mixed bool, casks map[string]bool) error {
	if mixed {
		for _, p := range pkgs {
			pkgCtx := manager.WithYes(ctx)
			if casks[p] {
				pkgCtx = manager.WithYes(manager.WithCask(ctx))
			}
			if err := adapter.Remove(pkgCtx, p); err != nil {
				return fmt.Errorf("remove failed: %w", err)
			}
		}
		return nil
	}
	if err := br.RemoveMany(manager.WithYes(removeCtx), pkgs...); err != nil {
		return fmt.Errorf("remove failed: %w", err)
	}
	return nil
}

// filterNoopRemoves drops packages whose removal would be a no-op (the package
// is not installed) from pkgs. Returns the remaining list and the skipped
// names. Adapters without a Previewer are left untouched. Used in -y mode,
// where the consent gate skips the per-package preview.
func filterNoopRemoves(ctx context.Context, adapter manager.Adapter, pkgs []string) (remaining, skipped []string) {
	p, ok := adapter.(manager.Previewer)
	if !ok {
		return pkgs, nil
	}
	for _, name := range pkgs {
		pv, err := p.PreviewRemove(ctx, name)
		if err == nil && pv.Noop {
			skipped = append(skipped, name)
			continue
		}
		remaining = append(remaining, name)
	}
	return remaining, skipped
}

// filterAbsentPkgs is the -y-mode wrapper around filterNoopRemoves that warns
// about skipped packages and handles early return. It returns nil when every
// package in the batch is a no-op (caller must check for nil).
func filterAbsentPkgs(ctx context.Context, w io.Writer, adapter manager.Adapter, pkgs []string) []string {
	_, caskAware := adapter.(caskDetector)
	if caskAware {
		return pkgs
	}
	remaining, skipped := filterNoopRemoves(ctx, adapter, pkgs)
	for _, name := range skipped {
		_, _ = fmt.Fprintf(w, "  nothing to do: %s via %s\n", name, adapter.Name())
	}
	if len(remaining) == 0 {
		return nil
	}
	return remaining
}

func newSearchCmd() *cobra.Command {
	var managerFlag string
	var groupSearch bool

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for packages across managers",
		Example: `  # search across all available managers
  stamp search htop

  # limit search to a specific manager
  stamp search lazygit -m brew

  # a tap-qualified query searches the brew tap
  stamp search nklmilojevic/sofka/sofka

  # search DNF package groups instead of individual packages
  stamp search Development -m dnf --group`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			query := args[0]

			targets, err := selectSearchTargets(app.adapters, managerFlag)
			if err != nil {
				return err
			}
			if managerFlag == "" && isTapQualifiedRef(query) {
				a := brewAdapter(app.adapters)
				if a == nil {
					return catErr(ErrUnavailable, "tap-qualified search %q requires brew", query)
				}
				targets = []manager.Adapter{a}
			}
			if err := validateGroupSearch(targets, managerFlag, groupSearch); err != nil {
				return err
			}

			results := searchManagers(cmd.Context(), targets, query, cmd.ErrOrStderr(), groupSearch)
			if len(results) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no results found")
				return nil
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), strings.Join(results, "\n"))
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to search")
	cmd.Flags().BoolVarP(&groupSearch, "group", "g", false, "search DNF package groups")
	return cmd
}

// selectSearchTargets returns the adapters to search: the -m manager when
// given, otherwise all adapters.
func selectSearchTargets(adapters []manager.Adapter, managerFlag string) ([]manager.Adapter, error) {
	if managerFlag == "" {
		return adapters, nil
	}
	want := manager.ResolveManager(managerFlag)
	for _, a := range adapters {
		if a.Name() == want {
			return []manager.Adapter{a}, nil
		}
	}
	return nil, fmt.Errorf("unknown manager %q", managerFlag)
}

// validateGroupSearch enforces the --group constraints: it requires --manager
// and only supports dnf/yum.
func validateGroupSearch(targets []manager.Adapter, managerFlag string, group bool) error {
	if group && managerFlag == "" {
		return fmt.Errorf("--group requires --manager <name>")
	}
	if group {
		for _, a := range targets {
			if a.Name() != "dnf" && a.Name() != "yum" {
				return fmt.Errorf("--group is only supported for dnf")
			}
		}
	}
	return nil
}

// searchManagers queries each adapter and returns the rendered results, warning
// to warnOut on per-adapter search failure without aborting the batch.
func searchManagers(ctx context.Context, targets []manager.Adapter, query string, warnOut io.Writer, group bool) []string {
	var results []string
	for _, a := range targets {
		searchCtx := ctx
		if group {
			searchCtx = manager.WithGroup(searchCtx)
		}
		pkgs, err := a.Search(searchCtx, query)
		if err != nil {
			_, _ = fmt.Fprintf(warnOut, "warning: %s search: %v\n", a.Name(), err)
			continue
		}
		for _, p := range pkgs {
			results = append(results, fmt.Sprintf("%s (%s)", p, a.Name()))
		}
	}
	return results
}
