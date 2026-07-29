package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newTapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tap <name>",
		Short: "Add a Homebrew tap (alias for repo add -m brew)",
		Example: `  # add a homebrew tap (equivalent to repo add <name> -m brew)
  stamp tap homebrew/cask`,
		Long: `Add a third-party Homebrew tap repository.
Equivalent to "stamp repo add <name> -m brew".`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			name := args[0]
			for _, a := range app.adapters {
				if a.Name() == "brew" {
					if err := a.AddRepo(cmd.Context(), name, ""); err != nil {
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
		Example: `  # remove a homebrew tap (equivalent to repo remove <name> -m brew)
  stamp untap homebrew/cask`,
		Long: `Remove a third-party Homebrew tap repository.
Equivalent to "stamp repo remove <name> -m brew".`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			name := args[0]
			for _, a := range app.adapters {
				if a.Name() == "brew" {
					if err := a.RemoveRepo(cmd.Context(), name); err != nil {
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
		Example: `  # list all homebrew taps (equivalent to repo list -m brew)
  stamp taps`,
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
