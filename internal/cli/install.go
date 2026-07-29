package cli

import (
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
		Example: "  stamp install htop\n  stamp install spotify --manager flatpak\n  stamp add lazygit -m brew --note \"better git TUI\"",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
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

			if err := adapter.Install(installCtx, pkgName); err != nil {
				return fmt.Errorf("install failed: %w", err)
			}

			app.manifest.AddPackage(manifest.Package{
				Name:    pkgName,
				Manager: adapter.Name(),
				Notes:   note,
				Cask:    cask,
				Group:   groupInstall,
			})

			if err := app.saveManifest(); err != nil {
				return fmt.Errorf("failed to save manifest: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "installed %s via %s\n", pkgName, adapter.Name())
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use")
	cmd.Flags().StringVarP(&note, "note", "n", "", "annotation for this package")
	cmd.Flags().BoolVarP(&groupInstall, "group", "g", false, "install a DNF package group")
	return cmd
}

func newRemoveCmd() *cobra.Command {
	var managerFlag string
	var groupRemove bool

	cmd := &cobra.Command{
		Use:     "remove <package>",
		Aliases: []string{"uninstall", "rm", "delete", "del"},
		Short:   "Remove a package and untrack it",
		Example: "  stamp remove htop\n  stamp remove -m brew lazygit\n  stamp uninstall htop\n  stamp rm htop\n  stamp delete htop",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
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
					return fmt.Errorf("no package managers available")
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

			if err := adapter.Remove(removeCtx, pkgName); err != nil {
				return fmt.Errorf("remove failed: %w", err)
			}

			app.manifest.RemovePackage(pkgName, adapter.Name())
			if err := app.saveManifest(); err != nil {
				return fmt.Errorf("failed to save manifest: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "removed %s via %s\n", pkgName, adapter.Name())
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use")
	cmd.Flags().BoolVarP(&groupRemove, "group", "g", false, "remove a DNF package group")
	return cmd
}

func newSearchCmd() *cobra.Command {
	var managerFlag string
	var groupSearch bool

	cmd := &cobra.Command{
		Use:     "search <query>",
		Short:   "Search for packages across managers",
		Example: "  stamp search htop\n  stamp search lazygit -m brew\n  stamp search ripgrep",
		Args:    cobra.ExactArgs(1),
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
