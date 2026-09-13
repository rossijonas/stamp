package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

func newTapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tap <name>",
		Short: "Add a Homebrew tap (alias for repo add -m brew)",
		Example: `  # add a homebrew tap (alias form)
  stamp tap homebrew/cask

  # equivalent canonical command
  stamp repo add homebrew/cask -m brew

  # the tap is recorded in the manifest and re-added by 'stamp restore'`,
		Long: `Add a third-party Homebrew tap repository.
Equivalent to "stamp repo add <name> -m brew".`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTap(cmd, args[0])
		},
	}
}

// runTap adds a Homebrew tap and records it in the manifest, mirroring the
// repo add command so taps added here are listed and restored.
func runTap(cmd *cobra.Command, name string) error {
	app := appFromCtx(cmd)
	if app.manifestErr != nil {
		return app.manifestErr
	}
	adapter := brewAdapter(app.adapters)
	if adapter == nil {
		return fmt.Errorf("brew is not available")
	}
	if err := requireConsent(cmd, fmt.Sprintf("Tap %s via brew", name)); err != nil {
		return handleConsent(err)
	}
	if err := adapter.AddRepo(manager.WithYes(cmd.Context()), name, ""); err != nil {
		// AddRepo already contextualizes with the tap name.
		return err
	}
	app.manifest.AddRepository(manifest.Repository{
		Name:    name,
		Manager: "brew",
		Origin:  manifest.OriginStamped,
	})
	if err := app.saveManifest(); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "added tap %s via brew\n", name)
	return nil
}

// brewAdapter returns the brew adapter, or nil when brew is unavailable.
func brewAdapter(adapters []manager.Adapter) manager.Adapter {
	for _, a := range adapters {
		if a.Name() == "brew" {
			return a
		}
	}
	return nil
}

func newUntapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "untap <name>",
		Short: "Remove a Homebrew tap (alias for repo remove -m brew)",
		Example: `  # remove a homebrew tap (alias form)
  stamp untap homebrew/cask

  # equivalent canonical command
  stamp repo remove homebrew/cask -m brew

  # removing a tap also drops its manifest entry`,
		Long: `Remove a third-party Homebrew tap repository.
Equivalent to "stamp repo remove <name> -m brew".`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUntap(cmd, args[0])
		},
	}
}

// runUntap removes a Homebrew tap and drops it from the manifest, mirroring
// the repo remove command.
func runUntap(cmd *cobra.Command, name string) error {
	app := appFromCtx(cmd)
	if app.manifestErr != nil {
		return app.manifestErr
	}
	adapter := brewAdapter(app.adapters)
	if adapter == nil {
		return fmt.Errorf("brew is not available")
	}
	if err := requireConsent(cmd, fmt.Sprintf("Untap %s via brew", name)); err != nil {
		return handleConsent(err)
	}
	if err := adapter.RemoveRepo(manager.WithYes(cmd.Context()), name); err != nil {
		// RemoveRepo already contextualizes with the tap name.
		return err
	}
	app.manifest.RemoveRepository(name, "brew")
	if err := app.saveManifest(); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "removed tap %s via brew\n", name)
	return nil
}

// listBrewTaps prints all Homebrew tap repositories. Returns an error
// when the brew adapter is not available.
func listBrewTaps(ctx context.Context, adapters []manager.Adapter, w io.Writer) error {
	for _, a := range adapters {
		if a.Name() == "brew" {
			repos, err := a.ListRepos(ctx)
			if err != nil {
				return fmt.Errorf("failed to list taps: %w", err)
			}
			if len(repos) == 0 {
				_, _ = fmt.Fprintln(w, "no taps added")
				return nil
			}
			for _, r := range repos {
				_, _ = fmt.Fprintf(w, "%s\n", r.Name)
			}
			return nil
		}
	}
	return fmt.Errorf("brew is not available")
}

func newTapsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "taps",
		Short: "List Homebrew taps (alias for repo list -m brew)",
		Example: `  # list homebrew taps (alias form)
  stamp taps

  # equivalent canonical command
  stamp repo list -m brew`,
		Long: `List all installed Homebrew tap repositories.
Equivalent to "stamp repo list -m brew".`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)
			return listBrewTaps(cmd.Context(), app.adapters, cmd.OutOrStdout())
		},
	}
}
