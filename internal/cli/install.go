package cli

import (
	"context"
	"fmt"
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

  # install a DNF package group (name may contain spaces)
  stamp install "Development Tools" -m dnf --group

  # add a note so you remember why later
  stamp add lazygit -m brew --note "better git TUI"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
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
				return fmt.Errorf("invalid package name: %w", err)
			}

			// Detect cask for brew packages before install so --cask is passed
			cask := false
			if adapter.Name() == "brew" {
				if brewAdapter, ok := adapter.(*manager.Brew); ok {
					if isCask, err := brewAdapter.IsCask(cmd.Context(), pkgName); err == nil && isCask {
						cask = true
					}
				}
			}

			if groupInstall && adapter.Name() != "dnf" && adapter.Name() != "yum" {
				return fmt.Errorf("--group is only supported for dnf")
			}

			installCtx := cmd.Context()
			if cask {
				installCtx = manager.WithCask(cmd.Context())
			}
			if groupInstall {
				installCtx = manager.WithGroup(installCtx)
			}

			// Confirmation gate: prompts unless -y is passed. Non-interactive
			// runs without -y abort (fail closed, non-zero exit).
			if err := confirmDestructive(installCtx, cmd.ErrOrStderr(), cmd.InOrStdin(), app.yes,
				adapter, previewInstall, "Install", pkgName); err != nil {
				return handleConsent(err)
			}

			errOut := cmd.ErrOrStderr()
			tty := isOutputTerminal(errOut)
			if line := statusLine(tty, false, "installing", pkgName, adapter.Name(), ""); line != "" {
				_, _ = fmt.Fprintln(errOut, line)
			}

			if err := adapter.Install(manager.WithYes(installCtx), pkgName); err != nil {
				return fmt.Errorf("install failed: %w", err)
			}

			app.manifest.AddPackage(manifest.Package{
				Name:    pkgName,
				Manager: adapter.Name(),
				Notes:   note,
				Cask:    cask,
				Group:   groupInstall,
				Origin:  manifest.OriginStamped,
			})

			if err := app.saveManifest(); err != nil {
				return fmt.Errorf("failed to save manifest: %w", err)
			}

			if line := statusLine(tty, true, "installed", pkgName, adapter.Name(), note); line != "" {
				_, _ = fmt.Fprintln(errOut, line)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use")
	cmd.Flags().StringVarP(&note, "note", "n", "", "annotation for this package")
	cmd.Flags().BoolVarP(&groupInstall, "group", "g", false, "install a DNF package group")
	return cmd
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
	for _, p := range pkgs {
		if err := manager.ValidatePackageForManager(adapter.Name(), p); err != nil {
			return fmt.Errorf("invalid package name %q: %w", p, err)
		}
	}

	bi, ok := adapter.(manager.BatchInstaller)
	if !ok {
		return catErr(ErrUnavailable, "manager %s does not support installing multiple packages at once", adapter.Name())
	}

	// Brew: --cask is batch-wide, so a mixed cask/formula batch falls back to
	// per-package single installs. A uniform batch uses one command.
	casks := brewCasks(cmd.Context(), adapter, pkgs)
	caskCount := 0
	for _, isCask := range casks {
		if isCask {
			caskCount++
		}
	}
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
	if line := statusLine(tty, false, "installing", target, adapter.Name(), ""); line != "" {
		_, _ = fmt.Fprintln(errOut, line)
	}

	if mixed {
		for _, p := range pkgs {
			ctx := manager.WithYes(cmd.Context())
			if casks[p] {
				ctx = manager.WithYes(manager.WithCask(cmd.Context()))
			}
			if err := adapter.Install(ctx, p); err != nil {
				return fmt.Errorf("install failed: %w", err)
			}
			app.manifest.AddPackage(manifest.Package{Name: p, Manager: adapter.Name(), Notes: note, Origin: manifest.OriginStamped})
		}
	} else {
		if err := bi.InstallMany(manager.WithYes(installCtx), pkgs...); err != nil {
			return fmt.Errorf("install failed: %w", err)
		}
		for _, p := range pkgs {
			app.manifest.AddPackage(manifest.Package{Name: p, Manager: adapter.Name(), Notes: note, Origin: manifest.OriginStamped})
		}
	}

	if err := app.saveManifest(); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}
	if line := statusLine(tty, true, "installed", target, adapter.Name(), note); line != "" {
		_, _ = fmt.Fprintln(errOut, line)
	}
	return nil
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

  # remove a DNF package group
  stamp remove "Development Tools" -m dnf --group

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

			var adapter manager.Adapter
			var isCask bool

			// Check manifest first: if package is tracked, use its recorded manager
			if managerFlag == "" {
				for _, p := range app.manifest.Packages {
					if p.Name == pkgName {
						for _, a := range app.adapters {
							if a.Name() == p.Manager {
								adapter = a
								isCask = p.Cask
								break
							}
						}
						break
					}
				}
			}

			// Fall back to explicit flag or first available adapter
			if adapter == nil {
				switch {
				case managerFlag != "":
					for _, a := range app.adapters {
						if a.Name() == manager.ResolveManager(managerFlag) {
							adapter = a
							break
						}
					}
					if adapter == nil {
						return fmt.Errorf("unknown manager %q", managerFlag)
					}
				case len(app.adapters) > 0:
					adapter = app.adapters[0]
				default:
					return catErr(ErrUnavailable, "no package managers available")
				}
			}

			removeCtx := cmd.Context()
			if isCask && adapter.Name() == "brew" {
				removeCtx = manager.WithCask(cmd.Context())
			}
			if groupRemove {
				if adapter.Name() != "dnf" && adapter.Name() != "yum" {
					return fmt.Errorf("--group is only supported for dnf")
				}
				removeCtx = manager.WithGroup(removeCtx)
			}

			// Confirmation gate: prompts unless -y is passed. Non-interactive
			// runs without -y abort (fail closed, non-zero exit).
			if err := confirmDestructive(removeCtx, cmd.ErrOrStderr(), cmd.InOrStdin(), app.yes,
				adapter, previewRemove, "Remove", pkgName); err != nil {
				return handleConsent(err)
			}

			errOut := cmd.ErrOrStderr()
			tty := isOutputTerminal(errOut)
			if line := statusLine(tty, false, "removing", pkgName, adapter.Name(), ""); line != "" {
				_, _ = fmt.Fprintln(errOut, line)
			}

			if err := adapter.Remove(manager.WithYes(removeCtx), pkgName); err != nil {
				return fmt.Errorf("remove failed: %w", err)
			}

			app.manifest.RemovePackage(pkgName, adapter.Name())
			if err := app.saveManifest(); err != nil {
				return fmt.Errorf("failed to save manifest: %w", err)
			}

			if line := statusLine(tty, true, "removed", pkgName, adapter.Name(), ""); line != "" {
				_, _ = fmt.Fprintln(errOut, line)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use")
	cmd.Flags().BoolVarP(&groupRemove, "group", "g", false, "remove a DNF package group")
	return cmd
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
	for _, p := range pkgs {
		if err := manager.ValidatePackageForManager(adapter.Name(), p); err != nil {
			return fmt.Errorf("invalid package name %q: %w", p, err)
		}
	}

	br, ok := adapter.(manager.BatchRemover)
	if !ok {
		return catErr(ErrUnavailable, "manager %s does not support removing multiple packages at once", adapter.Name())
	}

	// Brew: --cask is batch-wide; a mixed cask/formula batch falls back to
	// per-package single removals.
	casks := brewCasks(cmd.Context(), adapter, pkgs)
	caskCount := 0
	for _, isCask := range casks {
		if isCask {
			caskCount++
		}
	}
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
	if line := statusLine(tty, false, "removing", target, adapter.Name(), ""); line != "" {
		_, _ = fmt.Fprintln(errOut, line)
	}

	if mixed {
		for _, p := range pkgs {
			ctx := manager.WithYes(cmd.Context())
			if casks[p] {
				ctx = manager.WithYes(manager.WithCask(cmd.Context()))
			}
			if err := adapter.Remove(ctx, p); err != nil {
				return fmt.Errorf("remove failed: %w", err)
			}
			app.manifest.RemovePackage(p, adapter.Name())
		}
	} else {
		if err := br.RemoveMany(manager.WithYes(removeCtx), pkgs...); err != nil {
			return fmt.Errorf("remove failed: %w", err)
		}
		for _, p := range pkgs {
			app.manifest.RemovePackage(p, adapter.Name())
		}
	}

	if err := app.saveManifest(); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}
	if line := statusLine(tty, true, "removed", target, adapter.Name(), ""); line != "" {
		_, _ = fmt.Fprintln(errOut, line)
	}
	return nil
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

			targets := app.adapters
			if managerFlag != "" {
				var found bool
				for _, a := range app.adapters {
					if a.Name() == manager.ResolveManager(managerFlag) {
						targets = []manager.Adapter{a}
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("unknown manager %q", managerFlag)
				}
			}

			if groupSearch && managerFlag == "" {
				return fmt.Errorf("--group requires --manager <name>")
			}
			if groupSearch {
				for _, a := range targets {
					if a.Name() != "dnf" && a.Name() != "yum" {
						return fmt.Errorf("--group is only supported for dnf")
					}
				}
			}

			var results []string
			for _, a := range targets {
				searchCtx := cmd.Context()
				if groupSearch {
					searchCtx = manager.WithGroup(searchCtx)
				}
				pkgs, err := a.Search(searchCtx, query)
				if err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s search: %v\n", a.Name(), err)
					continue
				}
				for _, p := range pkgs {
					results = append(results, fmt.Sprintf("%s (%s)", p, a.Name()))
				}
			}

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
