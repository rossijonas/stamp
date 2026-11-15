package cli

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

func newUntrackCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:   "untrack <package>",
		Short: "Remove a package from the manifest without touching the system",
		Long: `Drop a package from the manifest without running the native package manager.

Use this to clear a stale or mis-attributed entry (for example a package
recorded under a name the native manager no longer resolves). The installed
package, if any, is left untouched. Normal uninstalls should use 'stamp remove'.

When the same name is tracked under more than one manager, pass -m to choose
the entry. -y skips the confirmation prompt; without it, non-interactive runs
fail closed.`,
		Example: `  # drop an entry the native manager cannot resolve
  stamp untrack yarn -m dnf

  # skip the confirmation prompt
  stamp untrack yarn -m dnf -y`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
			}

			pkg, err := selectUntrackedPackage(app.manifest.Packages, args[0], managerFlag)
			if err != nil {
				return err
			}

			proceed, err := confirmUntrack(cmd, app, pkg)
			if err != nil {
				return err
			}
			if !proceed {
				return nil
			}

			if !app.manifest.RemovePackage(pkg.Name, pkg.Manager) {
				return fmt.Errorf("package %q is not tracked under %q", pkg.Name, pkg.Manager)
			}
			if err := app.saveManifest(); err != nil {
				return fmt.Errorf("failed to save manifest: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "untracked %s (%s)\n", pkg.Name, pkg.Manager)
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "manager the package is tracked under")
	return cmd
}

// confirmUntrack gates a manifest-only removal: -y proceeds, an interactive
// decline stops cleanly, and a non-interactive run without -y fails closed.
// proceed=false with a nil error means the user declined; the caller stops
// without mutating and exits 0.
func confirmUntrack(cmd *cobra.Command, app *AppContext, pkg manifest.Package) (proceed bool, err error) {
	if app.yes {
		return true, nil
	}
	msg := fmt.Sprintf("Untrack %s (%s)? [y/N]: ", pkg.Name, pkg.Manager)
	if err := consentPrompt(cmd.Context(), cmd.ErrOrStderr(), cmd.InOrStdin(), msg); err != nil {
		if errors.Is(err, errNonInteractive) {
			return false, fmt.Errorf("%w (untrack %s via %s)", errNonInteractive, pkg.Name, pkg.Manager)
		}
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "aborted")
		return false, handleConsent(err)
	}
	return true, nil
}

// selectUntrackedPackage resolves which manifest entry to drop. With -m the
// manager must match; without it the name must be unambiguous. The lookup is
// manifest-only: no adapter needs to be installed.
func selectUntrackedPackage(pkgs []manifest.Package, name, managerFlag string) (manifest.Package, error) {
	if managerFlag != "" {
		want := manager.ResolveManager(managerFlag)
		for _, p := range pkgs {
			if p.Name == name && p.Manager == want {
				return p, nil
			}
		}
		return manifest.Package{}, fmt.Errorf("package %q is not tracked under %q", name, want)
	}

	var matches []manifest.Package
	for _, p := range pkgs {
		if p.Name == name {
			matches = append(matches, p)
		}
	}
	switch len(matches) {
	case 0:
		return manifest.Package{}, fmt.Errorf("package %q is not tracked", name)
	case 1:
		return matches[0], nil
	default:
		managers := make([]string, len(matches))
		for i, m := range matches {
			managers[i] = m.Manager
		}
		slices.Sort(managers)
		return manifest.Package{}, catErr(ErrUsage, "package %q is tracked under multiple managers (%s); use -m", name, strings.Join(managers, ", "))
	}
}
