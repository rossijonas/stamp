package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
)

func newTapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tap <name>",
		Short: "Add a Homebrew tap (alias for repo add -m brew)",
		Example: `  # add a homebrew tap (alias form)
  stamp tap homebrew/cask

  # equivalent canonical command
  stamp repo add homebrew/cask -m brew`,
		Long: `Add a third-party Homebrew tap repository.
Equivalent to "stamp repo add <name> -m brew".`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			name := args[0]
			for _, a := range app.adapters {
				if a.Name() == "brew" {
					if err := requireConsent(cmd, fmt.Sprintf("Tap %s via brew", name)); err != nil {
						return handleConsent(err)
					}
					if err := a.AddRepo(manager.WithYes(cmd.Context()), name, ""); err != nil {
						return fmt.Errorf("failed to tap %s: %w", name, err)
					}
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "added tap %s via brew\n", name)
					return nil
				}
			}
			return fmt.Errorf("brew is not available")
		},
	}
}

func newUntapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "untap <name>",
		Short: "Remove a Homebrew tap (alias for repo remove -m brew)",
		Example: `  # remove a homebrew tap (alias form)
  stamp untap homebrew/cask

  # equivalent canonical command
  stamp repo remove homebrew/cask -m brew`,
		Long: `Remove a third-party Homebrew tap repository.
Equivalent to "stamp repo remove <name> -m brew".`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			name := args[0]
			for _, a := range app.adapters {
				if a.Name() == "brew" {
					if err := requireConsent(cmd, fmt.Sprintf("Untap %s via brew", name)); err != nil {
						return handleConsent(err)
					}
					if err := a.RemoveRepo(manager.WithYes(cmd.Context()), name); err != nil {
						return fmt.Errorf("failed to untap %s: %w", name, err)
					}
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "removed tap %s via brew\n", name)
					return nil
				}
			}
			return fmt.Errorf("brew is not available")
		},
	}
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
			for _, a := range app.adapters {
				if a.Name() == "brew" {
					repos, err := a.ListRepos(cmd.Context())
					if err != nil {
						return fmt.Errorf("failed to list taps: %w", err)
					}
					if len(repos) == 0 {
						_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no taps added")
						return nil
					}
					for _, r := range repos {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", r.Name)
					}
					return nil
				}
			}
			return fmt.Errorf("brew is not available")
		},
	}
}
