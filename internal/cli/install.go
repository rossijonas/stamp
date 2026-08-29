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

  # install multiple packages in one command (per-manager batch, -m required)
  stamp install htop atop btop -m dnf

  # install a DNF package group (by group ID, see 'dnf group list')
  stamp install development-tools -m dnf --group

  # add a note so you remember why later
  stamp add lazygit -m brew --note "better git TUI"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
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

			r := NewResolver(app.adapters, app.config)
			adapter, err := r.Resolve(pkgName, managerFlag)
			if err != nil {
				return fmt.Errorf("cannot resolve package manager: %w", err)
			}

			if err := manager.ValidatePackageForManager(adapter.Name(), pkgName); err != nil {
				if groupInstall {
					return catErr(ErrUsage, "group IDs contain only a-z0-9_- (see 'dnf group list')")
				}
				return fmt.Errorf("invalid package name: %w", err)
			}

			cask := detectBrewCask(cmd.Context(), adapter, pkgName)
			return runSingleInstall(cmd, app, adapter, pkgName, note, cask, groupInstall)
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use")
	cmd.Flags().StringVarP(&note, "note", "n", "", "annotation for this package")
	cmd.Flags().BoolVarP(&groupInstall, "group", "g", false, "install a DNF package group (by group ID)")
	return cmd
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

	app.manifest.AddPackage(manifest.Package{
		Name:    pkgName,
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

	// Brew: --cask is batch-wide, so a mixed cask/formula batch falls back to
	// per-package single installs. A uniform batch uses one command.
	casks := brewCasks(cmd.Context(), adapter, pkgs)
	caskCount := countCasks(casks)
	mixed := caskCount > 0 && caskCount < len(pkgs)

	installCtx := cmd.Context()
	if caskCount == len(pkgs) {
		installCtx = manager.WithCask(cmd.Context())
	}

	if err := confirmDestructiveMany(installCtx, cmd.ErrOrStderr(), cmd.InOrStdin(), app.yes,
		adapter, previewInstall, "Install", pkgs); err != nil {
		return handleConsent(err)
	}

	errOut := cmd.ErrOrStderr()
	tty := isOutputTerminal(errOut)
	target := fmt.Sprintf("%d package(s)", len(pkgs))
	printStatus(errOut, tty, false, "installing", target, adapter.Name(), "")

	if mixed {
		for _, p := range pkgs {
			ctx := manager.WithYes(cmd.Context())
			if casks[p] {
				ctx = manager.WithYes(manager.WithCask(cmd.Context()))
			}
			if err := adapter.Install(ctx, p); err != nil {
				return fmt.Errorf("install failed: %w", err)
			}
		}
	} else {
		if err := bi.InstallMany(manager.WithYes(installCtx), pkgs...); err != nil {
			return fmt.Errorf("install failed: %w", err)
		}
	}
	addTrackedAll(app, pkgs, adapter.Name(), note)

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

// validateBatchPackages validates every package name against the chosen
// manager before any native command runs, so a bad name aborts the whole
// operation upfront.
func validateBatchPackages(adapter manager.Adapter, pkgs []string) error {
	for _, p := range pkgs {
		if err := manager.ValidatePackageForManager(adapter.Name(), p); err != nil {
			return fmt.Errorf("invalid package name %q: %w", p, err)
		}
	}
	return nil
}

// addTrackedAll records every installed package in the manifest. Called once
// after a batch install succeeds (saveManifest persists it).
func addTrackedAll(app *AppContext, pkgs []string, mgr, note string) {
	for _, p := range pkgs {
		app.manifest.AddPackage(manifest.Package{Name: p, Manager: mgr, Notes: note, Origin: manifest.OriginStamped})
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

			adapter, isCask, err := resolveRemoveTarget(app, pkgName, managerFlag)
			if err != nil {
				return err
			}

			return runSingleRemove(cmd, app, adapter, pkgName, isCask, groupRemove)
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
		if p.Name != name {
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
func resolveRemoveTarget(app *AppContext, pkgName, managerFlag string) (manager.Adapter, bool, error) {
	if managerFlag == "" {
		if a, cask, found := findTrackedAdapter(app.manifest.Packages, app.adapters, pkgName); found {
			return a, cask, nil
		}
		if len(app.adapters) > 0 {
			return app.adapters[0], false, nil
		}
		return nil, false, catErr(ErrUnavailable, "no package managers available")
	}

	want := manager.ResolveManager(managerFlag)
	for _, a := range app.adapters {
		if a.Name() == want {
			return a, false, nil
		}
	}
	return nil, false, fmt.Errorf("unknown manager %q", managerFlag)
}

// runSingleRemove performs one package removal: group validation, context flag
// stacking, confirmation gate, native remove, untrack, save, and the completion
// status line.
func runSingleRemove(cmd *cobra.Command, app *AppContext, adapter manager.Adapter, pkgName string, isCask, group bool) error {
	if err := validateGroupSupport(adapter, group); err != nil {
		return err
	}

	removeCtx := applyOpFlags(cmd.Context(), isCask && adapter.Name() == "brew", group)

	if err := confirmDestructive(removeCtx, cmd.ErrOrStderr(), cmd.InOrStdin(), app.yes,
		adapter, previewRemove, "Remove", pkgName); err != nil {
		return handleConsent(err)
	}

	// In -y mode the gate skipped the preview; pre-check so an absent package
	// is not falsely reported as removed and untracked.
	if app.yes {
		pv, pErr := previewOutput(removeCtx, adapter, previewRemove, pkgName)
		if pErr == nil && pv.Noop {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  nothing to do: %s via %s\n", pkgName, adapter.Name())
			return nil
		}
	}

	errOut := cmd.ErrOrStderr()
	tty := isOutputTerminal(errOut)
	printStatus(errOut, tty, false, "removing", pkgName, adapter.Name(), "")

	if err := adapter.Remove(manager.WithYes(removeCtx), pkgName); err != nil {
		return fmt.Errorf("remove failed: %w", err)
	}

	app.manifest.RemovePackage(pkgName, adapter.Name())
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
		app.manifest.RemovePackage(p, mgr)
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
