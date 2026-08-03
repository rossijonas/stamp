package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
)

func newHoldCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:   "hold <package>",
		Short: "Pin a package at its current version to prevent upgrades",
		Example: `  # hold with apt (via apt-mark)
  stamp hold nginx -m apt

  # hold with dnf (via dnf versionlock)
  stamp hold nginx -m dnf

  # hold on arch with pacman (adds to IgnorePkg in pacman.conf)
  stamp hold nginx -m pacman`,
		Long: `Pin a package at its current version to prevent accidental upgrades.

Scoped to a single manager with the --manager flag.
Supported managers: apt (apt-mark), dnf (dnf versionlock), pacman/paru (IgnorePkg).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			pkgName := args[0]

			targets := app.adapters
			if managerFlag != "" {
				resolved := manager.ResolveManager(managerFlag)
				var found bool
				for _, a := range targets {
					if a.Name() == resolved {
						targets = []manager.Adapter{a}
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("manager %q not available", managerFlag)
				}
			}

			for _, a := range targets {
				if !requireConsent(cmd, fmt.Sprintf("Hold %s via %s", pkgName, a.Name())) {
					return nil
				}
				err := a.Hold(manager.WithYes(cmd.Context()), pkgName)
				if err != nil {
					if errors.Is(err, manager.ErrNotSupported) {
						continue
					}
					return fmt.Errorf("failed to hold %s via %s: %w", pkgName, a.Name(), err)
				}
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "held %s via %s\n", pkgName, a.Name())
				return nil
			}

			return fmt.Errorf("no manager supports hold for package %s", pkgName)
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use")
	return cmd
}

func newUnholdCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:   "unhold <package>",
		Short: "Remove a version pin, allowing upgrades",
		Example: `  # unhold with apt (via apt-mark)
  stamp unhold nginx -m apt

  # unhold on arch with pacman (removes from IgnorePkg)
  stamp unhold nginx -m pacman`,
		Long: `Remove a version pin from a package, allowing it to be upgraded again.

Scoped to a single manager with the --manager flag.
Supported managers: apt (apt-mark), dnf (dnf versionlock), pacman/paru (IgnorePkg).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			pkgName := args[0]

			targets := app.adapters
			if managerFlag != "" {
				resolved := manager.ResolveManager(managerFlag)
				var found bool
				for _, a := range targets {
					if a.Name() == resolved {
						targets = []manager.Adapter{a}
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("manager %q not available", managerFlag)
				}
			}

			for _, a := range targets {
				if !requireConsent(cmd, fmt.Sprintf("Unhold %s via %s", pkgName, a.Name())) {
					return nil
				}
				err := a.Unhold(manager.WithYes(cmd.Context()), pkgName)
				if err != nil {
					if errors.Is(err, manager.ErrNotSupported) {
						continue
					}
					return fmt.Errorf("failed to unhold %s via %s: %w", pkgName, a.Name(), err)
				}
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "unheld %s via %s\n", pkgName, a.Name())
				return nil
			}

			return fmt.Errorf("no manager supports unhold for package %s", pkgName)
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use")
	return cmd
}

func newHeldCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:     "held",
		Short:   "List all held/pinned packages",
		Example: "  stamp held\n  stamp held -m apt",
		Long: `List all packages currently held/pinned across all managers.
Use --manager to scope to a single package manager.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)

			targets := app.adapters
			if managerFlag != "" {
				resolved := manager.ResolveManager(managerFlag)
				var found bool
				for _, a := range targets {
					if a.Name() == resolved {
						targets = []manager.Adapter{a}
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("manager %q not available", managerFlag)
				}
			}

			var results []string
			for _, a := range targets {
				pkgs, err := a.ListHeld(cmd.Context())
				if err != nil {
					if errors.Is(err, manager.ErrNotSupported) {
						continue
					}
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s held failed: %v\n", a.Name(), err)
					continue
				}
				if len(pkgs) == 0 {
					continue
				}
				for _, p := range pkgs {
					results = append(results, fmt.Sprintf("%s (%s)", p, a.Name()))
				}
			}

			if len(results) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no packages held")
				return nil
			}

			for _, r := range results {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), r)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to query")
	return cmd
}
