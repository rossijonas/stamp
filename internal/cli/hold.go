package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

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

			targets, err := resolveTargets(app.adapters, managerFlag)
			if err != nil {
				return err
			}

			return applyFirstSupported(cmd, args[0], targets, "Hold", "held", func(a manager.Adapter) error {
				return a.Hold(manager.WithYes(cmd.Context()), args[0])
			})
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

			targets, err := resolveTargets(app.adapters, managerFlag)
			if err != nil {
				return err
			}

			return applyFirstSupported(cmd, args[0], targets, "Unhold", "unheld", func(a manager.Adapter) error {
				return a.Unhold(manager.WithYes(cmd.Context()), args[0])
			})
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use")
	return cmd
}

func newHeldCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:   "held",
		Short: "List all held/pinned packages",
		Example: `  # list held/pinned packages across all managers
  stamp held

  # scope to a single package manager
  stamp held -m apt`,
		Long: `List all packages currently held/pinned across all managers.
Use --manager to scope to a single package manager.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)

			targets, err := resolveTargets(app.adapters, managerFlag)
			if err != nil {
				return err
			}

			printHeld(cmd.OutOrStdout(), collectHeld(cmd, targets))
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to query")
	return cmd
}

// resolveTargets scopes adapters to the single manager selected by the
// --manager flag, or returns them unchanged when the flag is empty.
func resolveTargets(adapters []manager.Adapter, managerFlag string) ([]manager.Adapter, error) {
	if managerFlag == "" {
		return adapters, nil
	}
	resolved := manager.ResolveManager(managerFlag)
	for _, a := range adapters {
		if a.Name() == resolved {
			return []manager.Adapter{a}, nil
		}
	}
	return nil, fmt.Errorf("manager %q not available", managerFlag)
}

// applyFirstSupported runs act against targets in precedence order until one
// succeeds, prompting consent before each attempt. Managers reporting
// ErrNotSupported are skipped; any other error aborts the walk. The first
// success prints a confirmation line and stops.
func applyFirstSupported(cmd *cobra.Command, pkg string, targets []manager.Adapter, verbTitle, verbPast string, act func(manager.Adapter) error) error {
	verb := strings.ToLower(verbTitle)
	for _, a := range targets {
		if err := requireConsent(cmd, fmt.Sprintf("%s %s via %s", verbTitle, pkg, a.Name())); err != nil {
			return handleConsent(err)
		}
		err := act(a)
		if err != nil {
			if errors.Is(err, manager.ErrNotSupported) {
				continue
			}
			return fmt.Errorf("failed to %s %s via %s: %w", verb, pkg, a.Name(), err)
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s %s via %s\n", verbPast, pkg, a.Name())
		return nil
	}
	return fmt.Errorf("no manager supports %s for package %s", verb, pkg)
}

// collectHeld gathers held packages from every target, formatted as
// "<pkg> (<manager>)". Unsupported managers are skipped silently; other
// failures print a warning and processing continues.
func collectHeld(cmd *cobra.Command, targets []manager.Adapter) []string {
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
		for _, p := range pkgs {
			results = append(results, fmt.Sprintf("%s (%s)", p, a.Name()))
		}
	}
	return results
}

// printHeld lists held packages, one per line, or prints a friendly message
// when none exist.
func printHeld(out io.Writer, results []string) {
	if len(results) == 0 {
		_, _ = fmt.Fprintln(out, "no packages held")
		return
	}
	for _, r := range results {
		_, _ = fmt.Fprintln(out, r)
	}
}
